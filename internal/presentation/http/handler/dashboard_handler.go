package handler

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/response"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"gorm.io/gorm"
)

// dashboardCache holds a cached DashboardStats with expiry.
type dashboardCache struct {
	mu        sync.RWMutex
	stats     DashboardStats
	expiresAt time.Time
}

var adminStatsCache dashboardCache

type DashboardHandler struct {
	db *gorm.DB
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

type DashboardStats struct {
	TotalPeserta       int64 `json:"total_peserta"`
	TotalGuru          int64 `json:"total_guru"`
	TotalRombel        int64 `json:"total_rombel"`
	TotalQuestionBanks int64 `json:"total_question_banks"`
	ActiveSchedules    int64 `json:"active_schedules"`
	TotalSessions      int64 `json:"total_sessions"`
	FinishedSessions   int64 `json:"finished_sessions"`
	OngoingSessions    int64 `json:"ongoing_sessions"`
}

// GET /admin/dashboard/stats
func (h *DashboardHandler) GetStats(c *gin.Context) {
	// Check cache first (30-second TTL).
	adminStatsCache.mu.RLock()
	cached := time.Now().Before(adminStatsCache.expiresAt)
	cachedStats := adminStatsCache.stats
	adminStatsCache.mu.RUnlock()
	if cached {
		response.Success(c, cachedStats)
		return
	}

	// Single query for user role counts.
	type roleCount struct {
		Role  string
		Count int64
	}
	var roleCounts []roleCount
	if err := h.db.Raw(`SELECT role, COUNT(*) as count FROM users WHERE deleted_at IS NULL AND role IN ('peserta','guru') GROUP BY role`).
		Scan(&roleCounts).Error; err != nil {
		logger.Log.Errorf("Dashboard: failed to query role counts: %v", err)
		response.InternalError(c, "Gagal memuat statistik")
		return
	}

	var stats DashboardStats
	for _, rc := range roleCounts {
		switch rc.Role {
		case entity.RolePeserta:
			stats.TotalPeserta = rc.Count
		case entity.RoleGuru:
			stats.TotalGuru = rc.Count
		}
	}

	// Single query for session status counts.
	type sessionCount struct {
		Status string
		Count  int64
	}
	var sessionCounts []sessionCount
	if err := h.db.Raw(`SELECT status, COUNT(*) as count FROM exam_sessions GROUP BY status`).
		Scan(&sessionCounts).Error; err != nil {
		logger.Log.Errorf("Dashboard: failed to query session counts: %v", err)
		response.InternalError(c, "Gagal memuat statistik")
		return
	}

	for _, sc := range sessionCounts {
		stats.TotalSessions += sc.Count
		switch sc.Status {
		case "finished":
			stats.FinishedSessions = sc.Count
		case "ongoing":
			stats.OngoingSessions = sc.Count
		}
	}

	if err := h.db.Model(&entity.Rombel{}).Count(&stats.TotalRombel).Error; err != nil {
		logger.Log.Errorf("Dashboard: failed to count rombels: %v", err)
	}
	if err := h.db.Model(&entity.QuestionBank{}).Where("deleted_at IS NULL").Count(&stats.TotalQuestionBanks).Error; err != nil {
		logger.Log.Errorf("Dashboard: failed to count question banks: %v", err)
	}
	if err := h.db.Model(&entity.ExamSchedule{}).Where("status IN ? AND deleted_at IS NULL", []string{"published", "active"}).Count(&stats.ActiveSchedules).Error; err != nil {
		logger.Log.Errorf("Dashboard: failed to count active schedules: %v", err)
	}

	// Store in cache.
	adminStatsCache.mu.Lock()
	adminStatsCache.stats = stats
	adminStatsCache.expiresAt = time.Now().Add(30 * time.Second)
	adminStatsCache.mu.Unlock()

	response.Success(c, stats)
}

// GET /guru/dashboard/stats — guru-specific stats
func (h *DashboardHandler) GetGuruStats(c *gin.Context) {
	userID := c.GetUint("user_id")

	type guruStats struct {
		TotalBanks      int64 `json:"total_banks"`
		TotalQuestions  int64 `json:"total_questions"`
		ActiveSchedules int64 `json:"active_schedules"`
		TotalSchedules  int64 `json:"total_schedules"`
	}

	var stats guruStats

	// Combine schedule counts into a single query.
	type schedCount struct {
		Active int64
		Total  int64
	}
	var sc schedCount
	if err := h.db.Raw(`
		SELECT
			COUNT(*) FILTER (WHERE status IN ('published','active')) AS active,
			COUNT(*) AS total
		FROM exam_schedules
		WHERE created_by = ? AND deleted_at IS NULL
	`, userID).Scan(&sc).Error; err != nil {
		logger.Log.Errorf("GuruStats: failed to query schedule counts: %v", err)
		response.InternalError(c, "Gagal memuat statistik")
		return
	}
	stats.ActiveSchedules = sc.Active
	stats.TotalSchedules = sc.Total

	if err := h.db.Model(&entity.QuestionBank{}).Where("created_by = ? AND deleted_at IS NULL", userID).Count(&stats.TotalBanks).Error; err != nil {
		logger.Log.Errorf("GuruStats: failed to count question banks: %v", err)
	}
	if err := h.db.Model(&entity.Question{}).
		Joins("JOIN question_banks qb ON qb.id = questions.question_bank_id").
		Where("qb.created_by = ? AND qb.deleted_at IS NULL", userID).
		Count(&stats.TotalQuestions).Error; err != nil {
		logger.Log.Errorf("GuruStats: failed to count questions: %v", err)
	}

	response.Success(c, stats)
}

// GET /guru/dashboard/essay-stats — ungraded essay count for the logged-in guru
func (h *DashboardHandler) GetGuruEssayStats(c *gin.Context) {
	userID := c.GetUint("user_id")

	type essayStats struct {
		UngradedEssays   int64 `json:"ungraded_essays"`
		TotalEssays      int64 `json:"total_essays"`
		SchedulesWithEssays int64 `json:"schedules_with_essays"`
	}

	var stats essayStats

	// Count total essay answers and ungraded ones across guru's schedules
	if err := h.db.Raw(`
		SELECT
			COUNT(*) AS total_essays,
			COUNT(*) FILTER (WHERE NOT (ea.answer::jsonb ? 'manual_score')) AS ungraded_essays
		FROM exam_answers ea
		JOIN exam_sessions es ON es.id = ea.exam_session_id
		JOIN exam_schedules esc ON esc.id = es.exam_schedule_id
		JOIN questions q ON q.id = ea.question_id
		WHERE esc.created_by = ?
			AND esc.deleted_at IS NULL
			AND q.question_type = 'esai'
			AND es.status IN ('finished', 'ongoing')
			AND ea.answer IS NOT NULL
	`, userID).Scan(&stats).Error; err != nil {
		logger.Log.Errorf("GuruEssayStats: failed to query essay counts: %v", err)
		response.InternalError(c, "Gagal memuat statistik esai")
		return
	}

	// Count schedules that have ungraded essays
	if err := h.db.Raw(`
		SELECT COUNT(DISTINCT esc.id) AS schedules_with_essays
		FROM exam_answers ea
		JOIN exam_sessions es ON es.id = ea.exam_session_id
		JOIN exam_schedules esc ON esc.id = es.exam_schedule_id
		JOIN questions q ON q.id = ea.question_id
		WHERE esc.created_by = ?
			AND esc.deleted_at IS NULL
			AND q.question_type = 'esai'
			AND es.status IN ('finished', 'ongoing')
			AND ea.answer IS NOT NULL
			AND NOT (ea.answer::jsonb ? 'manual_score')
	`, userID).Scan(&stats.SchedulesWithEssays).Error; err != nil {
		logger.Log.Errorf("GuruEssayStats: failed to count schedules: %v", err)
	}

	response.Success(c, stats)
}

// GET /guru/dashboard/ongoing-exams — real-time list of ongoing exams the guru manages
func (h *DashboardHandler) GetGuruOngoingExams(c *gin.Context) {
	userID := c.GetUint("user_id")

	type ongoingExam struct {
		ScheduleID    uint   `json:"schedule_id"`
		ScheduleName  string `json:"schedule_name"`
		Status        string `json:"status"`
		TotalStudents int64  `json:"total_students"`
		OngoingCount  int64  `json:"ongoing_count"`
		FinishedCount int64  `json:"finished_count"`
		StartTime     string `json:"start_time"`
		EndTime       string `json:"end_time"`
	}

	exams := make([]ongoingExam, 0)
	if err := h.db.Raw(`
		SELECT
			esc.id AS schedule_id,
			esc.name AS schedule_name,
			esc.status,
			esc.start_time,
			esc.end_time,
			COUNT(es.id) AS total_students,
			COUNT(es.id) FILTER (WHERE es.status = 'ongoing') AS ongoing_count,
			COUNT(es.id) FILTER (WHERE es.status = 'finished') AS finished_count
		FROM exam_schedules esc
		LEFT JOIN exam_sessions es ON es.exam_schedule_id = esc.id
		WHERE esc.created_by = ?
			AND esc.deleted_at IS NULL
			AND esc.status IN ('active', 'published')
		GROUP BY esc.id, esc.name, esc.status, esc.start_time, esc.end_time
		ORDER BY esc.start_time ASC
	`, userID).Scan(&exams).Error; err != nil {
		logger.Log.Errorf("GuruOngoingExams: failed to query: %v", err)
		response.InternalError(c, "Gagal memuat data ujian")
		return
	}

	response.Success(c, exams)
}

// GET /guru/dashboard/alerts — notifications/warnings for the guru
func (h *DashboardHandler) GetGuruAlerts(c *gin.Context) {
	userID := c.GetUint("user_id")

	type alert struct {
		Type       string `json:"type"`       // "ungraded_essay", "expiring_schedule"
		Message    string `json:"message"`
		ScheduleID uint   `json:"schedule_id"`
		Count      int64  `json:"count"`
	}

	alerts := make([]alert, 0)

	// Check for ungraded essays per schedule
	type ungradedRow struct {
		ScheduleID   uint   `json:"schedule_id"`
		ScheduleName string `json:"schedule_name"`
		Count        int64  `json:"count"`
	}
	var ungradedRows []ungradedRow
	if err := h.db.Raw(`
		SELECT esc.id AS schedule_id, esc.name AS schedule_name, COUNT(*) AS count
		FROM exam_answers ea
		JOIN exam_sessions es ON es.id = ea.exam_session_id
		JOIN exam_schedules esc ON esc.id = es.exam_schedule_id
		JOIN questions q ON q.id = ea.question_id
		WHERE esc.created_by = ?
			AND esc.deleted_at IS NULL
			AND q.question_type = 'esai'
			AND es.status IN ('finished', 'ongoing')
			AND ea.answer IS NOT NULL
			AND NOT (ea.answer::jsonb ? 'manual_score')
		GROUP BY esc.id, esc.name
	`, userID).Scan(&ungradedRows).Error; err == nil {
		for _, row := range ungradedRows {
			alerts = append(alerts, alert{
				Type:       "ungraded_essay",
				Message:    row.ScheduleName + ": " + formatCount(row.Count) + " esai belum dinilai",
				ScheduleID: row.ScheduleID,
				Count:      row.Count,
			})
		}
	}

	// Check for schedules expiring within 24 hours
	type expiringRow struct {
		ScheduleID   uint   `json:"schedule_id"`
		ScheduleName string `json:"schedule_name"`
		EndTime      string `json:"end_time"`
	}
	var expiringRows []expiringRow
	if err := h.db.Raw(`
		SELECT id AS schedule_id, name AS schedule_name, end_time
		FROM exam_schedules
		WHERE created_by = ?
			AND deleted_at IS NULL
			AND status IN ('active', 'published')
			AND end_time BETWEEN NOW() AND NOW() + INTERVAL '24 hours'
		ORDER BY end_time ASC
	`, userID).Scan(&expiringRows).Error; err == nil {
		for _, row := range expiringRows {
			alerts = append(alerts, alert{
				Type:       "expiring_schedule",
				Message:    row.ScheduleName + " akan berakhir dalam waktu dekat",
				ScheduleID: row.ScheduleID,
			})
		}
	}

	response.Success(c, alerts)
}

func formatCount(n int64) string {
	if n == 1 {
		return "1"
	}
	return fmt.Sprintf("%d", n)
}

// GET /admin/dashboard/alerts — actionable info cards for admin
func (h *DashboardHandler) GetAdminAlerts(c *gin.Context) {
	type adminAlert struct {
		Type    string `json:"type"`    // e.g. "peserta_tanpa_rombel", "rombel_kosong", "bank_soal_kosong"
		Title   string `json:"title"`
		Message string `json:"message"`
		Count   int64  `json:"count"`
		Link    string `json:"link"` // frontend route
		Icon    string `json:"icon"`
		Color   string `json:"color"` // warning, danger, info
	}

	alerts := make([]adminAlert, 0)

	// 1. Peserta tanpa rombel
	var pesertaTanpaRombel int64
	if err := h.db.Raw(`
		SELECT COUNT(*) FROM users
		WHERE role = 'peserta' AND deleted_at IS NULL AND is_active = true
		AND NOT EXISTS (SELECT 1 FROM user_rombels WHERE user_rombels.user_id = users.id)
	`).Scan(&pesertaTanpaRombel).Error; err == nil && pesertaTanpaRombel > 0 {
		alerts = append(alerts, adminAlert{
			Type:    "peserta_tanpa_rombel",
			Title:   "Peserta Belum Masuk Rombel",
			Message: fmt.Sprintf("%d peserta belum terdaftar di rombel manapun", pesertaTanpaRombel),
			Count:   pesertaTanpaRombel,
			Link:    "/admin/rombel-management",
			Icon:    "ti-users-minus",
			Color:   "warning",
		})
	}

	// 2. Rombel kosong (tanpa anggota)
	var rombelKosong int64
	if err := h.db.Raw(`
		SELECT COUNT(*) FROM rombels r
		WHERE r.deleted_at IS NULL
		AND NOT EXISTS (SELECT 1 FROM user_rombels ur WHERE ur.rombel_id = r.id)
	`).Scan(&rombelKosong).Error; err == nil && rombelKosong > 0 {
		alerts = append(alerts, adminAlert{
			Type:    "rombel_kosong",
			Title:   "Rombel Kosong",
			Message: fmt.Sprintf("%d rombel tidak memiliki anggota", rombelKosong),
			Count:   rombelKosong,
			Link:    "/admin/rombel-management",
			Icon:    "ti-users-group",
			Color:   "info",
		})
	}

	// 3. Bank soal kosong (tanpa soal)
	var bankSoalKosong int64
	if err := h.db.Raw(`
		SELECT COUNT(*) FROM question_banks qb
		WHERE qb.deleted_at IS NULL
		AND NOT EXISTS (SELECT 1 FROM questions q WHERE q.question_bank_id = qb.id)
	`).Scan(&bankSoalKosong).Error; err == nil && bankSoalKosong > 0 {
		alerts = append(alerts, adminAlert{
			Type:    "bank_soal_kosong",
			Title:   "Bank Soal Kosong",
			Message: fmt.Sprintf("%d bank soal belum memiliki soal", bankSoalKosong),
			Count:   bankSoalKosong,
			Link:    "/admin/question-banks",
			Icon:    "ti-book-off",
			Color:   "info",
		})
	}

	// 4. Peserta tidak aktif
	var pesertaNonaktif int64
	if err := h.db.Raw(`
		SELECT COUNT(*) FROM users
		WHERE role = 'peserta' AND deleted_at IS NULL AND is_active = false
	`).Scan(&pesertaNonaktif).Error; err == nil && pesertaNonaktif > 0 {
		alerts = append(alerts, adminAlert{
			Type:    "peserta_nonaktif",
			Title:   "Peserta Nonaktif",
			Message: fmt.Sprintf("%d peserta dalam status nonaktif", pesertaNonaktif),
			Count:   pesertaNonaktif,
			Link:    "/admin/users",
			Icon:    "ti-user-off",
			Color:   "secondary",
		})
	}

	response.Success(c, alerts)
}

// GET /admin/dashboard/server-stats
func (h *DashboardHandler) GetServerStats(c *gin.Context) {
	type serverStats struct {
		CPUPercent  float64 `json:"cpu_percent"`
		RAMPercent  float64 `json:"ram_percent"`
		RAMUsed     string  `json:"ram_used"`
		RAMTotal    string  `json:"ram_total"`
		DiskPercent float64 `json:"disk_percent"`
		DiskUsed    string  `json:"disk_used"`
		DiskTotal   string  `json:"disk_total"`
		GoVersion   string  `json:"go_version"`
		NumCPU      int     `json:"num_cpu"`
		NumGoroutine int    `json:"num_goroutine"`
	}

	stats := serverStats{
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
	}

	// CPU
	cpuPercents, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercents) > 0 {
		stats.CPUPercent = math.Round(cpuPercents[0])
	}

	// RAM
	vmem, err := mem.VirtualMemory()
	if err == nil {
		stats.RAMPercent = math.Round(vmem.UsedPercent)
		stats.RAMUsed = formatBytesGo(vmem.Used)
		stats.RAMTotal = formatBytesGo(vmem.Total)
	}

	// Disk
	diskStat, err := disk.Usage("/")
	if err == nil {
		stats.DiskPercent = math.Round(diskStat.UsedPercent)
		stats.DiskUsed = formatBytesGo(diskStat.Used)
		stats.DiskTotal = formatBytesGo(diskStat.Total)
	}

	response.Success(c, stats)
}

func formatBytesGo(b uint64) string {
	const gb = 1024 * 1024 * 1024
	const mb = 1024 * 1024
	if b >= gb {
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	}
	return fmt.Sprintf("%.0f MB", float64(b)/float64(mb))
}

// GET /admin/dashboard/ongoing-exams
func (h *DashboardHandler) GetOngoingExams(c *gin.Context) {
	type ongoingExam struct {
		ScheduleID    uint   `json:"schedule_id"`
		ScheduleName  string `json:"schedule_name"`
		SubjectName   string `json:"subject_name"`
		StartTime     string `json:"start_time"`
		EndTime       string `json:"end_time"`
		OngoingCount  int64  `json:"ongoing_count"`
		FinishedCount int64  `json:"finished_count"`
		TotalStudents int64  `json:"total_students"`
	}

	exams := make([]ongoingExam, 0)
	if err := h.db.Raw(`
		SELECT
			esc.id AS schedule_id,
			esc.name AS schedule_name,
			COALESCE(s.name, '') AS subject_name,
			esc.start_time,
			esc.end_time,
			COUNT(es.id) AS total_students,
			COUNT(es.id) FILTER (WHERE es.status = 'ongoing') AS ongoing_count,
			COUNT(es.id) FILTER (WHERE es.status = 'finished') AS finished_count
		FROM exam_schedules esc
		LEFT JOIN exam_sessions es ON es.exam_schedule_id = esc.id
		LEFT JOIN exam_schedule_question_banks esqb ON esqb.exam_schedule_id = esc.id
		LEFT JOIN question_banks qb ON qb.id = esqb.question_bank_id
		LEFT JOIN subjects s ON s.id = qb.subject_id
		WHERE esc.deleted_at IS NULL
			AND esc.status IN ('active', 'published')
			AND esc.start_time <= NOW()
			AND esc.end_time >= NOW()
		GROUP BY esc.id, esc.name, s.name, esc.start_time, esc.end_time
		ORDER BY esc.start_time ASC
	`).Scan(&exams).Error; err != nil {
		logger.Log.Errorf("OngoingExams: failed to query: %v", err)
		response.InternalError(c, "Gagal memuat data ujian")
		return
	}

	response.Success(c, exams)
}

// GET /admin/dashboard/upcoming-exams
func (h *DashboardHandler) GetUpcomingExams(c *gin.Context) {
	schedules := make([]entity.ExamSchedule, 0)
	h.db.Where("status IN ? AND start_time > ? AND deleted_at IS NULL", []string{"published", "draft"}, time.Now()).
		Order("start_time ASC").
		Limit(5).
		Find(&schedules)
	response.Success(c, schedules)
}

// GET /admin/dashboard/recent-activity
func (h *DashboardHandler) GetRecentActivity(c *gin.Context) {
	sessions := make([]entity.ExamSession, 0)
	h.db.Preload("User").Preload("ExamSchedule").
		Where("status IN ?", []string{"finished", "ongoing"}).
		Order("updated_at DESC").
		Limit(10).
		Find(&sessions)
	response.Success(c, sessions)
}

// GET /pengawas/dashboard/stats
func (h *DashboardHandler) GetPengawasStats(c *gin.Context) {
	// Combine all three counts into a single query.
	type pengawasStats struct {
		ActiveSchedules int64 `json:"active_schedules"`
		OngoingSessions int64 `json:"ongoing_sessions"`
		FinishedToday   int64 `json:"finished_today"`
	}
	var stats pengawasStats

	if err := h.db.Raw(`SELECT COUNT(*) AS active_schedules FROM exam_schedules WHERE status = 'active' AND deleted_at IS NULL`).
		Scan(&stats.ActiveSchedules).Error; err != nil {
		logger.Log.Errorf("PengawasStats: failed to query active schedules: %v", err)
		response.InternalError(c, "Gagal memuat statistik")
		return
	}
	if err := h.db.Raw(`
		SELECT
			COUNT(*) FILTER (WHERE status = 'ongoing') AS ongoing_sessions,
			COUNT(*) FILTER (WHERE status = 'finished' AND DATE(finished_at) = CURRENT_DATE) AS finished_today
		FROM exam_sessions
	`).Scan(&stats).Error; err != nil {
		logger.Log.Errorf("PengawasStats: failed to query session counts: %v", err)
		response.InternalError(c, "Gagal memuat statistik")
		return
	}

	response.Success(c, stats)
}

// GET /pengawas/dashboard/monitoring-summary
func (h *DashboardHandler) GetPengawasMonitoringSummary(c *gin.Context) {
	type monitoringSummary struct {
		OnlineStudents  int64 `json:"online_students"`
		ViolationsToday int64 `json:"violations_today"`
		ActiveSessions  int64 `json:"active_sessions"`
		ActiveSchedules int64 `json:"active_schedules"`
	}

	var summary monitoringSummary

	if err := h.db.Raw(`SELECT COUNT(*) FROM exam_sessions WHERE status = 'ongoing'`).
		Scan(&summary.OnlineStudents).Error; err != nil {
		logger.Log.Errorf("PengawasMonitoring: failed to query online students: %v", err)
	}

	if err := h.db.Raw(`SELECT COUNT(*) FROM violation_logs WHERE DATE(created_at) = CURRENT_DATE`).
		Scan(&summary.ViolationsToday).Error; err != nil {
		logger.Log.Errorf("PengawasMonitoring: failed to query violations today: %v", err)
	}

	summary.ActiveSessions = summary.OnlineStudents

	if err := h.db.Raw(`SELECT COUNT(*) FROM exam_schedules WHERE status = 'active' AND deleted_at IS NULL`).
		Scan(&summary.ActiveSchedules).Error; err != nil {
		logger.Log.Errorf("PengawasMonitoring: failed to query active schedules: %v", err)
	}

	response.Success(c, summary)
}

// GET /pengawas/dashboard/recent-violations
func (h *DashboardHandler) GetPengawasRecentViolations(c *gin.Context) {
	type violationEntry struct {
		ID            uint      `json:"id"`
		ExamSessionID uint      `json:"exam_session_id"`
		ViolationType string    `json:"violation_type"`
		Description   string    `json:"description"`
		CreatedAt     time.Time `json:"created_at"`
		StudentName   string    `json:"student_name"`
		ScheduleName  string    `json:"schedule_name"`
		ScheduleID    uint      `json:"schedule_id"`
	}

	var violations []violationEntry
	if err := h.db.Raw(`
		SELECT
			vl.id,
			vl.exam_session_id,
			vl.violation_type,
			vl.description,
			vl.created_at,
			u.name AS student_name,
			es2.name AS schedule_name,
			es2.id AS schedule_id
		FROM violation_logs vl
		JOIN exam_sessions e ON e.id = vl.exam_session_id
		JOIN users u ON u.id = e.user_id
		JOIN exam_schedules es2 ON es2.id = e.exam_schedule_id
		WHERE DATE(vl.created_at) = CURRENT_DATE
		ORDER BY vl.created_at DESC
		LIMIT 20
	`).Scan(&violations).Error; err != nil {
		logger.Log.Errorf("PengawasViolations: failed to query recent violations: %v", err)
		response.InternalError(c, "Gagal memuat data pelanggaran")
		return
	}

	if violations == nil {
		violations = []violationEntry{}
	}

	response.Success(c, violations)
}

// GET /pengawas/dashboard/active-rooms
func (h *DashboardHandler) GetPengawasActiveRooms(c *gin.Context) {
	type activeRoom struct {
		ScheduleID      uint   `json:"schedule_id"`
		ScheduleName    string `json:"schedule_name"`
		OnlineStudents  int64  `json:"online_students"`
		TotalStudents   int64  `json:"total_students"`
		ViolationCount  int64  `json:"violation_count"`
		EndTime         string `json:"end_time"`
		DurationMinutes int    `json:"duration_minutes"`
		Status          string `json:"status"`
	}

	var rooms []activeRoom
	if err := h.db.Raw(`
		SELECT
			es.id AS schedule_id,
			es.name AS schedule_name,
			COUNT(*) FILTER (WHERE s.status = 'ongoing') AS online_students,
			COUNT(s.id) AS total_students,
			COALESCE(SUM(s.violation_count), 0) AS violation_count,
			es.end_time,
			es.duration_minutes,
			es.status
		FROM exam_schedules es
		LEFT JOIN exam_sessions s ON s.exam_schedule_id = es.id
		WHERE es.status IN ('active', 'published')
			AND es.deleted_at IS NULL
			AND es.end_time >= NOW()
		GROUP BY es.id, es.name, es.end_time, es.duration_minutes, es.status
		ORDER BY es.start_time ASC
	`).Scan(&rooms).Error; err != nil {
		logger.Log.Errorf("PengawasActiveRooms: failed to query active rooms: %v", err)
		response.InternalError(c, "Gagal memuat data ruangan")
		return
	}

	if rooms == nil {
		rooms = []activeRoom{}
	}

	response.Success(c, rooms)
}

// GET /pengawas/dashboard/all-violations — full list with filters
func (h *DashboardHandler) GetPengawasAllViolations(c *gin.Context) {
	scheduleFilter := c.Query("schedule_id")
	severity := c.Query("severity")
	dateFilter := c.Query("date")

	type violationRow struct {
		ID            uint      `json:"id"`
		ExamSessionID uint      `json:"exam_session_id"`
		ViolationType string    `json:"violation_type"`
		Description   string    `json:"description"`
		CreatedAt     time.Time `json:"created_at"`
		StudentName   string    `json:"student_name"`
		ScheduleName  string    `json:"schedule_name"`
		ScheduleID    uint      `json:"schedule_id"`
	}

	query := `
		SELECT
			vl.id,
			vl.exam_session_id,
			vl.violation_type,
			vl.description,
			vl.created_at,
			u.name AS student_name,
			es2.name AS schedule_name,
			es2.id AS schedule_id
		FROM violation_logs vl
		JOIN exam_sessions e ON e.id = vl.exam_session_id
		JOIN users u ON u.id = e.user_id
		JOIN exam_schedules es2 ON es2.id = e.exam_schedule_id
		WHERE 1=1
	`
	var args []interface{}

	if scheduleFilter != "" {
		query += " AND es2.id = ?"
		args = append(args, scheduleFilter)
	}

	if severity != "" {
		switch severity {
		case "high":
			query += " AND vl.violation_type IN ('fullscreen_exit', 'multi_tab', 'alt_tab', 'external_paste')"
		case "medium":
			query += " AND vl.violation_type IN ('tab_switch', 'blur_extended', 'window_resize', 'focus_lost')"
		case "low":
			query += " AND vl.violation_type IN ('popup_detected', 'background_detected')"
		}
	}

	if dateFilter != "" {
		query += " AND DATE(vl.created_at) = ?"
		args = append(args, dateFilter)
	} else {
		query += " AND vl.created_at >= NOW() - INTERVAL '7 days'"
	}

	query += " ORDER BY vl.created_at DESC LIMIT 200"

	var violations []violationRow
	if err := h.db.Raw(query, args...).Scan(&violations).Error; err != nil {
		logger.Log.Errorf("PengawasAllViolations: failed to query: %v", err)
		response.InternalError(c, "Gagal memuat data pelanggaran")
		return
	}

	if violations == nil {
		violations = []violationRow{}
	}

	response.Success(c, violations)
}
