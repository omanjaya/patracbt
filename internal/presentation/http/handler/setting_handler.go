package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omanjaya/patra/internal/application/dto"
	settinguc "github.com/omanjaya/patra/internal/application/usecase/setting"
	"github.com/omanjaya/patra/internal/domain/repository"
	miniostorage "github.com/omanjaya/patra/internal/infrastructure/storage/minio"
	ws "github.com/omanjaya/patra/internal/infrastructure/websocket"
	"github.com/omanjaya/patra/pkg/response"
	"github.com/redis/go-redis/v9"
)

// restoreProgress tracks the progress of a restore operation
type restoreProgress struct {
	mu       sync.RWMutex
	Progress map[string]*RestoreStatus // keyed by restoreId
}

// RestoreStatus holds restore operation status
type RestoreStatus struct {
	Status   string `json:"status"`   // "processing", "completed", "failed"
	Progress int    `json:"progress"` // 0-100
	Message  string `json:"message"`
}

var globalRestoreProgress = &restoreProgress{
	Progress: make(map[string]*RestoreStatus),
}

func (rp *restoreProgress) Set(id string, status *RestoreStatus) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.Progress[id] = status
}

func (rp *restoreProgress) Get(id string) *RestoreStatus {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	if s, ok := rp.Progress[id]; ok {
		return s
	}
	return nil
}

type SettingHandler struct {
	uc          *settinguc.SettingUseCase
	settingRepo repository.SettingRepository
	rdb         *redis.Client
	hub         *ws.Hub
}

func NewSettingHandler(uc *settinguc.SettingUseCase, settingRepo repository.SettingRepository, rdb *redis.Client, hub ...*ws.Hub) *SettingHandler {
	h := &SettingHandler{uc: uc, settingRepo: settingRepo, rdb: rdb}
	if len(hub) > 0 {
		h.hub = hub[0]
	}
	return h
}

func (h *SettingHandler) GetAll(c *gin.Context) {
	settings, err := h.uc.GetAll()
	if err != nil {
		response.InternalError(c, "Gagal mengambil pengaturan")
		return
	}
	response.Success(c, settings)
}

func (h *SettingHandler) Update(c *gin.Context) {
	var req dto.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	if err := h.uc.Update(req); err != nil {
		response.InternalError(c, "Gagal menyimpan pengaturan")
		return
	}
	response.Success(c, nil)
}

// GET /settings/panic-mode/status
func (h *SettingHandler) PanicModeStatus(c *gin.Context) {
	setting, err := h.settingRepo.GetByKey("panic_mode_active")
	if err != nil || setting == nil || setting.Value == nil {
		response.Success(c, gin.H{"active": false})
		return
	}
	response.Success(c, gin.H{"active": *setting.Value == "1"})
}

// POST /settings/panic-mode/activate
func (h *SettingHandler) PanicModeActivate(c *gin.Context) {
	if err := h.settingRepo.Set("panic_mode_active", "1"); err != nil {
		response.InternalError(c, "Gagal mengaktifkan panic mode")
		return
	}

	// Broadcast panic mode event to all connected clients
	if h.hub != nil {
		h.hub.BroadcastAll(ws.Message{
			Event: ws.EventPanicMode,
			Data: ws.PanicModePayload{
				Active:  true,
				Message: "Panic Mode diaktifkan. Semua ujian dihentikan sementara.",
			},
		})
	}

	response.Success(c, gin.H{"active": true, "message": "Panic Mode diaktifkan"})
}

// POST /settings/panic-mode/deactivate
func (h *SettingHandler) PanicModeDeactivate(c *gin.Context) {
	if err := h.settingRepo.Set("panic_mode_active", "0"); err != nil {
		response.InternalError(c, "Gagal menonaktifkan panic mode")
		return
	}

	// Broadcast panic mode deactivation to all connected clients
	if h.hub != nil {
		h.hub.BroadcastAll(ws.Message{
			Event: ws.EventPanicMode,
			Data: ws.PanicModePayload{
				Active:  false,
				Message: "Panic Mode dinonaktifkan. Ujian dapat dilanjutkan.",
			},
		})
	}

	response.Success(c, gin.H{"active": false, "message": "Panic Mode dinonaktifkan"})
}

// POST /settings/minio — save MinIO configuration
func (h *SettingHandler) SaveMinioSettings(c *gin.Context) {
	var req struct {
		Endpoint  string `json:"minio_endpoint"`
		Bucket    string `json:"minio_bucket"`
		AccessKey string `json:"minio_access_key"`
		SecretKey string `json:"minio_secret_key"`
		UseSSL    string `json:"minio_use_ssl"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	if req.UseSSL == "" {
		req.UseSSL = "0"
	}
	pairs := map[string]string{
		"minio_endpoint":   req.Endpoint,
		"minio_bucket":     req.Bucket,
		"minio_access_key": req.AccessKey,
		"minio_secret_key": req.SecretKey,
		"minio_use_ssl":    req.UseSSL,
	}
	if err := h.settingRepo.SetMultiple(pairs); err != nil {
		response.InternalError(c, "Gagal menyimpan konfigurasi MinIO")
		return
	}
	response.Success(c, gin.H{"message": "Konfigurasi MinIO berhasil disimpan"})
}

// GET /settings/minio/test — test MinIO connection
func (h *SettingHandler) TestMinioConnection(c *gin.Context) {
	client, err := h.newMinioClient()
	if err != nil {
		response.Success(c, gin.H{"success": false, "message": err.Error()})
		return
	}

	if err := client.Ping(c.Request.Context()); err != nil {
		response.Success(c, gin.H{"success": false, "message": fmt.Sprintf("Gagal terhubung ke MinIO: %v", err)})
		return
	}

	response.Success(c, gin.H{"success": true, "message": "Koneksi MinIO berhasil"})
}

// newMinioClient builds a MinIO SDK client from DB settings.
func (h *SettingHandler) newMinioClient() (*miniostorage.Client, error) {
	getVal := func(key string) string {
		s, err := h.settingRepo.GetByKey(key)
		if err != nil || s == nil || s.Value == nil {
			return ""
		}
		return *s.Value
	}

	cfg := &miniostorage.Config{
		Endpoint:  getVal("minio_endpoint"),
		Bucket:    getVal("minio_bucket"),
		AccessKey: getVal("minio_access_key"),
		SecretKey: getVal("minio_secret_key"),
		UseSSL:    getVal("minio_use_ssl") == "1",
	}

	return miniostorage.NewClient(cfg)
}

// GET /settings/backup/minio/list — list backup files in MinIO bucket
func (h *SettingHandler) ListMinioBackups(c *gin.Context) {
	client, err := h.newMinioClient()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	objects, err := client.List(c.Request.Context(), "backup_")
	if err != nil {
		response.Error(c, http.StatusBadGateway, "MINIO_ERROR", fmt.Sprintf("Gagal list backup di MinIO: %v", err))
		return
	}

	type fileInfo struct {
		Key          string `json:"key"`
		LastModified string `json:"last_modified"`
		Size         int64  `json:"size"`
	}
	files := make([]fileInfo, 0, len(objects))
	for _, obj := range objects {
		files = append(files, fileInfo{
			Key:          obj.Key,
			LastModified: obj.LastModified.Format(time.RFC3339),
			Size:         obj.Size,
		})
	}
	response.Success(c, files)
}

// POST /settings/backup/minio — create backup and upload to MinIO
func (h *SettingHandler) BackupToMinio(c *gin.Context) {
	settings, err := h.settingRepo.GetAll()
	if err != nil {
		response.InternalError(c, "Gagal mengambil pengaturan")
		return
	}

	client, err := h.newMinioClient()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	// Ensure bucket exists before uploading
	if err := client.EnsureBucket(c.Request.Context()); err != nil {
		response.Error(c, http.StatusBadGateway, "MINIO_ERROR", err.Error())
		return
	}

	data := make(map[string]string)
	for _, s := range settings {
		if s.Value != nil {
			if s.Key == "ai_api_key" || s.Key == "minio_secret_key" {
				data[s.Key] = ""
			} else {
				data[s.Key] = *s.Value
			}
		}
	}

	backupPayload := map[string]interface{}{
		"version":    "1.0",
		"created_at": time.Now().Format(time.RFC3339),
		"settings":   data,
	}
	payloadBytes, err := json.MarshalIndent(backupPayload, "", "  ")
	if err != nil {
		response.InternalError(c, "Gagal membuat backup")
		return
	}

	filename := fmt.Sprintf("backup_%s.json", time.Now().Format("20060102_150405"))
	reader := bytes.NewReader(payloadBytes)

	if err := client.Upload(c.Request.Context(), filename, reader, int64(len(payloadBytes)), "application/json"); err != nil {
		response.Error(c, http.StatusBadGateway, "MINIO_ERROR", fmt.Sprintf("Gagal upload ke MinIO: %v", err))
		return
	}

	response.Success(c, gin.H{"filename": filename, "message": "Backup berhasil diupload ke MinIO"})
}

// POST /settings/restore/minio — restore settings from a file in MinIO
func (h *SettingHandler) RestoreFromMinio(c *gin.Context) {
	var req struct {
		Filename string `json:"filename" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	if strings.Contains(req.Filename, "..") || strings.Contains(req.Filename, "/") {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Nama file tidak valid")
		return
	}

	client, err := h.newMinioClient()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	reader, err := client.Download(c.Request.Context(), req.Filename)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "MINIO_ERROR", fmt.Sprintf("Gagal download dari MinIO: %v", err))
		return
	}
	defer reader.Close()

	fileData, err := io.ReadAll(reader)
	if err != nil {
		response.InternalError(c, "Gagal membaca file dari MinIO")
		return
	}

	var payload struct {
		Settings map[string]string `json:"settings"`
	}
	if err := json.Unmarshal(fileData, &payload); err != nil || len(payload.Settings) == 0 {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Format file backup tidak valid")
		return
	}

	if err := h.settingRepo.SetMultiple(payload.Settings); err != nil {
		response.InternalError(c, "Gagal memulihkan pengaturan")
		return
	}

	response.Success(c, gin.H{"restored": len(payload.Settings), "message": "Pengaturan berhasil dipulihkan dari MinIO"})
}

// GET /api/v1/branding — public, no auth required
func (h *SettingHandler) GetBranding(c *gin.Context) {
	keys := []string{
		"app_name", "app_logo", "app_favicon", "app_footer_text",
		"app_primary_color", "app_header_bg", "login_bg_image",
		"login_subtitle", "school_name",
	}
	result := make(map[string]string)
	for _, key := range keys {
		s, err := h.settingRepo.GetByKey(key)
		if err == nil && s != nil && s.Value != nil {
			result[key] = *s.Value
		}
	}
	// Defaults
	if result["app_name"] == "" {
		result["app_name"] = "CBT Patra"
	}
	response.Success(c, result)
}

// POST /admin/settings/upload-branding — upload logo/favicon
func (h *SettingHandler) UploadBranding(c *gin.Context) {
	field := c.PostForm("field") // "app_logo" atau "app_favicon" atau "login_bg_image"
	allowed := map[string]bool{"app_logo": true, "app_favicon": true, "login_bg_image": true}
	if !allowed[field] {
		response.BadRequest(c, "Field tidak valid")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "File tidak ditemukan")
		return
	}

	// Validate file size (max 2MB)
	if file.Size > 2*1024*1024 {
		response.BadRequest(c, "Ukuran file maksimal 2MB")
		return
	}

	// Validate extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	validExts := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".svg": true, ".ico": true, ".webp": true}
	if !validExts[ext] {
		response.BadRequest(c, "Format file tidak didukung. Gunakan PNG, JPG, SVG, ICO, atau WebP")
		return
	}

	// Save file
	filename := fmt.Sprintf("%s_%d%s", field, time.Now().Unix(), ext)
	savePath := filepath.Join("uploads", "branding", filename)
	os.MkdirAll(filepath.Dir(savePath), 0755)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		response.InternalError(c, "Gagal menyimpan file")
		return
	}

	// Save URL to settings
	url := "/" + savePath
	if err := h.settingRepo.Set(field, url); err != nil {
		response.InternalError(c, "Gagal menyimpan pengaturan")
		return
	}

	response.Success(c, gin.H{"url": url, "field": field})
}

// POST /settings/system/clear-cache — clear redis cache
func (h *SettingHandler) ClearCache(c *gin.Context) {
	if h.rdb != nil {
		if err := h.rdb.FlushDB(c.Request.Context()).Err(); err != nil {
			response.InternalError(c, "Gagal membersihkan cache redis")
			return
		}
	}
	response.Success(c, gin.H{"message": "Cache sistem berhasil dibersihkan. Sistem kini lebih fresh."})
}

// =============================================================================
// Chunked Restore
// =============================================================================

const restoreTempDir = "/tmp/patra-restore"

// POST /admin/settings/restore/chunk — receives a chunk of a backup file
func (h *SettingHandler) UploadRestoreChunk(c *gin.Context) {
	batchID := c.PostForm("batch_id")
	if batchID == "" {
		response.BadRequest(c, "batch_id wajib diisi")
		return
	}
	// Sanitize batch_id to prevent path traversal
	if strings.Contains(batchID, "..") || strings.Contains(batchID, "/") || strings.Contains(batchID, "\\") {
		response.BadRequest(c, "batch_id tidak valid")
		return
	}

	chunk, err := c.FormFile("chunk")
	if err != nil {
		response.BadRequest(c, "File chunk wajib diupload")
		return
	}

	// Ensure temp directory exists
	if err := os.MkdirAll(restoreTempDir, 0755); err != nil {
		response.InternalError(c, "Gagal membuat direktori temp")
		return
	}

	partPath := filepath.Join(restoreTempDir, batchID+".part")

	// Open destination file in append mode
	dst, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		response.InternalError(c, "Gagal membuka file temp")
		return
	}
	defer dst.Close()

	// Open uploaded chunk
	src, err := chunk.Open()
	if err != nil {
		response.InternalError(c, "Gagal membaca chunk")
		return
	}
	defer src.Close()

	// Copy chunk data to destination (append)
	if _, err := io.Copy(dst, src); err != nil {
		response.InternalError(c, "Gagal menulis chunk ke file")
		return
	}

	response.Success(c, gin.H{"status": "success"})
}

// POST /admin/settings/restore/process — starts processing the assembled backup file
func (h *SettingHandler) ProcessRestore(c *gin.Context) {
	var req struct {
		BatchID string `json:"batch_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	// Sanitize
	if strings.Contains(req.BatchID, "..") || strings.Contains(req.BatchID, "/") {
		response.BadRequest(c, "batch_id tidak valid")
		return
	}

	partPath := filepath.Join(restoreTempDir, req.BatchID+".part")
	if _, err := os.Stat(partPath); os.IsNotExist(err) {
		response.NotFound(c, "File part tidak ditemukan. Upload ulang.")
		return
	}

	// Rename .part -> .patrabak
	finalPath := filepath.Join(restoreTempDir, req.BatchID+".patrabak")
	if err := os.Rename(partPath, finalPath); err != nil {
		response.InternalError(c, "Gagal memproses file backup")
		return
	}

	// Generate restore ID for progress tracking
	restoreID := uuid.New().String()

	// Set initial progress
	globalRestoreProgress.Set(restoreID, &RestoreStatus{
		Status:   "processing",
		Progress: 0,
		Message:  "Memulai proses restore...",
	})

	// Process restore asynchronously
	go h.executeChunkedRestore(finalPath, restoreID)

	response.Success(c, gin.H{
		"restore_id": restoreID,
		"message":    "Proses restore dimulai",
	})
}

// GET /admin/settings/restore/progress/:restoreId — returns current restore progress
func (h *SettingHandler) RestoreProgress(c *gin.Context) {
	restoreID := c.Param("restoreId")
	if restoreID == "" {
		response.BadRequest(c, "restore_id wajib diisi")
		return
	}

	// Try Redis first (if available)
	if h.rdb != nil {
		data, err := h.rdb.Get(c.Request.Context(), "restore_progress_"+restoreID).Result()
		if err == nil && data != "" {
			var status RestoreStatus
			if err := json.Unmarshal([]byte(data), &status); err == nil {
				response.Success(c, status)
				return
			}
		}
	}

	// Fallback to in-memory progress
	status := globalRestoreProgress.Get(restoreID)
	if status == nil {
		response.Success(c, RestoreStatus{
			Status:   "processing",
			Progress: 0,
			Message:  "Menunggu antrian...",
		})
		return
	}

	response.Success(c, status)
}

// executeChunkedRestore performs the actual restore from assembled backup file
func (h *SettingHandler) executeChunkedRestore(filePath, restoreID string) {
	updateProgress := func(progress int, status, message string) {
		rs := &RestoreStatus{
			Status:   status,
			Progress: progress,
			Message:  message,
		}
		globalRestoreProgress.Set(restoreID, rs)

		// Also store in Redis if available (10 min TTL)
		if h.rdb != nil {
			data, _ := json.Marshal(rs)
			h.rdb.Set(context.Background(), "restore_progress_"+restoreID, string(data), 10*time.Minute)
		}
	}

	defer func() {
		// Cleanup the backup file
		os.Remove(filePath)
	}()

	// Step 1: Read and parse the backup file (10-30%)
	updateProgress(10, "processing", "Membaca file backup...")

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		updateProgress(0, "failed", "Gagal membaca file backup: "+err.Error())
		return
	}

	updateProgress(30, "processing", "Parsing file backup...")

	var payload struct {
		Version   string            `json:"version"`
		CreatedAt string            `json:"created_at"`
		Settings  map[string]string `json:"settings"`
	}
	if err := json.Unmarshal(fileData, &payload); err != nil {
		updateProgress(0, "failed", "Format file backup tidak valid: "+err.Error())
		return
	}

	if len(payload.Settings) == 0 {
		updateProgress(0, "failed", "Backup tidak berisi pengaturan")
		return
	}

	// Step 2: Restore settings (30-90%)
	updateProgress(50, "processing", fmt.Sprintf("Memulihkan %d pengaturan...", len(payload.Settings)))

	if err := h.settingRepo.SetMultiple(payload.Settings); err != nil {
		updateProgress(0, "failed", "Gagal memulihkan pengaturan: "+err.Error())
		return
	}

	updateProgress(90, "processing", "Membersihkan cache...")

	// Step 3: Clear cache if Redis available
	if h.rdb != nil {
		// Don't flush the restore progress key
		_ = h.rdb.FlushDB(context.Background())
	}

	// Step 4: Done
	updateProgress(100, "completed", fmt.Sprintf("Selesai! %d pengaturan berhasil dipulihkan.", len(payload.Settings)))
}
