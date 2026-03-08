# Clean Code Practices - CBT Patra (Golang)

**Tanggal:** 2026-03-05

---

## Naming Conventions

### Files & Packages
```
- Package: lowercase, singkat, no underscores
  GOOD: package usecase, package handler, package entity
  BAD:  package UseCase, package exam_handler

- File: snake_case
  GOOD: user_repository.go, exam_session.go
  BAD:  UserRepository.go, examSession.go
```

### Types & Functions
```go
// Struct/Interface/Type: PascalCase
type ExamSession struct { ... }
type UserRepository interface { ... }
type QuestionType string

// Function: PascalCase untuk exported, camelCase untuk unexported
func CreateExamSession() { ... }   // GOOD - exported
func calculateScore() { ... }       // GOOD - unexported, verb-first

// Method: PascalCase untuk exported
func (s *ExamSession) IsCompleted() bool { ... }
func (s *ExamSession) MarkAsTerminated() { ... }

// Variable: camelCase
var examSession *ExamSession
var totalScore float64
var isEligible bool

// Constant: PascalCase atau ALL_CAPS untuk package-level constants
const MaxViolationCount = 3
const DefaultExamDuration = 90

// Boolean: prefix is, has, can
var isActive bool
var hasToken bool
var canFinish bool
```

### Error Types
```go
// Custom error: ErrXxx pattern
var ErrSessionNotFound = errors.New("exam session not found")
var ErrSessionTerminated = errors.New("exam session has been terminated")
var ErrNotEligible = errors.New("user is not eligible for this exam")
```

---

## Function Design

### Single Responsibility
```go
// GOOD: Function hanya melakukan 1 hal
func (uc *StartExamUseCase) checkEligibility(ctx context.Context, userID, scheduleID uint) error {
    schedule, err := uc.scheduleRepo.FindByID(scheduleID)
    if err != nil {
        return fmt.Errorf("schedule not found: %w", err)
    }
    
    if !schedule.IsActive() {
        return ErrExamNotActive
    }
    
    return uc.eligibilityChecker.Check(ctx, userID, schedule)
}

// GOOD: Pisahkan concern shuffle dari start session
func (uc *StartExamUseCase) generateQuestionOrder(questions []*entity.Question, shouldShuffle bool) []uint {
    ids := make([]uint, len(questions))
    for i, q := range questions {
        ids[i] = q.ID
    }
    
    if shouldShuffle {
        rand.Shuffle(len(ids), func(i, j int) {
            ids[i], ids[j] = ids[j], ids[i]
        })
    }
    
    return ids
}
```

### Max 30 Baris Per Function
```go
// GOOD: Pecah function panjang menjadi helper
func (uc *FinishExamUseCase) Execute(ctx context.Context, req dto.FinishExamRequest) (*dto.ExamResultDTO, error) {
    session, err := uc.loadAndValidateSession(ctx, req)
    if err != nil {
        return nil, err
    }
    
    score, err := uc.calculateAndSave(ctx, session)
    if err != nil {
        return nil, err
    }
    
    uc.notifyCompletion(session)
    
    return uc.buildResult(session, score), nil
}
```

### Max 4 Parameter (gunakan struct jika lebih)
```go
// BAD: terlalu banyak parameter
func CreateExamSession(scheduleID, userID uint, shouldShuffle bool, duration int, token string) error { ... }

// GOOD: gunakan DTO/options struct
type StartExamInput struct {
    ScheduleID    uint
    UserID        uint
    Token         string
}

func (uc *StartExamUseCase) Execute(ctx context.Context, input StartExamInput) (*entity.ExamSession, error) { ... }
```

---

## Error Handling

### Custom Error Hierarchy
```go
// pkg/apperror/errors.go

package apperror

import "net/http"

type AppError struct {
    Code       string
    Message    string
    StatusCode int
    Err        error
}

func (e *AppError) Error() string {
    if e.Err != nil {
        return e.Message + ": " + e.Err.Error()
    }
    return e.Message
}

func (e *AppError) Unwrap() error {
    return e.Err
}

// Constructor helpers
func NotFound(code, message string) *AppError {
    return &AppError{Code: code, Message: message, StatusCode: http.StatusNotFound}
}

func Forbidden(code, message string) *AppError {
    return &AppError{Code: code, Message: message, StatusCode: http.StatusForbidden}
}

func BadRequest(code, message string) *AppError {
    return &AppError{Code: code, Message: message, StatusCode: http.StatusBadRequest}
}

func Internal(code, message string, err error) *AppError {
    return &AppError{Code: code, Message: message, StatusCode: http.StatusInternalServerError, Err: err}
}

// Domain error constants
var (
    ErrSessionNotFound    = NotFound("SESSION_NOT_FOUND", "Sesi ujian tidak ditemukan")
    ErrSessionTerminated  = Forbidden("SESSION_TERMINATED", "Sesi ujian telah dihentikan")
    ErrExamNotActive      = Forbidden("EXAM_NOT_ACTIVE", "Ujian tidak sedang berlangsung")
    ErrNotEligible        = Forbidden("NOT_ELIGIBLE", "Anda tidak terdaftar dalam ujian ini")
    ErrMinTimeNotReached  = Forbidden("MIN_TIME_NOT_REACHED", "Waktu minimal pengerjaan belum tercapai")
)
```

### Error Handling Pattern
```go
// GOOD: wrap error dengan context
func (r *ExamSessionRepo) FindByID(id uint) (*entity.ExamSession, error) {
    var session entity.ExamSession
    result := r.db.First(&session, id)
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            return nil, apperror.ErrSessionNotFound
        }
        return nil, apperror.Internal("DB_ERROR", "gagal mengambil sesi", result.Error)
    }
    return &session, nil
}

// GOOD: handle error di handler, tidak di use case
func (h *ExamHandler) SaveAnswer(c *gin.Context) {
    var req dto.SaveAnswerRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, apperror.BadRequest("INVALID_REQUEST", err.Error()))
        return
    }
    
    if err := h.saveAnswerUC.Execute(c.Request.Context(), req); err != nil {
        response.Error(c, err)
        return
    }
    
    response.Success(c, nil)
}
```

### Handler Error Response
```go
// pkg/response/response.go

func Error(c *gin.Context, err error) {
    var appErr *apperror.AppError
    if errors.As(err, &appErr) {
        c.JSON(appErr.StatusCode, gin.H{
            "success": false,
            "code":    appErr.Code,
            "message": appErr.Message,
        })
        return
    }
    
    // Unexpected error: log + return generic 500
    logger.Error("unexpected error", zap.Error(err))
    c.JSON(http.StatusInternalServerError, gin.H{
        "success": false,
        "code":    "INTERNAL_ERROR",
        "message": "Terjadi kesalahan internal",
    })
}
```

---

## Type Safety

### Golang Config
```go
// Strict typing, no interface{} yang tidak perlu
// GOOD: gunakan typed constants
type QuestionType string

const (
    QuestionTypePG          QuestionType = "pg"
    QuestionTypePGK         QuestionType = "pgk"
    QuestionTypeEssay       QuestionType = "esai"
    QuestionTypeMatching    QuestionType = "menjodohkan"
    QuestionTypeFillIn      QuestionType = "singkat"
    QuestionTypeMatrix      QuestionType = "matrix"
    QuestionTypeTrueFalse   QuestionType = "bs"
)

func (qt QuestionType) IsValid() bool {
    valid := map[QuestionType]bool{
        QuestionTypePG: true, QuestionTypePGK: true,
        QuestionTypeEssay: true, QuestionTypeMatching: true,
        QuestionTypeFillIn: true, QuestionTypeMatrix: true,
        QuestionTypeTrueFalse: true,
    }
    return valid[qt]
}
```

### TypeScript Frontend Config
```typescript
// tsconfig.json
{
  "compilerOptions": {
    "strict": true,
    "noImplicitAny": true,
    "strictNullChecks": true
  }
}

// GOOD: typed API response
interface ExamSession {
    id: string
    scheduleId: number
    status: 'ongoing' | 'completed' | 'terminated'
    violationCount: number
    examEndTime: string
}

// GOOD: typed Pinia store
const useExamStore = defineStore('exam', () => {
    const session = ref<ExamSession | null>(null)
    const questions = ref<Question[]>([])
    
    return { session, questions }
})
```

---

## Testing Standards

### Pattern AAA (Arrange-Act-Assert)
```go
func TestStartExamUseCase_ShouldCreateNewSession(t *testing.T) {
    // Arrange
    mockScheduleRepo := mocks.NewExamScheduleRepository(t)
    mockSessionRepo := mocks.NewExamSessionRepository(t)
    mockQuestionRepo := mocks.NewQuestionRepository(t)
    
    schedule := &entity.ExamSchedule{
        ID:               1,
        StartTime:        time.Now().Add(-1 * time.Hour),
        EndTime:          time.Now().Add(1 * time.Hour),
        ShuffleQuestions: false,
    }
    
    mockScheduleRepo.On("FindByID", uint(1)).Return(schedule, nil)
    mockSessionRepo.On("FindActiveByUserAndSchedule", uint(100), uint(1)).Return(nil, apperror.ErrSessionNotFound)
    mockSessionRepo.On("Create", mock.AnythingOfType("*entity.ExamSession")).Return(nil)
    mockQuestionRepo.On("ListBySchedule", uint(1)).Return([]*entity.Question{{ID: 1}, {ID: 2}}, nil)
    
    uc := exam.NewStartExamUseCase(mockScheduleRepo, mockSessionRepo, mockQuestionRepo)
    
    // Act
    session, err := uc.Execute(context.Background(), dto.StartExamInput{
        ScheduleID: 1,
        UserID:     100,
    })
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, session)
    assert.Equal(t, "ongoing", session.Status)
    mockSessionRepo.AssertExpectations(t)
}
```

---

## Logging Standards

```go
// JANGAN gunakan fmt.Println atau log.Println di production code
// GUNAKAN zap logger

// pkg/logger/logger.go
var log *zap.Logger

func Info(msg string, fields ...zap.Field) {
    log.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
    log.Error(msg, fields...)
}

// Penggunaan
logger.Info("exam session started",
    zap.Uint("schedule_id", scheduleID),
    zap.Uint("user_id", userID),
    zap.String("session_id", session.HashID),
)

logger.Error("failed to calculate score",
    zap.Uint("session_id", sessionID),
    zap.Error(err),
)
```

---

## Code Review Checklist

- [ ] Tidak ada `fmt.Println` atau `log.Println` di production code
- [ ] Error di-handle atau di-wrap dengan context yang jelas
- [ ] Tidak ada `interface{}` yang bisa diganti dengan tipe spesifik
- [ ] Function tidak lebih dari 30 baris
- [ ] Parameter function tidak lebih dari 4 (gunakan struct jika lebih)
- [ ] Nama boolean menggunakan prefix is/has/can
- [ ] Tidak ada magic string/number tanpa constant
- [ ] Ada unit test untuk setiap use case baru
- [ ] Repository interface didefinisikan di domain layer
- [ ] Handler tidak berisi business logic (semua di use case)
- [ ] Tidak ada query N+1 (gunakan preload/join)
- [ ] Endpoint baru terdokumentasi di 08-API-DESIGN.md
