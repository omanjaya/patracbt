package handler

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	examuc "github.com/omanjaya/patra/internal/application/usecase/exam"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/infrastructure/persistence/postgres"
	ws "github.com/omanjaya/patra/internal/infrastructure/websocket"
	"github.com/omanjaya/patra/pkg/ginhelper"
	"github.com/omanjaya/patra/pkg/response"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SupervisionActionsHandler struct {
	uc         *examuc.ExamSessionUseCase
	scheduleUc *examuc.ExamScheduleUseCase
	hub        *ws.Hub
	db         *gorm.DB
	auditRepo  *postgres.AuditLogRepo
}

func NewSupervisionActionsHandler(uc *examuc.ExamSessionUseCase, scheduleUc *examuc.ExamScheduleUseCase, hub *ws.Hub, auditRepo *postgres.AuditLogRepo, db ...*gorm.DB) *SupervisionActionsHandler {
	h := &SupervisionActionsHandler{uc: uc, scheduleUc: scheduleUc, hub: hub, auditRepo: auditRepo}
	if len(db) > 0 {
		h.db = db[0]
	}
	return h
}

// logAudit is a helper to create an audit log entry. It silently ignores errors
// so that audit failures never block the main operation.
func (h *SupervisionActionsHandler) logAudit(c *gin.Context, action string, targetID uint, targetType, details string) {
	if h.auditRepo == nil {
		return
	}
	_ = h.auditRepo.Create(&entity.AuditLog{
		UserID:     c.GetUint("user_id"),
		Action:     action,
		TargetID:   targetID,
		TargetType: targetType,
		IPAddress:  c.ClientIP(),
		Details:    details,
	})
}

// POST /monitoring/:scheduleId/sessions/:sessionId/force-finish
func (h *SupervisionActionsHandler) ForceFinish(c *gin.Context) {
	sessionID, ok := ginhelper.ParseID(c, "sessionId")
	if !ok {
		return
	}
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	session, err := h.uc.ForceTerminate(sessionID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Notify student
	h.hub.SendToUser(scheduleID, session.UserID, ws.Message{
		Event: ws.EventForceFinish,
		Data: ws.ForceFinishPayload{
			SessionID: session.ID,
			UserID:    session.UserID,
			Message:   "Ujian Anda telah diselesaikan oleh pengawas.",
		},
	})
	// Notify supervisors
	h.hub.BroadcastToSupervisors(scheduleID, ws.Message{
		Event: ws.EventSessionFinished,
		Data: ws.SessionFinishedPayload{
			SessionID: session.ID,
			UserID:    session.UserID,
			Score:     session.Score,
			MaxScore:  session.MaxScore,
		},
	})

	h.logAudit(c, "force_terminate", sessionID, "exam_session",
		fmt.Sprintf(`{"schedule_id":%d,"target_user_id":%d}`, scheduleID, session.UserID))

	response.Success(c, session)
}

// POST /monitoring/:scheduleId/sessions/:sessionId/extend-time
func (h *SupervisionActionsHandler) ExtendTime(c *gin.Context) {
	sessionID, ok := ginhelper.ParseID(c, "sessionId")
	if !ok {
		return
	}
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	var req struct {
		Minutes int `json:"minutes" binding:"required,min=1,max=120"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	session, err := h.uc.ExtendTime(sessionID, req.Minutes)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	newEndTime := ""
	if session.EndTime != nil {
		newEndTime = session.EndTime.Format(time.RFC3339)
	}

	// Notify student
	h.hub.SendToUser(scheduleID, session.UserID, ws.Message{
		Event: ws.EventTimeExtended,
		Data: ws.TimeExtendedPayload{
			SessionID:  session.ID,
			UserID:     session.UserID,
			AddMinutes: req.Minutes,
			NewEndTime: newEndTime,
		},
	})

	h.logAudit(c, "extend_time", sessionID, "exam_session",
		fmt.Sprintf(`{"schedule_id":%d,"target_user_id":%d,"minutes":%d}`, scheduleID, session.UserID, req.Minutes))

	response.Success(c, session)
}

// POST /monitoring/:scheduleId/sessions/:sessionId/send-message
func (h *SupervisionActionsHandler) SendMessage(c *gin.Context) {
	sessionID, ok := ginhelper.ParseID(c, "sessionId")
	if !ok {
		return
	}
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	var req struct {
		Message    string `json:"message" binding:"required"`
		SenderName string `json:"sender_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}
	if req.SenderName == "" {
		req.SenderName = "Pengawas"
	}

	session, err := h.uc.GetSessionByID(sessionID)
	if err != nil {
		response.NotFound(c, "Sesi tidak ditemukan")
		return
	}

	h.hub.SendToUser(scheduleID, session.UserID, ws.Message{
		Event: ws.EventChatMessage,
		Data: ws.ChatMessagePayload{
			SenderName: req.SenderName,
			Message:    req.Message,
		},
	})

	response.Success(c, gin.H{"sent": true})
}

// POST /monitoring/:scheduleId/sessions/:sessionId/unlock
func (h *SupervisionActionsHandler) Unlock(c *gin.Context) {
	sessionID, ok := ginhelper.ParseID(c, "sessionId")
	if !ok {
		return
	}
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	session, err := h.uc.UnlockSession(sessionID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Notify supervisors about status change
	h.hub.BroadcastToSupervisors(scheduleID, ws.Message{
		Event: ws.EventStudentJoined,
		Data:  gin.H{"user_id": session.UserID, "session_id": session.ID, "status": "ongoing"},
	})

	h.logAudit(c, "unlock_session", sessionID, "exam_session",
		fmt.Sprintf(`{"schedule_id":%d,"target_user_id":%d}`, scheduleID, session.UserID))

	response.Success(c, session)
}

// POST /monitoring/:scheduleId/sessions/:sessionId/reset
func (h *SupervisionActionsHandler) Reset(c *gin.Context) {
	sessionID, ok := ginhelper.ParseID(c, "sessionId")
	if !ok {
		return
	}

	if err := h.uc.ResetSession(sessionID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	h.logAudit(c, "reset_session", sessionID, "exam_session", "")

	response.Success(c, gin.H{"reset": true})
}

// bulkFailure represents a single failed operation in a bulk action.
type bulkFailure struct {
	ID    uint   `json:"id"`
	Error string `json:"error"`
}

// POST /monitoring/:scheduleId/bulk-action — bulk action on multiple sessions
func (h *SupervisionActionsHandler) BulkAction(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	var req struct {
		Action     string `json:"action" binding:"required"`
		SessionIDs []uint `json:"session_ids" binding:"required"`
		Minutes    int    `json:"minutes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	// Limit bulk operation size to prevent abuse
	const maxBulkSize = 1000
	if len(req.SessionIDs) > maxBulkSize {
		response.BadRequest(c, fmt.Sprintf("Jumlah session_ids maksimal %d", maxBulkSize))
		return
	}

	const maxWorkers = 10

	var (
		mu        sync.Mutex
		succeeded []uint
		failed    []bulkFailure
	)

	switch req.Action {
	case "force_finish":
		sem := make(chan struct{}, maxWorkers)
		var wg sync.WaitGroup
		for _, sid := range req.SessionIDs {
			sid := sid
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				session, err := h.uc.ForceTerminate(sid)
				if err != nil {
					mu.Lock()
					failed = append(failed, bulkFailure{ID: sid, Error: err.Error()})
					mu.Unlock()
					return
				}
				mu.Lock()
				succeeded = append(succeeded, sid)
				mu.Unlock()
				h.hub.SendToUser(scheduleID, session.UserID, ws.Message{
					Event: ws.EventForceFinish,
					Data: ws.ForceFinishPayload{
						SessionID: session.ID,
						UserID:    session.UserID,
						Message:   "Ujian Anda telah diselesaikan oleh pengawas.",
					},
				})
			}()
		}
		wg.Wait()
	case "extend_time":
		if req.Minutes <= 0 {
			req.Minutes = 15
		}
		sem := make(chan struct{}, maxWorkers)
		var wg sync.WaitGroup
		for _, sid := range req.SessionIDs {
			sid := sid
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				session, err := h.uc.ExtendTime(sid, req.Minutes)
				if err != nil {
					mu.Lock()
					failed = append(failed, bulkFailure{ID: sid, Error: err.Error()})
					mu.Unlock()
					return
				}
				mu.Lock()
				succeeded = append(succeeded, sid)
				mu.Unlock()
				newEndTime := ""
				if session.EndTime != nil {
					newEndTime = session.EndTime.Format(time.RFC3339)
				}
				h.hub.SendToUser(scheduleID, session.UserID, ws.Message{
					Event: ws.EventTimeExtended,
					Data: ws.TimeExtendedPayload{
						SessionID:  session.ID,
						UserID:     session.UserID,
						AddMinutes: req.Minutes,
						NewEndTime: newEndTime,
					},
				})
			}()
		}
		wg.Wait()
	case "unlock":
		sem := make(chan struct{}, maxWorkers)
		var wg sync.WaitGroup
		for _, sid := range req.SessionIDs {
			sid := sid
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				_, err := h.uc.UnlockSession(sid)
				if err != nil {
					mu.Lock()
					failed = append(failed, bulkFailure{ID: sid, Error: err.Error()})
					mu.Unlock()
					return
				}
				mu.Lock()
				succeeded = append(succeeded, sid)
				mu.Unlock()
			}()
		}
		wg.Wait()
	default:
		response.BadRequest(c, "Aksi tidak valid")
		return
	}

	h.logAudit(c, "bulk_action", scheduleID, "exam_schedule",
		fmt.Sprintf(`{"action":"%s","session_count":%d,"success":%d,"failed":%d}`, req.Action, len(req.SessionIDs), len(succeeded), len(failed)))

	response.Success(c, gin.H{
		"success": succeeded,
		"failed":  failed,
	})
}

// GET /monitoring/:scheduleId/unfinished — list sessions that haven't finished
func (h *SupervisionActionsHandler) GetUnfinishedList(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	ongoing, err := h.uc.ListOngoingBySchedule(scheduleID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	notStarted, err := h.uc.ListNotStartedBySchedule(scheduleID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"ongoing":     ongoing,
		"not_started": notStarted,
		"total":       len(ongoing) + len(notStarted),
	})
}

// POST /reports/:scheduleId/finish-all — force finish all ongoing sessions
func (h *SupervisionActionsHandler) FinishAllOngoing(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	finished, err := h.uc.FinishAllOngoing(scheduleID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	for _, session := range finished {
		h.hub.SendToUser(scheduleID, session.UserID, ws.Message{
			Event: ws.EventForceFinish,
			Data: ws.ForceFinishPayload{
				SessionID: session.ID,
				UserID:    session.UserID,
				Message:   "Ujian telah diselesaikan.",
			},
		})
	}

	response.Success(c, gin.H{"finished": len(finished)})
}

// POST /supervision/claim — supervisor joining a room
func (h *SupervisionActionsHandler) Claim(c *gin.Context) {
	var req struct {
		ExamScheduleIDs []uint `json:"exam_schedule_ids" binding:"required"`
		RoomID          string `json:"room_id" binding:"required"`
		Token           string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	// Validate role constraint for NO token
	userRole := c.GetString("role")
	if userRole != entity.RoleAdmin && userRole != entity.RoleGuru && req.Token == "" {
		response.BadRequest(c, "Token ruangan diperlukan untuk pengawas")
		return
	}

	// Admin and guru skip token validation
	if userRole == entity.RoleAdmin || userRole == entity.RoleGuru {
		// Build room info for response
		roomInfo := gin.H{"room_id": req.RoomID}
		if req.RoomID != "GLOBAL_ALL" {
			roomIDUint, err := strconv.ParseUint(req.RoomID, 10, 64)
			if err != nil {
				response.BadRequest(c, "Room ID tidak valid")
				return
			}
			roomInfo["room_id"] = uint(roomIDUint)
		}
		response.Success(c, gin.H{
			"message":      "Akses diberikan",
			"schedule_ids": req.ExamScheduleIDs,
			"room":         roomInfo,
		})
		return
	}

	// Pengawas: validate token against ExamSupervision table and ExamScheduleRoom
	if req.RoomID != "GLOBAL_ALL" && req.Token != "" {
		roomIDUint, err := strconv.ParseUint(req.RoomID, 10, 64)
		if err != nil {
			response.BadRequest(c, "Room ID tidak valid")
			return
		}
		isValid := false

		for _, sid := range req.ExamScheduleIDs {
			// Check ExamSupervision table first
			if h.db != nil {
				var supervision entity.ExamSupervision
				err := h.db.Where("exam_schedule_id = ? AND room_id = ? AND token = ?", sid, uint(roomIDUint), req.Token).First(&supervision).Error
				if err == nil {
					isValid = true
					break
				}
			}

			// Fallback: check ExamScheduleRoom and global token
			schedule, err := h.scheduleUc.GetByID(sid)
			if err != nil {
				continue
			}

			hasToken := false
			for _, r := range schedule.ExamRooms {
				if r.RoomID == uint(roomIDUint) && r.SupervisionToken == req.Token {
					hasToken = true
					break
				}
			}
			if !hasToken && schedule.SupervisionToken == req.Token {
				hasToken = true
			}

			if hasToken {
				isValid = true
				break
			}
		}

		if !isValid {
			response.Error(c, 403, "FORBIDDEN", "Token tidak valid untuk ruangan yang dipilih")
			return
		}
	}

	roomInfo := gin.H{"room_id": req.RoomID}
	if req.RoomID != "GLOBAL_ALL" {
		roomIDUint, err := strconv.ParseUint(req.RoomID, 10, 64)
		if err != nil {
			response.BadRequest(c, "Room ID tidak valid")
			return
		}
		roomInfo["room_id"] = uint(roomIDUint)
	}
	response.Success(c, gin.H{
		"message":      "Akses diberikan",
		"schedule_ids": req.ExamScheduleIDs,
		"room":         roomInfo,
	})
}

// GET /supervision/tokens/:scheduleId
func (h *SupervisionActionsHandler) GetRoomTokens(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}
	schedule, err := h.scheduleUc.GetByID(scheduleID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	type RoomTokenRes struct {
		RoomID uint   `json:"room_id"`
		Token  string `json:"token"`
	}

	// Try to fetch from ExamSupervision table first
	if h.db != nil {
		var supervisions []entity.ExamSupervision
		if err := h.db.Where("exam_schedule_id = ?", scheduleID).Find(&supervisions).Error; err == nil && len(supervisions) > 0 {
			res := make([]RoomTokenRes, 0, len(supervisions))
			for _, s := range supervisions {
				res = append(res, RoomTokenRes{RoomID: s.RoomID, Token: s.Token})
			}
			response.Success(c, gin.H{
				"global_token": schedule.SupervisionToken,
				"rooms":        res,
			})
			return
		}
	}

	// Fallback: from ExamScheduleRoom
	res := []RoomTokenRes{}
	for _, r := range schedule.ExamRooms {
		res = append(res, RoomTokenRes{RoomID: r.RoomID, Token: r.SupervisionToken})
	}

	response.Success(c, gin.H{
		"global_token": schedule.SupervisionToken,
		"rooms":        res,
	})
}

// POST /supervision/tokens/:scheduleId
func (h *SupervisionActionsHandler) SaveRoomTokens(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}
	var req struct {
		GlobalToken string `json:"global_token"`
		Rooms       []struct {
			RoomID uint   `json:"room_id"`
			Token  string `json:"token"`
		} `json:"rooms"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	// Generate 6-char alphanumeric tokens for rooms that have no token
	type RoomTokenResult struct {
		RoomID uint   `json:"room_id"`
		Token  string `json:"token"`
	}
	results := make([]RoomTokenResult, 0, len(req.Rooms))
	roomTokensMap := make(map[uint]string)
	for _, r := range req.Rooms {
		token := r.Token
		if token == "" {
			token = generateAlphanumericToken(6)
		}
		roomTokensMap[r.RoomID] = token
		results = append(results, RoomTokenResult{RoomID: r.RoomID, Token: token})
	}

	// Save to ExamScheduleRoom via use case
	err := h.scheduleUc.SaveSupervisionTokens(scheduleID, req.GlobalToken, roomTokensMap)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// Upsert into ExamSupervision table
	if h.db != nil {
		for _, r := range results {
			supervision := entity.ExamSupervision{
				ExamScheduleID: scheduleID,
				RoomID:         r.RoomID,
				Token:          r.Token,
			}
			h.db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "exam_schedule_id"}, {Name: "room_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"token", "updated_at"}),
			}).Create(&supervision)
		}
	}

	response.Success(c, gin.H{
		"message": "Token pengawas berhasil disimpan",
		"rooms":   results,
	})
}

// POST /monitoring/:scheduleId/sessions/:sessionId/return-to-exam
func (h *SupervisionActionsHandler) ReturnToExam(c *gin.Context) {
	sessionID, ok := ginhelper.ParseID(c, "sessionId")
	if !ok {
		return
	}
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	session, err := h.uc.ReturnToExam(sessionID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Notify student that they can resume
	h.hub.SendToUser(scheduleID, session.UserID, ws.Message{
		Event: ws.EventStudentJoined,
		Data:  gin.H{"user_id": session.UserID, "session_id": session.ID, "status": "ongoing", "message": "Anda dapat melanjutkan ujian."},
	})

	// Notify supervisors
	h.hub.BroadcastToSupervisors(scheduleID, ws.Message{
		Event: ws.EventStudentJoined,
		Data:  gin.H{"user_id": session.UserID, "session_id": session.ID, "status": "ongoing"},
	})

	h.logAudit(c, "return_to_exam", sessionID, "exam_session",
		fmt.Sprintf(`{"schedule_id":%d,"target_user_id":%d}`, scheduleID, session.UserID))

	response.Success(c, session)
}

// POST /monitoring/:scheduleId/sessions/:sessionId/force-logout
func (h *SupervisionActionsHandler) ForceLogout(c *gin.Context) {
	sessionID, ok := ginhelper.ParseID(c, "sessionId")
	if !ok {
		return
	}
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	session, err := h.uc.GetSessionByID(sessionID)
	if err != nil {
		response.NotFound(c, "Sesi tidak ditemukan")
		return
	}

	// Send force logout to the specific user across all rooms
	h.hub.SendToUser(scheduleID, session.UserID, ws.Message{
		Event: ws.EventForceLogout,
		Data: ws.ForceLogoutPayload{
			UserID:  session.UserID,
			Message: "Anda telah dikeluarkan dari ujian oleh pengawas.",
		},
	})

	// Also broadcast globally in case user is connected on multiple rooms
	h.hub.SendToUserGlobal(session.UserID, ws.Message{
		Event: ws.EventForceLogout,
		Data: ws.ForceLogoutPayload{
			UserID:  session.UserID,
			Message: "Anda telah dikeluarkan dari ujian oleh pengawas.",
		},
	})

	h.logAudit(c, "force_logout", sessionID, "exam_session",
		fmt.Sprintf(`{"schedule_id":%d,"target_user_id":%d}`, scheduleID, session.UserID))

	response.Success(c, gin.H{"force_logout": true, "user_id": session.UserID})
}

// generateAlphanumericToken generates a random alphanumeric string of given length.
func generateAlphanumericToken(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[n.Int64()]
	}
	return string(result)
}
