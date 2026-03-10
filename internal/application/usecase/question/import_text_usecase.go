package question

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/sanitizer"
	"github.com/omanjaya/patra/pkg/types"
)

// Regex patterns used by the text parser.
var (
	reSoalPrefix    = regexp.MustCompile(`(?mi)^Soal\s*:\s*(\d+)\)\s*`)
	reWacanaBlock   = regexp.MustCompile(`(?mi)^\[(WACANA|STIMULUS)\]\s*`)
	reSoalMandiri   = regexp.MustCompile(`(?mi)^\[(SOAL-MANDIRI|SOAL-BIASA)\]\s*`)
	reTypeTag       = regexp.MustCompile(`(?i)\[(<[^>]*>)*(BENAR-SALAH|MENJODOHKAN|ISIAN-SINGKAT|ESAI|PGK|PG|MATRIX)(<[^>]*>)*\]`)
	reOptionLine    = regexp.MustCompile(`(?m)^([A-Z])[\.\)]\s+(.+)$`)
	reKunci         = regexp.MustCompile(`(?mi)^Kunci\s*:\s*(.+)$`)
	rePoin          = regexp.MustCompile(`(?mi)^Poin\s*:\s*(.+)$`)
	reKolom         = regexp.MustCompile(`(?mi)^Kolom\s*:\s*(.+)$`)
	reBaris         = regexp.MustCompile(`(?mi)^Baris\s*:\s*(.+)$`)
	reOptionWeight  = regexp.MustCompile(`^\[(\d+)%\]\s*(.+)$`)
	reHTMLComment   = regexp.MustCompile(`<!--[\s\S]*?-->`)
	reStyleBlock    = regexp.MustCompile(`(?i)<style[^>]*>[\s\S]*?</style>`)
	reMsoCSS        = regexp.MustCompile(`(?i)\s*mso-[^;:"]+:[^;:"]+;?`)
	reMsoClass      = regexp.MustCompile(`(?i)\s+class="Mso[^"]*"`)
	reEmptySpan     = regexp.MustCompile(`(?i)<span[^>]*>\s*</span>`)
	reEmptyDiv      = regexp.MustCompile(`(?i)<div[^>]*>\s*</div>`)
	reBrVariants    = regexp.MustCompile(`(?i)<br\s*/?>`)
	reEmptyStyle    = regexp.MustCompile(`(?i)\s+style="\s*"`)
	reSplitSoal = regexp.MustCompile(`(?mi)^(?:Soal\s*:\s*\d+\)|\[(WACANA|STIMULUS|SOAL-MANDIRI|SOAL-BIASA)\])`)
)

// cleanWordHTML removes Word/mso-specific markup from pasted HTML content.
func cleanWordHTML(html string) string {
	s := html

	// Remove HTML comments (<!-- ... -->)
	s = reHTMLComment.ReplaceAllString(s, "")

	// Remove <style> blocks
	s = reStyleBlock.ReplaceAllString(s, "")

	// Remove mso-* CSS properties
	s = reMsoCSS.ReplaceAllString(s, "")

	// Remove class="Mso*" attributes
	s = reMsoClass.ReplaceAllString(s, "")

	// Remove leftover empty style attributes
	s = reEmptyStyle.ReplaceAllString(s, "")

	// Remove empty spans and divs
	s = reEmptySpan.ReplaceAllString(s, "")
	s = reEmptyDiv.ReplaceAllString(s, "")

	// Normalize <br> variants to \n
	s = reBrVariants.ReplaceAllString(s, "\n")

	// Convert <p>...</p> to text with newlines (simple approach)
	s = regexp.MustCompile(`(?i)<p[^>]*>`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?i)</p>`).ReplaceAllString(s, "\n")

	return strings.TrimSpace(s)
}

// ImportQuestionsFromText parses a copy-paste text format (including Word)
// and bulk-creates questions and stimuli in the given question bank.
func ImportQuestionsFromText(
	content string,
	bankID uint,
	questionRepo repository.QuestionRepository,
	stimulusRepo repository.QuestionRepository, // stimulus methods are on QuestionRepository
) (*QuestionImportResult, error) {

	content = cleanWordHTML(content)

	result := &QuestionImportResult{}
	blocks := splitTextBlocks(content)

	var currentStimulusID *uint
	orderIndex := 0

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		// --- Wacana / Stimulus block ---
		if reWacanaBlock.MatchString(block) {
			stimContent := reWacanaBlock.ReplaceAllString(block, "")
			stimContent = sanitizer.SanitizeHTML(strings.TrimSpace(stimContent))
			if stimContent == "" {
				result.Errors = append(result.Errors, "Blok WACANA kosong, dilewati")
				result.Skipped++
				continue
			}
			stim := &entity.Stimulus{
				QuestionBankID: bankID,
				Content:        stimContent,
			}
			if err := stimulusRepo.CreateStimulus(stim); err != nil {
				result.Errors = append(result.Errors, "Gagal menyimpan stimulus: "+err.Error())
				result.Skipped++
				currentStimulusID = nil
				continue
			}
			currentStimulusID = &stim.ID
			continue
		}

		// --- SOAL-MANDIRI / SOAL-BIASA: reset stimulus link ---
		if reSoalMandiri.MatchString(block) {
			currentStimulusID = nil
			// The block may also contain a question – strip the tag and fall through
			block = reSoalMandiri.ReplaceAllString(block, "")
			block = strings.TrimSpace(block)
			if block == "" {
				continue
			}
		}

		// --- Question block ---
		q, errMsg := parseTextQuestionBlock(block, bankID)
		if errMsg != "" {
			result.Errors = append(result.Errors, errMsg)
			result.Skipped++
			continue
		}

		q.StimulusID = currentStimulusID
		orderIndex++
		q.OrderIndex = orderIndex

		if err := questionRepo.Create(q); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Soal #%d: %s", orderIndex, err.Error()))
			result.Skipped++
			continue
		}
		result.Imported++
	}

	return result, nil
}

// splitTextBlocks splits the cleaned content into discrete blocks.
// Each block starts with "Soal:N)" or a bracket tag like [WACANA].
func splitTextBlocks(content string) []string {
	indices := reSplitSoal.FindAllStringIndex(content, -1)
	if len(indices) == 0 {
		// No structure found – treat entire content as one block
		return []string{content}
	}

	var blocks []string
	for i, idx := range indices {
		start := idx[0]
		var end int
		if i+1 < len(indices) {
			end = indices[i+1][0]
		} else {
			end = len(content)
		}
		blocks = append(blocks, content[start:end])
	}

	// If there is content before the first block (preamble), prepend it
	if indices[0][0] > 0 {
		preamble := strings.TrimSpace(content[:indices[0][0]])
		if preamble != "" {
			blocks = append([]string{preamble}, blocks...)
		}
	}

	return blocks
}

// parseTextQuestionBlock parses a single question block and returns an entity.
func parseTextQuestionBlock(block string, bankID uint) (*entity.Question, string) {
	// Remove the "Soal:N) " prefix
	body := reSoalPrefix.ReplaceAllString(block, "")

	// Detect question type from tag like [ESAI], [BENAR-SALAH], etc.
	qType := entity.QuestionTypePG // default
	if match := reTypeTag.FindStringSubmatch(body); len(match) > 0 {
		qType = normalizeTypeTag(match[2])
		body = reTypeTag.ReplaceAllString(body, "")
	}

	// Extract Poin (score)
	score := 1.0
	if m := rePoin.FindStringSubmatch(body); len(m) > 0 {
		if v, err := strconv.ParseFloat(strings.TrimSpace(m[1]), 64); err == nil {
			score = v
		}
		body = rePoin.ReplaceAllString(body, "")
	}

	// Extract Kunci (correct answer)
	var kunciRaw string
	if m := reKunci.FindStringSubmatch(body); len(m) > 0 {
		kunciRaw = strings.TrimSpace(m[1])
		body = reKunci.ReplaceAllString(body, "")
	}

	// Extract matrix columns and rows
	var matrixKolom []string
	var matrixBaris []string
	if qType == entity.QuestionTypeMatrix {
		if m := reKolom.FindStringSubmatch(body); len(m) > 0 {
			for _, col := range strings.Split(m[1], ",") {
				matrixKolom = append(matrixKolom, strings.TrimSpace(col))
			}
			body = reKolom.ReplaceAllString(body, "")
		}
		for _, m := range reBaris.FindAllStringSubmatch(body, -1) {
			matrixBaris = append(matrixBaris, strings.TrimSpace(m[1]))
		}
		body = reBaris.ReplaceAllString(body, "")
	}

	// Extract options (A. ... B. ... etc.)
	optionMatches := reOptionLine.FindAllStringSubmatch(body, -1)
	var optionsJSON types.JSON
	if len(optionMatches) > 0 {
		optionsJSON = buildOptions(optionMatches, qType)
		// Remove option lines from body
		body = reOptionLine.ReplaceAllString(body, "")
	}

	// Clean up the body
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, "Soal dengan body kosong, dilewati"
	}
	body = sanitizer.SanitizeHTML(body)

	// Sanitize options
	optionsJSON = sanitizeImportOptions(optionsJSON)

	// Build correct answer
	var correctJSON types.JSON
	switch qType {
	case entity.QuestionTypeMatrix:
		correctJSON = buildMatrixAnswer(matrixKolom, matrixBaris)
	case entity.QuestionTypeEsai:
		// Esai typically has no key; if provided, store as text
		if kunciRaw != "" {
			b, _ := json.Marshal(map[string]interface{}{"text": kunciRaw})
			correctJSON = types.JSON(b)
		}
	case entity.QuestionTypeIsian:
		correctJSON = buildIsianAnswer(kunciRaw)
	case entity.QuestionTypeMenjodohkan:
		correctJSON = buildMenjodohkanAnswer(optionMatches)
	default:
		if kunciRaw != "" {
			correctJSON = parseCorrectAnswer(kunciRaw, qType)
		}
	}

	return &entity.Question{
		QuestionBankID: bankID,
		QuestionType:   qType,
		Body:           body,
		Options:        optionsJSON,
		CorrectAnswer:  correctJSON,
		Score:          score,
		Difficulty:     entity.DifficultyMedium,
	}, ""
}

// normalizeTypeTag converts a bracket tag name to the entity constant.
func normalizeTypeTag(tag string) string {
	tag = strings.ToUpper(strings.TrimSpace(tag))
	switch tag {
	case "BENAR-SALAH":
		return entity.QuestionTypeBenarSalah
	case "MENJODOHKAN":
		return entity.QuestionTypeMenjodohkan
	case "ISIAN-SINGKAT":
		return entity.QuestionTypeIsian
	case "ESAI":
		return entity.QuestionTypeEsai
	case "PGK":
		return entity.QuestionTypePGK
	case "MATRIX":
		return entity.QuestionTypeMatrix
	case "PG":
		return entity.QuestionTypePG
	default:
		return entity.QuestionTypePG
	}
}

// buildOptions creates a JSON array of option objects from regex matches.
func buildOptions(matches [][]string, qType string) types.JSON {
	opts := make([]map[string]interface{}, 0, len(matches))
	for i, m := range matches {
		text := strings.TrimSpace(m[2])
		opt := map[string]interface{}{
			"index": i,
			"label": m[1],
			"text":  text,
		}

		// Check for weight: [75%] Text
		if wm := reOptionWeight.FindStringSubmatch(text); len(wm) > 0 {
			if w, err := strconv.Atoi(wm[1]); err == nil {
				opt["weight"] = w
			}
			opt["text"] = strings.TrimSpace(wm[2])
		}

		// For menjodohkan, split on " = "
		if qType == entity.QuestionTypeMenjodohkan {
			parts := strings.SplitN(text, " = ", 2)
			if len(parts) == 2 {
				opt["left"] = strings.TrimSpace(parts[0])
				opt["right"] = strings.TrimSpace(parts[1])
			}
		}

		opts = append(opts, opt)
	}
	b, _ := json.Marshal(opts)
	return types.JSON(b)
}

// buildMatrixAnswer creates the correct_answer JSON for matrix questions.
// Format: "Baris: Earth is round = 1" means row "Earth is round" maps to column index 1.
func buildMatrixAnswer(kolom []string, baris []string) types.JSON {
	answer := map[string]interface{}{
		"columns": kolom,
	}
	rows := make([]map[string]interface{}, 0, len(baris))
	for _, b := range baris {
		parts := strings.SplitN(b, " = ", 2)
		if len(parts) != 2 {
			continue
		}
		colIdx := 0
		if v, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			colIdx = v
		}
		rows = append(rows, map[string]interface{}{
			"text":         strings.TrimSpace(parts[0]),
			"column_index": colIdx,
		})
	}
	answer["rows"] = rows
	b, _ := json.Marshal(answer)
	return types.JSON(b)
}

// buildIsianAnswer creates the correct_answer JSON for short-answer questions.
// Supports weighted answers: "Jakarta=100%, DKI Jakarta=80%"
func buildIsianAnswer(raw string) types.JSON {
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	answers := make([]map[string]interface{}, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Check for weight: Answer=80%
		if eqIdx := strings.LastIndex(p, "="); eqIdx > 0 {
			valPart := strings.TrimSpace(p[eqIdx+1:])
			if strings.HasSuffix(valPart, "%") {
				text := strings.TrimSpace(p[:eqIdx])
				weightStr := strings.TrimSuffix(valPart, "%")
				weight, err := strconv.Atoi(strings.TrimSpace(weightStr))
				if err == nil {
					answers = append(answers, map[string]interface{}{
						"text":   text,
						"weight": weight,
					})
					continue
				}
			}
		}
		answers = append(answers, map[string]interface{}{
			"text":   p,
			"weight": 100,
		})
	}

	b, _ := json.Marshal(map[string]interface{}{"answers": answers})
	return types.JSON(b)
}

// buildMenjodohkanAnswer creates the correct_answer JSON for matching questions.
// The answer is derived from the option pairs (left = right).
func buildMenjodohkanAnswer(optionMatches [][]string) types.JSON {
	pairs := make([]map[string]interface{}, 0, len(optionMatches))
	for i, m := range optionMatches {
		text := strings.TrimSpace(m[2])
		parts := strings.SplitN(text, " = ", 2)
		if len(parts) == 2 {
			pairs = append(pairs, map[string]interface{}{
				"index": i,
				"left":  strings.TrimSpace(parts[0]),
				"right": strings.TrimSpace(parts[1]),
			})
		}
	}
	b, _ := json.Marshal(map[string]interface{}{"pairs": pairs})
	return types.JSON(b)
}
