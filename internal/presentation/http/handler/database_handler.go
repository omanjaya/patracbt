package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/config"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/response"
)

type DatabaseHandler struct {
	cfg *config.Config
}

func NewDatabaseHandler(cfg *config.Config) *DatabaseHandler {
	return &DatabaseHandler{cfg: cfg}
}

// buildPgEnv returns environment variables for pg_dump/pg_restore with PGPASSWORD set.
func (h *DatabaseHandler) buildPgEnv() []string {
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", h.cfg.DB.Password))
	return env
}

// buildConnArgs returns common connection arguments for pg_dump/pg_restore.
func (h *DatabaseHandler) buildConnArgs() []string {
	return []string{
		"-h", h.cfg.DB.Host,
		"-p", h.cfg.DB.Port,
		"-U", h.cfg.DB.Username,
		"-d", h.cfg.DB.Database,
	}
}

// ExportDatabase streams a pg_dump custom-format backup as a file download.
// POST /admin/settings/database/export
func (h *DatabaseHandler) ExportDatabase(c *gin.Context) {
	if _, err := exec.LookPath("pg_dump"); err != nil {
		logger.Log.Errorf("Database export: pg_dump not found: %v", err)
		response.InternalError(c, "pg_dump tidak ditemukan di server")
		return
	}

	args := append([]string{"-Fc"}, h.buildConnArgs()...)
	cmd := exec.Command("pg_dump", args...)
	cmd.Env = h.buildPgEnv()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Log.Errorf("Database export: failed to create stdout pipe: %v", err)
		response.InternalError(c, "Gagal memulai proses export")
		return
	}

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		logger.Log.Errorf("Database export: failed to start pg_dump: %v", err)
		response.InternalError(c, "Gagal menjalankan pg_dump")
		return
	}

	filename := fmt.Sprintf("cbt_patra_%s.dump", time.Now().Format("20060102_150405"))

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	if _, err := io.Copy(c.Writer, stdout); err != nil {
		logger.Log.Errorf("Database export: failed to stream output: %v", err)
		// Headers already sent, cannot return JSON error
		return
	}

	if err := cmd.Wait(); err != nil {
		logger.Log.Errorf("Database export: pg_dump failed: %v, stderr: %s", err, stderrBuf.String())
		// Headers already sent at this point; log only
		return
	}

	logger.Log.Infow("Database exported successfully", "filename", filename)
}

// ImportDatabase restores a database from an uploaded pg_dump custom-format file.
// POST /admin/settings/database/import
func (h *DatabaseHandler) ImportDatabase(c *gin.Context) {
	if _, err := exec.LookPath("pg_restore"); err != nil {
		logger.Log.Errorf("Database import: pg_restore not found: %v", err)
		response.InternalError(c, "pg_restore tidak ditemukan di server")
		return
	}

	// Limit to 500MB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 500<<20)

	file, header, err := c.Request.FormFile("backup")
	if err != nil {
		logger.Log.Errorf("Database import: failed to read uploaded file: %v", err)
		response.BadRequest(c, "File backup wajib diupload (max 500MB)")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".dump" && ext != ".sql" && ext != ".gz" {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Format file harus .dump, .sql, atau .gz")
		return
	}

	// Save to temp file
	tmpFile, err := os.CreateTemp("", "patra_import_*.dump")
	if err != nil {
		logger.Log.Errorf("Database import: failed to create temp file: %v", err)
		response.InternalError(c, "Gagal membuat file sementara")
		return
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, file); err != nil {
		tmpFile.Close()
		logger.Log.Errorf("Database import: failed to write temp file: %v", err)
		response.InternalError(c, "Gagal menyimpan file upload")
		return
	}
	tmpFile.Close()

	// Run pg_restore
	args := append([]string{"--clean", "--if-exists"}, h.buildConnArgs()...)
	args = append(args, tmpPath)
	cmd := exec.Command("pg_restore", args...)
	cmd.Env = h.buildPgEnv()

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Log.Errorf("Database import: pg_restore failed: %v, output: %s", err, string(output))
		response.InternalError(c, "Import database gagal. Silakan periksa format file backup.")
		return
	}

	logger.Log.Infow("Database imported successfully")
	response.Success(c, gin.H{"message": "Database berhasil diimport"})
}

// ListBackups returns a list of backup files in the ./backups/ directory.
// GET /admin/settings/database/backups
func (h *DatabaseHandler) ListBackups(c *gin.Context) {
	backupDir := "./backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		logger.Log.Errorf("ListBackups: failed to create backup dir: %v", err)
		response.InternalError(c, "Gagal mengakses direktori backup")
		return
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		logger.Log.Errorf("ListBackups: failed to read backup dir: %v", err)
		response.InternalError(c, "Gagal membaca direktori backup")
		return
	}

	type backupFile struct {
		Filename   string `json:"filename"`
		Size       int64  `json:"size"`
		SizeHuman  string `json:"size_human"`
		ModifiedAt string `json:"modified_at"`
	}

	backups := make([]backupFile, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".dump" && ext != ".sql" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, backupFile{
			Filename:   name,
			Size:       info.Size(),
			SizeHuman:  formatFileSize(info.Size()),
			ModifiedAt: info.ModTime().Format(time.RFC3339),
		})
	}

	// Sort by date descending (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModifiedAt > backups[j].ModifiedAt
	})

	response.Success(c, gin.H{"backups": backups})
}

// DeleteBackup removes a backup file from ./backups/.
// DELETE /admin/settings/database/backups/:filename
func (h *DatabaseHandler) DeleteBackup(c *gin.Context) {
	filename := c.Param("filename")

	// Security: prevent path traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		response.BadRequest(c, "Nama file tidak valid")
		return
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".dump" && ext != ".sql" {
		response.BadRequest(c, "Hanya file .dump dan .sql yang boleh dihapus")
		return
	}

	path := filepath.Join("backups", filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		response.NotFound(c, "File backup tidak ditemukan")
		return
	}

	if err := os.Remove(path); err != nil {
		logger.Log.Errorf("DeleteBackup: failed to delete %s: %v", filename, err)
		response.InternalError(c, "Gagal menghapus file backup")
		return
	}

	logger.Log.Infow("Backup deleted", "filename", filename)
	response.Success(c, gin.H{"message": "File backup berhasil dihapus"})
}

// ExportAndSave creates a pg_dump backup and saves it to ./backups/.
// POST /admin/settings/database/export-save
func (h *DatabaseHandler) ExportAndSave(c *gin.Context) {
	if _, err := exec.LookPath("pg_dump"); err != nil {
		logger.Log.Errorf("ExportAndSave: pg_dump not found: %v", err)
		response.InternalError(c, "pg_dump tidak ditemukan di server")
		return
	}

	backupDir := "./backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		logger.Log.Errorf("ExportAndSave: failed to create backup dir: %v", err)
		response.InternalError(c, "Gagal membuat direktori backup")
		return
	}

	filename := fmt.Sprintf("cbt_patra_%s.dump", time.Now().Format("20060102_150405"))
	filePath := filepath.Join(backupDir, filename)

	args := append([]string{"-Fc", "-f", filePath}, h.buildConnArgs()...)
	cmd := exec.Command("pg_dump", args...)
	cmd.Env = h.buildPgEnv()

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Log.Errorf("ExportAndSave: pg_dump failed: %v, output: %s", err, string(output))
		// Clean up partial file
		if removeErr := os.Remove(filePath); removeErr != nil {
			logger.Log.Errorf("ExportAndSave: failed to clean up partial file %s: %v", filePath, removeErr)
		}
		response.InternalError(c, "Export database gagal. Silakan hubungi administrator.")
		return
	}

	info, err := os.Stat(filePath)
	if err != nil {
		logger.Log.Errorf("ExportAndSave: failed to stat backup file: %v", err)
		response.InternalError(c, "Backup dibuat tapi gagal membaca info file")
		return
	}

	logger.Log.Infow("Database backup saved", "filename", filename, "size", info.Size())
	response.Success(c, gin.H{
		"filename":   filename,
		"size":       info.Size(),
		"size_human": formatFileSize(info.Size()),
	})
}

// formatFileSize returns a human-readable file size string.
func formatFileSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
