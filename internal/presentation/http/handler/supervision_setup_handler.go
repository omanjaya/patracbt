package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	examuc "github.com/omanjaya/patra/internal/application/usecase/exam"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/infrastructure/persistence/postgres"
	"github.com/omanjaya/patra/pkg/ginhelper"
	"github.com/omanjaya/patra/pkg/response"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SupervisionSetupHandler handles supervision setup, token generation,
// global stats, student fetching, and session exit.
type SupervisionSetupHandler struct {
	scheduleUc *examuc.ExamScheduleUseCase
	sessionUc  *examuc.ExamSessionUseCase
	db         *gorm.DB
	auditRepo  *postgres.AuditLogRepo
}

func NewSupervisionSetupHandler(
	scheduleUc *examuc.ExamScheduleUseCase,
	sessionUc *examuc.ExamSessionUseCase,
	auditRepo *postgres.AuditLogRepo,
	db ...*gorm.DB,
) *SupervisionSetupHandler {
	h := &SupervisionSetupHandler{
		scheduleUc: scheduleUc,
		sessionUc:  sessionUc,
		auditRepo:  auditRepo,
	}
	if len(db) > 0 {
		h.db = db[0]
	}
	return h
}

func (h *SupervisionSetupHandler) logAudit(c *gin.Context, action string, targetID uint, targetType, details string) {
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

// ──────────────────────────────────────────────────────────────────
// 1. GET /admin/supervision/:scheduleId/setup
// ──────────────────────────────────────────────────────────────────

type roomSetupItem struct {
	RoomID   uint   `json:"room_id"`
	RoomName string `json:"room_name"`
	Token    string `json:"token"`
	Capacity int    `json:"capacity"`
}

// GetSupervisionSetup returns the supervision configuration for a schedule:
// rooms that have eligible students, existing tokens, and assigned pengawas.
func (h *SupervisionSetupHandler) GetSupervisionSetup(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	schedule, err := h.scheduleUc.GetByID(scheduleID)
	if err != nil {
		response.NotFound(c, "Jadwal ujian tidak ditemukan")
		return
	}

	if h.db == nil {
		response.InternalError(c, "Database tidak tersedia")
		return
	}

	// Find rooms that have eligible peserta based on schedule's rombel/tag/user filters.
	// We get all rooms that have at least one peserta assigned to them.
	roomIDs := h.getEligibleRoomIDs(schedule)

	var rooms []entity.Room
	if len(roomIDs) > 0 {
		h.db.Where("id IN ?", roomIDs).Order("name ASC").Find(&rooms)
	}

	// Fetch existing supervision tokens from ExamSupervision table
	var supervisions []entity.ExamSupervision
	h.db.Where("exam_schedule_id = ?", scheduleID).Find(&supervisions)
	tokenMap := make(map[uint]string, len(supervisions))
	for _, s := range supervisions {
		tokenMap[s.RoomID] = s.Token
	}

	// Also check ExamScheduleRoom tokens as fallback
	for _, r := range schedule.ExamRooms {
		if _, exists := tokenMap[r.RoomID]; !exists && r.SupervisionToken != "" {
			tokenMap[r.RoomID] = r.SupervisionToken
		}
	}

	// Build response
	roomItems := make([]roomSetupItem, 0, len(rooms))
	for _, room := range rooms {
		roomItems = append(roomItems, roomSetupItem{
			RoomID:   room.ID,
			RoomName: room.Name,
			Token:    tokenMap[room.ID],
			Capacity: room.Capacity,
		})
	}

	response.Success(c, gin.H{
		"schedule": gin.H{
			"id":                schedule.ID,
			"name":              schedule.Name,
			"status":            schedule.Status,
			"supervision_token": schedule.SupervisionToken,
		},
		"rooms": roomItems,
	})
}

// getEligibleRoomIDs returns room IDs that contain peserta who are eligible
// for the given exam schedule (based on rombel/tag/user inclusion/exclusion).
func (h *SupervisionSetupHandler) getEligibleRoomIDs(schedule *entity.ExamSchedule) []uint {
	// Collect inclusion/exclusion IDs
	var incRombelIDs, excRombelIDs []uint
	for _, r := range schedule.Rombels {
		incRombelIDs = append(incRombelIDs, r.RombelID)
	}

	var incTagIDs []uint
	for _, t := range schedule.Tags {
		incTagIDs = append(incTagIDs, t.TagID)
	}

	var incUserIDs, excUserIDs []uint
	for _, u := range schedule.Users {
		if u.Type == "include" {
			incUserIDs = append(incUserIDs, u.UserID)
		} else if u.Type == "exclude" {
			excUserIDs = append(excUserIDs, u.UserID)
		}
	}

	// If no inclusion criteria, return all rooms with peserta
	hasInclusions := len(incRombelIDs) > 0 || len(incTagIDs) > 0 || len(incUserIDs) > 0
	if !hasInclusions {
		var roomIDs []uint
		h.db.Table("user_profiles").
			Joins("JOIN users ON users.id = user_profiles.user_id").
			Where("users.role = ? AND users.deleted_at IS NULL AND user_profiles.room_id IS NOT NULL", entity.RolePeserta).
			Distinct("user_profiles.room_id").
			Pluck("user_profiles.room_id", &roomIDs)
		return roomIDs
	}

	// Build query for eligible user IDs
	query := h.db.Table("users").
		Select("DISTINCT user_profiles.room_id").
		Joins("JOIN user_profiles ON user_profiles.user_id = users.id").
		Where("users.role = ? AND users.deleted_at IS NULL AND user_profiles.room_id IS NOT NULL", entity.RolePeserta)

	// Apply inclusion filter (OR logic)
	query = query.Where(h.db.Where("1=0"). // start with false, then OR
							Or("user_profiles.rombel_id IN ?", incRombelIDs).
							Or("users.id IN ?", incUserIDs))

	if len(incTagIDs) > 0 {
		query = query.Or("users.id IN (?)",
			h.db.Table("user_tags").Select("user_id").Where("tag_id IN ?", incTagIDs))
	}

	// Apply exclusion filter
	if len(excUserIDs) > 0 {
		query = query.Where("users.id NOT IN ?", excUserIDs)
	}
	if len(excRombelIDs) > 0 {
		query = query.Where("user_profiles.rombel_id NOT IN ?", excRombelIDs)
	}

	var roomIDs []uint
	query.Pluck("user_profiles.room_id", &roomIDs)
	return roomIDs
}

// ──────────────────────────────────────────────────────────────────
// 2. POST /admin/supervision/:scheduleId/generate-tokens
// ──────────────────────────────────────────────────────────────────

// GenerateTokens creates supervision tokens for all rooms in a schedule.
// Only rooms that don't already have a token will get a new one.
func (h *SupervisionSetupHandler) GenerateTokens(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	schedule, err := h.scheduleUc.GetByID(scheduleID)
	if err != nil {
		response.NotFound(c, "Jadwal ujian tidak ditemukan")
		return
	}

	if h.db == nil {
		response.InternalError(c, "Database tidak tersedia")
		return
	}

	// Get all rooms
	var rooms []entity.Room
	h.db.Order("name ASC").Find(&rooms)

	tokensGenerated := 0
	type tokenResult struct {
		RoomID   uint   `json:"room_id"`
		RoomName string `json:"room_name"`
		Token    string `json:"token"`
		IsNew    bool   `json:"is_new"`
	}
	results := make([]tokenResult, 0, len(rooms))

	for _, room := range rooms {
		var existing entity.ExamSupervision
		err := h.db.Where("exam_schedule_id = ? AND room_id = ?", scheduleID, room.ID).First(&existing).Error

		if err == nil {
			// Already exists
			results = append(results, tokenResult{
				RoomID:   room.ID,
				RoomName: room.Name,
				Token:    existing.Token,
				IsNew:    false,
			})
			continue
		}

		// Create new token
		newToken := generateAlphanumericToken(6)
		supervision := entity.ExamSupervision{
			ExamScheduleID: scheduleID,
			RoomID:         room.ID,
			Token:          newToken,
		}
		h.db.Create(&supervision)

		// Also update ExamScheduleRoom
		h.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "exam_schedule_id"}, {Name: "room_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"supervision_token"}),
		}).Create(&entity.ExamScheduleRoom{
			ExamScheduleID:   scheduleID,
			RoomID:           room.ID,
			SupervisionToken: newToken,
		})

		tokensGenerated++
		results = append(results, tokenResult{
			RoomID:   room.ID,
			RoomName: room.Name,
			Token:    newToken,
			IsNew:    true,
		})
	}

	h.logAudit(c, "generate_supervision_tokens", scheduleID, "exam_schedule",
		fmt.Sprintf(`{"tokens_generated":%d,"total_rooms":%d}`, tokensGenerated, len(rooms)))

	response.Success(c, gin.H{
		"message":          fmt.Sprintf("Proses selesai. %d token baru dibuat.", tokensGenerated),
		"tokens_generated": tokensGenerated,
		"rooms":            results,
		"global_token":     schedule.SupervisionToken,
	})
}

// ──────────────────────────────────────────────────────────────────
// 3. POST /admin/supervision/:scheduleId/rooms/:roomId/regenerate-token
// ──────────────────────────────────────────────────────────────────

// RegenerateToken regenerates the supervision token for a specific room.
func (h *SupervisionSetupHandler) RegenerateToken(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}
	roomID, ok := ginhelper.ParseID(c, "roomId")
	if !ok {
		return
	}

	if h.db == nil {
		response.InternalError(c, "Database tidak tersedia")
		return
	}

	newToken := generateAlphanumericToken(6)

	// Update ExamSupervision table
	result := h.db.Model(&entity.ExamSupervision{}).
		Where("exam_schedule_id = ? AND room_id = ?", scheduleID, roomID).
		Update("token", newToken)

	if result.RowsAffected == 0 {
		// Create if not exists
		h.db.Create(&entity.ExamSupervision{
			ExamScheduleID: scheduleID,
			RoomID:         roomID,
			Token:          newToken,
		})
	}

	// Also update ExamScheduleRoom
	h.db.Model(&entity.ExamScheduleRoom{}).
		Where("exam_schedule_id = ? AND room_id = ?", scheduleID, roomID).
		Update("supervision_token", newToken)

	h.logAudit(c, "regenerate_supervision_token", scheduleID, "exam_schedule",
		fmt.Sprintf(`{"room_id":%d,"new_token":"%s"}`, roomID, newToken))

	response.Success(c, gin.H{
		"message":   "Token berhasil diperbarui",
		"room_id":   roomID,
		"new_token": newToken,
	})
}

// ──────────────────────────────────────────────────────────────────
// 4. GET /admin/supervision/:scheduleId/global-stats
// ──────────────────────────────────────────────────────────────────

// GetGlobalStats returns overall supervision statistics for a schedule.
func (h *SupervisionSetupHandler) GetGlobalStats(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	if h.db == nil {
		response.InternalError(c, "Database tidak tersedia")
		return
	}

	schedule, err := h.scheduleUc.GetByID(scheduleID)
	if err != nil {
		response.NotFound(c, "Jadwal ujian tidak ditemukan")
		return
	}

	roomIDParam := c.Query("room_id")

	// Count total eligible peserta
	totalPeserta := h.countEligiblePeserta(schedule, roomIDParam)

	// Count sessions by status
	statsQuery := h.db.Table("exam_sessions").
		Select("status, COUNT(*) as total").
		Where("exam_schedule_id = ?", scheduleID)

	if roomIDParam != "" && roomIDParam != "GLOBAL_ALL" {
		statsQuery = statsQuery.Where("user_id IN (?)",
			h.db.Table("user_profiles").Select("user_id").Where("room_id = ?", roomIDParam))
	}

	type statusCount struct {
		Status string
		Total  int64
	}
	var statsCounts []statusCount
	statsQuery.Group("status").Scan(&statsCounts)

	statsMap := make(map[string]int64)
	var hasSessionCount int64
	for _, sc := range statsCounts {
		statsMap[sc.Status] = sc.Total
		hasSessionCount += sc.Total
	}

	completed := statsMap["finished"]
	if completed == 0 {
		completed = statsMap["completed"]
	}
	ongoing := statsMap["ongoing"]
	terminated := statsMap["terminated"]
	notStarted := int64(totalPeserta) - hasSessionCount
	if notStarted < 0 {
		notStarted = 0
	}

	// Count total violations
	var totalViolations int64
	violationQuery := h.db.Table("exam_sessions").
		Select("COALESCE(SUM(violation_count), 0)").
		Where("exam_schedule_id = ?", scheduleID)

	if roomIDParam != "" && roomIDParam != "GLOBAL_ALL" {
		violationQuery = violationQuery.Where("user_id IN (?)",
			h.db.Table("user_profiles").Select("user_id").Where("room_id = ?", roomIDParam))
	}
	violationQuery.Scan(&totalViolations)

	// Per-room breakdown
	type roomStat struct {
		RoomID     uint   `json:"room_id"`
		RoomName   string `json:"room_name"`
		Total      int64  `json:"total"`
		Ongoing    int64  `json:"ongoing"`
		Completed  int64  `json:"completed"`
		Terminated int64  `json:"terminated"`
		NotStarted int64  `json:"not_started"`
		Violations int64  `json:"violations"`
	}
	var roomBreakdown []roomStat

	if roomIDParam == "" || roomIDParam == "GLOBAL_ALL" {
		// Get all rooms with students
		type roomSessionRow struct {
			RoomID     uint
			RoomName   string
			Status     string
			Total      int64
			Violations int64
		}

		var rows []roomSessionRow
		h.db.Raw(`
			SELECT up.room_id, r.name as room_name, es.status,
				COUNT(DISTINCT es.id) as total,
				COALESCE(SUM(es.violation_count), 0) as violations
			FROM user_profiles up
			JOIN users u ON u.id = up.user_id AND u.role = ? AND u.deleted_at IS NULL
			JOIN rooms r ON r.id = up.room_id
			LEFT JOIN exam_sessions es ON es.user_id = up.user_id AND es.exam_schedule_id = ?
			WHERE up.room_id IS NOT NULL
			GROUP BY up.room_id, r.name, es.status
			ORDER BY r.name
		`, entity.RolePeserta, scheduleID).Scan(&rows)

		// Aggregate by room
		roomMap := make(map[uint]*roomStat)
		var roomOrder []uint
		for _, row := range rows {
			rs, exists := roomMap[row.RoomID]
			if !exists {
				rs = &roomStat{RoomID: row.RoomID, RoomName: row.RoomName}
				roomMap[row.RoomID] = rs
				roomOrder = append(roomOrder, row.RoomID)
			}
			switch row.Status {
			case "ongoing":
				rs.Ongoing = row.Total
			case "finished", "completed":
				rs.Completed = row.Total
			case "terminated":
				rs.Terminated = row.Total
			}
			rs.Violations += row.Violations
		}

		// Count total peserta per room and compute not_started
		type roomPesertaCount struct {
			RoomID uint
			Total  int64
		}
		var pesertaCounts []roomPesertaCount
		h.db.Table("user_profiles").
			Select("user_profiles.room_id, COUNT(DISTINCT user_profiles.user_id) as total").
			Joins("JOIN users ON users.id = user_profiles.user_id").
			Where("users.role = ? AND users.deleted_at IS NULL AND user_profiles.room_id IS NOT NULL", entity.RolePeserta).
			Group("user_profiles.room_id").
			Scan(&pesertaCounts)

		pesertaMap := make(map[uint]int64)
		for _, pc := range pesertaCounts {
			pesertaMap[pc.RoomID] = pc.Total
		}

		for _, rid := range roomOrder {
			rs := roomMap[rid]
			totalInRoom := pesertaMap[rid]
			rs.Total = totalInRoom
			sessionCount := rs.Ongoing + rs.Completed + rs.Terminated
			rs.NotStarted = totalInRoom - sessionCount
			if rs.NotStarted < 0 {
				rs.NotStarted = 0
			}
			roomBreakdown = append(roomBreakdown, *rs)
		}
	}

	response.Success(c, gin.H{
		"total":       totalPeserta,
		"ongoing":     ongoing,
		"completed":   completed,
		"terminated":  terminated,
		"not_started": notStarted,
		"violations":  totalViolations,
		"rooms":       roomBreakdown,
	})
}

// countEligiblePeserta counts peserta eligible for the schedule, optionally filtered by room.
func (h *SupervisionSetupHandler) countEligiblePeserta(schedule *entity.ExamSchedule, roomID string) int {
	query := h.db.Table("users").
		Joins("JOIN user_profiles ON user_profiles.user_id = users.id").
		Where("users.role = ? AND users.deleted_at IS NULL", entity.RolePeserta)

	if roomID != "" && roomID != "GLOBAL_ALL" {
		query = query.Where("user_profiles.room_id = ?", roomID)
	}

	// Apply inclusion/exclusion filters from schedule
	var incRombelIDs []uint
	for _, r := range schedule.Rombels {
		incRombelIDs = append(incRombelIDs, r.RombelID)
	}
	var incTagIDs []uint
	for _, t := range schedule.Tags {
		incTagIDs = append(incTagIDs, t.TagID)
	}
	var incUserIDs, excUserIDs []uint
	for _, u := range schedule.Users {
		if u.Type == "include" {
			incUserIDs = append(incUserIDs, u.UserID)
		} else {
			excUserIDs = append(excUserIDs, u.UserID)
		}
	}

	hasInclusions := len(incRombelIDs) > 0 || len(incTagIDs) > 0 || len(incUserIDs) > 0
	if hasInclusions {
		query = query.Where(
			h.db.Where("user_profiles.rombel_id IN ?", incRombelIDs).
				Or("users.id IN ?", incUserIDs),
		)
	}

	if len(excUserIDs) > 0 {
		query = query.Where("users.id NOT IN ?", excUserIDs)
	}

	var count int64
	query.Count(&count)
	return int(count)
}

// ──────────────────────────────────────────────────────────────────
// 5. GET /admin/supervision/:scheduleId/rooms/:roomId/students
// ──────────────────────────────────────────────────────────────────

type studentItem struct {
	UserID         uint    `json:"user_id"`
	Name           string  `json:"name"`
	NIS            string  `json:"nis"`
	RoomName       string  `json:"room_name"`
	SessionID      *uint   `json:"session_id"`
	Status         string  `json:"status"`
	ViolationCount int     `json:"violation_count"`
	Score          float64 `json:"score"`
	MaxScore       float64 `json:"max_score"`
	Answered       int     `json:"answered"`
	TotalQuestions int     `json:"total_questions"`
	ExtraTime      int     `json:"extra_time"`
	AvatarPath     string  `json:"avatar_path"`
}

// FetchStudents returns students in a specific room for supervision monitoring.
func (h *SupervisionSetupHandler) FetchStudents(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}
	roomID, ok := ginhelper.ParseID(c, "roomId")
	if !ok {
		return
	}

	if h.db == nil {
		response.InternalError(c, "Database tidak tersedia")
		return
	}

	_, err := h.scheduleUc.GetByID(scheduleID)
	if err != nil {
		response.NotFound(c, "Jadwal ujian tidak ditemukan")
		return
	}

	// Count total questions for this schedule
	var totalQuestions int64
	h.db.Raw(`
		SELECT COALESCE(SUM(
			CASE WHEN esqb.question_count > 0 THEN esqb.question_count
			ELSE (SELECT COUNT(*) FROM questions q WHERE q.question_bank_id = esqb.question_bank_id AND q.deleted_at IS NULL)
			END
		), 0)
		FROM exam_schedule_question_banks esqb
		WHERE esqb.exam_schedule_id = ?
	`, scheduleID).Scan(&totalQuestions)

	// Get peserta in this room
	type userRow struct {
		UserID     uint
		Name       string
		NIS        string
		RoomName   string
		AvatarPath string
	}
	var users []userRow
	h.db.Raw(`
		SELECT u.id as user_id, u.name,
			COALESCE(up.nis, '') as nis,
			COALESCE(r.name, '-') as room_name,
			COALESCE(u.avatar_path, '') as avatar_path
		FROM users u
		JOIN user_profiles up ON up.user_id = u.id
		LEFT JOIN rooms r ON r.id = up.room_id
		WHERE u.role = ? AND u.deleted_at IS NULL AND up.room_id = ?
		ORDER BY u.name ASC
	`, entity.RolePeserta, roomID).Scan(&users)

	// Get sessions for these users in this schedule
	userIDs := make([]uint, len(users))
	for i, u := range users {
		userIDs[i] = u.UserID
	}

	type sessionRow struct {
		ID             uint
		UserID         uint
		Status         string
		ViolationCount int
		Score          float64
		MaxScore       float64
		ExtraTime      int
		AnswerCount    int64
	}
	sessionMap := make(map[uint]*sessionRow)

	if len(userIDs) > 0 {
		var sessions []sessionRow
		h.db.Raw(`
			SELECT es.id, es.user_id, es.status, es.violation_count, es.score, es.max_score, es.extra_time,
				(SELECT COUNT(*) FROM exam_answers ea WHERE ea.exam_session_id = es.id) as answer_count
			FROM exam_sessions es
			WHERE es.exam_schedule_id = ? AND es.user_id IN ?
		`, scheduleID, userIDs).Scan(&sessions)

		for i := range sessions {
			sessionMap[sessions[i].UserID] = &sessions[i]
		}
	}

	// Build response
	items := make([]studentItem, 0, len(users))
	for _, u := range users {
		item := studentItem{
			UserID:         u.UserID,
			Name:           u.Name,
			NIS:            u.NIS,
			RoomName:       u.RoomName,
			Status:         "not_started",
			TotalQuestions: int(totalQuestions),
			AvatarPath:     u.AvatarPath,
		}

		if sess, ok := sessionMap[u.UserID]; ok {
			item.SessionID = &sess.ID
			item.Status = sess.Status
			item.ViolationCount = sess.ViolationCount
			item.Score = sess.Score
			item.MaxScore = sess.MaxScore
			item.Answered = int(sess.AnswerCount)
			item.ExtraTime = sess.ExtraTime
		}

		items = append(items, item)
	}

	// Sort: terminated first, then ongoing, then not_started, then finished
	statusOrder := map[string]int{
		"terminated":  0,
		"ongoing":     1,
		"not_started": 2,
		"finished":    3,
		"completed":   3,
	}
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			oi := statusOrder[items[i].Status]
			oj := statusOrder[items[j].Status]
			if oi > oj || (oi == oj && items[i].Name > items[j].Name) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	response.Success(c, gin.H{
		"students":        items,
		"total":           len(items),
		"total_questions": totalQuestions,
	})
}

// ──────────────────────────────────────────────────────────────────
// 6. POST /pengawas/supervision/:scheduleId/exit
// ──────────────────────────────────────────────────────────────────

// ExitSession allows a pengawas to exit their supervision session.
// In the Go version (stateless JWT), this is a no-op acknowledgment
// since there's no server-side session to clear. The frontend handles
// clearing its local state.
func (h *SupervisionSetupHandler) ExitSession(c *gin.Context) {
	scheduleID, ok := ginhelper.ParseID(c, "scheduleId")
	if !ok {
		return
	}

	userID := c.GetUint("user_id")

	h.logAudit(c, "exit_supervision_session", scheduleID, "exam_schedule",
		fmt.Sprintf(`{"user_id":%d}`, userID))

	response.Success(c, gin.H{
		"message": "Sesi pengawasan berhasil diakhiri",
	})
}
