package http

import (
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/config"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/internal/presentation/http/handler"
	"github.com/omanjaya/patra/internal/presentation/http/middleware"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/response"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

type Handlers struct {
	Auth               *handler.AuthHandler
	User               *handler.UserHandler
	Rombel             *handler.RombelHandler
	Subject            *handler.SubjectHandler
	Tag                *handler.TagHandler
	Room               *handler.RoomHandler
	Setting            *handler.SettingHandler
	Backup             *handler.BackupHandler
	QuestionBank       *handler.QuestionBankHandler
	Question           *handler.QuestionHandler
	QuestionImport     *handler.QuestionImportHandler
	ExamSchedule       *handler.ExamScheduleHandler
	ExamSession        *handler.ExamSessionHandler
	WS                 *handler.WSHandler
	Report             *handler.ReportHandler
	Export             *handler.ExportHandler
	Dashboard          *handler.DashboardHandler
	Profile            *handler.ProfileHandler
	SupervisionActions *handler.SupervisionActionsHandler
	Audio              *handler.AudioHandler
	Permission         *handler.PermissionHandler
	Role               *handler.RoleHandler
	Card               *handler.CardHandler
	AuditLog           *handler.AuditLogHandler
	Database           *handler.DatabaseHandler
	QuestionExport     *handler.QuestionExportHandler
	LiveScore          *handler.LiveScoreHandler
	SupervisionSetup   *handler.SupervisionSetupHandler
	PWA                *handler.PWAHandler
}

func NewRouter(cfg *config.Config, h Handlers, settingRepo repository.SettingRepository, db *gorm.DB) *gin.Engine {
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.SecurityHeaders(cfg.App.Env))

	if cfg.App.Env == "production" && cfg.CORS.AllowedOrigins == "*" {
		logger.Log.Warn("CORS: wildcard origin '*' is not allowed in production, using default")
		cfg.CORS.AllowedOrigins = "https://localhost"
	}

	origins := strings.Split(cfg.CORS.AllowedOrigins, ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}

	if cfg.App.Env == "production" {
		filtered := make([]string, 0, len(origins))
		for _, origin := range origins {
			lower := strings.ToLower(origin)
			if strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1") {
				logger.Log.Warnf("CORS: localhost origin '%s' detected in production, removing", origin)
				continue
			}
			filtered = append(filtered, origin)
		}
		if len(filtered) == 0 {
			logger.Log.Warn("CORS: no valid origins after filtering localhost in production")
			filtered = append(filtered, "https://example.com")
		}
		origins = filtered
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Serve uploaded files (avatars, etc.)
	r.Static("/uploads", "./uploads")

	// Public audio streaming (no auth required for exam audio)
	r.GET("/audio-stream/:filename", h.Audio.Stream)

	r.GET("/api/v1/health", func(c *gin.Context) {
		status := "healthy"
		checks := gin.H{}

		// Check DB
		sqlDB, _ := db.DB()
		if err := sqlDB.Ping(); err != nil {
			status = "degraded"
			checks["database"] = "down"
		} else {
			checks["database"] = "up"
		}

		response.Success(c, gin.H{
			"status":  status,
			"version": "1.0.0",
			"checks":  checks,
		})
	})

	// PWA validation (no auth required)
	r.GET("/pwa/validate", h.PWA.ValidatePWA)

	api := r.Group("/api/v1")

	// Public branding (no auth)
	api.GET("/branding", h.Setting.GetBranding)

	// Public — rate limited
	authGroup := api.Group("")
	authGroup.Use(middleware.RateLimit(rate.Limit(5), 10))
	{
		authGroup.POST("/auth/login", h.Auth.Login)
		authGroup.POST("/auth/refresh", h.Auth.RefreshToken)
	}

	// Helper: permission check shorthand
	perm := func(permissions ...string) gin.HandlerFunc {
		return middleware.PermissionMiddleware(db, permissions...)
	}

	// Protected
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(cfg.JWT.AccessSecret))
	protected.Use(middleware.ActivityLogger())
	{
		// Auth
		protected.POST("/auth/logout", h.Auth.Logout)
		protected.GET("/auth/me", h.Auth.Me)

		// Profile (all roles)
		protected.GET("/profile", h.Profile.Get)
		protected.PUT("/profile", h.Profile.Update)
		protected.PUT("/profile/password", h.Profile.ChangePassword)
		protected.POST("/profile/avatar", h.Profile.UploadAvatar)

		// Admin only
		admin := protected.Group("")
		admin.Use(middleware.RoleMiddleware(entity.RoleAdmin))
		{
			// Users
			admin.GET("/admin/users", perm("user-list"), h.User.List)
			admin.GET("/admin/users/search-peserta", perm("user-list"), h.User.SearchPeserta)
			admin.GET("/admin/users/trashed", perm("user-view-trash"), h.User.ListTrashed)
			admin.POST("/admin/users", perm("user-create"), h.User.Create)
			admin.PUT("/admin/users/:id", perm("user-edit"), h.User.Update)
			admin.DELETE("/admin/users/:id", perm("user-delete"), h.User.Delete)
			admin.POST("/admin/users/:id/restore", perm("user-restore"), h.User.Restore)
			admin.DELETE("/admin/users/:id/force", perm("user-force-delete"), h.User.ForceDelete)
			admin.POST("/admin/users/import", perm("user-create"), h.User.ImportExcel)
			admin.GET("/admin/users/import/template", perm("user-create"), h.User.DownloadTemplate)
			admin.POST("/admin/users/bulk-action", perm("user-delete"), h.User.BulkAction)

			// Dashboard
			admin.GET("/admin/dashboard/stats", h.Dashboard.GetStats)
			admin.GET("/admin/dashboard/server-stats", h.Dashboard.GetServerStats)
			admin.GET("/admin/dashboard/ongoing-exams", h.Dashboard.GetOngoingExams)
			admin.GET("/admin/dashboard/upcoming-exams", h.Dashboard.GetUpcomingExams)
			admin.GET("/admin/dashboard/recent-activity", h.Dashboard.GetRecentActivity)
			admin.GET("/admin/dashboard/alerts", h.Dashboard.GetAdminAlerts)

			// Rombels
			admin.GET("/admin/rombels", perm("rombel-list"), h.Rombel.List)
			admin.POST("/admin/rombels", perm("rombel-create"), h.Rombel.Create)
			admin.PUT("/admin/rombels/:id", perm("rombel-edit"), h.Rombel.Update)
			admin.DELETE("/admin/rombels/:id", perm("rombel-delete"), h.Rombel.Delete)
			admin.POST("/admin/rombels/bulk-delete", perm("rombel-delete"), h.Rombel.BulkDelete)
			admin.POST("/admin/rombels/:id/assign-users", perm("rombel-edit"), h.Rombel.AssignUsers)
			admin.POST("/admin/rombels/:id/remove-users", perm("rombel-edit"), h.Rombel.RemoveUsers)

			// Subjects
			admin.GET("/admin/subjects", perm("subject-list"), h.Subject.List)
			admin.GET("/admin/subjects/all", perm("subject-list"), h.Subject.ListAll)
			admin.POST("/admin/subjects", perm("subject-create"), h.Subject.Create)
			admin.PUT("/admin/subjects/:id", perm("subject-edit"), h.Subject.Update)
			admin.DELETE("/admin/subjects/:id", perm("subject-delete"), h.Subject.Delete)
			admin.POST("/admin/subjects/bulk-delete", perm("subject-delete"), h.Subject.BulkDelete)

			// Tags
			admin.GET("/admin/tags", perm("tag-list"), h.Tag.List)
			admin.GET("/admin/tags/all", perm("tag-list"), h.Tag.ListAll)
			admin.POST("/admin/tags", perm("tag-create"), h.Tag.Create)
			admin.PUT("/admin/tags/:id", perm("tag-edit"), h.Tag.Update)
			admin.DELETE("/admin/tags/:id", perm("tag-delete"), h.Tag.Delete)
			admin.POST("/admin/tags/bulk-delete", perm("tag-delete"), h.Tag.BulkDelete)
			admin.POST("/admin/tags/:id/assign-users", perm("tag-edit"), h.Tag.AssignUsers)
			admin.POST("/admin/tags/:id/remove-users", perm("tag-edit"), h.Tag.RemoveUsers)
			admin.POST("/admin/tags/import-users", perm("tag-edit"), h.Tag.ImportUserTags)
			admin.GET("/admin/tags/export-template", perm("tag-list"), h.Tag.ExportTemplate)

			// Permissions (for MasterPermissionsPage & UserPermissionsPage)
			admin.GET("/admin/permissions", perm("permission-list"), h.Permission.List)
			admin.GET("/admin/permissions/all", perm("permission-list"), h.Permission.ListAll)
			admin.GET("/admin/permissions/groups", perm("permission-list"), h.Permission.ListGroups)
			admin.POST("/admin/permissions", perm("permission-create"), h.Permission.Create)
			admin.PUT("/admin/permissions/:id", perm("permission-edit"), h.Permission.Update)
			admin.DELETE("/admin/permissions/:id", perm("permission-delete"), h.Permission.Delete)

			// User-Permission management
			admin.GET("/admin/user-permissions", perm("permission-list"), h.Permission.ListUsersWithPermissions)
			admin.POST("/admin/user-permissions/assign", perm("permission-edit"), h.Permission.AssignPermissionToUsers)
			admin.POST("/admin/user-permissions/remove", perm("permission-edit"), h.Permission.RemovePermissionFromUsers)

			// Roles
			admin.GET("/admin/roles", perm("role-list"), h.Role.List)
			admin.POST("/admin/roles", perm("role-create"), h.Role.Create)
			admin.PUT("/admin/roles/:id", perm("role-edit"), h.Role.Update)
			admin.DELETE("/admin/roles/:id", perm("role-delete"), h.Role.Delete)
			admin.GET("/admin/roles/:id/permissions", perm("role-list"), h.Role.GetPermissions)
			admin.POST("/admin/roles/:id/permissions", perm("role-edit"), h.Role.AssignPermissions)

			// Rooms
			admin.GET("/admin/rooms", perm("room-list"), h.Room.List)
			admin.POST("/admin/rooms", perm("room-create"), h.Room.Create)
			admin.PUT("/admin/rooms/:id", perm("room-edit"), h.Room.Update)
			admin.DELETE("/admin/rooms/:id", perm("room-delete"), h.Room.Delete)
			admin.POST("/admin/rooms/bulk-delete", perm("room-delete"), h.Room.BulkDelete)
			admin.GET("/admin/rooms/:id/users", perm("room-list"), h.Room.GetUsers)
			admin.POST("/admin/rooms/:id/assign-users", perm("room-edit"), h.Room.AssignUsers)
			admin.POST("/admin/rooms/:id/remove-users", perm("room-edit"), h.Room.RemoveUsers)

			// Settings
			admin.GET("/admin/settings", perm("setting-manage"), h.Setting.GetAll)
			admin.POST("/admin/settings", perm("setting-manage"), h.Setting.Update)
			admin.POST("/admin/settings/upload-branding", perm("setting-manage"), h.Setting.UploadBranding)

			// Settings backup/restore + AI test
			admin.POST("/settings/backup", perm("setting-manage"), h.Backup.CreateBackup)
			admin.GET("/settings/backup/download/:filename", perm("setting-manage"), h.Backup.DownloadBackup)
			admin.POST("/settings/restore", perm("setting-manage"), h.Backup.RestoreBackup)
			admin.POST("/settings/ai/test", perm("setting-manage"), h.Backup.TestAIConnection)

			// Database export/import (pg_dump/pg_restore)
			admin.POST("/admin/settings/database/export", perm("setting-manage"), h.Database.ExportDatabase)
			admin.POST("/admin/settings/database/import", perm("setting-manage"), h.Database.ImportDatabase)
			admin.GET("/admin/settings/database/backups", perm("setting-manage"), h.Database.ListBackups)
			admin.DELETE("/admin/settings/database/backups/:filename", perm("setting-manage"), h.Database.DeleteBackup)
			admin.POST("/admin/settings/database/export-save", perm("setting-manage"), h.Database.ExportAndSave)

			// Panic Mode
			admin.GET("/settings/panic-mode/status", perm("setting-manage"), h.Setting.PanicModeStatus)
			admin.POST("/settings/panic-mode/activate", perm("setting-manage"), h.Setting.PanicModeActivate)
			admin.POST("/settings/panic-mode/deactivate", perm("setting-manage"), h.Setting.PanicModeDeactivate)

			// System Actions
			admin.POST("/settings/system/clear-cache", perm("setting-manage"), h.Setting.ClearCache)

			// MinIO Settings & Backup
			admin.POST("/settings/minio", perm("setting-manage"), h.Setting.SaveMinioSettings)
			admin.GET("/settings/minio/test", perm("setting-manage"), h.Setting.TestMinioConnection)
			admin.POST("/settings/backup/minio", perm("setting-manage"), h.Setting.BackupToMinio)
			admin.GET("/settings/backup/minio/list", perm("setting-manage"), h.Setting.ListMinioBackups)
			admin.POST("/settings/restore/minio", perm("setting-manage"), h.Setting.RestoreFromMinio)

			// Admin-only supervision: reset session
			admin.POST("/monitoring/:scheduleId/sessions/:sessionId/reset", perm("supervision-action"), h.SupervisionActions.Reset)

			// Admin: preview as peserta
			admin.POST("/admin/preview-as-peserta", h.Auth.PreviewAsPeserta)
			admin.POST("/admin/preview-back", h.Auth.PreviewBack)

			// Audit Logs
			admin.GET("/admin/audit-logs", h.AuditLog.List)

			// Card Generator
			admin.GET("/admin/cards", h.Card.GetCards)
			admin.GET("/admin/cards/settings", h.Card.GetCardSettings)
			admin.GET("/admin/cards/qr", h.Card.GenerateQR)
			admin.GET("/admin/cards/with-qr", h.Card.GetCardsWithQR)

			// Chunked Restore
			admin.POST("/admin/settings/restore/chunk", perm("setting-manage"), h.Setting.UploadRestoreChunk)
			admin.POST("/admin/settings/restore/process", perm("setting-manage"), h.Setting.ProcessRestore)
			admin.GET("/admin/settings/restore/progress/:restoreId", perm("setting-manage"), h.Setting.RestoreProgress)

			// Supervision Setup (admin only)
			admin.GET("/admin/supervision/:scheduleId/setup", perm("exam-schedule-edit"), h.SupervisionSetup.GetSupervisionSetup)
			admin.POST("/admin/supervision/:scheduleId/generate-tokens", perm("exam-schedule-edit"), h.SupervisionSetup.GenerateTokens)
			admin.POST("/admin/supervision/:scheduleId/rooms/:roomId/regenerate-token", perm("exam-schedule-edit"), h.SupervisionSetup.RegenerateToken)
		}

		// Guru + Admin access
		staff := protected.Group("")
		staff.Use(middleware.RoleMiddleware(entity.RoleAdmin, entity.RoleGuru))
		{
			staff.GET("/subjects", h.Subject.ListAll)
			staff.GET("/rombels/all", h.Rombel.List)

			// Guru dashboard stats
			staff.GET("/guru/dashboard/stats", h.Dashboard.GetGuruStats)
			staff.GET("/guru/dashboard/upcoming-exams", h.Dashboard.GetUpcomingExams)
			staff.GET("/guru/dashboard/recent-activity", h.Dashboard.GetRecentActivity)
			staff.GET("/guru/dashboard/essay-stats", h.Dashboard.GetGuruEssayStats)
			staff.GET("/guru/dashboard/ongoing-exams", h.Dashboard.GetGuruOngoingExams)
			staff.GET("/guru/dashboard/alerts", h.Dashboard.GetGuruAlerts)

			// Question Banks
			staff.GET("/question-banks", perm("question-bank-list"), h.QuestionBank.List)
			staff.GET("/question-banks/:id", perm("question-bank-list"), h.QuestionBank.GetByID)
			staff.POST("/question-banks", perm("question-bank-create"), h.QuestionBank.Create)
			staff.PUT("/question-banks/:id", perm("question-bank-edit"), h.QuestionBank.Update)
			staff.DELETE("/question-banks/:id", perm("question-bank-delete"), h.QuestionBank.Delete)
			staff.POST("/question-banks/bulk-delete", perm("question-bank-delete"), h.QuestionBank.BulkDelete)
			staff.PATCH("/question-banks/:id/toggle-status", perm("question-bank-edit"), h.QuestionBank.ToggleStatus)

			// Questions
			staff.GET("/question-banks/:id/questions", perm("question-bank-list"), h.Question.List)
			staff.POST("/question-banks/:id/questions", perm("question-bank-create"), h.Question.Create)
			staff.PUT("/questions/:id", perm("question-bank-edit"), h.Question.Update)
			staff.DELETE("/questions/:id", perm("question-bank-delete"), h.Question.Delete)
			staff.POST("/questions/bulk-action", perm("question-bank-edit"), h.Question.BulkAction)
			staff.PATCH("/question-banks/:id/questions/reorder", perm("question-bank-edit"), h.Question.Reorder)
			staff.POST("/question-banks/:id/clone", perm("question-bank-create"), h.QuestionBank.Clone)
			staff.POST("/question-banks/:id/import", perm("question-bank-create"), h.QuestionImport.Import)
			staff.POST("/question-banks/:id/import/text", perm("question-bank-create"), h.QuestionImport.ImportText)
			staff.GET("/question-banks/:id/import/template", perm("question-bank-list"), h.QuestionImport.DownloadTemplate)

			// Stimuli
			staff.GET("/question-banks/:id/stimuli", perm("question-bank-list"), h.Question.ListStimuli)
			staff.POST("/question-banks/:id/stimuli", perm("question-bank-create"), h.Question.CreateStimulus)
			staff.PUT("/stimuli/:stimulusId", perm("question-bank-edit"), h.Question.UpdateStimulus)
			staff.DELETE("/stimuli/:stimulusId", perm("question-bank-delete"), h.Question.DeleteStimulus)

			// Exam Schedules
			staff.GET("/exam-schedules", perm("exam-schedule-list"), h.ExamSchedule.List)
			staff.GET("/exam-schedules/trashed", perm("exam-schedule-list"), h.ExamSchedule.ListTrashed)
			staff.GET("/exam-schedules/:id", perm("exam-schedule-list"), h.ExamSchedule.GetByID)
			staff.GET("/exam-schedules/:id/preview", perm("exam-schedule-list"), h.ExamSchedule.Preview)
			staff.POST("/exam-schedules", perm("exam-schedule-create"), h.ExamSchedule.Create)
			staff.PUT("/exam-schedules/:id", perm("exam-schedule-edit"), h.ExamSchedule.Update)
			staff.PATCH("/exam-schedules/:id/status", perm("exam-schedule-edit"), h.ExamSchedule.UpdateStatus)
			staff.DELETE("/exam-schedules/:id", perm("exam-schedule-delete"), h.ExamSchedule.Delete)
			staff.POST("/exam-schedules/:id/restore", perm("exam-schedule-edit"), h.ExamSchedule.Restore)
			staff.DELETE("/exam-schedules/:id/force", perm("exam-schedule-delete"), h.ExamSchedule.ForceDelete)
			staff.POST("/exam-schedules/:id/clone", perm("exam-schedule-create"), h.ExamSchedule.Clone)
			staff.GET("/exam-schedules/:id/sessions", perm("exam-schedule-list"), h.ExamSession.GetAvailable)
			staff.GET("/exam-schedules/:id/sessions/ongoing", perm("exam-schedule-list"), h.ExamSession.ListOngoingBySchedule)
			staff.GET("/exam-schedules/:id/sessions/not-started", perm("exam-schedule-list"), h.ExamSession.ListNotStartedBySchedule)
			staff.POST("/exam-schedules/:id/warm-cache", perm("exam-schedule-edit"), h.ExamSchedule.WarmCache)
			staff.GET("/exam-schedules/:id/cache-status", perm("exam-schedule-list"), h.ExamSchedule.CacheStatus)

			// Reports
			staff.GET("/reports/:scheduleId", perm("report-view"), h.Report.GetScheduleReport)
			staff.GET("/reports/:scheduleId/analysis", perm("report-view"), h.Report.GetExamAnalysis)
			staff.GET("/reports/:scheduleId/sessions/:sessionId", perm("report-view"), h.Report.GetPersonalReport)
			staff.GET("/reports/:scheduleId/export", perm("report-export"), h.Export.LedgerExcel)
			staff.GET("/reports/:scheduleId/unfinished/export", perm("report-export"), h.Export.UnfinishedExcel)
			staff.POST("/reports/:scheduleId/regrade", perm("report-regrade"), h.Report.Regrade)
			staff.GET("/reports/:scheduleId/key-changes", perm("report-view"), h.Report.GetKeyChanges)
			staff.GET("/reports/:scheduleId/regrade-logs", perm("report-view"), h.Report.GetRegradeLogs)
			staff.POST("/exam-sessions/:id/grade-essay", perm("report-regrade"), h.Report.GradeEssay)
			staff.POST("/exam-sessions/:id/ai-grade", perm("report-regrade"), h.Report.AIGradeEssay)
			staff.POST("/exam-sessions/:id/ai-grade-batch", perm("report-regrade"), h.Report.AIGradeBatchEssay)

			// Question Print, Export/Import ZIP, Get All IDs
			staff.GET("/question-banks/:id/questions/print", perm("question-bank-list"), h.Question.PrintQuestions)
			staff.GET("/question-banks/:id/questions/ids", perm("question-bank-list"), h.Question.GetAllIDs)
			staff.GET("/question-banks/:id/export-zip", perm("question-bank-list"), h.QuestionExport.ExportQuestionsZIP)
			staff.POST("/question-banks/:id/import-zip", perm("question-bank-create"), h.QuestionExport.ImportQuestionsZIP)

			// AI Question Generation & MathML & Upload Image
			staff.POST("/admin/questions/generate-ai", perm("question-bank-create"), h.QuestionImport.GenerateAI)
			staff.POST("/admin/questions/convert-mathml", perm("question-bank-create"), h.QuestionImport.ConvertMathML)
			staff.POST("/admin/questions/import/upload-image", perm("question-bank-create"), h.QuestionImport.UploadImage)

			// Live Score
			staff.GET("/admin/live-score/:scheduleId", h.LiveScore.GetLiveData)
			staff.GET("/admin/live-score/:scheduleId/update", h.LiveScore.GetUpdate)
		}

		// Pengawas + Admin access
		pengawas := protected.Group("")
		pengawas.Use(middleware.RoleMiddleware(entity.RoleAdmin, entity.RolePengawas))
		{
			pengawas.GET("/pengawas/dashboard/stats", h.Dashboard.GetPengawasStats)
			pengawas.GET("/pengawas/dashboard/monitoring-summary", h.Dashboard.GetPengawasMonitoringSummary)
			pengawas.GET("/pengawas/dashboard/recent-violations", h.Dashboard.GetPengawasRecentViolations)
			pengawas.GET("/pengawas/dashboard/active-rooms", h.Dashboard.GetPengawasActiveRooms)
			pengawas.GET("/pengawas/dashboard/all-violations", h.Dashboard.GetPengawasAllViolations)

			// Pengawas can also do supervision actions
			pengawas.GET("/monitoring/:scheduleId/clients", perm("supervision-view"), h.WS.GetRoomClients)
			pengawas.GET("/monitoring/:scheduleId/unfinished", perm("supervision-view"), h.SupervisionActions.GetUnfinishedList)
			pengawas.POST("/monitoring/:scheduleId/lock", perm("supervision-action"), h.WS.LockClient)
			pengawas.POST("/monitoring/:scheduleId/sessions/:sessionId/force-finish", perm("supervision-action"), h.SupervisionActions.ForceFinish)
			pengawas.POST("/monitoring/:scheduleId/sessions/:sessionId/extend-time", perm("supervision-action"), h.SupervisionActions.ExtendTime)
			pengawas.POST("/monitoring/:scheduleId/sessions/:sessionId/send-message", perm("supervision-action"), h.SupervisionActions.SendMessage)
			pengawas.POST("/monitoring/:scheduleId/sessions/:sessionId/unlock", perm("supervision-action"), h.SupervisionActions.Unlock)
			pengawas.POST("/monitoring/:scheduleId/sessions/:sessionId/return-to-exam", perm("supervision-action"), h.SupervisionActions.ReturnToExam)
			pengawas.POST("/monitoring/:scheduleId/sessions/:sessionId/force-logout", perm("supervision-action"), h.SupervisionActions.ForceLogout)
			pengawas.POST("/monitoring/:scheduleId/bulk-action", perm("supervision-action"), h.SupervisionActions.BulkAction)
			pengawas.POST("/reports/:scheduleId/finish-all", perm("supervision-action"), h.SupervisionActions.FinishAllOngoing)

			// Room Token Management
			pengawas.GET("/supervision/tokens/:scheduleId", h.SupervisionActions.GetRoomTokens)
			pengawas.POST("/supervision/tokens/:scheduleId", h.SupervisionActions.SaveRoomTokens)

			// Supervision Setup (pengawas access)
			pengawas.GET("/admin/supervision/:scheduleId/global-stats", perm("supervision-view"), h.SupervisionSetup.GetGlobalStats)
			pengawas.GET("/admin/supervision/:scheduleId/rooms/:roomId/students", perm("supervision-view"), h.SupervisionSetup.FetchStudents)
			pengawas.POST("/pengawas/supervision/:scheduleId/exit", h.SupervisionSetup.ExitSession)
		}

		// All authenticated users (exam taking) — panic mode blocks peserta
		// However, Supervisor claim is done by admin/guru/pengawas, so put it under staff or protected
		protected.POST("/supervision/claim", h.SupervisionActions.Claim)

		exam := protected.Group("/exam")
		exam.Use(middleware.PanicMode(settingRepo))
		exam.Use(middleware.IPWhitelist(settingRepo))
		exam.Use(middleware.EnforcePWA(settingRepo))
		exam.Use(middleware.SingleSession(db, cfg.JWT.AccessSecret))
		{
			exam.GET("/available", h.ExamSession.GetAvailable)
			exam.GET("/history", h.ExamSession.GetMyHistory)
			exam.POST("/start", h.ExamSession.Start)
			exam.GET("/sessions/:id", h.ExamSession.LoadSession)
			exam.POST("/sessions/:id/answers", middleware.UserRateLimiter(5, 10), h.ExamSession.SaveAnswer)
			exam.POST("/sessions/:id/answers/batch", middleware.UserRateLimiter(1, 3), h.ExamSession.BatchSaveAnswers)
			exam.GET("/sessions/:id/transition", h.ExamSession.GetTransition)
			exam.POST("/sessions/:id/start-section", h.ExamSession.StartSection)
			exam.POST("/sessions/:id/finish", h.ExamSession.Finish)
			exam.POST("/sessions/:id/violations", middleware.UserRateLimiter(2, 5), h.ExamSession.LogViolation)
			exam.POST("/sessions/:id/questions/:questionId/flag", h.ExamSession.ToggleFlag)
			exam.GET("/sessions/:id/lock-status", h.ExamSession.CheckLockStatus)
			exam.POST("/sessions/:id/beacon-sync", h.ExamSession.BeaconSync)
			exam.GET("/sessions/:id/report", h.Report.GetMyReport)
		}

		// WebSocket endpoint (all authenticated)
		protected.GET("/ws/exam/:scheduleId", h.WS.HandleExam)
	}

	// Serve frontend static files (production)
	if cfg.App.Env == "production" {
		r.Static("/assets", "./web/dist/assets")
		r.Static("/css", "./web/dist/css")
		r.Static("/js", "./web/dist/js")

		// SPA fallback: try static file first, then index.html
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			// Return 404 for API and WS routes
			if strings.HasPrefix(path, "/api/") ||
				strings.HasPrefix(path, "/ws/") {
				response.NotFound(c, "Not found")
				return
			}
			// Try serving as a static file from web/dist
			filePath := "./web/dist" + path
			if _, err := os.Stat(filePath); err == nil {
				c.File(filePath)
				return
			}
			// SPA fallback
			c.File("./web/dist/index.html")
		})
	}

	return r
}
