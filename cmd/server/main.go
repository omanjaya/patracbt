package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/omanjaya/patra/config"
	aiuc "github.com/omanjaya/patra/internal/application/usecase/ai"
	"github.com/omanjaya/patra/internal/application/usecase/auth"
	examuc "github.com/omanjaya/patra/internal/application/usecase/exam"
	masteruc "github.com/omanjaya/patra/internal/application/usecase/master"
	questionuc "github.com/omanjaya/patra/internal/application/usecase/question"
	reportuc "github.com/omanjaya/patra/internal/application/usecase/report"
	settinguc "github.com/omanjaya/patra/internal/application/usecase/setting"
	useruc "github.com/omanjaya/patra/internal/application/usecase/user"
	examcache "github.com/omanjaya/patra/internal/infrastructure/cache"
	redisclient "github.com/omanjaya/patra/internal/infrastructure/cache/redis"
	"github.com/omanjaya/patra/internal/infrastructure/persistence/postgres"
	"github.com/omanjaya/patra/internal/infrastructure/scheduler"
	wsinfra "github.com/omanjaya/patra/internal/infrastructure/websocket"
	httpserver "github.com/omanjaya/patra/internal/presentation/http"
	"github.com/omanjaya/patra/internal/presentation/http/handler"
	"github.com/omanjaya/patra/pkg/hashid"
	"github.com/omanjaya/patra/pkg/logger"

	"github.com/omanjaya/patra/internal/domain/entity"
	pkgbcrypt "github.com/omanjaya/patra/pkg/bcrypt"
	"gorm.io/gorm"
)

func main() {
	// Tune GOMAXPROCS — honour env var; otherwise use all available cores.
	if maxProcs := os.Getenv("GOMAXPROCS"); maxProcs == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	// Load .env
	_ = godotenv.Load()

	// Config
	cfg := config.Load()
	cfg.Validate()

	// Logger
	logger.Init(cfg.App.Env)
	defer logger.Sync()

	// HashID
	hashid.Init(cfg.HashID.Salt, cfg.HashID.MinLength)

	logger.Log.Infof("CBT Patra starting... (env: %s)", cfg.App.Env)

	// Database
	db := postgres.NewDB(cfg)

	// Auto-migrate (dev only)
	if cfg.App.Env == "development" {
		runMigrations(db)
	}

	// Redis
	rdb := redisclient.NewClient(cfg)

	// Repositories
	userRepo := postgres.NewUserRepository(db)
	rombelRepo := postgres.NewRombelRepository(db)
	subjectRepo := postgres.NewSubjectRepository(db)
	tagRepo := postgres.NewTagRepository(db)
	roomRepo := postgres.NewRoomRepository(db)
	settingRepo := postgres.NewSettingRepository(db)
	questionBankRepo := postgres.NewQuestionBankRepository(db)
	questionRepo := postgres.NewQuestionRepository(db)
	examScheduleRepo := postgres.NewExamScheduleRepository(db)
	examSessionRepo := postgres.NewExamSessionRepository(db)
	permissionRepo := postgres.NewPermissionRepository(db)
	roleRepo := postgres.NewRoleRepository(db)
	auditLogRepo := postgres.NewAuditLogRepository(db)

	// Exam Cache (Redis write-behind for answers)
	examCache := examcache.NewExamCache(rdb)
	answerFlusher := examcache.NewAnswerFlusher(examCache, examSessionRepo, cfg.App.FlushInterval)
	answerFlusher.Start()

	// Use Cases
	loginUC := auth.NewLoginUseCase(userRepo, examSessionRepo, cfg, rdb)
	refreshTokenUC := auth.NewRefreshTokenUseCase(userRepo, cfg)
	userUC := useruc.NewUserUseCase(userRepo, examSessionRepo)
	rombelUC := masteruc.NewRombelUseCase(rombelRepo)
	subjectUC := masteruc.NewSubjectUseCase(subjectRepo)
	tagUC := masteruc.NewTagUseCase(tagRepo)
	roomUC := masteruc.NewRoomUseCase(roomRepo)
	settingUC := settinguc.NewSettingUseCase(settingRepo)
	questionBankUC := questionuc.NewQuestionBankUseCase(questionBankRepo, questionRepo)
	questionUC := questionuc.NewQuestionUseCase(questionRepo, questionBankRepo)
	examScheduleUC := examuc.NewExamScheduleUseCase(examScheduleRepo, examSessionRepo)
	examSessionUC := examuc.NewExamSessionUseCase(examSessionRepo, examScheduleRepo, questionRepo, examCache, answerFlusher)
	reportUC := reportuc.NewReportUseCase(examSessionRepo, examScheduleRepo, questionRepo)
	gradingUC := aiuc.NewGradingUseCase(settingRepo)
	permissionUC := masteruc.NewPermissionUseCase(permissionRepo)
	roleUC := masteruc.NewRoleUseCase(roleRepo)

	// WebSocket Hub
	hub := wsinfra.NewHub()
	go hub.Run()

	// Auto-finish scheduler
	scheduler.StartAutoFinish(examSessionRepo, questionRepo, hub)

	// Ensure required upload directories exist
	// TODO: implement periodic cleanup of orphaned upload files (avatars, audio)
	// when users/questions are soft-deleted
	_ = os.MkdirAll("./uploads/audio", 0755)

	// Handlers + Router
	h := httpserver.Handlers{
		Auth:               handler.NewAuthHandler(loginUC, refreshTokenUC, userRepo, cfg, auditLogRepo),
		User:               handler.NewUserHandler(userUC, db),
		Rombel:             handler.NewRombelHandler(rombelUC),
		Subject:            handler.NewSubjectHandler(subjectUC),
		Tag:                handler.NewTagHandler(tagUC, db),
		Room:               handler.NewRoomHandler(roomUC),
		Setting:            handler.NewSettingHandler(settingUC, settingRepo, rdb, hub),
		Backup:             handler.NewBackupHandler(settingRepo),
		QuestionBank:       handler.NewQuestionBankHandler(questionBankUC),
		Question:           handler.NewQuestionHandler(questionUC),
		QuestionImport:     handler.NewQuestionImportHandler(questionRepo, settingRepo),
		ExamSchedule:       handler.NewExamScheduleHandler(examScheduleUC, questionRepo, rdb),
		ExamSession:        handler.NewExamSessionHandler(examSessionUC, hub),
		WS:                 handler.NewWSHandler(hub, cfg.CORS.AllowedOrigins),
		Report:             handler.NewReportHandler(reportUC, gradingUC, questionRepo, examSessionRepo, userRepo),
		Export:             handler.NewExportHandler(examSessionRepo, examScheduleRepo, questionRepo),
		Dashboard:          handler.NewDashboardHandler(db),
		Profile:            handler.NewProfileHandler(userRepo),
		SupervisionActions: handler.NewSupervisionActionsHandler(examSessionUC, examScheduleUC, hub, auditLogRepo, db),
		Audio:              handler.NewAudioHandler(),
		Permission:         handler.NewPermissionHandler(permissionUC),
		Role:               handler.NewRoleHandler(roleUC),
		Card:               handler.NewCardHandler(db, settingRepo),
		AuditLog:           handler.NewAuditLogHandler(auditLogRepo),
		Database:           handler.NewDatabaseHandler(cfg),
		QuestionExport:     handler.NewQuestionExportHandler(questionUC, questionBankRepo, questionRepo),
		LiveScore:          handler.NewLiveScoreHandler(examSessionRepo, examScheduleRepo, questionRepo),
		SupervisionSetup:   handler.NewSupervisionSetupHandler(examScheduleUC, examSessionUC, auditLogRepo, db),
		PWA:                handler.NewPWAHandler(),
	}
	router := httpserver.NewRouter(cfg, h, settingRepo, db)

	seedAdmin(db)
	seedSettings(settingRepo)
	seedPermissions(db)

	// Start server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.App.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Log.Errorf("Server panic: %v", r)
			}
		}()
		logger.Log.Infof("Server running at http://localhost:%s", cfg.App.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down server...")

	// 1. Stop accepting new WebSocket connections (close all client send channels)
	hub.Stop()

	// 2. Flush Redis answer buffer to PostgreSQL
	answerFlusher.Stop()

	// 3. Shutdown HTTP server (drains active connections)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Errorf("Server forced shutdown: %v", err)
	}

	// 4. Close database connection pool
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}

	logger.Log.Info("Server exited cleanly")
}

func runMigrations(db *gorm.DB) {
	logger.Log.Info("Running auto-migrations...")
	err := db.AutoMigrate(
		&entity.User{},
		&entity.UserProfile{},
		&entity.Rombel{},
		&entity.UserRombel{},
		&entity.Subject{},
		&entity.Tag{},
		&entity.UserTag{},
		&entity.Room{},
		&entity.Setting{},
		&entity.QuestionBank{},
		&entity.Stimulus{},
		&entity.Question{},
		&entity.ExamSchedule{},
		&entity.ExamScheduleQuestionBank{},
		&entity.ExamScheduleRombel{},
		&entity.ExamScheduleTag{},
		&entity.ExamScheduleRoom{},
		&entity.ExamScheduleUser{},
		&entity.ExamSession{},
		&entity.ExamAnswer{},
		&entity.ViolationLog{},
		&entity.RegradeLog{},
		&entity.Permission{},
		&entity.UserPermission{},
		&entity.Role{},
		&entity.ExamSupervision{},
		&entity.AuditLog{},
	)
	if err != nil {
		logger.Log.Fatalf("Migration failed: %v", err)
	}
	logger.Log.Info("Migrations completed")
}

func seedSettings(repo *postgres.SettingRepo) {
	defaults := map[string]string{
		"app_name":          "CBT Patra",
		"ai_api_url":        "",
		"ai_api_key":        "",
		"ai_api_header":     "Authorization",
		"ai_model_params":   "{}",
		"app_footer_text":   "",
		"app_logo":          "",
		"app_favicon":       "",
		"app_primary_color": "",
		"app_header_bg":     "",
		"login_bg_image":    "",
		"login_subtitle":    "Masuk ke akun Anda untuk melanjutkan",
		"school_name":       "",
		"websocket_enabled": "1",
		"panic_mode_active": "0",
		"login_method":      "normal",
		"enforce_pwa_mode":  "0",
	}
	for k, v := range defaults {
		existing, _ := repo.GetByKey(k)
		if existing == nil {
			_ = repo.Set(k, v)
		}
	}
}

func generateRandomPassword(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Use URL-safe base64, trim padding, take first `length` chars
	pw := base64.URLEncoding.EncodeToString(b)
	pw = strings.TrimRight(pw, "=")
	if len(pw) > length {
		pw = pw[:length]
	}
	return pw, nil
}

func seedAdmin(db *gorm.DB) {
	var count int64
	db.Model(&entity.User{}).Where("role = ?", "admin").Count(&count)
	if count > 0 {
		return
	}

	plainPassword, err := generateRandomPassword(16)
	if err != nil {
		logger.Log.Fatalf("Failed to generate admin password: %v", err)
	}

	hashed, _ := pkgbcrypt.HashPassword(plainPassword)
	admin := entity.User{
		Name:                "Administrator",
		Username:            "admin",
		Password:            hashed,
		Role:                "admin",
		ForcePasswordChange: true,
	}
	if err := db.Create(&admin).Error; err != nil {
		logger.Log.Errorf("Gagal seed admin: %v", err)
		return
	}

	logger.Log.Infow("Initial admin account created",
		"username", "admin",
		"password", plainPassword,
		"notice", "Note this password now — it will NOT be shown again! You will be required to change it on first login.",
	)
}

func seedPermissions(db *gorm.DB) {
	type permDef struct {
		Name      string
		GroupName string
	}

	permissions := []permDef{
		// Manajemen User
		{"user-list", "Manajemen User"},
		{"user-create", "Manajemen User"},
		{"user-edit", "Manajemen User"},
		{"user-delete", "Manajemen User"},
		{"user-view-trash", "Manajemen User"},
		{"user-restore", "Manajemen User"},
		{"user-force-delete", "Manajemen User"},

		// Manajemen Rombel
		{"rombel-list", "Manajemen Rombel"},
		{"rombel-create", "Manajemen Rombel"},
		{"rombel-edit", "Manajemen Rombel"},
		{"rombel-delete", "Manajemen Rombel"},

		// Manajemen Mapel
		{"subject-list", "Manajemen Mapel"},
		{"subject-create", "Manajemen Mapel"},
		{"subject-edit", "Manajemen Mapel"},
		{"subject-delete", "Manajemen Mapel"},

		// Manajemen Tag
		{"tag-list", "Manajemen Tag"},
		{"tag-create", "Manajemen Tag"},
		{"tag-edit", "Manajemen Tag"},
		{"tag-delete", "Manajemen Tag"},

		// Manajemen Ruangan
		{"room-list", "Manajemen Ruangan"},
		{"room-create", "Manajemen Ruangan"},
		{"room-edit", "Manajemen Ruangan"},
		{"room-delete", "Manajemen Ruangan"},

		// Manajemen Permission & Role
		{"permission-list", "Manajemen Permission"},
		{"permission-create", "Manajemen Permission"},
		{"permission-edit", "Manajemen Permission"},
		{"permission-delete", "Manajemen Permission"},
		{"role-list", "Manajemen Permission"},
		{"role-create", "Manajemen Permission"},
		{"role-edit", "Manajemen Permission"},
		{"role-delete", "Manajemen Permission"},

		// Manajemen Bank Soal
		{"question-bank-list", "Manajemen Bank Soal"},
		{"question-bank-create", "Manajemen Bank Soal"},
		{"question-bank-edit", "Manajemen Bank Soal"},
		{"question-bank-delete", "Manajemen Bank Soal"},

		// Manajemen Jadwal
		{"exam-schedule-list", "Manajemen Jadwal"},
		{"exam-schedule-create", "Manajemen Jadwal"},
		{"exam-schedule-edit", "Manajemen Jadwal"},
		{"exam-schedule-delete", "Manajemen Jadwal"},

		// Laporan
		{"report-view", "Laporan"},
		{"report-export", "Laporan"},
		{"report-regrade", "Laporan"},

		// Pengaturan
		{"setting-manage", "Pengaturan"},

		// Pengawasan
		{"supervision-view", "Pengawasan"},
		{"supervision-action", "Pengawasan"},
	}

	var created int
	for _, p := range permissions {
		var count int64
		db.Model(&entity.Permission{}).Where("name = ?", p.Name).Count(&count)
		if count == 0 {
			perm := entity.Permission{
				Name:      p.Name,
				GroupName: p.GroupName,
			}
			if err := db.Create(&perm).Error; err != nil {
				logger.Log.Errorf("Gagal seed permission %s: %v", p.Name, err)
			} else {
				created++
			}
		}
	}
	if created > 0 {
		logger.Log.Infof("Seeded %d new permissions", created)
	}
}
