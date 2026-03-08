# Layered Architecture - CBT Patra (Golang)

**Tanggal:** 2026-03-05
**Diperbarui:** 2026-03-05 (v2 - single project structure)

---

## Architecture Overview

Menggunakan Clean Architecture dengan 4 layer utama:

```
+========================================+
|         PRESENTATION LAYER             |
|  HTTP Handlers, WebSocket Handlers,    |
|  Middleware, Request/Response DTOs     |
+========================================+
             |    ^
             v    |
+========================================+
|         APPLICATION LAYER             |
|  Use Cases (Interactors)              |
|  Application Services                  |
|  DTOs, Mappers                        |
+========================================+
             |    ^
             v    |
+========================================+
|          DOMAIN LAYER                  |
|  Entities, Value Objects              |
|  Domain Services, Domain Events        |
|  Repository Interfaces                 |
+========================================+
             |    ^
             v    |
+========================================+
|       INFRASTRUCTURE LAYER            |
|  GORM Repositories, Redis Client      |
|  MinIO Client, AI HTTP Client         |
|  DB Migrations, External Services     |
+========================================+
```

**Aturan Dependency:**
- Dependency hanya boleh mengarah KE DALAM (presentation -> application -> domain)
- Infrastructure implement interface yang didefinisikan di domain
- Domain TIDAK boleh import infrastructure

---

## Struktur Project (Single Project)

> **Keputusan Arsitektur:** Single project di satu root folder (tidak dipisah backend/frontend).
> Go di root level, Vue.js di subfolder `web/`. Cocok untuk solo developer.
> Satu repo, satu docker-compose, satu deploy.

### Mode Development
```
Terminal 1: go run ./cmd/server     → API di :8080
Terminal 2: cd web && npm run dev   → Vue dev server di :5173 (proxy ke :8080)
```

### Mode Production
```
npm run build           → output ke web/dist/
go build ./cmd/server   → binary baca static dari web/dist/ via embed
Jalankan binary saja    → semua di :8080 (1 proses)
```

---

## Folder Structure

```
patra/                                 # Root project
├── cmd/
│   └── server/
│       └── main.go                    # Entry point Go
│
├── internal/                          # Semua Go code (private)
│   ├── domain/                        # DOMAIN LAYER
│   │   ├── entity/
│   │   │   ├── user.go
│   │   │   ├── question_bank.go
│   │   │   ├── question.go
│   │   │   ├── exam_schedule.go
│   │   │   ├── exam_session.go
│   │   │   ├── exam_answer.go
│   │   │   ├── rombel.go
│   │   │   ├── subject.go
│   │   │   ├── tag.go
│   │   │   ├── room.go
│   │   │   ├── stimulus.go
│   │   │   └── setting.go
│   │   ├── valueobject/
│   │   │   ├── question_type.go       # pg, pgk, esai, etc.
│   │   │   ├── exam_status.go         # upcoming, active, finished
│   │   │   ├── session_status.go      # ongoing, completed, terminated
│   │   │   ├── show_score_after.go    # immediately, after_end_time, manual
│   │   │   └── late_policy.go        # cut_time, full_time
│   │   ├── repository/
│   │   │   ├── user_repository.go     # Interface
│   │   │   ├── question_bank_repository.go
│   │   │   ├── question_repository.go
│   │   │   ├── exam_schedule_repository.go
│   │   │   ├── exam_session_repository.go
│   │   │   ├── exam_answer_repository.go
│   │   │   ├── rombel_repository.go
│   │   │   ├── subject_repository.go
│   │   │   ├── tag_repository.go
│   │   │   └── setting_repository.go
│   │   └── service/
│   │       ├── score_calculator.go    # Domain service: hitung skor
│   │       └── eligibility_checker.go # Domain service: cek peserta eligible
│   │
│   ├── application/                   # APPLICATION LAYER
│   │   ├── usecase/
│   │   │   ├── auth/
│   │   │   │   ├── login_usecase.go
│   │   │   │   ├── logout_usecase.go
│   │   │   │   └── refresh_token_usecase.go
│   │   │   ├── user/
│   │   │   │   ├── create_user_usecase.go
│   │   │   │   ├── update_user_usecase.go
│   │   │   │   ├── delete_user_usecase.go
│   │   │   │   ├── list_users_usecase.go
│   │   │   │   └── import_users_usecase.go
│   │   │   ├── question_bank/
│   │   │   │   ├── create_bank_usecase.go
│   │   │   │   ├── update_bank_usecase.go
│   │   │   │   ├── delete_bank_usecase.go
│   │   │   │   └── list_banks_usecase.go
│   │   │   ├── question/
│   │   │   │   ├── create_question_usecase.go
│   │   │   │   ├── update_question_usecase.go
│   │   │   │   ├── delete_question_usecase.go
│   │   │   │   ├── list_questions_usecase.go
│   │   │   │   ├── import_questions_usecase.go
│   │   │   │   └── ai_generate_questions_usecase.go
│   │   │   ├── exam_schedule/
│   │   │   │   ├── create_schedule_usecase.go
│   │   │   │   ├── update_schedule_usecase.go
│   │   │   │   ├── delete_schedule_usecase.go
│   │   │   │   └── list_schedules_usecase.go
│   │   │   ├── exam/
│   │   │   │   ├── confirm_exam_usecase.go
│   │   │   │   ├── start_exam_usecase.go
│   │   │   │   ├── load_session_usecase.go
│   │   │   │   ├── save_answer_usecase.go
│   │   │   │   ├── toggle_flag_usecase.go
│   │   │   │   ├── log_violation_usecase.go
│   │   │   │   ├── finish_exam_usecase.go
│   │   │   │   └── get_result_usecase.go
│   │   │   ├── monitoring/
│   │   │   │   ├── get_live_status_usecase.go
│   │   │   │   └── lock_client_usecase.go
│   │   │   └── report/
│   │   │       ├── get_schedule_report_usecase.go
│   │   │       ├── regrade_usecase.go
│   │   │       ├── ai_grade_essay_usecase.go
│   │   │       └── export_pdf_usecase.go
│   │   └── dto/
│   │       ├── auth_dto.go
│   │       ├── user_dto.go
│   │       ├── question_dto.go
│   │       ├── exam_schedule_dto.go
│   │       ├── exam_session_dto.go
│   │       └── report_dto.go
│   │
│   ├── infrastructure/                # INFRASTRUCTURE LAYER
│   │   ├── persistence/
│   │   │   ├── postgres/
│   │   │   │   ├── db.go
│   │   │   │   ├── user_repo.go
│   │   │   │   ├── question_bank_repo.go
│   │   │   │   ├── question_repo.go
│   │   │   │   ├── exam_schedule_repo.go
│   │   │   │   ├── exam_session_repo.go
│   │   │   │   ├── exam_answer_repo.go
│   │   │   │   └── ...
│   │   │   └── migration/
│   │   │       └── migrate.go
│   │   ├── cache/
│   │   │   └── redis/
│   │   │       ├── client.go
│   │   │       ├── token_store.go
│   │   │       └── question_cache.go
│   │   ├── storage/
│   │   │   └── minio/
│   │   │       └── client.go
│   │   ├── ai/
│   │   │   └── ai_client.go
│   │   └── websocket/
│   │       ├── hub.go
│   │       ├── client.go
│   │       └── events.go
│   │
│   └── presentation/                  # PRESENTATION LAYER
│       ├── http/
│       │   ├── router.go
│       │   ├── static.go              # Serve Vue build output (embed.FS)
│       │   ├── middleware/
│       │   │   ├── auth_middleware.go
│       │   │   ├── role_middleware.go
│       │   │   ├── rate_limit.go
│       │   │   └── logger.go
│       │   └── handler/
│       │       ├── auth_handler.go
│       │       ├── user_handler.go
│       │       ├── rombel_handler.go
│       │       ├── subject_handler.go
│       │       ├── tag_handler.go
│       │       ├── question_bank_handler.go
│       │       ├── question_handler.go
│       │       ├── exam_schedule_handler.go
│       │       ├── exam_handler.go
│       │       ├── monitoring_handler.go
│       │       ├── report_handler.go
│       │       └── setting_handler.go
│       └── websocket/
│           └── ws_handler.go
│
├── pkg/                               # Shared utilities
│   ├── jwt/
│   │   └── jwt.go
│   ├── bcrypt/
│   │   └── bcrypt.go
│   ├── hashid/
│   │   └── hashid.go
│   ├── pagination/
│   │   └── pagination.go
│   ├── validator/
│   │   └── validator.go
│   ├── response/
│   │   └── response.go
│   └── logger/
│       └── logger.go
│
├── config/
│   └── config.go
│
├── migrations/                        # SQL migration files
│   ├── 001_create_users.sql
│   ├── 002_create_question_banks.sql
│   └── ...
│
├── web/                               # Vue.js SPA (bukan "frontend/")
│   ├── src/
│   │   ├── main.ts
│   │   ├── router/
│   │   │   └── index.ts
│   │   ├── stores/
│   │   │   ├── auth.store.ts
│   │   │   ├── exam.store.ts
│   │   │   └── monitoring.store.ts
│   │   ├── composables/
│   │   │   ├── useExamTimer.ts
│   │   │   ├── useWebSocket.ts
│   │   │   └── useApi.ts
│   │   ├── pages/
│   │   │   ├── auth/
│   │   │   │   └── LoginPage.vue
│   │   │   ├── admin/
│   │   │   │   ├── dashboard/
│   │   │   │   ├── users/
│   │   │   │   ├── rombels/
│   │   │   │   └── settings/
│   │   │   ├── guru/
│   │   │   │   ├── question-bank/
│   │   │   │   ├── exam-schedule/
│   │   │   │   └── report/
│   │   │   ├── pengawas/
│   │   │   │   └── monitoring/
│   │   │   └── peserta/
│   │   │       ├── dashboard/
│   │   │       ├── exam/
│   │   │       │   ├── ConfirmPage.vue
│   │   │       │   ├── ExamPage.vue
│   │   │       │   └── ResultPage.vue
│   │   │       └── history/
│   │   ├── components/
│   │   │   ├── ui/
│   │   │   ├── exam/
│   │   │   │   ├── QuestionPG.vue
│   │   │   │   ├── QuestionPGK.vue
│   │   │   │   ├── QuestionEssay.vue
│   │   │   │   ├── QuestionMatching.vue
│   │   │   │   ├── QuestionFillIn.vue
│   │   │   │   ├── QuestionMatrix.vue
│   │   │   │   └── QuestionTrueFalse.vue
│   │   │   ├── monitoring/
│   │   │   │   └── StudentCard.vue
│   │   │   └── layout/
│   │   │       ├── AdminLayout.vue
│   │   │       └── ExamLayout.vue
│   │   ├── api/
│   │   │   ├── auth.api.ts
│   │   │   ├── exam.api.ts
│   │   │   └── ...
│   │   └── types/
│   │       └── index.ts
│   ├── dist/                          # Build output (gitignore, di-embed ke Go binary)
│   ├── index.html
│   ├── vite.config.ts
│   └── package.json
│
├── Docs/                              # Dokumentasi project
├── exampatra/                         # Referensi ExamPatra clone
├── docker-compose.yml                 # Dev infra: PostgreSQL + Redis + MinIO
├── go.mod
├── go.sum
├── .env                               # Config lokal (gitignore)
├── .env.example
├── .gitignore
└── Makefile                           # Shortcut commands
```

### Makefile (Shortcut Commands)
```makefile
# Makefile

.PHONY: dev infra build

# Jalankan infra (Docker: PostgreSQL + Redis + MinIO)
infra:
	docker compose up -d

# Jalankan backend Go (development)
dev-api:
	go run ./cmd/server

# Jalankan frontend Vue (development)
dev-web:
	cd web && npm run dev

# Build Vue untuk production
build-web:
	cd web && npm run build

# Build binary final (embed Vue dist)
build:
	$(MAKE) build-web
	CGO_ENABLED=0 go build -ldflags='-w -s' -o cbt-patra ./cmd/server

# Migrate database
migrate:
	go run ./cmd/server migrate

# Seed data awal
seed:
	go run ./cmd/server seed
```

### Cara Serve Static Files di Go (embed.FS)
```go
// internal/presentation/http/static.go

package http

import (
    "embed"
    "io/fs"
    "net/http"
)

//go:embed ../../../web/dist
var webDist embed.FS

func RegisterStaticRoutes(r *gin.Engine) {
    distFS, _ := fs.Sub(webDist, "web/dist")
    
    // Serve static assets (/assets/*, /favicon.ico)
    r.StaticFS("/assets", http.FS(distFS))
    
    // SPA fallback: semua route non-API return index.html
    r.NoRoute(func(c *gin.Context) {
        if strings.HasPrefix(c.Request.URL.Path, "/api") {
            c.JSON(404, gin.H{"message": "not found"})
            return
        }
        // Return index.html untuk Vue Router handle
        data, _ := fs.ReadFile(distFS, "index.html")
        c.Data(http.StatusOK, "text/html", data)
    })
}
```

---

## Use Case Listing Per Domain

### AUTH
- LoginUseCase: validate credential, issue JWT
- LogoutUseCase: revoke refresh token di Redis
- RefreshTokenUseCase: issue access token baru dari refresh token

### USER
- CreateUserUseCase: buat user + assign role + hash password
- UpdateUserUseCase: update data user
- DeleteUserUseCase: soft delete user
- ListUsersUseCase: list dengan filter + pagination
- ImportUsersUseCase: bulk create dari Excel

### QUESTION BANK
- CreateBankUseCase: buat bank soal baru
- UpdateBankUseCase: update bank soal
- DeleteBankUseCase: soft delete bank soal
- ListBanksUseCase: list dengan filter

### QUESTION
- CreateQuestionUseCase: buat soal per tipe dengan validasi
- UpdateQuestionUseCase: update soal (trigger options_updated_at)
- DeleteQuestionUseCase: hapus soal
- ListQuestionsUseCase: list soal dalam bank
- ImportQuestionsUseCase: bulk import dari Excel/CSV
- AiGenerateQuestionsUseCase: generate soal via AI API

### EXAM SCHEDULE
- CreateScheduleUseCase: buat jadwal ujian
- UpdateScheduleUseCase: update jadwal
- DeleteScheduleUseCase: soft delete jadwal
- ListSchedulesUseCase: list jadwal dengan filter status

### EXAM (Peserta Flow)
- ConfirmExamUseCase: ambil info ujian sebelum mulai
- StartExamUseCase: buat/resume ExamSession, generate question order
- LoadSessionUseCase: load soal + jawaban yang sudah ada
- SaveAnswerUseCase: simpan/update jawaban per soal
- ToggleFlagUseCase: toggle flag ragu-ragu
- LogViolationUseCase: catat pelanggaran, terminate jika melebihi limit
- FinishExamUseCase: selesaikan sesi, hitung skor
- GetResultUseCase: ambil hasil ujian

### MONITORING
- GetLiveStatusUseCase: ambil status semua peserta dalam jadwal
- LockClientUseCase: kirim perintah lock ke client via WebSocket

### REPORT
- GetScheduleReportUseCase: rekap nilai semua peserta
- RegradeUseCase: hitung ulang nilai semua sesi dalam jadwal
- AiGradeEssayUseCase: nilai esai via AI API
- ExportPdfUseCase: generate PDF laporan / kartu peserta

---

## Entity Listing

| Entity | Kolom Utama | Relasi |
|--------|-------------|--------|
| User | id, name, username, email, password, role, last_login_at | HasMany: ExamSession, ExamAnswer |
| UserProfile | id, user_id, nis, class, major | BelongsTo: User |
| Rombel | id, name, grade_level | ManyToMany: User, ExamSchedule |
| Subject | id, name | HasMany: QuestionBank |
| Tag | id, name | ManyToMany: User, ExamSchedule |
| QuestionBank | id, name, subject_id, user_id | HasMany: Question, ManyToMany: ExamSchedule |
| Stimulus | id, question_bank_id, content | HasMany: Question |
| Question | id, question_bank_id, stimulus_id, type, question_body, options, correct_answer_text, sort_order, default_mark, audio_path | BelongsTo: QuestionBank |
| ExamSchedule | id, name, user_id, duration_minutes, start_time, end_time, token, ... | ManyToMany: QuestionBank, Rombel, Tag, User; HasMany: ExamSession |
| ExamSession | id, exam_schedule_id, user_id, start_time, end_time, score, status, violation_count, session_details | BelongsTo: ExamSchedule, User; HasMany: ExamAnswer |
| ExamAnswer | id, exam_session_id, question_id, answer, is_doubtful, score | BelongsTo: ExamSession, Question |
| ExamSupervision | id, exam_schedule_id, room_id, token | BelongsTo: ExamSchedule |
| Room | id, name, capacity | HasMany: ExamSupervision |
| Setting | id, key, value | - |
| RegradeLog | id, exam_session_id, old_score, new_score, reason | BelongsTo: ExamSession |

---

## Repository Interface (Contoh)

```go
// domain/repository/exam_session_repository.go

package repository

import "github.com/cbt-patra/internal/domain/entity"

type ExamSessionRepository interface {
    Create(session *entity.ExamSession) error
    FindByID(id uint) (*entity.ExamSession, error)
    FindByHashID(hashID string) (*entity.ExamSession, error)
    FindActiveByUserAndSchedule(userID, scheduleID uint) (*entity.ExamSession, error)
    Update(session *entity.ExamSession) error
    ListBySchedule(scheduleID uint) ([]*entity.ExamSession, error)
    ListByUser(userID uint, status string) ([]*entity.ExamSession, error)
}
```

```go
// domain/repository/question_repository.go

package repository

import "github.com/cbt-patra/internal/domain/entity"

type QuestionRepository interface {
    Create(question *entity.Question) error
    FindByID(id uint) (*entity.Question, error)
    FindByIDs(ids []uint) ([]*entity.Question, error)
    Update(question *entity.Question) error
    Delete(id uint) error
    ListByBankID(bankID uint, opts ListOptions) ([]*entity.Question, int64, error)
    CountByBankID(bankID uint) (int64, error)
    BulkCreate(questions []*entity.Question) error
}
```

---

## Domain Events

| Event | Trigger | Handler |
|-------|---------|---------|
| ExamSessionStarted | StartExamUseCase | Notify WebSocket hub: student joined |
| ExamSessionFinished | FinishExamUseCase | Notify WebSocket hub: student finished |
| ViolationLogged | LogViolationUseCase | Notify WebSocket hub: violation logged |
| SessionTerminated | LogViolationUseCase | Notify WebSocket hub: student terminated |
| ScoreCalculated | FinishExamUseCase | (optional) send notification |

---

## Data Flow Diagrams

### Save Answer Flow
```
Client (Vue) 
  |
  | POST /api/exam/session/{id}/save-answer
  | Body: { question_id, answer, is_doubtful }
  |
  v
auth_middleware.go (validate JWT)
  |
  v
exam_handler.go (parse request, call use case)
  |
  v
SaveAnswerUseCase
  |-- Load ExamSession from repo (validate ownership + status ongoing)
  |-- Load Question from repo (validate belongs to session's schedule)
  |-- Upsert ExamAnswer
  |
  v
ExamAnswerRepository.Upsert()
  |
  v
PostgreSQL (UPSERT exam_answers WHERE session_id + question_id)
  |
  v
Response: { success: true }
  |
  v
Client (optional) Broadcast ke WebSocket hub
```

---

## DI Configuration

Menggunakan manual DI (no framework, keep it simple):

```go
// cmd/server/main.go (pattern)

// Infrastructure
db := postgres.NewDB(cfg)
redisClient := redis.NewClient(cfg)
wsHub := websocket.NewHub()

// Repositories
userRepo := postgres.NewUserRepository(db)
questionRepo := postgres.NewQuestionRepository(db)
examSessionRepo := postgres.NewExamSessionRepository(db)
examAnswerRepo := postgres.NewExamAnswerRepository(db)

// Domain Services
scoreCalc := domain.NewScoreCalculator()

// Use Cases
loginUC := auth.NewLoginUseCase(userRepo, redisClient, cfg)
startExamUC := exam.NewStartExamUseCase(examSessionRepo, questionRepo, ...)
saveAnswerUC := exam.NewSaveAnswerUseCase(examSessionRepo, examAnswerRepo, ...)

// Handlers
authHandler := handler.NewAuthHandler(loginUC)
examHandler := handler.NewExamHandler(startExamUC, saveAnswerUC, ...)

// Router
router := presentation.NewRouter(authHandler, examHandler, wsHub, ...)
```

---

## Testing Strategy

| Level | Tool | Coverage Target |
|-------|------|-----------------|
| Unit Test | Go testing + testify | Domain service, Use case logic |
| Integration Test | Go testing + testcontainers | Repository, DB queries |
| E2E Test | Playwright (frontend) | Critical user flows |

**Pattern:** Arrange-Act-Assert

```go
// Contoh unit test use case
func TestSaveAnswerUseCase_ShouldSaveAnswer(t *testing.T) {
    // Arrange
    mockSessionRepo := mocks.NewExamSessionRepository(t)
    mockAnswerRepo := mocks.NewExamAnswerRepository(t)
    
    session := &entity.ExamSession{ID: 1, Status: "ongoing", UserID: 100}
    mockSessionRepo.On("FindByID", uint(1)).Return(session, nil)
    mockAnswerRepo.On("Upsert", mock.AnythingOfType("*entity.ExamAnswer")).Return(nil)
    
    uc := exam.NewSaveAnswerUseCase(mockSessionRepo, mockAnswerRepo)
    
    // Act
    err := uc.Execute(context.Background(), dto.SaveAnswerDTO{
        SessionID:  1,
        UserID:     100,
        QuestionID: 5,
        Answer:     "A",
        IsDoubtful: false,
    })
    
    // Assert
    assert.NoError(t, err)
    mockAnswerRepo.AssertExpectations(t)
}
```
