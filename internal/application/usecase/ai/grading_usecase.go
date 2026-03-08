package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/urlvalidator"
)

type GradeResult struct {
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

type GradingUseCase struct {
	settingRepo repository.SettingRepository
}

func NewGradingUseCase(settingRepo repository.SettingRepository) *GradingUseCase {
	return &GradingUseCase{settingRepo: settingRepo}
}

// GradeEssay sends essay answer to configured AI API and returns score + reason.
func (uc *GradingUseCase) GradeEssay(question *entity.Question, answerText string) (*GradeResult, error) {
	apiURL := uc.getSetting("ai_api_url")
	apiKey := uc.getSetting("ai_api_key")
	apiHeader := uc.getSetting("ai_api_header")
	if apiHeader == "" {
		apiHeader = "Authorization"
	}

	if apiURL == "" || apiKey == "" {
		return nil, errors.New("konfigurasi AI belum lengkap. Atur di menu Pengaturan")
	}

	if err := urlvalidator.ValidateExternalURL(apiURL); err != nil {
		return nil, fmt.Errorf("AI API URL tidak aman: %w", err)
	}

	prompt := uc.buildPrompt(question, answerText)
	payload := uc.buildPayload(apiURL, prompt)

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(apiHeader, apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gagal menghubungi AI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("AI API error (%d)", resp.StatusCode)
	}

	respBody, _ := io.ReadAll(resp.Body)
	var respData map[string]interface{}
	_ = json.Unmarshal(respBody, &respData)

	text := uc.extractText(respData, string(respBody))
	cleaned := uc.cleanJSON(text)

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, errors.New("format respons AI tidak valid")
	}

	score := 0.0
	if v, ok := result["score"]; ok {
		if f, ok := v.(float64); ok {
			score = f
		}
	} else if v, ok := result["nilai"]; ok {
		if f, ok := v.(float64); ok {
			score = f
		}
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	reason := ""
	if v, ok := result["reason"]; ok {
		if s, ok := v.(string); ok {
			reason = s
		}
	} else if v, ok := result["alasan"]; ok {
		if s, ok := v.(string); ok {
			reason = s
		}
	}

	return &GradeResult{Score: score, Reason: reason}, nil
}

func (uc *GradingUseCase) getSetting(key string) string {
	s, err := uc.settingRepo.GetByKey(key)
	if err != nil || s == nil || s.Value == nil {
		return ""
	}
	return strings.TrimSpace(*s.Value)
}

func (uc *GradingUseCase) buildPrompt(q *entity.Question, answer string) string {
	soal := strings.TrimSpace(q.Body)
	answer = strings.TrimSpace(answer)
	kriteria := "Gunakan pengetahuan umum untuk menilai."

	return fmt.Sprintf(`Bertindaklah sebagai Guru Pemeriksa Ujian yang objektif.
Nilai jawaban esai siswa berdasarkan soal dan kriteria kunci jawaban.

[SOAL]
%s

[KRITERIA JAWABAN]
%s

[JAWABAN SISWA]
%s

[INSTRUKSI]
1. Berikan nilai (score) 0-100.
2. Berikan alasan singkat (reason, maks 2 kalimat).
3. PENTING: Output HANYA JSON valid tanpa teks lain.

Format: {"score": 85, "reason": "..."}`, soal, kriteria, answer)
}

func (uc *GradingUseCase) getModelParams() map[string]interface{} {
	raw := uc.getSetting("ai_model_params")
	if raw == "" {
		return nil
	}
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return nil
	}
	return params
}

func (uc *GradingUseCase) buildPayload(apiURL, prompt string) map[string]interface{} {
	params := uc.getModelParams()

	// Gemini format
	if strings.Contains(apiURL, "googleapis.com") {
		return map[string]interface{}{
			"contents": []map[string]interface{}{
				{"parts": []map[string]string{{"text": prompt}}},
			},
			"generationConfig": map[string]interface{}{
				"temperature": 0.4, "maxOutputTokens": 512,
			},
		}
	}

	// Anthropic Messages API format
	if strings.Contains(apiURL, "anthropic.com") {
		model := "claude-sonnet-4-6"
		if params != nil {
			if m, ok := params["model"].(string); ok && m != "" {
				model = m
			}
		}
		payload := map[string]interface{}{
			"model": model,
			"messages": []map[string]interface{}{
				{"role": "user", "content": prompt},
			},
			"max_tokens": 512,
		}
		// Merge extra params (excluding model/messages/max_tokens already set)
		if params != nil {
			for k, v := range params {
				if k != "model" && k != "messages" && k != "max_tokens" {
					payload[k] = v
				}
			}
		}
		return payload
	}

	// OpenAI chat completions format (default for openai.com and others)
	model := "gpt-4"
	if params != nil {
		if m, ok := params["model"].(string); ok && m != "" {
			model = m
		}
	}
	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  512,
		"temperature": 0.4,
	}
	// Merge extra params
	if params != nil {
		for k, v := range params {
			if k != "model" && k != "messages" && k != "max_tokens" {
				payload[k] = v
			}
		}
	}
	return payload
}

func (uc *GradingUseCase) extractText(data map[string]interface{}, raw string) string {
	// Gemini format
	if cands, ok := data["candidates"].([]interface{}); ok && len(cands) > 0 {
		if c, ok := cands[0].(map[string]interface{}); ok {
			if cont, ok := c["content"].(map[string]interface{}); ok {
				if parts, ok := cont["parts"].([]interface{}); ok && len(parts) > 0 {
					if p, ok := parts[0].(map[string]interface{}); ok {
						if t, ok := p["text"].(string); ok {
							return t
						}
					}
				}
			}
		}
	}
	// OpenAI format
	if choices, ok := data["choices"].([]interface{}); ok && len(choices) > 0 {
		if c, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := c["message"].(map[string]interface{}); ok {
				if t, ok := msg["content"].(string); ok {
					return t
				}
			}
		}
	}
	// Anthropic format: {"content": [{"type": "text", "text": "..."}]}
	if content, ok := data["content"].([]interface{}); ok && len(content) > 0 {
		if block, ok := content[0].(map[string]interface{}); ok {
			if t, ok := block["text"].(string); ok {
				return t
			}
		}
	}
	// Simple format
	for _, key := range []string{"text", "response", "output", "jawaban"} {
		if v, ok := data[key].(string); ok {
			return v
		}
	}
	return raw
}

func (uc *GradingUseCase) cleanJSON(text string) string {
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	re := regexp.MustCompile(`\{[\s\S]*\}`)
	if m := re.FindString(text); m != "" {
		return m
	}
	return text
}
