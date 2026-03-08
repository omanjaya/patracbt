package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	aiuc "github.com/omanjaya/patra/internal/application/usecase/ai"
	reportuc "github.com/omanjaya/patra/internal/application/usecase/report"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/ginhelper"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/response"
)

type ReportHandler struct {
	uc           *reportuc.ReportUseCase
	gradingUC    *aiuc.GradingUseCase
	questionRepo repository.QuestionRepository
	sessionRepo  repository.ExamSessionRepository
	userRepo     repository.UserRepository
}

func NewReportHandler(
	uc *reportuc.ReportUseCase,
	gradingUC *aiuc.GradingUseCase,
	questionRepo repository.QuestionRepository,
	sessionRepo repository.ExamSessionRepository,
	userRepo repository.UserRepository,
) *ReportHandler {
	return &ReportHandler{
		uc:           uc,
		gradingUC:    gradingUC,
		questionRepo: questionRepo,
		sessionRepo:  sessionRepo,
		userRepo:     userRepo,
	}
}

// GET /reports/:scheduleId
func (h *ReportHandler) GetScheduleReport(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}
	report, err := h.uc.GetScheduleReport(id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, report)
}

// GET /reports/:scheduleId/sessions/:sessionId
func (h *ReportHandler) GetPersonalReport(c *gin.Context) {
	sessionID, ok := ginhelper.ParseID(c, "sessionId")
	if !ok {
		return
	}
	report, err := h.uc.GetPersonalReport(sessionID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, report)
}

// GET /exam/sessions/:id/report — student-facing personal report
func (h *ReportHandler) GetMyReport(c *gin.Context) {
	sessionID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}

	// Verify ownership: the session must belong to the requesting user
	userID := c.GetUint("user_id")
	session, err := h.sessionRepo.FindByID(sessionID)
	if err != nil {
		response.NotFound(c, "Sesi ujian tidak ditemukan")
		return
	}
	if session.UserID != userID {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Anda tidak memiliki akses ke sesi ini")
		return
	}

	// Check allow_see_result on the schedule
	if !session.ExamSchedule.AllowSeeResult {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Pembahasan belum diizinkan untuk ujian ini")
		return
	}

	// Check session is finished
	if session.Status != "finished" && session.Status != "terminated" {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Ujian belum selesai")
		return
	}

	report, err := h.uc.GetPersonalReport(sessionID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, report)
}

// GET /reports/:scheduleId/analysis
func (h *ReportHandler) GetExamAnalysis(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}
	analysis, err := h.uc.GetExamAnalysis(id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, analysis)
}

// POST /reports/:scheduleId/regrade
func (h *ReportHandler) Regrade(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	// Get requesting user ID from JWT context
	userID := uint(0)
	if uid, exists := c.Get("user_id"); exists {
		if v, ok := uid.(float64); ok {
			userID = uint(v)
		} else if v, ok := uid.(uint); ok {
			userID = v
		}
	}

	summary, err := h.uc.RegradeSchedule(id, userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, summary)
}

// GET /reports/:scheduleId/regrade-logs
func (h *ReportHandler) GetRegradeLogs(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}
	logs, err := h.uc.ListRegradeLogs(id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// Resolve user names for requested_by
	type RegradeLogDTO struct {
		ID            uint        `json:"id"`
		ScheduleID    uint        `json:"exam_schedule_id"`
		RequestedBy   uint        `json:"requested_by"`
		RequestedName string      `json:"requested_name"`
		SessionsCount int         `json:"sessions_count"`
		ScoreChanges  interface{} `json:"score_changes"`
		CreatedAt     string      `json:"created_at"`
	}

	result := make([]RegradeLogDTO, 0, len(logs))
	for _, l := range logs {
		name := "Unknown"
		if user, err := h.userRepo.FindByID(l.RequestedBy); err == nil && user != nil {
			name = user.Name
		}

		// Parse score_changes from JSON
		var changes interface{}
		if len(l.ScoreChanges) > 0 {
			_ = json.Unmarshal(l.ScoreChanges, &changes)
		}

		result = append(result, RegradeLogDTO{
			ID:            l.ID,
			ScheduleID:    l.ExamScheduleID,
			RequestedBy:   l.RequestedBy,
			RequestedName: name,
			SessionsCount: l.SessionsCount,
			ScoreChanges:  changes,
			CreatedAt:     l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	response.Success(c, result)
}

// GET /reports/:scheduleId/key-changes
func (h *ReportHandler) GetKeyChanges(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}
	changes, err := h.uc.GetKeyChanges(id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, changes)
}

// POST /exam-sessions/:id/grade-essay
func (h *ReportHandler) GradeEssay(c *gin.Context) {
	sessionID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req struct {
		QuestionID uint    `json:"question_id" binding:"required"`
		Score      float64 `json:"score" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if err := h.uc.SetEssayScore(sessionID, req.QuestionID, req.Score); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"graded": true})
}

// POST /exam-sessions/:id/ai-grade
func (h *ReportHandler) AIGradeEssay(c *gin.Context) {
	sessionID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req struct {
		QuestionID uint   `json:"question_id" binding:"required"`
		Answer     string `json:"answer" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	question, err := h.questionRepo.FindByID(req.QuestionID)
	if err != nil || question == nil {
		response.NotFound(c, "Soal tidak ditemukan")
		return
	}

	result, err := h.gradingUC.GradeEssay(question, req.Answer)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// Optionally auto-apply the score
	_ = h.uc.SetEssayScore(sessionID, req.QuestionID, result.Score)

	response.Success(c, result)
}

// POST /exam-sessions/:id/ai-grade-batch
func (h *ReportHandler) AIGradeBatchEssay(c *gin.Context) {
	sessionID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}

	session, err := h.sessionRepo.FindByID(sessionID)
	if err != nil || session == nil {
		response.NotFound(c, "Sesi ujian tidak ditemukan")
		return
	}

	var order []uint
	_ = json.Unmarshal(session.QuestionOrder, &order)
	if len(order) == 0 {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Tidak ada soal di sesi ini")
		return
	}

	// Load all questions
	questions, err := h.questionRepo.FindByIDs(order)
	if err != nil {
		response.InternalError(c, "Gagal memuat soal")
		return
	}

	// Load all answers for this session
	answers, err := h.sessionRepo.GetAllAnswers(sessionID)
	if err != nil {
		response.InternalError(c, "Gagal memuat jawaban")
		return
	}
	answerMap := make(map[uint]string)
	for _, a := range answers {
		var parsed map[string]interface{}
		if err := json.Unmarshal(a.Answer, &parsed); err == nil {
			if text, ok := parsed["text"].(string); ok {
				answerMap[a.QuestionID] = text
			} else {
				answerMap[a.QuestionID] = string(a.Answer)
			}
		} else {
			answerMap[a.QuestionID] = string(a.Answer)
		}
	}

	type batchResult struct {
		QuestionID uint    `json:"question_id"`
		Score      float64 `json:"score"`
		Reason     string  `json:"reason"`
		Error      string  `json:"error,omitempty"`
	}

	results := make([]batchResult, 0)
	graded := 0
	for _, q := range questions {
		if q.QuestionType != "esai" {
			continue
		}
		ansText, hasAnswer := answerMap[q.ID]
		if !hasAnswer || ansText == "" || ansText == "null" || ansText == `""` {
			results = append(results, batchResult{
				QuestionID: q.ID,
				Score:      0,
				Reason:     "Siswa tidak menjawab",
			})
			continue
		}

		result, err := h.gradingUC.GradeEssay(q, ansText)
		if err != nil {
			logger.Log.Warnw("ai batch grade failed", "session_id", sessionID, "question_id", q.ID, "error", err.Error())
			results = append(results, batchResult{
				QuestionID: q.ID,
				Error:      err.Error(),
			})
			continue
		}

		// Auto-apply score
		if setErr := h.uc.SetEssayScore(sessionID, q.ID, result.Score); setErr != nil {
			logger.Log.Warnw("ai batch grade: failed to set score", "session_id", sessionID, "question_id", q.ID, "error", setErr.Error())
		}

		results = append(results, batchResult{
			QuestionID: q.ID,
			Score:      result.Score,
			Reason:     result.Reason,
		})
		graded++
	}

	logger.Log.Infow("ai batch grade completed", "session_id", sessionID, "graded", graded, "total", len(results))
	response.Success(c, results)
}
