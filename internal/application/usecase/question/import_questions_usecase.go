package question

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/sanitizer"
	"github.com/omanjaya/patra/pkg/types"
	"github.com/xuri/excelize/v2"
)

type QuestionImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// ImportQuestionsFromExcel parses an Excel file and bulk-creates questions.
// Expected columns:
//
//	A = question_type (pg/pgk/esai/isian_singkat/benar_salah)
//	B = body (question text, can be HTML)
//	C = options JSON array or pipe-separated (e.g. "Opsi A|Opsi B|Opsi C|Opsi D")
//	D = correct_answer (index 0-based for pg, or JSON)
//	E = score (default 1)
//	F = difficulty (easy/medium/hard, default medium)
//	G = audio_path (optional, filename of pre-uploaded audio)
//	H = audio_limit (optional, max playback count, default 2)
func ImportQuestionsFromExcel(data []byte, bankID uint, questionRepo repository.QuestionRepository) (*QuestionImportResult, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("gagal membaca file Excel: " + err.Error())
	}

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("file Excel tidak memiliki sheet")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}

	result := &QuestionImportResult{}
	orderIndex := 0

	// Note: Import uses per-row error handling (skip on error, continue).
	// This means partial imports are possible on DB errors. This is by design
	// to maximize imported rows. Use the import result to see skipped rows.
	for i, row := range rows {
		if i == 0 {
			continue // skip header
		}
		if len(row) < 2 {
			result.Errors = append(result.Errors, fmt.Sprintf("Baris %d: kolom tidak lengkap", i+1))
			result.Skipped++
			continue
		}

		qType := strings.TrimSpace(strings.ToLower(row[0]))
		body := strings.TrimSpace(row[1])
		if body == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Baris %d: body soal kosong", i+1))
			result.Skipped++
			continue
		}

		// Normalize question type
		switch qType {
		case "pg", "pgk", "esai", "isian_singkat", "isian", "singkat", "benar_salah", "bs", "menjodohkan", "matrix":
		default:
			qType = entity.QuestionTypePG
		}
		if qType == "isian" || qType == "singkat" {
			qType = entity.QuestionTypeIsian
		}
		if qType == "bs" {
			qType = entity.QuestionTypeBenarSalah
		}

		// Parse options
		var optionsJSON types.JSON
		if len(row) >= 3 && row[2] != "" {
			optionsJSON = parseOptions(row[2])
		}

		// Parse correct answer
		var correctJSON types.JSON
		if len(row) >= 4 && row[3] != "" {
			correctJSON = parseCorrectAnswer(row[3], qType)
		}

		// Parse score
		score := 1.0
		if len(row) >= 5 && row[4] != "" {
			fmt.Sscanf(row[4], "%f", &score)
		}

		// Parse difficulty
		difficulty := entity.DifficultyMedium
		if len(row) >= 6 && row[5] != "" {
			d := strings.ToLower(strings.TrimSpace(row[5]))
			if d == "easy" || d == "mudah" {
				difficulty = entity.DifficultyEasy
			} else if d == "hard" || d == "sulit" || d == "susah" {
				difficulty = entity.DifficultyHard
			}
		}

		// Parse audio_path (column G)
		var audioPath *string
		if len(row) >= 7 && strings.TrimSpace(row[6]) != "" {
			ap := strings.TrimSpace(row[6])
			audioPath = &ap
		}

		// Parse audio_limit (column H, default 2)
		audioLimit := 2
		if len(row) >= 8 && strings.TrimSpace(row[7]) != "" {
			if v, err := strconv.Atoi(strings.TrimSpace(row[7])); err == nil {
				audioLimit = v
			}
		}

		// Sanitize HTML in body and options
		body = sanitizer.SanitizeHTML(body)
		optionsJSON = sanitizeImportOptions(optionsJSON)

		orderIndex++
		q := &entity.Question{
			QuestionBankID: bankID,
			QuestionType:   qType,
			Body:           body,
			Options:        optionsJSON,
			CorrectAnswer:  correctJSON,
			Score:          score,
			Difficulty:     difficulty,
			OrderIndex:     orderIndex,
			AudioPath:      audioPath,
			AudioLimit:     audioLimit,
		}

		if err := questionRepo.Create(q); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Baris %d: %s", i+1, err.Error()))
			result.Skipped++
			continue
		}
		result.Imported++
	}

	return result, nil
}

func parseOptions(raw string) types.JSON {
	raw = strings.TrimSpace(raw)
	// If it looks like JSON array, use as-is
	if strings.HasPrefix(raw, "[") {
		return types.JSON(raw)
	}
	// Pipe-separated: "Opsi A|Opsi B|Opsi C|Opsi D"
	parts := strings.Split(raw, "|")
	opts := make([]map[string]interface{}, 0, len(parts))
	for i, p := range parts {
		opts = append(opts, map[string]interface{}{
			"index": i,
			"text":  strings.TrimSpace(p),
		})
	}
	b, _ := json.Marshal(opts)
	return types.JSON(b)
}

// sanitizeImportOptions sanitizes HTML in option text fields within imported options.
func sanitizeImportOptions(raw types.JSON) types.JSON {
	if raw == nil {
		return nil
	}
	var opts []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &opts); err != nil {
		return types.JSON(sanitizer.SanitizeHTML(string(raw)))
	}
	for i, opt := range opts {
		if text, ok := opt["text"].(string); ok {
			opts[i]["text"] = sanitizer.SanitizeHTML(text)
		}
	}
	b, _ := json.Marshal(opts)
	return types.JSON(b)
}

func parseCorrectAnswer(raw, qType string) types.JSON {
	raw = strings.TrimSpace(raw)
	// If JSON, use as-is
	if strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[") {
		return types.JSON(raw)
	}
	switch qType {
	case entity.QuestionTypePG, entity.QuestionTypeBenarSalah:
		// Single option index or letter A-E
		idx := 0
		if len(raw) == 1 && raw[0] >= 'A' && raw[0] <= 'E' {
			idx = int(raw[0] - 'A')
		} else {
			fmt.Sscanf(raw, "%d", &idx)
		}
		b, _ := json.Marshal(map[string]interface{}{"option_index": idx})
		return types.JSON(b)
	case entity.QuestionTypePGK:
		// Comma-separated letters: A,C,D
		letters := strings.Split(raw, ",")
		var indices []int
		for _, l := range letters {
			l = strings.TrimSpace(l)
			if len(l) == 1 && l[0] >= 'A' && l[0] <= 'E' {
				indices = append(indices, int(l[0]-'A'))
			}
		}
		b, _ := json.Marshal(map[string]interface{}{"option_indices": indices})
		return types.JSON(b)
	default:
		b, _ := json.Marshal(map[string]interface{}{"text": raw})
		return types.JSON(b)
	}
}
