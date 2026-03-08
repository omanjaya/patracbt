package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/application/dto"
	examuc "github.com/omanjaya/patra/internal/application/usecase/exam"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/ginhelper"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/response"
	"github.com/redis/go-redis/v9"
)

type ExamScheduleHandler struct {
	uc          *examuc.ExamScheduleUseCase
	questionRepo repository.QuestionRepository
	redis       *redis.Client
}

func NewExamScheduleHandler(uc *examuc.ExamScheduleUseCase, questionRepo repository.QuestionRepository, rdb *redis.Client) *ExamScheduleHandler {
	return &ExamScheduleHandler{uc: uc, questionRepo: questionRepo, redis: rdb}
}

func (h *ExamScheduleHandler) List(c *gin.Context) {
	p := pagination.FromQuery(c)
	filter := repository.ExamScheduleFilter{
		Search: c.Query("search"),
		Status: c.Query("status"),
	}
	schedules, total, err := h.uc.List(filter, p)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	ginhelper.RespondPaginated(c, schedules, p, total)
}

func (h *ExamScheduleHandler) GetByID(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	s, err := h.uc.GetByID(id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, s)
}

func (h *ExamScheduleHandler) Create(c *gin.Context) {
	var req dto.CreateExamScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	userID := c.GetUint("user_id")
	s, err := h.uc.Create(req, userID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Created(c, s)
}

func (h *ExamScheduleHandler) Update(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateExamScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	s, err := h.uc.Update(id, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	// Invalidate cached questions for this schedule after update
	ctx := context.Background()
	pattern := fmt.Sprintf("exam:cache:%d:*", id)
	iter := h.redis.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		h.redis.Del(ctx, iter.Val())
	}

	response.Success(c, s)
}

func (h *ExamScheduleHandler) UpdateStatus(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var body struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if err := h.uc.UpdateStatus(id, body.Status); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Status diperbarui"})
}

func (h *ExamScheduleHandler) Delete(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.Delete(id); err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Jadwal dihapus"})
}

func (h *ExamScheduleHandler) ListTrashed(c *gin.Context) {
	p := pagination.FromQuery(c)
	filter := repository.ExamScheduleFilter{
		Search: c.Query("search"),
	}
	schedules, total, err := h.uc.ListTrashed(filter, p)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	ginhelper.RespondPaginated(c, schedules, p, total)
}

func (h *ExamScheduleHandler) Restore(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.Restore(id); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Jadwal dipulihkan"})
}

func (h *ExamScheduleHandler) ForceDelete(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	if err := h.uc.ForceDelete(id); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Jadwal dihapus permanen"})
}

func (h *ExamScheduleHandler) Clone(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	userID := c.GetUint("user_id")
	s, err := h.uc.Clone(id, userID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Created(c, s)
}

// Preview returns schedule info with sample questions from each bank for teacher preview.
func (h *ExamScheduleHandler) Preview(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}

	schedule, err := h.uc.GetByID(id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	type PreviewQuestion struct {
		ID            uint        `json:"id"`
		QuestionType  string      `json:"question_type"`
		Body          string      `json:"body"`
		Score         float64     `json:"score"`
		Difficulty    string      `json:"difficulty"`
		Options       interface{} `json:"options"`
		CorrectAnswer interface{} `json:"correct_answer"`
		OrderIndex    int         `json:"order_index"`
	}

	type PreviewBank struct {
		QuestionBankID   uint              `json:"question_bank_id"`
		QuestionBankName string            `json:"question_bank_name"`
		TotalQuestions   int64             `json:"total_questions"`
		QuestionCount    int               `json:"question_count"`
		Weight           float64           `json:"weight"`
		SampleQuestions  []PreviewQuestion `json:"sample_questions"`
	}

	banks := make([]PreviewBank, 0)
	for _, bank := range schedule.QuestionBanks {
		p := pagination.Params{Page: 1, PerPage: 5}
		questions, total, err := h.questionRepo.ListByBank(bank.QuestionBankID, p)
		if err != nil {
			response.InternalError(c, "Gagal mengambil soal: "+err.Error())
			return
		}

		samples := make([]PreviewQuestion, 0, len(questions))
		for _, q := range questions {
			samples = append(samples, PreviewQuestion{
				ID:            q.ID,
				QuestionType:  q.QuestionType,
				Body:          q.Body,
				Score:         q.Score,
				Difficulty:    q.Difficulty,
				Options:       q.Options,
				CorrectAnswer: q.CorrectAnswer,
				OrderIndex:    q.OrderIndex,
			})
		}

		bankName := ""
		if bank.QuestionBank.Name != "" {
			bankName = bank.QuestionBank.Name
		}

		banks = append(banks, PreviewBank{
			QuestionBankID:   bank.QuestionBankID,
			QuestionBankName: bankName,
			TotalQuestions:   total,
			QuestionCount:    bank.QuestionCount,
			Weight:           bank.Weight,
			SampleQuestions:  samples,
		})
	}

	response.Success(c, gin.H{
		"schedule": schedule,
		"banks":    banks,
	})
}

// WarmCache loads all questions for an exam schedule into Redis cache.
func (h *ExamScheduleHandler) WarmCache(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}

	schedule, err := h.uc.GetByID(id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	// Collect all questions from all question banks in this schedule
	allQuestions := make([]interface{}, 0)
	for _, bank := range schedule.QuestionBanks {
		p := pagination.Params{Page: 1, PerPage: 10000}
		questions, _, err := h.questionRepo.ListByBank(bank.QuestionBankID, p)
		if err != nil {
			response.InternalError(c, "Gagal mengambil soal: "+err.Error())
			return
		}
		for _, q := range questions {
			allQuestions = append(allQuestions, q)
		}
	}

	data, err := json.Marshal(allQuestions)
	if err != nil {
		response.InternalError(c, "Gagal encode soal: "+err.Error())
		return
	}

	cacheKey := fmt.Sprintf("exam:cache:%d:questions", id)
	ctx := context.Background()
	if err := h.redis.Set(ctx, cacheKey, string(data), 24*time.Hour).Err(); err != nil {
		response.InternalError(c, "Gagal menyimpan cache: "+err.Error())
		return
	}

	response.Success(c, gin.H{"cached": len(allQuestions), "status": "ok"})
}

// CacheStatus checks whether the cache for an exam schedule exists.
func (h *ExamScheduleHandler) CacheStatus(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}

	cacheKey := fmt.Sprintf("exam:cache:%d:questions", id)
	ctx := context.Background()

	val, err := h.redis.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		response.Success(c, gin.H{"cached": false, "count": 0, "expires_at": nil})
		return
	}
	if err != nil {
		response.InternalError(c, "Gagal memeriksa cache: "+err.Error())
		return
	}

	// Count items — if cache data is corrupted, return degraded response
	items := make([]interface{}, 0)
	if err := json.Unmarshal([]byte(val), &items); err != nil {
		logger.Log.Errorf("CacheStatus: failed to unmarshal cache for schedule %d: %v", id, err)
		response.Success(c, gin.H{"cached": true, "count": -1, "expires_at": nil, "warning": "cache data corrupted"})
		return
	}

	ttl, _ := h.redis.TTL(ctx, cacheKey).Result()
	expiresAt := time.Now().Add(ttl).UTC().Format(time.RFC3339)

	response.Success(c, gin.H{"cached": true, "count": len(items), "expires_at": expiresAt})
}
