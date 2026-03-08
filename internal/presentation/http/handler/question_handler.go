package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/application/dto"
	questionuc "github.com/omanjaya/patra/internal/application/usecase/question"
	"github.com/omanjaya/patra/pkg/audio"
	"github.com/omanjaya/patra/pkg/ginhelper"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/response"
)

type QuestionHandler struct {
	uc *questionuc.QuestionUseCase
}

func NewQuestionHandler(uc *questionuc.QuestionUseCase) *QuestionHandler {
	return &QuestionHandler{uc: uc}
}

func (h *QuestionHandler) List(c *gin.Context) {
	bankID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	p := pagination.FromQuery(c)
	questions, total, err := h.uc.List(bankID, p)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	ginhelper.RespondPaginated(c, questions, p, total)
}

func (h *QuestionHandler) Create(c *gin.Context) {
	bankID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}

	var req dto.CreateQuestionRequest

	// Support both JSON and multipart form data
	contentType := c.ContentType()
	if contentType == "application/json" {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	} else {
		// Multipart form data
		req.QuestionType = c.PostForm("question_type")
		req.Body = c.PostForm("body")
		req.Difficulty = c.PostForm("difficulty")
		if s := c.PostForm("score"); s != "" {
			v, _ := strconv.ParseFloat(s, 64)
			req.Score = v
		}
		if s := c.PostForm("order_index"); s != "" {
			v, _ := strconv.Atoi(s)
			req.OrderIndex = v
		}
		if s := c.PostForm("stimulus_id"); s != "" {
			v, _ := strconv.ParseUint(s, 10, 64)
			uid := uint(v)
			req.StimulusID = &uid
		}
		if s := c.PostForm("options"); s != "" {
			var opts interface{}
			if err := json.Unmarshal([]byte(s), &opts); err != nil {
				response.BadRequest(c, "options: invalid JSON")
				return
			}
			req.Options = json.RawMessage(s)
		}
		if s := c.PostForm("correct_answer"); s != "" {
			var ans interface{}
			if err := json.Unmarshal([]byte(s), &ans); err != nil {
				response.BadRequest(c, "correct_answer: invalid JSON")
				return
			}
			req.CorrectAnswer = json.RawMessage(s)
		}
		if s := c.PostForm("audio_limit"); s != "" {
			v, _ := strconv.Atoi(s)
			req.AudioLimit = v
		}
		if s := c.PostForm("bloom_level"); s != "" {
			v, _ := strconv.Atoi(s)
			req.BloomLevel = v
		}
		req.TopicCode = c.PostForm("topic_code")

		if req.QuestionType == "" || req.Body == "" {
			response.BadRequest(c, "question_type dan body wajib diisi")
			return
		}
	}

	// Handle audio file upload
	audioPath, err := h.handleAudioUpload(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if audioPath != "" {
		req.AudioPath = &audioPath
	}

	q, err := h.uc.Create(bankID, req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, q)
}

func (h *QuestionHandler) Update(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}

	var req dto.UpdateQuestionRequest

	contentType := c.ContentType()
	if contentType == "application/json" {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	} else {
		// Multipart form data
		req.QuestionType = c.PostForm("question_type")
		req.Body = c.PostForm("body")
		req.Difficulty = c.PostForm("difficulty")
		if s := c.PostForm("score"); s != "" {
			v, _ := strconv.ParseFloat(s, 64)
			req.Score = v
		}
		if s := c.PostForm("order_index"); s != "" {
			v, _ := strconv.Atoi(s)
			req.OrderIndex = v
		}
		if s := c.PostForm("stimulus_id"); s != "" {
			v, _ := strconv.ParseUint(s, 10, 64)
			uid := uint(v)
			req.StimulusID = &uid
		}
		if s := c.PostForm("options"); s != "" {
			var opts interface{}
			if err := json.Unmarshal([]byte(s), &opts); err != nil {
				response.BadRequest(c, "options: invalid JSON")
				return
			}
			req.Options = json.RawMessage(s)
		}
		if s := c.PostForm("correct_answer"); s != "" {
			var ans interface{}
			if err := json.Unmarshal([]byte(s), &ans); err != nil {
				response.BadRequest(c, "correct_answer: invalid JSON")
				return
			}
			req.CorrectAnswer = json.RawMessage(s)
		}
		if s := c.PostForm("audio_limit"); s != "" {
			v, _ := strconv.Atoi(s)
			req.AudioLimit = v
		}
		if c.PostForm("remove_audio") == "true" {
			req.RemoveAudio = true
		}
		if s := c.PostForm("bloom_level"); s != "" {
			v, _ := strconv.Atoi(s)
			req.BloomLevel = v
		}
		req.TopicCode = c.PostForm("topic_code")
	}

	// Handle audio file upload
	audioPath, err := h.handleAudioUpload(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if audioPath != "" {
		req.AudioPath = &audioPath
	}

	q, err := h.uc.Update(id, req)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, q)
}

// handleAudioUpload processes the "audio" file field from a multipart form.
// Returns the generated filename or empty string if no audio uploaded.
func (h *QuestionHandler) handleAudioUpload(c *gin.Context) (string, error) {
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		// No file uploaded — not an error
		return "", nil
	}
	defer file.Close()

	if header.Size > int64(audio.MaxFileSize) {
		return "", fmt.Errorf("ukuran file audio maksimal 10MB")
	}

	if !audio.IsAllowedType(header.Filename) {
		return "", fmt.Errorf("format audio tidak didukung (gunakan mp3, wav, m4a, ogg, aac)")
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("gagal membaca file audio")
	}

	filename := audio.GenerateFilename(header.Filename)
	if err := audio.SaveFile(data, filename); err != nil {
		return "", fmt.Errorf("gagal menyimpan file audio")
	}

	return filename, nil
}

func (h *QuestionHandler) Reorder(c *gin.Context) {
	bankID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.uc.Reorder(bankID, req.Items); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Urutan soal berhasil disimpan"})
}

func (h *QuestionHandler) Delete(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.Delete(id); err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Soal dihapus"})
}

func (h *QuestionHandler) BulkAction(c *gin.Context) {
	var body struct {
		Action       string `json:"action" binding:"required"`
		IDs          []uint `json:"ids" binding:"required"`
		TargetBankID *uint  `json:"target_bank_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var err error
	switch body.Action {
	case "delete":
		err = h.uc.BulkDelete(body.IDs)
	case "move":
		if body.TargetBankID == nil {
			response.BadRequest(c, "target_bank_id wajib diisi untuk aksi pindah")
			return
		}
		err = h.uc.MoveToBank(body.IDs, *body.TargetBankID)
	case "copy":
		if body.TargetBankID == nil {
			response.BadRequest(c, "target_bank_id wajib diisi untuk aksi salin")
			return
		}
		err = h.uc.CopyToBank(body.IDs, *body.TargetBankID)
	default:
		response.BadRequest(c, "Aksi tidak valid")
		return
	}

	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// PrintQuestions returns all questions for a bank (no pagination) for print view.
func (h *QuestionHandler) PrintQuestions(c *gin.Context) {
	bankID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	questions, err := h.uc.ListAllByBank(bankID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, questions)
}

// GetAllIDs returns all question IDs for a bank (ordered), used for navigation.
func (h *QuestionHandler) GetAllIDs(c *gin.Context) {
	bankID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	search := c.Query("q")
	ids, err := h.uc.ListIDsByBank(bankID, search)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, ids)
}

// Stimulus

func (h *QuestionHandler) ListStimuli(c *gin.Context) {
	bankID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	stimuli, err := h.uc.ListStimuli(bankID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, stimuli)
}

func (h *QuestionHandler) CreateStimulus(c *gin.Context) {
	bankID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateStimulusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	s, err := h.uc.CreateStimulus(bankID, req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, s)
}

func (h *QuestionHandler) UpdateStimulus(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "stimulusId")
	if !ok {
		return
	}
	var req dto.UpdateStimulusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	s, err := h.uc.UpdateStimulus(id, req)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, s)
}

func (h *QuestionHandler) DeleteStimulus(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "stimulusId")
	if !ok {
		return
	}
	if err := h.uc.DeleteStimulus(id); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Stimulus dihapus"})
}
