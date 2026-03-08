package service

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/types"
)

// ─── Benchmark Helpers ──────────────────────────────────────────

func benchQuestion(qType string, score float64, options, correctAnswer interface{}) *entity.Question {
	q := &entity.Question{
		QuestionType: qType,
		Score:        score,
	}
	if options != nil {
		b, _ := json.Marshal(options)
		q.Options = types.JSON(b)
	}
	if correctAnswer != nil {
		b, _ := json.Marshal(correctAnswer)
		q.CorrectAnswer = types.JSON(b)
	}
	return q
}

func benchJSON(v interface{}) types.JSON {
	b, _ := json.Marshal(v)
	return types.JSON(b)
}

// ─── BenchmarkScorePG ───────────────────────────────────────────

func BenchmarkScorePG(b *testing.B) {
	calc := NewScoreCalculator()
	q := benchQuestion(entity.QuestionTypePG, 10, []pgOption{
		{Text: "A", IsCorrect: false},
		{Text: "B", IsCorrect: true},
		{Text: "C", IsCorrect: false},
		{Text: "D", IsCorrect: false},
	}, nil)
	answer := benchJSON(map[string]int{"option_index": 1})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(q, answer)
	}
}

func BenchmarkScorePG_WithWeight(b *testing.B) {
	calc := NewScoreCalculator()
	q := benchQuestion(entity.QuestionTypePG, 10, []pgOption{
		{Text: "A", Weight: ptrFloat(0)},
		{Text: "B", Weight: ptrFloat(1.0)},
		{Text: "C", Weight: ptrFloat(0.5)},
		{Text: "D", Weight: ptrFloat(0.25)},
	}, nil)
	answer := benchJSON(map[string]int{"option_index": 2})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(q, answer)
	}
}

// ─── BenchmarkScorePGK ─────────────────────────────────────────

func BenchmarkScorePGK(b *testing.B) {
	calc := NewScoreCalculator()
	q := benchQuestion(entity.QuestionTypePGK, 4, []pgOption{
		{Text: "A", Weight: ptrFloat(1.0)},
		{Text: "B", Weight: ptrFloat(1.0)},
		{Text: "C", Weight: ptrFloat(0)},
		{Text: "D", Weight: ptrFloat(-0.5)},
		{Text: "E", Weight: ptrFloat(1.0)},
	}, nil)
	answer := benchJSON(map[string][]int{"option_indices": {0, 1, 4}})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(q, answer)
	}
}

func BenchmarkScorePGK_ManyOptions(b *testing.B) {
	calc := NewScoreCalculator()
	opts := make([]pgOption, 20)
	indices := make([]int, 0, 10)
	for i := range opts {
		w := 0.0
		if i%2 == 0 {
			w = 1.0
			indices = append(indices, i)
		}
		opts[i] = pgOption{Text: fmt.Sprintf("Option %d", i), Weight: &w}
	}
	q := benchQuestion(entity.QuestionTypePGK, 10, opts, nil)
	answer := benchJSON(map[string][]int{"option_indices": indices})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(q, answer)
	}
}

// ─── BenchmarkScoreMenjodohkan ──────────────────────────────────

func BenchmarkScoreMenjodohkan(b *testing.B) {
	calc := NewScoreCalculator()
	opts := []matchingOption{
		{Prompt: "Ibukota Indonesia", Answer: "Jakarta"},
		{Prompt: "Ibukota Jepang", Answer: "Tokyo"},
		{Prompt: "Ibukota Prancis", Answer: "Paris"},
		{Prompt: "Ibukota Jerman", Answer: "Berlin"},
		{Prompt: "Ibukota Inggris", Answer: "London"},
	}
	q := benchQuestion(entity.QuestionTypeMenjodohkan, 5, opts, nil)
	answer := benchJSON(map[string]interface{}{
		"pairs": map[string]string{
			"Ibukota Indonesia": "Jakarta",
			"Ibukota Jepang":    "Tokyo",
			"Ibukota Prancis":   "Paris",
			"Ibukota Jerman":    "Berlin",
			"Ibukota Inggris":   "London",
		},
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(q, answer)
	}
}

func BenchmarkScoreMenjodohkan_Large(b *testing.B) {
	calc := NewScoreCalculator()
	opts := make([]matchingOption, 20)
	pairs := make(map[string]string, 20)
	for i := range opts {
		prompt := fmt.Sprintf("Prompt %d", i)
		ans := fmt.Sprintf("Answer %d", i)
		opts[i] = matchingOption{Prompt: prompt, Answer: ans}
		pairs[prompt] = ans
	}
	q := benchQuestion(entity.QuestionTypeMenjodohkan, 20, opts, nil)
	answer := benchJSON(map[string]interface{}{"pairs": pairs})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(q, answer)
	}
}

// ─── BenchmarkScoreIsian ────────────────────────────────────────

func BenchmarkScoreIsian(b *testing.B) {
	calc := NewScoreCalculator()
	variants := []shortAnswerVariant{
		{Text: "Jakarta", Weight: 1.0},
		{Text: "DKI Jakarta", Weight: 0.8},
		{Text: "Daerah Khusus Ibukota Jakarta", Weight: 0.6},
	}
	q := benchQuestion(entity.QuestionTypeIsian, 2, nil, variants)
	answer := benchJSON(map[string]string{"text": "Jakarta"})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(q, answer)
	}
}

func BenchmarkScoreIsian_ManyVariants(b *testing.B) {
	calc := NewScoreCalculator()
	variants := make([]shortAnswerVariant, 50)
	for i := range variants {
		variants[i] = shortAnswerVariant{
			Text:   fmt.Sprintf("variant %d", i),
			Weight: float64(50-i) / 50.0,
		}
	}
	q := benchQuestion(entity.QuestionTypeIsian, 5, nil, variants)
	// Match the last variant to force full scan
	answer := benchJSON(map[string]string{"text": "variant 49"})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(q, answer)
	}
}

// ─── BenchmarkScoreMatrix ───────────────────────────────────────

func BenchmarkScoreMatrix(b *testing.B) {
	calc := NewScoreCalculator()
	md := matrixData{
		Columns: []string{"Setuju", "Netral", "Tidak Setuju"},
		Rows: []matrixRow{
			{StatementText: "Pernyataan 1", CorrectColumnIndex: 0},
			{StatementText: "Pernyataan 2", CorrectColumnIndex: 2},
			{StatementText: "Pernyataan 3", CorrectColumnIndex: 1},
			{StatementText: "Pernyataan 4", CorrectColumnIndex: 0},
			{StatementText: "Pernyataan 5", CorrectColumnIndex: 2},
		},
	}
	q := benchQuestion(entity.QuestionTypeMatrix, 5, md, nil)
	answer := benchJSON(map[string]interface{}{
		"answers": map[string]int{"0": 0, "1": 2, "2": 1, "3": 0, "4": 2},
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(q, answer)
	}
}

func BenchmarkScoreMatrix_Large(b *testing.B) {
	calc := NewScoreCalculator()
	rows := make([]matrixRow, 20)
	answers := make(map[string]int, 20)
	for i := range rows {
		rows[i] = matrixRow{
			StatementText:      fmt.Sprintf("Statement %d", i),
			CorrectColumnIndex: i % 4,
		}
		answers[fmt.Sprintf("%d", i)] = i % 4
	}
	md := matrixData{
		Columns: []string{"A", "B", "C", "D"},
		Rows:    rows,
	}
	q := benchQuestion(entity.QuestionTypeMatrix, 20, md, nil)
	answer := benchJSON(map[string]interface{}{"answers": answers})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(q, answer)
	}
}

// ─── BenchmarkScoreBenarSalah ───────────────────────────────────

func BenchmarkScoreBenarSalah(b *testing.B) {
	calc := NewScoreCalculator()
	q := benchQuestion(entity.QuestionTypeBenarSalah, 1, []pgOption{
		{Text: "Benar", IsCorrect: true},
		{Text: "Salah", IsCorrect: false},
	}, nil)
	answer := benchJSON(map[string]int{"option_index": 0})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(q, answer)
	}
}

// ─── Parallel Benchmarks (simulate concurrent scoring) ──────────

func BenchmarkScorePG_Parallel(b *testing.B) {
	calc := NewScoreCalculator()
	q := benchQuestion(entity.QuestionTypePG, 10, []pgOption{
		{Text: "A", IsCorrect: false},
		{Text: "B", IsCorrect: true},
		{Text: "C", IsCorrect: false},
		{Text: "D", IsCorrect: false},
	}, nil)
	answer := benchJSON(map[string]int{"option_index": 1})

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			calc.Calculate(q, answer)
		}
	})
}

func BenchmarkScorePGK_Parallel(b *testing.B) {
	calc := NewScoreCalculator()
	q := benchQuestion(entity.QuestionTypePGK, 4, []pgOption{
		{Text: "A", Weight: ptrFloat(1.0)},
		{Text: "B", Weight: ptrFloat(1.0)},
		{Text: "C", Weight: ptrFloat(0)},
		{Text: "D", Weight: ptrFloat(-0.5)},
	}, nil)
	answer := benchJSON(map[string][]int{"option_indices": {0, 1}})

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			calc.Calculate(q, answer)
		}
	})
}

func BenchmarkScoreMenjodohkan_Parallel(b *testing.B) {
	calc := NewScoreCalculator()
	opts := []matchingOption{
		{Prompt: "A", Answer: "1"},
		{Prompt: "B", Answer: "2"},
		{Prompt: "C", Answer: "3"},
	}
	q := benchQuestion(entity.QuestionTypeMenjodohkan, 3, opts, nil)
	answer := benchJSON(map[string]interface{}{
		"pairs": map[string]string{"A": "1", "B": "2", "C": "3"},
	})

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			calc.Calculate(q, answer)
		}
	})
}
