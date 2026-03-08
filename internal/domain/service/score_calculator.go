package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/types"
)

type ScoreCalculator struct{}

func NewScoreCalculator() *ScoreCalculator {
	return &ScoreCalculator{}
}

func (s *ScoreCalculator) Calculate(q *entity.Question, answer types.JSON) float64 {
	if len(answer) == 0 || string(answer) == "null" {
		return 0
	}
	switch q.QuestionType {
	case entity.QuestionTypePG, entity.QuestionTypeBenarSalah:
		return s.scorePG(q, answer)
	case entity.QuestionTypePGK:
		return s.scorePGK(q, answer)
	case entity.QuestionTypeMenjodohkan:
		return s.scoreMenjodohkan(q, answer)
	case entity.QuestionTypeIsian:
		return s.scoreIsian(q, answer)
	case entity.QuestionTypeMatrix:
		return s.scoreMatrix(q, answer)
	case entity.QuestionTypeEsai:
		return 0 // requires manual/AI grading
	}
	return 0
}

// --- Option / Answer helper structs ---

// pgOption represents a single PG/BS/PGK option from the JSONB options array.
type pgOption struct {
	Text      string   `json:"text"`
	Weight    *float64 `json:"weight"`    // pointer so we can detect absence
	IsCorrect bool     `json:"is_correct"`
}

// matchingOption represents one prompt-answer pair in Menjodohkan options.
type matchingOption struct {
	Prompt string `json:"prompt"`
	Answer string `json:"answer"`
}

// shortAnswerVariant represents one accepted answer for Isian Singkat.
type shortAnswerVariant struct {
	Text   string  `json:"text"`
	Weight float64 `json:"weight"`
}

// matrixData represents the JSONB structure stored in Options for Matrix questions.
type matrixData struct {
	Columns []string    `json:"columns"`
	Rows    []matrixRow `json:"rows"`
}

type matrixRow struct {
	StatementText      string `json:"statement_text"`
	CorrectColumnIndex int    `json:"correct_column_index"`
}

// --- Scoring methods ---

// scorePG handles PG (Multiple Choice) and Benar/Salah (True/False).
//
// Answer formats supported:
//   {"option_index": 2}          — index into the options array
//   {"option_id": 2}             — treated as index (legacy compat)
//   {"option_id": "a"}           — legacy string match against CorrectAnswer
//
// Scoring:
//   If the selected option has a weight field  -> score = question.Score * weight
//   If weight is absent but is_correct == true  -> score = question.Score
//   Otherwise                                   -> 0
func (s *ScoreCalculator) scorePG(q *entity.Question, answer types.JSON) float64 {
	// Parse the student answer — supports both option_index (int) and option_id (int or string).
	idx, strID, ok := parseSingleOptionAnswer(answer)
	if !ok {
		logger.Log.Warnf("scorePG: question %d: failed to parse single option answer: %s", q.ID, string(answer))
		return 0
	}

	// Try index-based lookup first (preferred path).
	if idx >= 0 {
		return s.scorePGByIndex(q, idx)
	}

	// Fallback: legacy string-based comparison against CorrectAnswer.
	if strID != "" {
		var correct string
		if err := json.Unmarshal(q.CorrectAnswer, &correct); err != nil {
			return 0
		}
		if strID == correct {
			return q.Score
		}
	}
	return 0
}

// scorePGByIndex scores a PG answer by option index, using weight when available.
func (s *ScoreCalculator) scorePGByIndex(q *entity.Question, idx int) float64 {
	var opts []pgOption
	if err := json.Unmarshal(q.Options, &opts); err != nil {
		logger.Log.Warnf("scorePGByIndex: question %d: failed to unmarshal options: %v", q.ID, err)
		return 0
	}
	if idx < 0 || idx >= len(opts) {
		logger.Log.Warnf("scorePGByIndex: question %d: option index %d out of range (total %d)", q.ID, idx, len(opts))
		return 0
	}

	selected := opts[idx]

	// If weight is explicitly set, use it.
	if selected.Weight != nil {
		w := *selected.Weight
		if w < 0 {
			w = 0 // clamp negative to 0 for single-choice
		}
		return q.Score * w
	}

	// Fallback: boolean is_correct.
	if selected.IsCorrect {
		return q.Score
	}
	return 0
}

// scorePGK handles PGK (Complex Multiple Choice) with per-option weights.
//
// Answer format:
//   {"option_indices": [0, 2]}   — preferred
//   {"option_ids": [0, 2]}       — legacy (numeric array)
//   {"option_ids": ["a","c"]}    — legacy string array
//
// Scoring:
//   sum_selected_weights / sum_correct_weights * question.Score
//   Clamped to [0, question.Score].
func (s *ScoreCalculator) scorePGK(q *entity.Question, answer types.JSON) float64 {
	indices, strIDs := parseMultiOptionAnswer(answer)

	// --- Index-based path (preferred) ---
	if len(indices) > 0 {
		return s.scorePGKByIndices(q, indices)
	}

	// --- Legacy string-based path ---
	if len(strIDs) > 0 {
		return s.scorePGKByStringIDs(q, strIDs)
	}

	logger.Log.Warnf("scorePGK: question %d: failed to parse multi-option answer: %s", q.ID, string(answer))
	return 0
}

// scorePGKByIndices scores PGK using option indices and weights from Options JSONB.
func (s *ScoreCalculator) scorePGKByIndices(q *entity.Question, indices []int) float64 {
	var opts []pgOption
	if err := json.Unmarshal(q.Options, &opts); err != nil {
		logger.Log.Warnf("scorePGKByIndices: question %d: failed to unmarshal options: %v", q.ID, err)
		return 0
	}

	// Calculate sum of correct weights (denominator).
	var sumCorrectWeights float64
	for i := range opts {
		w := optionWeight(&opts[i])
		if w > 0 {
			sumCorrectWeights += w
		}
	}
	if sumCorrectWeights == 0 {
		return 0
	}

	// Sum weights of selected options (can include negative weights for wrong picks).
	var sumSelectedWeights float64
	for _, idx := range indices {
		if idx < 0 || idx >= len(opts) {
			continue
		}
		sumSelectedWeights += optionWeight(&opts[idx])
	}

	// Clamp ratio to [0, 1].
	ratio := sumSelectedWeights / sumCorrectWeights
	ratio = math.Max(0, math.Min(1, ratio))
	return q.Score * ratio
}

// scorePGKByStringIDs is the legacy fallback using string option_ids and CorrectAnswer array.
func (s *ScoreCalculator) scorePGKByStringIDs(q *entity.Question, selectedIDs []string) float64 {
	var correct []string
	if err := json.Unmarshal(q.CorrectAnswer, &correct); err != nil {
		return 0
	}
	correctSet := make(map[string]bool, len(correct))
	for _, id := range correct {
		correctSet[id] = true
	}

	// Try to get per-option scores from Options JSONB.
	var opts []struct {
		ID    string  `json:"id"`
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal(q.Options, &opts); err != nil {
		// Fallback: equal weight per correct option.
		if len(correct) == 0 {
			return 0
		}
		matched := 0
		for _, id := range selectedIDs {
			if correctSet[id] {
				matched++
			}
		}
		return q.Score * float64(matched) / float64(len(correct))
	}

	optScores := make(map[string]float64, len(opts))
	for _, opt := range opts {
		optScores[opt.ID] = opt.Score
	}
	var total float64
	for _, id := range selectedIDs {
		if correctSet[id] {
			total += optScores[id]
		}
	}
	return total
}

// scoreMenjodohkan handles Matching questions with partial credit.
//
// Options JSONB: [{"prompt":"A","answer":"1"}, {"prompt":"B","answer":"2"}]
// Answer:        {"pairs": {"A":"1", "B":"2"}}
//
// Score = question.Score * (correct_pairs / total_pairs)
func (s *ScoreCalculator) scoreMenjodohkan(q *entity.Question, answer types.JSON) float64 {
	var ans struct {
		Pairs map[string]string `json:"pairs"`
	}
	if err := json.Unmarshal(answer, &ans); err != nil {
		logger.Log.Warnf("scoreMenjodohkan: question %d: failed to parse answer pairs: %v", q.ID, err)
		return 0
	}
	if len(ans.Pairs) == 0 {
		return 0
	}

	// Try structured options first.
	var opts []matchingOption
	if err := json.Unmarshal(q.Options, &opts); err == nil && len(opts) > 0 {
		totalPairs := len(opts)
		correctCount := 0

		// Build key map: prompt -> answer (case-insensitive trimmed).
		keyMap := make(map[string]string, totalPairs)
		for _, opt := range opts {
			keyMap[strings.TrimSpace(strings.ToLower(opt.Prompt))] = strings.TrimSpace(strings.ToLower(opt.Answer))
		}

		for prompt, userAnswer := range ans.Pairs {
			pNorm := strings.TrimSpace(strings.ToLower(prompt))
			uNorm := strings.TrimSpace(strings.ToLower(userAnswer))
			if correctAnswer, exists := keyMap[pNorm]; exists && uNorm == correctAnswer {
				correctCount++
			}
		}

		if totalPairs == 0 {
			return 0
		}
		return q.Score * (float64(correctCount) / float64(totalPairs))
	}

	// Fallback: CorrectAnswer is a map[string]string.
	var correct map[string]string
	if err := json.Unmarshal(q.CorrectAnswer, &correct); err != nil {
		return 0
	}
	if len(correct) == 0 {
		return 0
	}
	var matched float64
	for k, v := range ans.Pairs {
		if correct[k] == v {
			matched++
		}
	}
	return q.Score * (matched / float64(len(correct)))
}

// scoreIsian handles Isian Singkat (Short Answer) with weighted answer variants.
//
// CorrectAnswer JSONB: [{"text":"jawaban1","weight":1.0}, {"text":"jawaban2","weight":0.8}]
//   OR legacy Options JSONB: {"accepted_answers": ["jawaban1","jawaban2"]}
//
// Answer: {"text": "jawaban"}
//
// Scoring: Match against all variants (case-insensitive, trimmed). Use the highest weight.
func (s *ScoreCalculator) scoreIsian(q *entity.Question, answer types.JSON) float64 {
	var ans struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(answer, &ans); err != nil {
		logger.Log.Warnf("scoreIsian: question %d: failed to parse answer text: %v", q.ID, err)
		return 0
	}
	ansNorm := strings.TrimSpace(strings.ToLower(ans.Text))
	if ansNorm == "" {
		return 0
	}

	// Path 1: Weighted variants from CorrectAnswer JSONB.
	var variants []shortAnswerVariant
	if err := json.Unmarshal(q.CorrectAnswer, &variants); err == nil && len(variants) > 0 {
		bestWeight := 0.0
		found := false
		for _, v := range variants {
			vNorm := strings.TrimSpace(strings.ToLower(v.Text))
			if vNorm == ansNorm {
				w := v.Weight
				if w == 0 && !found {
					// If weight is zero/unset but text matches, treat as full credit.
					w = 1.0
				}
				if w > bestWeight {
					bestWeight = w
					found = true
				}
			}
		}
		if found {
			return q.Score * math.Max(0, math.Min(1, bestWeight))
		}
		// Variants existed but no match — score 0.
		return 0
	}

	// Path 2: Legacy — simple accepted_answers list from Options JSONB.
	var opts struct {
		AcceptedAnswers []string `json:"accepted_answers"`
	}
	if err := json.Unmarshal(q.Options, &opts); err != nil {
		return 0
	}
	for _, accepted := range opts.AcceptedAnswers {
		if ansNorm == strings.TrimSpace(strings.ToLower(accepted)) {
			return q.Score
		}
	}
	return 0
}

// scoreMatrix handles Matrix questions with partial credit per row.
//
// Options JSONB: {"columns":["Col1","Col2"],"rows":[{"statement_text":"...","correct_column_index":0}]}
// Answer:        {"answers":{"0":1,"1":0}}  (row_index -> selected_column_index)
//
// Score = question.Score * (correct_rows / total_rows)
func (s *ScoreCalculator) scoreMatrix(q *entity.Question, answer types.JSON) float64 {
	// Parse answer — supports both int and string values for column indices.
	var rawAns struct {
		Answers map[string]json.RawMessage `json:"answers"`
	}
	if err := json.Unmarshal(answer, &rawAns); err != nil {
		logger.Log.Warnf("scoreMatrix: question %d: failed to parse answer map: %v", q.ID, err)
		return 0
	}
	if len(rawAns.Answers) == 0 {
		return 0
	}

	// Try structured matrix data from Options.
	var md matrixData
	if err := json.Unmarshal(q.Options, &md); err == nil && len(md.Rows) > 0 {
		totalRows := len(md.Rows)
		correctCount := 0

		for rowIdx, row := range md.Rows {
			rowKey := fmt.Sprintf("%d", rowIdx)
			rawVal, exists := rawAns.Answers[rowKey]
			if !exists {
				continue
			}
			userCol, ok := parseIntFromRaw(rawVal)
			if !ok {
				logger.Log.Warnf("scoreMatrix: question %d row %d: failed to parse answer value: %s", q.ID, rowIdx, string(rawVal))
				continue
			}
			if userCol < 0 || userCol >= len(md.Columns) {
				logger.Log.Warnf("scoreMatrix: question %d row %d: column index %d out of range (total %d)", q.ID, rowIdx, userCol, len(md.Columns))
				continue
			}
			if row.CorrectColumnIndex < 0 || row.CorrectColumnIndex >= len(md.Columns) {
				logger.Log.Warnf("scoreMatrix: question %d row %d: correct_column_index %d out of range (total %d)", q.ID, rowIdx, row.CorrectColumnIndex, len(md.Columns))
				continue
			}
			if userCol == row.CorrectColumnIndex {
				correctCount++
			}
		}

		if totalRows == 0 {
			return 0
		}
		return q.Score * (float64(correctCount) / float64(totalRows))
	}

	// Fallback: CorrectAnswer is map[string]string.
	var correct map[string]string
	if err := json.Unmarshal(q.CorrectAnswer, &correct); err != nil {
		return 0
	}
	if len(correct) == 0 {
		return 0
	}
	var matched float64
	for k, rawVal := range rawAns.Answers {
		// Normalize the raw value to a string for comparison.
		userVal := strings.Trim(strings.TrimSpace(string(rawVal)), "\"")
		if correct[k] == userVal {
			matched++
		}
	}
	return q.Score * (matched / float64(len(correct)))
}

// --- Helper functions ---

// optionWeight extracts the effective weight from a pgOption.
// If Weight is set, returns it. If Weight is nil, returns 1.0 for correct, 0 for incorrect.
func optionWeight(opt *pgOption) float64 {
	if opt.Weight != nil {
		return *opt.Weight
	}
	if opt.IsCorrect {
		return 1.0
	}
	return 0
}

// parseSingleOptionAnswer parses a PG/BS answer.
// Returns (index, stringID, ok). At most one of index/stringID is meaningful.
func parseSingleOptionAnswer(answer types.JSON) (int, string, bool) {
	// Try option_index first (int).
	var byIndex struct {
		OptionIndex *int `json:"option_index"`
	}
	if err := json.Unmarshal(answer, &byIndex); err == nil && byIndex.OptionIndex != nil {
		return *byIndex.OptionIndex, "", true
	}

	// Try option_id as int.
	var byIDInt struct {
		OptionID *int `json:"option_id"`
	}
	if err := json.Unmarshal(answer, &byIDInt); err == nil && byIDInt.OptionID != nil {
		return *byIDInt.OptionID, "", true
	}

	// Try option_id as string.
	var byIDStr struct {
		OptionID string `json:"option_id"`
	}
	if err := json.Unmarshal(answer, &byIDStr); err == nil && byIDStr.OptionID != "" {
		return -1, byIDStr.OptionID, true
	}

	return -1, "", false
}

// parseMultiOptionAnswer parses a PGK answer.
// Returns (indices, stringIDs). At most one slice is non-empty.
func parseMultiOptionAnswer(answer types.JSON) ([]int, []string) {
	// Try option_indices first.
	var byIndices struct {
		OptionIndices []int `json:"option_indices"`
	}
	if err := json.Unmarshal(answer, &byIndices); err == nil && len(byIndices.OptionIndices) > 0 {
		return byIndices.OptionIndices, nil
	}

	// Try option_ids as int array.
	var byIDsInt struct {
		OptionIDs []int `json:"option_ids"`
	}
	if err := json.Unmarshal(answer, &byIDsInt); err == nil && len(byIDsInt.OptionIDs) > 0 {
		return byIDsInt.OptionIDs, nil
	}

	// Try option_ids as string array.
	var byIDsStr struct {
		OptionIDs []string `json:"option_ids"`
	}
	if err := json.Unmarshal(answer, &byIDsStr); err == nil && len(byIDsStr.OptionIDs) > 0 {
		return nil, byIDsStr.OptionIDs
	}

	return nil, nil
}

// parseIntFromRaw extracts an integer from a json.RawMessage that could be a number or a quoted string.
// Returns the parsed integer and true on success, or -1 and false on failure.
func parseIntFromRaw(raw json.RawMessage) (int, bool) {
	// Reject null, empty, or non-value types early.
	if len(raw) == 0 || string(raw) == "null" {
		return -1, false
	}

	var i int
	if err := json.Unmarshal(raw, &i); err == nil {
		return i, true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		var n int
		if _, err := fmt.Sscanf(s, "%d", &n); err == nil {
			return n, true
		}
	}
	return -1, false // sentinel: will not match any valid column index (0+)
}
