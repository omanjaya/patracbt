package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/response"
	"github.com/omanjaya/patra/pkg/urlvalidator"
)

type BackupHandler struct {
	settingRepo repository.SettingRepository
}

func NewBackupHandler(settingRepo repository.SettingRepository) *BackupHandler {
	return &BackupHandler{settingRepo: settingRepo}
}

// POST /settings/backup — create settings backup JSON file
func (h *BackupHandler) CreateBackup(c *gin.Context) {
	settings, err := h.settingRepo.GetAll()
	if err != nil {
		response.InternalError(c, "Gagal mengambil pengaturan")
		return
	}

	data := make(map[string]string)
	for _, s := range settings {
		if s.Value != nil {
			// Skip sensitive keys
			if s.Key == "ai_api_key" {
				data[s.Key] = ""
			} else {
				data[s.Key] = *s.Value
			}
		}
	}

	payload := gin.H{
		"version":    "1.0",
		"created_at": time.Now().Format(time.RFC3339),
		"settings":   data,
	}

	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		response.InternalError(c, "Gagal membuat backup")
		return
	}

	backupDir := "backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		response.InternalError(c, "Gagal membuat direktori backup")
		return
	}

	filename := fmt.Sprintf("backup_settings_%s.json", time.Now().Format("20060102_150405"))
	savePath := filepath.Join(backupDir, filename)
	if err := os.WriteFile(savePath, b, 0644); err != nil {
		response.InternalError(c, "Gagal menyimpan file backup")
		return
	}

	response.Success(c, gin.H{"filename": filename, "path": savePath})
}

// GET /settings/backup/download/:filename — download backup file
func (h *BackupHandler) DownloadBackup(c *gin.Context) {
	filename := c.Param("filename")
	// Security: prevent path traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Nama file tidak valid")
		return
	}

	path := filepath.Join("backups", filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		response.NotFound(c, "File backup tidak ditemukan")
		return
	}

	c.FileAttachment(path, filename)
}

// POST /settings/restore — restore settings from uploaded JSON
func (h *BackupHandler) RestoreBackup(c *gin.Context) {
	file, _, err := c.Request.FormFile("backup")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "File backup wajib diupload")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		response.InternalError(c, "Gagal membaca file backup")
		return
	}

	var payload struct {
		Settings map[string]string `json:"settings"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Format file backup tidak valid")
		return
	}

	if len(payload.Settings) == 0 {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Backup tidak berisi pengaturan")
		return
	}

	// Whitelist of restorable setting keys — sensitive keys like ai_api_key are excluded
	allowedKeys := map[string]bool{
		"app_name":          true,
		"app_logo":          true,
		"app_description":   true,
		"institution_name":  true,
		"institution_logo":  true,
		"ai_api_url":        true,
		"ai_api_header":     true,
		"ai_model_params":   true,
		"ai_prompt_analyze": true,
		"ai_prompt_grade":   true,
		"exam_auto_submit":  true,
		"exam_max_violations": true,
		"exam_violation_action": true,
		"backup_auto":       true,
		"backup_interval":   true,
	}

	filtered := make(map[string]string)
	skipped := 0
	for k, v := range payload.Settings {
		if allowedKeys[k] {
			filtered[k] = v
		} else {
			skipped++
		}
	}

	if len(filtered) == 0 {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Tidak ada pengaturan yang dapat dipulihkan")
		return
	}

	if err := h.settingRepo.SetMultiple(filtered); err != nil {
		response.InternalError(c, "Gagal memulihkan pengaturan")
		return
	}

	response.Success(c, gin.H{"restored": len(filtered), "skipped": skipped, "message": "Pengaturan berhasil dipulihkan"})
}

// POST /settings/ai/test — test AI API connection
func (h *BackupHandler) TestAIConnection(c *gin.Context) {
	apiURL, _ := h.settingRepo.GetByKey("ai_api_url")
	apiKey, _ := h.settingRepo.GetByKey("ai_api_key")
	apiHeader, _ := h.settingRepo.GetByKey("ai_api_header")
	modelParams, _ := h.settingRepo.GetByKey("ai_model_params")

	if apiURL == nil || apiURL.Value == nil || *apiURL.Value == "" {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "AI API URL belum dikonfigurasi")
		return
	}

	if err := urlvalidator.ValidateExternalURL(*apiURL.Value); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", fmt.Sprintf("AI API URL tidak aman: %v", err))
		return
	}

	headerName := "Authorization"
	if apiHeader != nil && apiHeader.Value != nil && *apiHeader.Value != "" {
		headerName = *apiHeader.Value
	}

	var params map[string]interface{}
	if modelParams != nil && modelParams.Value != nil {
		_ = json.Unmarshal([]byte(*modelParams.Value), &params)
	}
	if params == nil {
		params = make(map[string]interface{})
	}

	// Build a minimal test request (OpenAI-compatible format)
	testBody := map[string]interface{}{
		"model": func() string {
			if m, ok := params["model"].(string); ok {
				return m
			}
			return "gpt-3.5-turbo"
		}(),
		"messages": []map[string]string{
			{"role": "user", "content": "Balas dengan 'OK' saja untuk tes koneksi."},
		},
		"max_tokens": 10,
	}

	bodyBytes, _ := json.Marshal(testBody)
	req, err := http.NewRequest("POST", *apiURL.Value, bytes.NewReader(bodyBytes))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "URL AI tidak valid")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != nil && apiKey.Value != nil && *apiKey.Value != "" {
		req.Header.Set(headerName, "Bearer "+*apiKey.Value)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "AI_ERROR", fmt.Sprintf("Gagal terhubung ke AI: %v", err))
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logger.Log.Errorf("AI API error (%d): %s", resp.StatusCode, string(respBody))
		response.Error(c, http.StatusBadGateway, "AI_ERROR", fmt.Sprintf("AI mengembalikan error (HTTP %d). Periksa konfigurasi API.", resp.StatusCode))
		return
	}

	response.Success(c, gin.H{"status": "ok", "http_status": resp.StatusCode, "message": "Koneksi AI berhasil"})
}
