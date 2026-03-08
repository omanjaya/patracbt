package audio

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var AllowedTypes = map[string]bool{
	".mp3": true, ".wav": true, ".m4a": true, ".ogg": true, ".aac": true,
}

const MaxFileSize = 10 * 1024 * 1024 // 10MB

func IsAllowedType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return AllowedTypes[ext]
}

func GenerateFilename(originalName string) string {
	b := make([]byte, 20)
	rand.Read(b)
	ext := strings.ToLower(filepath.Ext(originalName))
	return hex.EncodeToString(b) + ext
}

func SaveFile(data []byte, filename string) error {
	dir := "./uploads/audio"
	os.MkdirAll(dir, 0755)
	return os.WriteFile(filepath.Join(dir, filename), data, 0644)
}

func DeleteFile(filename string) error {
	if filename == "" {
		return nil
	}
	path := filepath.Join("./uploads/audio", filepath.Base(filename))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("gagal menghapus file audio: %w", err)
	}
	return nil
}
