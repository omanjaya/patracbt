package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/pkg/response"
)

type AudioHandler struct{}

func NewAudioHandler() *AudioHandler {
	return &AudioHandler{}
}

// GET /audio-stream/:filename
// Streams audio files from ./uploads/audio/ directory
func (h *AudioHandler) Stream(c *gin.Context) {
	filename := c.Param("filename")

	// Security: reject filenames containing path separators or traversal sequences
	if strings.ContainsAny(filename, "/\\") || strings.Contains(filename, "..") {
		response.NotFound(c, "File tidak ditemukan")
		return
	}

	// Clean the filename to normalise any remaining oddities
	filename = filepath.Clean(filename)

	// Re-check after Clean (e.g. Clean("..") == "..")
	if strings.ContainsAny(filename, "/\\") || strings.Contains(filename, "..") {
		response.NotFound(c, "File tidak ditemukan")
		return
	}

	ext := strings.ToLower(filepath.Ext(filename))
	allowedExts := map[string]string{
		".mp3": "audio/mpeg",
		".wav": "audio/wav",
		".ogg": "audio/ogg",
		".m4a": "audio/mp4",
	}

	contentType, ok := allowedExts[ext]
	if !ok {
		response.BadRequest(c, "Format audio tidak didukung")
		return
	}

	// Resolve the expected uploads directory to an absolute path
	uploadsDir, err := filepath.Abs("./uploads/audio")
	if err != nil {
		response.InternalError(c, "Gagal menentukan direktori upload")
		return
	}

	filePath := filepath.Join(uploadsDir, filename)

	// Resolve symlinks so we can verify the real location
	resolvedPath, err := filepath.EvalSymlinks(filePath)
	if err != nil {
		response.NotFound(c, "File audio tidak ditemukan")
		return
	}

	// Ensure the resolved path is still within the uploads directory
	if !strings.HasPrefix(resolvedPath, uploadsDir+string(os.PathSeparator)) && resolvedPath != uploadsDir {
		response.NotFound(c, "File tidak ditemukan")
		return
	}

	f, err := os.Open(resolvedPath)
	if err != nil {
		response.NotFound(c, "File audio tidak ditemukan")
		return
	}
	defer f.Close()

	stat, _ := f.Stat()
	c.Header("Content-Type", contentType)
	c.Header("Accept-Ranges", "bytes")
	http.ServeContent(c.Writer, c.Request, filename, stat.ModTime(), f)
}
