package handler

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	questionuc "github.com/omanjaya/patra/internal/application/usecase/question"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	audioutils "github.com/omanjaya/patra/pkg/audio"
	"github.com/omanjaya/patra/pkg/ginhelper"
	"github.com/omanjaya/patra/pkg/response"
	"github.com/omanjaya/patra/pkg/types"
)

type QuestionExportHandler struct {
	questionUC *questionuc.QuestionUseCase
	bankRepo   repository.QuestionBankRepository
	questRepo  repository.QuestionRepository
}

func NewQuestionExportHandler(
	questionUC *questionuc.QuestionUseCase,
	bankRepo repository.QuestionBankRepository,
	questRepo repository.QuestionRepository,
) *QuestionExportHandler {
	return &QuestionExportHandler{
		questionUC: questionUC,
		bankRepo:   bankRepo,
		questRepo:  questRepo,
	}
}

// portableQuestion is the JSON structure stored in data.patra inside the ZIP.
type portableQuestion struct {
	QuestionType  string          `json:"question_type"`
	Body          string          `json:"body"`
	Score         float64         `json:"score"`
	Difficulty    string          `json:"difficulty"`
	Options       json.RawMessage `json:"options"`
	CorrectAnswer json.RawMessage `json:"correct_answer"`
	AudioPath     *string         `json:"audio_path"`
	AudioLimit    int             `json:"audio_limit"`
	BloomLevel    int             `json:"bloom_level"`
	TopicCode     string          `json:"topic_code"`
	OrderIndex    int             `json:"order_index"`
}

// ExportQuestionsZIP exports all questions from a bank as a .bnkpatra (ZIP) file.
// GET /question-banks/:id/export-zip
func (h *QuestionExportHandler) ExportQuestionsZIP(c *gin.Context) {
	bankID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}

	bank, err := h.bankRepo.FindByID(bankID)
	if err != nil {
		response.NotFound(c, "Bank soal tidak ditemukan")
		return
	}

	questions, err := h.questRepo.ListAllByBank(bankID)
	if err != nil {
		response.InternalError(c, "Gagal memuat soal: "+err.Error())
		return
	}

	// Build ZIP in memory
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	exportData := make([]portableQuestion, 0, len(questions))

	for _, q := range questions {
		pq := portableQuestion{
			QuestionType:  q.QuestionType,
			Body:          q.Body,
			Score:         q.Score,
			Difficulty:    q.Difficulty,
			Options:       json.RawMessage(q.Options),
			CorrectAnswer: json.RawMessage(q.CorrectAnswer),
			AudioLimit:    q.AudioLimit,
			BloomLevel:    q.BloomLevel,
			TopicCode:     q.TopicCode,
			OrderIndex:    q.OrderIndex,
		}

		// Handle audio files
		if q.AudioPath != nil && *q.AudioPath != "" {
			audioFullPath := filepath.Join("./uploads/audio", filepath.Base(*q.AudioPath))
			if data, err := os.ReadFile(audioFullPath); err == nil {
				zipAudioPath := "assets/audio/" + filepath.Base(*q.AudioPath)
				fw, err := zw.Create(zipAudioPath)
				if err == nil {
					fw.Write(data)
					pq.AudioPath = &zipAudioPath
				}
			}
		}

		// Handle embedded images in body
		pq.Body = h.processExportImages(q.Body, zw)

		// Handle embedded images in options
		if len(q.Options) > 0 {
			pq.Options = h.processExportOptionsImages(json.RawMessage(q.Options), zw)
		}

		exportData = append(exportData, pq)
	}

	// Encode data with obfuscation (same as Laravel: JSON -> gzip -> base64)
	jsonBytes, err := json.Marshal(exportData)
	if err != nil {
		zw.Close()
		response.InternalError(c, "Gagal mengenkode data soal")
		return
	}

	var gzBuf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&gzBuf, gzip.BestCompression)
	gz.Write(jsonBytes)
	gz.Close()

	encoded := base64.StdEncoding.EncodeToString(gzBuf.Bytes())

	// Write data.patra
	fw, _ := zw.Create("data.patra")
	fw.Write([]byte(encoded))

	// Write meta.info
	meta := map[string]string{
		"version":    "1.0",
		"app":        "CBT Patra",
		"created_at": time.Now().Format("2006-01-02 15:04:05"),
	}
	metaBytes, _ := json.Marshal(meta)
	fw2, _ := zw.Create("meta.info")
	fw2.Write(metaBytes)

	zw.Close()

	// Send as download
	slug := strings.ReplaceAll(strings.ToLower(bank.Name), " ", "-")
	filename := fmt.Sprintf("BANK-%s.bnkpatra", slug)

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/octet-stream")
	c.Data(http.StatusOK, "application/octet-stream", buf.Bytes())
}

// ImportQuestionsZIP imports questions from a .bnkpatra (ZIP) file.
// POST /question-banks/:id/import-zip
func (h *QuestionExportHandler) ImportQuestionsZIP(c *gin.Context) {
	bankID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}

	bank, err := h.bankRepo.FindByID(bankID)
	if err != nil {
		response.NotFound(c, "Bank soal tidak ditemukan")
		return
	}

	if h.bankRepo.IsBankUsedInSchedule(bank.ID) {
		response.BadRequest(c, "Bank soal terkunci (sedang dipakai dalam ujian)")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "File tidak ditemukan. Upload field 'file' berformat .bnkpatra")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".bnkpatra" && ext != ".zip" {
		response.BadRequest(c, "Format file salah. Harap upload file .bnkpatra")
		return
	}

	// Read entire file into memory for zip processing
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		response.InternalError(c, "Gagal membaca file upload")
		return
	}

	reader := bytes.NewReader(fileBytes)
	zr, err := zip.NewReader(reader, int64(len(fileBytes)))
	if err != nil {
		response.BadRequest(c, "File rusak atau format tidak valid")
		return
	}

	// Find and read data.patra
	var dataContent []byte
	zipFiles := make(map[string]*zip.File) // store zip entries for asset lookup

	for _, f := range zr.File {
		zipFiles[f.Name] = f
		if f.Name == "data.patra" {
			rc, err := f.Open()
			if err != nil {
				response.InternalError(c, "Gagal membuka data.patra")
				return
			}
			dataContent, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				response.InternalError(c, "Gagal membaca data.patra")
				return
			}
		}
	}

	if dataContent == nil {
		response.BadRequest(c, "File tidak valid (data.patra tidak ditemukan)")
		return
	}

	// Decode: base64 -> gzip decompress -> JSON
	decoded, err := base64.StdEncoding.DecodeString(string(dataContent))
	if err != nil {
		response.BadRequest(c, "File korup atau dimodifikasi (base64 decode gagal)")
		return
	}

	gzReader, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		response.BadRequest(c, "File korup atau dimodifikasi (gzip decode gagal)")
		return
	}
	jsonBytes, err := io.ReadAll(gzReader)
	gzReader.Close()
	if err != nil {
		response.BadRequest(c, "File korup atau dimodifikasi (gzip read gagal)")
		return
	}

	var importData []portableQuestion
	if err := json.Unmarshal(jsonBytes, &importData); err != nil {
		response.BadRequest(c, "Gagal mendekode data soal: "+err.Error())
		return
	}

	// Import questions
	imported := 0
	for i, pq := range importData {
		// Restore audio
		var audioPath *string
		if pq.AudioPath != nil && *pq.AudioPath != "" {
			if zf, ok := zipFiles[*pq.AudioPath]; ok {
				rc, err := zf.Open()
				if err == nil {
					data, err := io.ReadAll(rc)
					rc.Close()
					if err == nil {
						// Generate new audio filename
						newName := audioutils.GenerateFilename(filepath.Base(*pq.AudioPath))
						if saveErr := audioutils.SaveFile(data, newName); saveErr == nil {
							audioPath = &newName
						}
					}
				}
			}
		}

		// Restore images in body
		body := h.processImportImages(pq.Body, zipFiles)

		// Restore images in options
		var options types.JSON
		if len(pq.Options) > 0 {
			options = types.JSON(h.processImportOptionsImages(pq.Options, zipFiles))
		}

		q := &entity.Question{
			QuestionBankID: bankID,
			QuestionType:   pq.QuestionType,
			Body:           body,
			Score:          pq.Score,
			Difficulty:     pq.Difficulty,
			Options:        options,
			CorrectAnswer:  types.JSON(pq.CorrectAnswer),
			AudioPath:      audioPath,
			AudioLimit:     pq.AudioLimit,
			BloomLevel:     pq.BloomLevel,
			TopicCode:      pq.TopicCode,
			OrderIndex:     i,
		}

		if err := h.questRepo.Create(q); err == nil {
			imported++
		}
	}

	response.Success(c, gin.H{
		"message": fmt.Sprintf("Berhasil mengimpor %d soal beserta asetnya.", imported),
		"count":   imported,
	})
}

// --- Image processing helpers ---

var imgSrcRegex = regexp.MustCompile(`src="(/uploads/[^"]+)"`)
var zipImgSrcRegex = regexp.MustCompile(`src="(assets/images/[^"]+)"`)

// processExportImages finds local image references in HTML, adds them to the ZIP,
// and replaces src with relative ZIP paths.
func (h *QuestionExportHandler) processExportImages(html string, zw *zip.Writer) string {
	if html == "" {
		return html
	}
	return imgSrcRegex.ReplaceAllStringFunc(html, func(match string) string {
		subs := imgSrcRegex.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		relPath := subs[1] // e.g. /uploads/question-images/file.jpg
		realPath := "." + relPath

		data, err := os.ReadFile(realPath)
		if err != nil {
			return match
		}

		zipPath := "assets/images/" + filepath.Base(relPath)
		fw, err := zw.Create(zipPath)
		if err != nil {
			return match
		}
		fw.Write(data)
		return fmt.Sprintf(`src="%s"`, zipPath)
	})
}

// processImportImages restores images from ZIP and updates src to local paths.
func (h *QuestionExportHandler) processImportImages(html string, zipFiles map[string]*zip.File) string {
	if html == "" {
		return html
	}
	return zipImgSrcRegex.ReplaceAllStringFunc(html, func(match string) string {
		subs := zipImgSrcRegex.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		zipPath := subs[1]
		zf, ok := zipFiles[zipPath]
		if !ok {
			return `src="" alt="Image Broken"`
		}

		rc, err := zf.Open()
		if err != nil {
			return `src="" alt="Image Broken"`
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return `src="" alt="Image Broken"`
		}

		// Save to uploads directory
		ext := filepath.Ext(filepath.Base(zipPath))
		newName := fmt.Sprintf("import_%d_%s%s", time.Now().UnixNano(), uuid.New().String()[:8], ext)
		destDir := "./uploads/question-images"
		os.MkdirAll(destDir, 0755)
		destPath := filepath.Join(destDir, newName)

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return `src="" alt="Image Broken"`
		}

		return fmt.Sprintf(`src="/uploads/question-images/%s"`, newName)
	})
}

// processExportOptionsImages processes images inside options JSON.
func (h *QuestionExportHandler) processExportOptionsImages(raw json.RawMessage, zw *zip.Writer) json.RawMessage {
	s := string(raw)
	s = imgSrcRegex.ReplaceAllStringFunc(s, func(match string) string {
		subs := imgSrcRegex.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		relPath := subs[1]
		realPath := "." + relPath

		data, err := os.ReadFile(realPath)
		if err != nil {
			return match
		}

		zipPath := "assets/images/" + filepath.Base(relPath)
		fw, err := zw.Create(zipPath)
		if err != nil {
			return match
		}
		fw.Write(data)
		return fmt.Sprintf(`src="%s"`, zipPath)
	})
	return json.RawMessage(s)
}

// processImportOptionsImages restores images inside options JSON from ZIP.
func (h *QuestionExportHandler) processImportOptionsImages(raw json.RawMessage, zipFiles map[string]*zip.File) json.RawMessage {
	s := string(raw)
	s = zipImgSrcRegex.ReplaceAllStringFunc(s, func(match string) string {
		subs := zipImgSrcRegex.FindStringSubmatch(match)
		if len(subs) < 2 {
			return match
		}
		zipPath := subs[1]
		zf, ok := zipFiles[zipPath]
		if !ok {
			return `src="" alt="Image Broken"`
		}

		rc, err := zf.Open()
		if err != nil {
			return `src="" alt="Image Broken"`
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return `src="" alt="Image Broken"`
		}

		ext := filepath.Ext(filepath.Base(zipPath))
		newName := fmt.Sprintf("import_%d_%s%s", time.Now().UnixNano(), uuid.New().String()[:8], ext)
		destDir := "./uploads/question-images"
		os.MkdirAll(destDir, 0755)
		destPath := filepath.Join(destDir, newName)

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return `src="" alt="Image Broken"`
		}

		return fmt.Sprintf(`src="/uploads/question-images/%s"`, newName)
	})
	return json.RawMessage(s)
}
