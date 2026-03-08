package handler

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	questionuc "github.com/omanjaya/patra/internal/application/usecase/question"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/response"
)

type QuestionImportHandler struct {
	questionRepo repository.QuestionRepository
	settingRepo  repository.SettingRepository
}

func NewQuestionImportHandler(questionRepo repository.QuestionRepository, settingRepo ...repository.SettingRepository) *QuestionImportHandler {
	h := &QuestionImportHandler{questionRepo: questionRepo}
	if len(settingRepo) > 0 {
		h.settingRepo = settingRepo[0]
	}
	return h
}

// POST /question-banks/:bankId/import
func (h *QuestionImportHandler) Import(c *gin.Context) {
	bankID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || bankID == 0 {
		response.BadRequest(c, "bank id tidak valid")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file wajib diupload")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".xlsx" && ext != ".xls" {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Format file harus .xlsx atau .xls")
		return
	}
	if header.Size > 10*1024*1024 { // 10MB
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Ukuran file maksimal 10MB")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		response.InternalError(c, "Gagal membaca file")
		return
	}

	// Validate actual content type (not just extension)
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "application/zip") &&
		!strings.HasPrefix(contentType, "application/vnd.openxmlformats") &&
		!strings.HasPrefix(contentType, "application/vnd.ms-excel") &&
		!strings.HasPrefix(contentType, "application/octet-stream") {
		response.BadRequest(c, "File bukan format Excel yang valid")
		return
	}

	result, err := questionuc.ImportQuestionsFromExcel(data, uint(bankID), h.questionRepo)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, result)
}

// GET /question-banks/:id/import/template
func (h *QuestionImportHandler) DownloadTemplate(c *gin.Context) {
	csvContent := "type,body,option_a,option_b,option_c,option_d,option_e,correct_answer,score,difficulty,explanation\r\n" +
		"multiple_choice,Ibukota negara Indonesia adalah...,Jakarta,Bandung,Surabaya,Medan,,A,1,easy,Jakarta adalah ibukota Indonesia\r\n" +
		"essay,Jelaskan dampak pemanasan global bagi kehidupan manusia!,,,,,,,1,medium,Jawaban dinilai manual oleh guru\r\n" +
		"true_false,Matahari terbit dari arah barat.,Benar,Salah,,,,B,1,easy,Matahari terbit dari timur\r\n"

	c.Header("Content-Disposition", `attachment; filename="template-import-soal.csv"`)
	c.Header("Content-Type", "text/csv")
	c.String(http.StatusOK, csvContent)
}

// =============================================================================
// AI Question Generation
// =============================================================================

// POST /admin/questions/generate-ai — generates questions using AI
func (h *QuestionImportHandler) GenerateAI(c *gin.Context) {
	if h.settingRepo == nil {
		response.InternalError(c, "Setting repository tidak tersedia")
		return
	}

	var req struct {
		Topic      string `json:"topic"`
		Prompt     string `json:"prompt"`
		Count      int    `json:"count"`
		Type       string `json:"type"`
		Difficulty string `json:"difficulty"`
		Language   string `json:"language"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	// Validate topic/prompt
	topic := req.Topic
	if topic == "" {
		topic = req.Prompt
	}
	if topic == "" {
		response.BadRequest(c, "Topik/Prompt tidak boleh kosong")
		return
	}

	if req.Count <= 0 {
		req.Count = 5
	}
	if req.Count > 50 {
		req.Count = 50
	}
	if req.Type == "" {
		req.Type = "pg"
	}
	if req.Difficulty == "" {
		req.Difficulty = "Sedang"
	}

	// Read AI settings from DB
	getVal := func(key string) string {
		s, err := h.settingRepo.GetByKey(key)
		if err != nil || s == nil || s.Value == nil {
			return ""
		}
		return *s.Value
	}

	apiURL := strings.TrimSpace(getVal("ai_api_url"))
	apiKey := strings.TrimSpace(getVal("ai_api_key"))
	apiHeader := strings.TrimSpace(getVal("ai_api_header"))
	modelParamsStr := getVal("ai_model_params")

	if apiURL == "" || apiKey == "" {
		response.BadRequest(c, "Konfigurasi AI belum lengkap. Silakan atur di Pengaturan.")
		return
	}

	if apiHeader == "" {
		apiHeader = "Authorization"
	}

	var modelParams map[string]interface{}
	if modelParamsStr != "" {
		_ = json.Unmarshal([]byte(modelParamsStr), &modelParams)
	}
	if modelParams == nil {
		modelParams = make(map[string]interface{})
	}

	// Construct generation prompt
	prompt := constructGenerationPrompt(topic, req.Count, req.Type, req.Difficulty)

	// Build request payload — support both Google API and OpenAI-compatible APIs
	isGoogleAPI := strings.Contains(apiURL, "googleapis.com")

	var payload map[string]interface{}
	if isGoogleAPI {
		payload = map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]string{
						{"text": prompt},
					},
				},
			},
			"generationConfig": map[string]interface{}{
				"temperature": 0.7,
				"topK":        40,
				"topP":        0.95,
			},
		}
		// Merge model params
		for k, v := range modelParams {
			payload[k] = v
		}
	} else {
		// OpenAI-compatible or custom proxy
		payload = map[string]interface{}{
			"model": func() string {
				if m, ok := modelParams["model"].(string); ok {
					return m
				}
				return "gpt-3.5-turbo"
			}(),
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
		}
		// Merge extra model params (except "model" which is already set)
		for k, v := range modelParams {
			if k != "model" {
				payload[k] = v
			}
		}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		response.InternalError(c, "Gagal membuat request AI")
		return
	}

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		response.BadRequest(c, "URL AI tidak valid")
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Set auth header
	if strings.EqualFold(apiHeader, "Authorization") {
		if !strings.HasPrefix(apiKey, "Bearer ") {
			httpReq.Header.Set(apiHeader, "Bearer "+apiKey)
		} else {
			httpReq.Header.Set(apiHeader, apiKey)
		}
	} else {
		httpReq.Header.Set(apiHeader, apiKey)
	}

	client := &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "AI_ERROR", fmt.Sprintf("Gagal menghubungi AI: %v", err))
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		response.InternalError(c, "Gagal membaca respon AI")
		return
	}

	if resp.StatusCode >= 400 {
		logger.Log.Errorf("AI API error (%d): %s", resp.StatusCode, string(respBody))
		response.Error(c, http.StatusBadGateway, "AI_ERROR",
			fmt.Sprintf("AI mengembalikan error (HTTP %d). Periksa konfigurasi API.", resp.StatusCode))
		return
	}

	// Extract text from AI response (support multiple formats)
	text := extractTextFromAIResponse(respBody)

	// Clean up the generated text
	text = cleanGeneratedText(text)

	if text == "" {
		response.InternalError(c, "Gagal generate soal dari AI (respon kosong)")
		return
	}

	response.Success(c, gin.H{
		"status":  "success",
		"content": text,
	})
}

// constructGenerationPrompt builds the AI prompt for question generation
func constructGenerationPrompt(topic string, count int, qType, difficulty string) string {
	typeInstruction := ""
	switch qType {
	case "pg":
		typeInstruction = "Format: Pilihan Ganda Biasa (1 Jawaban Benar). Opsi A-E. Sertakan Kunci."
	case "pgk":
		typeInstruction = "Format: Pilihan Ganda Kompleks (Jawaban Benar > 1). Opsi A-E. Kunci pisahkan koma (contoh: Kunci:A,C)."
	case "benar_salah", "bs":
		typeInstruction = "Format: [BENAR-SALAH]. Opsi A=Benar, B=Salah. Sertakan Kunci."
	case "menjodohkan":
		typeInstruction = "Format: [MENJODOHKAN]. Pasangkan premis kiri dan kanan. Contoh: A. Kiri = Kanan."
	case "isian_singkat", "singkat":
		typeInstruction = "Format: [ISIAN-SINGKAT]. Jawaban pendek pasti. Contoh: Kunci:Jawabannya."
	case "matrix":
		typeInstruction = "Format: [MATRIX]. Tentukan Kolom (misal: Benar,Salah) dan Baris pernyataan."
	case "esai":
		typeInstruction = "Format: Esai/Uraian. Sertakan tag [ESAI]."
	default:
		typeInstruction = "Format: Pilihan Ganda."
	}

	examples := `[CONTOH FORMAT SESUAI TIPE]:

JIKA PG / PGK:
Soal:1) Pertanyaan...<br>
A. Opsi<br>
...
Kunci:A (atau A,C jika PGK)<br>

JIKA BENAR-SALAH:
Soal:1) [BENAR-SALAH] Pernyataan...<br>
A. Benar<br>
B. Salah<br>
Kunci:A<br>

JIKA MENJODOHKAN:
Soal:1) [MENJODOHKAN] Pasangkan ibukota berikut...<br>
A. Indonesia = Jakarta<br>
B. Jepang = Tokyo<br>

JIKA ISIAN SINGKAT:
Soal:1) [ISIAN-SINGKAT] Presiden pertama RI adalah...<br>
Kunci:Soekarno<br>

JIKA MATRIX:
Soal:1) [MATRIX] Tentukan fakta/opini.<br>
Kolom:Fakta, Opini<br>
Baris:Matahari terbit timur = 1<br>
Baris:Bakso itu enak = 2<br>`

	return fmt.Sprintf(`Bertindaklah sebagai Mesin Generator Soal (Bukan Asisten Chat).
Tugas: Buat %d soal ujian topik "%s" (%s).
%s

ATURAN OUTPUT MUTLAK:
1. JANGAN ADA TEKS PEMBUKA (Seperti "Tentu", "Berikut adalah", dll). Langsung mulai dengan soal no 1.
2. Gunakan tag HTML <br> untuk ganti baris.
3. Gunakan format persis seperti contoh di bawah.

%s

Hasilkan HANYA raw string soal tersebut tanpa markdown code block.`, count, topic, difficulty, typeInstruction, examples)
}

// extractTextFromAIResponse extracts text content from various AI API response formats
func extractTextFromAIResponse(body []byte) string {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// If not JSON, return as plain text
		return string(body)
	}

	// Custom wrapper format: { data: {...} }
	if d, ok := data["data"]; ok {
		if dm, ok := d.(map[string]interface{}); ok {
			b, _ := json.Marshal(dm)
			return string(b)
		}
	}

	// Simple formats: text, response, output
	for _, key := range []string{"text", "response", "output", "jawaban"} {
		if v, ok := data[key].(string); ok {
			return v
		}
	}

	// Gemini format: candidates[0].content.parts[0].text
	if candidates, ok := data["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if cand, ok := candidates[0].(map[string]interface{}); ok {
			if content, ok := cand["content"].(map[string]interface{}); ok {
				if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
					if part, ok := parts[0].(map[string]interface{}); ok {
						if text, ok := part["text"].(string); ok {
							return text
						}
					}
				}
			}
		}
	}

	// OpenAI format: choices[0].message.content
	if choices, ok := data["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok {
					return content
				}
			}
		}
	}

	// Fallback: return raw JSON
	return string(body)
}

// cleanGeneratedText cleans up AI-generated question text
func cleanGeneratedText(text string) string {
	// Remove markdown code blocks
	re := regexp.MustCompile("(?m)^```.*\n?")
	text = re.ReplaceAllString(text, "")

	text = strings.TrimSpace(text)

	// Normalize newlines
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Remove excessive newlines (3+ -> 2)
	reMultiNewline := regexp.MustCompile(`\n{3,}`)
	text = reMultiNewline.ReplaceAllString(text, "\n\n")

	return text
}

// =============================================================================
// MathML Conversion
// =============================================================================

// POST /admin/questions/convert-mathml — converts MathML to SVG
func (h *QuestionImportHandler) ConvertMathML(c *gin.Context) {
	var req struct {
		MathML string `json:"mathml" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	if strings.TrimSpace(req.MathML) == "" {
		response.BadRequest(c, "MathML tidak boleh kosong")
		return
	}

	// Try to find Node.js binary
	nodePath := findNodeBinary()
	if nodePath == "" {
		response.Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE",
			"Node.js tidak ditemukan. Pastikan Node.js terinstall untuk konversi MathML.")
		return
	}

	// Try to find the converter script
	scriptPath := findMathMLScript()
	if scriptPath == "" {
		// Fallback: return the MathML as-is with a note that conversion is not available
		response.Success(c, gin.H{
			"success": false,
			"message": "MathML converter script tidak ditemukan. Silakan install: npm install mathml-to-svg",
			"mathml":  req.MathML,
		})
		return
	}

	// Execute Node.js script with MathML input
	cmd := exec.Command(nodePath, scriptPath, req.MathML)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logger.Log.Errorf("MathML conversion error: %v, stderr: %s", err, stderr.String())
		response.InternalError(c, "Gagal mengkonversi persamaan: "+stderr.String())
		return
	}

	svg := stdout.String()

	// Save SVG to storage
	filename := fmt.Sprintf("equation_%d_%s.svg", time.Now().Unix(), uuid.New().String()[:8])
	savePath := filepath.Join("uploads", "question-uploads", "general", filename)
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		response.InternalError(c, "Gagal membuat direktori penyimpanan")
		return
	}

	if err := os.WriteFile(savePath, []byte(svg), 0644); err != nil {
		response.InternalError(c, "Gagal menyimpan file SVG")
		return
	}

	url := "/" + savePath

	response.Success(c, gin.H{
		"success": true,
		"url":     url,
		"svg":     svg,
	})
}

// findNodeBinary looks for the Node.js binary in common locations
func findNodeBinary() string {
	// Try common paths
	paths := []string{
		"/usr/local/bin/node",
		"/opt/homebrew/bin/node",
		"/usr/bin/node",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Try PATH lookup
	path, err := exec.LookPath("node")
	if err == nil {
		return path
	}

	return ""
}

// findMathMLScript looks for the MathML converter script
func findMathMLScript() string {
	candidates := []string{
		"scripts/mathml-to-svg.cjs",
		"scripts/mathml-to-svg.js",
		"web/scripts/mathml-to-svg.cjs",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	return ""
}

// =============================================================================
// Upload Image during Import
// =============================================================================

// POST /admin/questions/import/upload-image — uploads an image for use in question import
func (h *QuestionImportHandler) UploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "File gambar wajib diupload")
		return
	}
	defer file.Close()

	// Validate file size (max 10MB)
	if header.Size > 10*1024*1024 {
		response.BadRequest(c, "Ukuran file maksimal 10MB")
		return
	}

	// Validate extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	validExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true,
		".gif": true, ".svg": true, ".webp": true,
	}
	if !validExts[ext] {
		response.BadRequest(c, "Format file tidak didukung. Gunakan JPG, PNG, GIF, SVG, atau WebP")
		return
	}

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		response.InternalError(c, "Gagal membaca file")
		return
	}

	// Validate content type
	contentType := http.DetectContentType(data)
	validTypes := map[string]bool{
		"image/jpeg":    true,
		"image/png":     true,
		"image/gif":     true,
		"image/svg+xml": true,
		"image/webp":    true,
	}
	// SVG may be detected as text/xml or application/xml
	if !validTypes[contentType] && !strings.Contains(contentType, "xml") && contentType != "text/plain" {
		response.BadRequest(c, "File bukan format gambar yang valid")
		return
	}

	// Generate unique filename
	filename := fmt.Sprintf("import_%d_%s%s", time.Now().Unix(), uuid.New().String()[:8], ext)
	savePath := filepath.Join("uploads", "question-uploads", "general", filename)

	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		response.InternalError(c, "Gagal membuat direktori penyimpanan")
		return
	}

	if err := os.WriteFile(savePath, data, 0644); err != nil {
		response.InternalError(c, "Gagal menyimpan file")
		return
	}

	url := "/" + savePath

	response.Success(c, gin.H{
		"location": url,
		"url":      url,
		"filename": filename,
	})
}
