package service

import (
	"encoding/json"
	"testing"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/types"
)

func init() {
	logger.Init("test")
}

// --- helpers ---

func toJSON(v interface{}) types.JSON {
	b, _ := json.Marshal(v)
	return types.JSON(b)
}

func ptrFloat(f float64) *float64 { return &f }

func makeQuestion(qType string, score float64, options, correctAnswer interface{}) *entity.Question {
	q := &entity.Question{
		QuestionType: qType,
		Score:        score,
	}
	if options != nil {
		q.Options = toJSON(options)
	}
	if correctAnswer != nil {
		q.CorrectAnswer = toJSON(correctAnswer)
	}
	return q
}

// ─── PG (Multiple Choice) ──────────────────────────────────────

func TestScorePG(t *testing.T) {
	opts := []pgOption{
		{Text: "A", IsCorrect: false},
		{Text: "B", IsCorrect: true},
		{Text: "C", IsCorrect: false},
	}
	q := makeQuestion(entity.QuestionTypePG, 1.0, opts, nil)
	calc := NewScoreCalculator()

	tests := []struct {
		name   string
		answer interface{}
		want   float64
	}{
		{"correct answer", map[string]int{"option_index": 1}, 1.0},
		{"wrong answer", map[string]int{"option_index": 0}, 0},
		{"another wrong", map[string]int{"option_index": 2}, 0},
		{"empty object", map[string]interface{}{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calc.Calculate(q, toJSON(tt.answer))
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScorePG_WithWeight(t *testing.T) {
	opts := []pgOption{
		{Text: "A", Weight: ptrFloat(0)},
		{Text: "B", Weight: ptrFloat(1.0)},
		{Text: "C", Weight: ptrFloat(0.5)},
	}
	q := makeQuestion(entity.QuestionTypePG, 2.0, opts, nil)
	calc := NewScoreCalculator()

	tests := []struct {
		name   string
		answer interface{}
		want   float64
	}{
		{"full weight", map[string]int{"option_index": 1}, 2.0},
		{"partial weight", map[string]int{"option_index": 2}, 1.0},
		{"zero weight", map[string]int{"option_index": 0}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calc.Calculate(q, toJSON(tt.answer))
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ─── PGK (Complex Multiple Choice) ─────────────────────────────

func TestScorePGK(t *testing.T) {
	opts := []pgOption{
		{Text: "A", Weight: ptrFloat(1.0)},
		{Text: "B", Weight: ptrFloat(1.0)},
		{Text: "C", Weight: ptrFloat(0)},
		{Text: "D", Weight: ptrFloat(-0.5)},
	}
	q := makeQuestion(entity.QuestionTypePGK, 4.0, opts, nil)
	calc := NewScoreCalculator()

	tests := []struct {
		name   string
		answer interface{}
		want   float64
	}{
		{"all correct selected", map[string][]int{"option_indices": {0, 1}}, 4.0},
		{"partial correct", map[string][]int{"option_indices": {0}}, 2.0},
		{"all wrong", map[string][]int{"option_indices": {2, 3}}, 0},
		{"mix correct and negative", map[string][]int{"option_indices": {0, 3}}, 1.0}, // (1.0 + -0.5)/2.0 = 0.25 * 4 = 1.0
		{"empty selection", map[string][]int{"option_indices": {}}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calc.Calculate(q, toJSON(tt.answer))
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ─── Benar/Salah (True/False) ──────────────────────────────────

func TestScoreBenarSalah(t *testing.T) {
	opts := []pgOption{
		{Text: "Benar", IsCorrect: true},
		{Text: "Salah", IsCorrect: false},
	}
	q := makeQuestion(entity.QuestionTypeBenarSalah, 1.0, opts, nil)
	calc := NewScoreCalculator()

	tests := []struct {
		name   string
		answer interface{}
		want   float64
	}{
		{"correct (benar)", map[string]int{"option_index": 0}, 1.0},
		{"wrong (salah)", map[string]int{"option_index": 1}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calc.Calculate(q, toJSON(tt.answer))
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ─── Menjodohkan (Matching) ────────────────────────────────────

func TestScoreMenjodohkan(t *testing.T) {
	opts := []matchingOption{
		{Prompt: "Ibukota Indonesia", Answer: "Jakarta"},
		{Prompt: "Ibukota Jepang", Answer: "Tokyo"},
		{Prompt: "Ibukota Prancis", Answer: "Paris"},
	}
	q := makeQuestion(entity.QuestionTypeMenjodohkan, 3.0, opts, nil)
	calc := NewScoreCalculator()

	tests := []struct {
		name   string
		pairs  map[string]string
		want   float64
	}{
		{"all correct", map[string]string{"Ibukota Indonesia": "Jakarta", "Ibukota Jepang": "Tokyo", "Ibukota Prancis": "Paris"}, 3.0},
		{"some correct", map[string]string{"Ibukota Indonesia": "Jakarta", "Ibukota Jepang": "Paris", "Ibukota Prancis": "Tokyo"}, 1.0},
		{"none correct", map[string]string{"Ibukota Indonesia": "Tokyo", "Ibukota Jepang": "Paris", "Ibukota Prancis": "Jakarta"}, 0},
		{"case insensitive", map[string]string{"ibukota indonesia": "jakarta", "ibukota jepang": "tokyo", "ibukota prancis": "paris"}, 3.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answer := toJSON(map[string]interface{}{"pairs": tt.pairs})
			got := calc.Calculate(q, answer)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ─── Isian Singkat (Short Answer) ──────────────────────────────

func TestScoreIsian(t *testing.T) {
	variants := []shortAnswerVariant{
		{Text: "Jakarta", Weight: 1.0},
		{Text: "DKI Jakarta", Weight: 0.8},
	}
	q := makeQuestion(entity.QuestionTypeIsian, 2.0, nil, variants)
	calc := NewScoreCalculator()

	tests := []struct {
		name string
		text string
		want float64
	}{
		{"exact match full weight", "Jakarta", 2.0},
		{"case insensitive", "jakarta", 2.0},
		{"partial weight variant", "DKI Jakarta", 1.6},
		{"wrong answer", "Bandung", 0},
		{"empty answer", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answer := toJSON(map[string]string{"text": tt.text})
			got := calc.Calculate(q, answer)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ─── Matrix ────────────────────────────────────────────────────

func TestScoreMatrix(t *testing.T) {
	md := matrixData{
		Columns: []string{"Setuju", "Netral", "Tidak Setuju"},
		Rows: []matrixRow{
			{StatementText: "Pernyataan 1", CorrectColumnIndex: 0},
			{StatementText: "Pernyataan 2", CorrectColumnIndex: 2},
			{StatementText: "Pernyataan 3", CorrectColumnIndex: 1},
		},
	}
	q := makeQuestion(entity.QuestionTypeMatrix, 3.0, md, nil)
	calc := NewScoreCalculator()

	tests := []struct {
		name    string
		answers map[string]int
		want    float64
	}{
		{"all correct", map[string]int{"0": 0, "1": 2, "2": 1}, 3.0},
		{"partial (2/3)", map[string]int{"0": 0, "1": 2, "2": 0}, 2.0},
		{"all wrong", map[string]int{"0": 1, "1": 0, "2": 2}, 0},
		{"partial (1/3)", map[string]int{"0": 0, "1": 0, "2": 0}, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answer := toJSON(map[string]interface{}{"answers": tt.answers})
			got := calc.Calculate(q, answer)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ─── Esai ──────────────────────────────────────────────────────

func TestScoreEsai(t *testing.T) {
	q := makeQuestion(entity.QuestionTypeEsai, 10.0, nil, nil)
	calc := NewScoreCalculator()

	answer := toJSON(map[string]string{"text": "This is a long essay answer about the topic."})
	got := calc.Calculate(q, answer)
	if got != 0 {
		t.Errorf("esai should return 0 (manual grading), got %v", got)
	}
}

// ─── Empty/Nil/Null Answer ─────────────────────────────────────

func TestScoreEmptyAnswer(t *testing.T) {
	q := makeQuestion(entity.QuestionTypePG, 1.0,
		[]pgOption{{Text: "A", IsCorrect: true}}, nil)
	calc := NewScoreCalculator()

	tests := []struct {
		name   string
		answer types.JSON
		want   float64
	}{
		{"nil answer", nil, 0},
		{"empty bytes", types.JSON{}, 0},
		{"null string", types.JSON("null"), 0},
		{"empty string json", types.JSON(`""`), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calc.Calculate(q, tt.answer)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ─── Edge Cases: Malformed JSON ────────────────────────────────

func TestScorePG_MalformedJSON(t *testing.T) {
	opts := []pgOption{
		{Text: "A", IsCorrect: true},
		{Text: "B", IsCorrect: false},
	}
	q := makeQuestion(entity.QuestionTypePG, 1.0, opts, nil)
	calc := NewScoreCalculator()

	tests := []struct {
		name   string
		answer types.JSON
		want   float64
	}{
		{"garbage bytes", types.JSON(`{not valid json`), 0},
		{"array instead of object", types.JSON(`[1,2,3]`), 0},
		{"number literal", types.JSON(`42`), 0},
		{"string literal", types.JSON(`"hello"`), 0},
		{"empty object", types.JSON(`{}`), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calc.Calculate(q, tt.answer)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScorePGK_MalformedJSON(t *testing.T) {
	opts := []pgOption{
		{Text: "A", Weight: ptrFloat(1.0)},
		{Text: "B", Weight: ptrFloat(0)},
	}
	q := makeQuestion(entity.QuestionTypePGK, 2.0, opts, nil)
	calc := NewScoreCalculator()

	tests := []struct {
		name   string
		answer types.JSON
		want   float64
	}{
		{"garbage JSON", types.JSON(`{broken`), 0},
		{"wrong key name", types.JSON(`{"selected":[0]}`), 0},
		{"empty object", types.JSON(`{}`), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calc.Calculate(q, tt.answer)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScoreMenjodohkan_MalformedJSON(t *testing.T) {
	opts := []matchingOption{
		{Prompt: "A", Answer: "1"},
	}
	q := makeQuestion(entity.QuestionTypeMenjodohkan, 1.0, opts, nil)
	calc := NewScoreCalculator()

	tests := []struct {
		name   string
		answer types.JSON
		want   float64
	}{
		{"garbage JSON", types.JSON(`not json`), 0},
		{"empty pairs", types.JSON(`{"pairs":{}}`), 0},
		{"missing pairs key", types.JSON(`{"answers":{"A":"1"}}`), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calc.Calculate(q, tt.answer)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScoreIsian_MalformedJSON(t *testing.T) {
	variants := []shortAnswerVariant{
		{Text: "Jakarta", Weight: 1.0},
	}
	q := makeQuestion(entity.QuestionTypeIsian, 2.0, nil, variants)
	calc := NewScoreCalculator()

	tests := []struct {
		name   string
		answer types.JSON
		want   float64
	}{
		{"garbage JSON", types.JSON(`{broken`), 0},
		{"wrong key", types.JSON(`{"answer":"Jakarta"}`), 0},
		{"number instead of text", types.JSON(`{"text":123}`), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calc.Calculate(q, tt.answer)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScoreMatrix_MalformedJSON(t *testing.T) {
	md := matrixData{
		Columns: []string{"A", "B"},
		Rows:    []matrixRow{{StatementText: "Q1", CorrectColumnIndex: 0}},
	}
	q := makeQuestion(entity.QuestionTypeMatrix, 1.0, md, nil)
	calc := NewScoreCalculator()

	tests := []struct {
		name   string
		answer types.JSON
		want   float64
	}{
		{"garbage JSON", types.JSON(`{broken`), 0},
		{"empty answers map", types.JSON(`{"answers":{}}`), 0},
		{"wrong key", types.JSON(`{"selections":{"0":0}}`), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calc.Calculate(q, tt.answer)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ─── Edge Cases: Out-of-Bounds Option Index ────────────────────

func TestScorePG_OutOfBoundsIndex(t *testing.T) {
	opts := []pgOption{
		{Text: "A", IsCorrect: true},
		{Text: "B", IsCorrect: false},
	}
	q := makeQuestion(entity.QuestionTypePG, 1.0, opts, nil)
	calc := NewScoreCalculator()

	tests := []struct {
		name   string
		answer interface{}
		want   float64
	}{
		{"index too high", map[string]int{"option_index": 99}, 0},
		{"negative index", map[string]int{"option_index": -1}, 0},
		{"index at boundary", map[string]int{"option_index": 2}, 0}, // len=2, valid is 0,1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calc.Calculate(q, toJSON(tt.answer))
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScorePGK_OutOfBoundsIndices(t *testing.T) {
	opts := []pgOption{
		{Text: "A", Weight: ptrFloat(1.0)},
		{Text: "B", Weight: ptrFloat(0)},
	}
	q := makeQuestion(entity.QuestionTypePGK, 2.0, opts, nil)
	calc := NewScoreCalculator()

	tests := []struct {
		name   string
		answer interface{}
		want   float64
	}{
		{"all out of bounds", map[string][]int{"option_indices": {5, 10}}, 0},
		{"mix valid and invalid", map[string][]int{"option_indices": {0, 99}}, 2.0}, // only idx 0 is valid (weight=1.0)
		{"negative indices", map[string][]int{"option_indices": {-1, -2}}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calc.Calculate(q, toJSON(tt.answer))
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// ─── Edge Cases: Empty Options Array ───────────────────────────

func TestScorePG_EmptyOptions(t *testing.T) {
	q := makeQuestion(entity.QuestionTypePG, 1.0, []pgOption{}, nil)
	calc := NewScoreCalculator()

	got := calc.Calculate(q, toJSON(map[string]int{"option_index": 0}))
	if got != 0 {
		t.Errorf("expected 0 for empty options, got %v", got)
	}
}

func TestScorePGK_EmptyOptions(t *testing.T) {
	q := makeQuestion(entity.QuestionTypePGK, 2.0, []pgOption{}, nil)
	calc := NewScoreCalculator()

	got := calc.Calculate(q, toJSON(map[string][]int{"option_indices": {0}}))
	if got != 0 {
		t.Errorf("expected 0 for empty options, got %v", got)
	}
}

func TestScoreMenjodohkan_EmptyOptions(t *testing.T) {
	q := makeQuestion(entity.QuestionTypeMenjodohkan, 1.0, []matchingOption{}, nil)
	calc := NewScoreCalculator()

	answer := toJSON(map[string]interface{}{"pairs": map[string]string{"A": "1"}})
	got := calc.Calculate(q, answer)
	if got != 0 {
		t.Errorf("expected 0 for empty options, got %v", got)
	}
}

func TestScoreMatrix_EmptyRows(t *testing.T) {
	md := matrixData{
		Columns: []string{"A", "B"},
		Rows:    []matrixRow{}, // no rows
	}
	q := makeQuestion(entity.QuestionTypeMatrix, 1.0, md, nil)
	calc := NewScoreCalculator()

	answer := toJSON(map[string]interface{}{"answers": map[string]int{"0": 0}})
	got := calc.Calculate(q, answer)
	if got != 0 {
		t.Errorf("expected 0 for empty matrix rows, got %v", got)
	}
}

// ─── Edge Cases: Nil/Null Answer Per Question Type ─────────────

func TestScoreNilAnswer_AllTypes(t *testing.T) {
	calc := NewScoreCalculator()

	pgOpts := []pgOption{{Text: "A", IsCorrect: true}}
	matchOpts := []matchingOption{{Prompt: "P", Answer: "A"}}
	matrixOpts := matrixData{
		Columns: []string{"C1"},
		Rows:    []matrixRow{{StatementText: "S1", CorrectColumnIndex: 0}},
	}
	isianVariants := []shortAnswerVariant{{Text: "answer", Weight: 1.0}}

	questions := map[string]*entity.Question{
		"pg":          makeQuestion(entity.QuestionTypePG, 1.0, pgOpts, nil),
		"pgk":         makeQuestion(entity.QuestionTypePGK, 2.0, pgOpts, nil),
		"benar_salah": makeQuestion(entity.QuestionTypeBenarSalah, 1.0, pgOpts, nil),
		"menjodohkan": makeQuestion(entity.QuestionTypeMenjodohkan, 1.0, matchOpts, nil),
		"isian":       makeQuestion(entity.QuestionTypeIsian, 2.0, nil, isianVariants),
		"matrix":      makeQuestion(entity.QuestionTypeMatrix, 1.0, matrixOpts, nil),
		"esai":        makeQuestion(entity.QuestionTypeEsai, 10.0, nil, nil),
	}

	nullAnswers := []struct {
		name   string
		answer types.JSON
	}{
		{"nil", nil},
		{"empty", types.JSON{}},
		{"null", types.JSON("null")},
		{"empty string json", types.JSON(`""`)},
	}

	for qType, q := range questions {
		for _, na := range nullAnswers {
			t.Run(qType+"/"+na.name, func(t *testing.T) {
				got := calc.Calculate(q, na.answer)
				if got != 0 {
					t.Errorf("%s with %s answer: expected 0, got %v", qType, na.name, got)
				}
			})
		}
	}
}
