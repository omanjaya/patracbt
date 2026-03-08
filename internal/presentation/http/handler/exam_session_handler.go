package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/application/dto"
	examuc "github.com/omanjaya/patra/internal/application/usecase/exam"
	ws "github.com/omanjaya/patra/internal/infrastructure/websocket"
	"github.com/omanjaya/patra/pkg/ginhelper"
	"github.com/omanjaya/patra/pkg/response"
)

type ExamSessionHandler struct {
	uc  *examuc.ExamSessionUseCase
	hub *ws.Hub
}

func NewExamSessionHandler(uc *examuc.ExamSessionUseCase, hub *ws.Hub) *ExamSessionHandler {
	return &ExamSessionHandler{uc: uc, hub: hub}
}

// GET /exam/available
func (h *ExamSessionHandler) GetAvailable(c *gin.Context) {
	userID := c.GetUint("user_id")
	schedules, err := h.uc.GetAvailableExams(userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, schedules)
}

// GET /exam/history — student's finished sessions
func (h *ExamSessionHandler) GetMyHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	sessions, err := h.uc.GetMyHistory(userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, sessions)
}

// POST /exam/start
func (h *ExamSessionHandler) Start(c *gin.Context) {
	var req dto.StartExamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	userID := c.GetUint("user_id")
	result, err := h.uc.StartExam(userID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// GET /exam/sessions/:id
func (h *ExamSessionHandler) LoadSession(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	userID := c.GetUint("user_id")
	result, err := h.uc.LoadSession(id, userID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Success(c, result)
}

// POST /exam/sessions/:id/answers
func (h *ExamSessionHandler) SaveAnswer(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.SaveAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	userID := c.GetUint("user_id")
	session, answered, total, err := h.uc.SaveAnswer(id, userID, req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	// Broadcast WS event using batched delivery to reduce marshal+lock overhead
	if h.hub != nil && session != nil {
		h.hub.BroadcastAnswerToSupervisors(session.ExamScheduleID, ws.AnswerSavedPayload{
			SessionID:  session.ID,
			UserID:     userID,
			QuestionID: req.QuestionID,
			Answered:   answered,
			Total:      total,
		})
	}

	response.Success(c, gin.H{"saved": true})
}

// POST /exam/sessions/:id/finish
func (h *ExamSessionHandler) Finish(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	userID := c.GetUint("user_id")
	session, err := h.uc.FinishExam(id, userID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	// Emit WS event
	if h.hub != nil && session != nil {
		h.hub.BroadcastToSupervisors(session.ExamScheduleID, ws.Message{
			Event: ws.EventSessionFinished,
			Data: ws.SessionFinishedPayload{
				SessionID: session.ID,
				UserID:    userID,
				Score:     session.Score,
				MaxScore:  session.MaxScore,
			},
		})
	}

	response.Success(c, session)
}

// POST /exam/sessions/:id/answers/batch — offline sync / navigator.sendBeacon fallback
func (h *ExamSessionHandler) BatchSaveAnswers(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var answers []dto.SaveAnswerRequest
	if err := c.ShouldBindJSON(&answers); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	userID := c.GetUint("user_id")
	if err := h.uc.BatchSaveAnswers(id, userID, answers); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Success(c, gin.H{"saved": len(answers)})
}

// GET /exam/sessions/:id/transition — get next section info (multi-stage)
func (h *ExamSessionHandler) GetTransition(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	userID := c.GetUint("user_id")
	next, err := h.uc.GetTransition(id, userID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Success(c, next)
}

// POST /exam/sessions/:id/start-section — move to next section (multi-stage)
func (h *ExamSessionHandler) StartSection(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	userID := c.GetUint("user_id")
	result, err := h.uc.StartSection(id, userID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Success(c, result)
}

// POST /exam/sessions/:id/questions/:questionId/flag
func (h *ExamSessionHandler) ToggleFlag(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	questionID, ok := ginhelper.ParseID(c, "questionId")
	if !ok {
		return
	}
	userID := c.GetUint("user_id")
	req := dto.ToggleFlagRequest{
		QuestionID: questionID,
		IsFlagged:  true, // default to true
	}
	// Allow body to override is_flagged (for toggle off)
	var body struct {
		IsFlagged *bool `json:"is_flagged"`
	}
	if err := c.ShouldBindJSON(&body); err == nil && body.IsFlagged != nil {
		req.IsFlagged = *body.IsFlagged
	}
	if err := h.uc.ToggleFlag(id, userID, req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Success(c, gin.H{"is_flagged": req.IsFlagged})
}

// GET /exam/sessions/:id/lock-status
func (h *ExamSessionHandler) CheckLockStatus(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	userID := c.GetUint("user_id")
	status, err := h.uc.CheckLockStatus(id, userID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Success(c, gin.H{"status": status})
}

// POST /exam/sessions/:id/beacon-sync
func (h *ExamSessionHandler) BeaconSync(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.BeaconSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	userID := c.GetUint("user_id")
	saved, err := h.uc.BeaconSync(id, userID, req.Answers)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	response.Success(c, gin.H{"saved": saved})
}

// GET /exam-schedules/:id/sessions/ongoing
func (h *ExamSessionHandler) ListOngoingBySchedule(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	sessions, err := h.uc.ListOngoingBySchedule(scheduleID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, sessions)
}

// GET /exam-schedules/:id/sessions/not-started
func (h *ExamSessionHandler) ListNotStartedBySchedule(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	sessions, err := h.uc.ListNotStartedBySchedule(scheduleID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, sessions)
}

// POST /exam/sessions/:id/violations
func (h *ExamSessionHandler) LogViolation(c *gin.Context) {
	id, ok := ginhelper.ParseID(c, "id")
	if !ok {
		return
	}
	var req dto.LogViolationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	userID := c.GetUint("user_id")
	if err := h.uc.LogViolation(id, userID, req); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// Emit WS event
	if h.hub != nil {
		session, _ := h.uc.GetSessionByID(id)
		if session != nil {
			count, _ := h.uc.GetViolationCount(id)
			h.hub.BroadcastToSupervisors(session.ExamScheduleID, ws.Message{
				Event: ws.EventViolationLogged,
				Data: ws.ViolationPayload{
					SessionID:     session.ID,
					UserID:        userID,
					ViolationType: req.ViolationType,
					Count:         count,
				},
			})
		}
	}

	response.Success(c, gin.H{"logged": true})
}
