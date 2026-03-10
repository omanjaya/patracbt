package exam

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/internal/domain/service"
	"github.com/omanjaya/patra/internal/infrastructure/cache"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/types"
)

func init() {
	logger.Init("test")
}

// ─── Mock Repository ───────────────────────────────────────────

type mockSessionRepo struct {
	findByIDBasicFn      func(id uint) (*entity.ExamSession, error)
	findByIDFn           func(id uint) (*entity.ExamSession, error)
	upsertAnswerFn       func(answer *entity.ExamAnswer) error
	countNonEmptyFn      func(sessionID uint) (int, error)
	updateFn             func(s *entity.ExamSession) error
	getAllAnswersFn       func(sessionID uint) ([]entity.ExamAnswer, error)
	batchUpsertAnswersFn func(answers []*entity.ExamAnswer) error
	atomicFinishFn       func(id uint) (int64, error)
}

func (m *mockSessionRepo) FindByIDBasic(id uint) (*entity.ExamSession, error) {
	if m.findByIDBasicFn != nil {
		return m.findByIDBasicFn(id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSessionRepo) FindByID(id uint) (*entity.ExamSession, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSessionRepo) UpsertAnswer(answer *entity.ExamAnswer) error {
	if m.upsertAnswerFn != nil {
		return m.upsertAnswerFn(answer)
	}
	return nil
}

func (m *mockSessionRepo) CountNonEmptyAnswers(sessionID uint) (int, error) {
	if m.countNonEmptyFn != nil {
		return m.countNonEmptyFn(sessionID)
	}
	return 0, nil
}

func (m *mockSessionRepo) Update(s *entity.ExamSession) error {
	if m.updateFn != nil {
		return m.updateFn(s)
	}
	return nil
}

func (m *mockSessionRepo) GetAllAnswers(sessionID uint) ([]entity.ExamAnswer, error) {
	if m.getAllAnswersFn != nil {
		return m.getAllAnswersFn(sessionID)
	}
	return nil, nil
}

func (m *mockSessionRepo) BatchUpsertAnswers(answers []*entity.ExamAnswer) error {
	if m.batchUpsertAnswersFn != nil {
		return m.batchUpsertAnswersFn(answers)
	}
	return nil
}

// Stubs for remaining interface methods — not used in SaveAnswer tests.
func (m *mockSessionRepo) Create(_ *entity.ExamSession) error                        { return nil }
func (m *mockSessionRepo) FindByUserAndSchedule(_, _ uint) (*entity.ExamSession, error) {
	return nil, nil
}
func (m *mockSessionRepo) UpdateStatus(_ uint, _ string, _ *time.Time) error { return nil }
func (m *mockSessionRepo) UpdateScore(_ uint, _, _ float64) error            { return nil }
func (m *mockSessionRepo) IncrementViolation(_ uint) error                   { return nil }
func (m *mockSessionRepo) UpdateExtraTime(_ uint, _ int) error               { return nil }
func (m *mockSessionRepo) ListBySchedule(_ uint, _ pagination.Params) ([]*entity.ExamSession, int64, error) {
	return nil, 0, nil
}
func (m *mockSessionRepo) ListByUser(_ uint) ([]*entity.ExamSession, error) { return nil, nil }
func (m *mockSessionRepo) GetAnswer(_, _ uint) (*entity.ExamAnswer, error)  { return nil, nil }
func (m *mockSessionRepo) GetAllAnswersBySchedule(_ uint) (map[uint][]entity.ExamAnswer, error) {
	return nil, nil
}
func (m *mockSessionRepo) DeleteAnswersBySession(_ uint) error { return nil }
func (m *mockSessionRepo) LogViolation(_ *entity.ViolationLog) error { return nil }
func (m *mockSessionRepo) CountViolations(_ uint) (int, error)      { return 0, nil }
func (m *mockSessionRepo) UserInRombels(_ uint, _ []uint) (bool, error) { return false, nil }
func (m *mockSessionRepo) UserHasTags(_ uint, _ []uint) (bool, error)   { return false, nil }
func (m *mockSessionRepo) GetUserRombelIDs(_ uint) ([]uint, error)      { return nil, nil }
func (m *mockSessionRepo) GetUserTagIDs(_ uint) ([]uint, error)         { return nil, nil }
func (m *mockSessionRepo) FindOngoingByUser(_ uint) (*entity.ExamSession, error) { return nil, nil }
func (m *mockSessionRepo) FindExpiredOngoing() ([]*entity.ExamSession, error)    { return nil, nil }
func (m *mockSessionRepo) ListFinishedBySchedule(_ uint) ([]*entity.ExamSession, error) {
	return nil, nil
}
func (m *mockSessionRepo) ListOngoingBySchedule(_ uint) ([]*entity.ExamSession, error) {
	return nil, nil
}
func (m *mockSessionRepo) ListNotStartedBySchedule(_ uint) ([]*entity.ExamSession, error) {
	return nil, nil
}
func (m *mockSessionRepo) Delete(_ uint) error                              { return nil }
func (m *mockSessionRepo) CountByScheduleAndStatus(_ uint, _ string) (int64, error) { return 0, nil }
func (m *mockSessionRepo) AtomicFinish(id uint) (int64, error) {
	if m.atomicFinishFn != nil {
		return m.atomicFinishFn(id)
	}
	return 1, nil
}
func (m *mockSessionRepo) CreateRegradeLog(_ *entity.RegradeLog) error      { return nil }
func (m *mockSessionRepo) ListRegradeLogs(_ uint) ([]entity.RegradeLog, error) { return nil, nil }
func (m *mockSessionRepo) GetUserRombelNames(_ []uint) (map[uint][]string, error) { return nil, nil }

// Verify interface compliance at compile time.
var _ repository.ExamSessionRepository = (*mockSessionRepo)(nil)

// ─── Mock Schedule Repo (minimal) ──────────────────────────────

type mockScheduleRepo struct {
	findByIDFn func(id uint) (*entity.ExamSchedule, error)
}

func (m *mockScheduleRepo) FindByID(id uint) (*entity.ExamSchedule, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return nil, errors.New("not found")
}

func (m *mockScheduleRepo) Create(_ *entity.ExamSchedule) error { return nil }
func (m *mockScheduleRepo) Update(_ *entity.ExamSchedule) error { return nil }
func (m *mockScheduleRepo) Delete(_ uint) error                 { return nil }
func (m *mockScheduleRepo) List(_ repository.ExamScheduleFilter, _ pagination.Params) ([]*entity.ExamSchedule, int64, error) {
	return nil, 0, nil
}
func (m *mockScheduleRepo) FindByToken(_ string) (*entity.ExamSchedule, error) { return nil, nil }
func (m *mockScheduleRepo) UpdateStatus(_ uint, _ string) error                { return nil }
func (m *mockScheduleRepo) UpdateSupervisionToken(_ uint, _ string) error      { return nil }
func (m *mockScheduleRepo) CountByStatus(_ string) (int64, error)              { return 0, nil }
func (m *mockScheduleRepo) Restore(_ uint) error                               { return nil }
func (m *mockScheduleRepo) ForceDelete(_ uint) error                           { return nil }
func (m *mockScheduleRepo) ListTrashed(_ repository.ExamScheduleFilter, _ pagination.Params) ([]*entity.ExamSchedule, int64, error) {
	return nil, 0, nil
}
func (m *mockScheduleRepo) SetQuestionBanks(_ uint, _ []entity.ExamScheduleQuestionBank) error {
	return nil
}
func (m *mockScheduleRepo) SetRombels(_ uint, _ []uint) error                        { return nil }
func (m *mockScheduleRepo) SetTags(_ uint, _ []uint) error                           { return nil }
func (m *mockScheduleRepo) SetExamRooms(_ uint, _ []entity.ExamScheduleRoom) error   { return nil }
func (m *mockScheduleRepo) SetUsers(_ uint, _ []entity.ExamScheduleUser) error       { return nil }
func (m *mockScheduleRepo) GetUsersBySchedule(_ uint) ([]entity.ExamScheduleUser, error) {
	return nil, nil
}

var _ repository.ExamScheduleRepository = (*mockScheduleRepo)(nil)

// ─── Mock Question Repo (minimal) ──────────────────────────────

type mockQuestionRepo struct{}

func (m *mockQuestionRepo) Create(_ *entity.Question) error { return nil }
func (m *mockQuestionRepo) Update(_ *entity.Question) error { return nil }
func (m *mockQuestionRepo) Delete(_ uint) error             { return nil }
func (m *mockQuestionRepo) FindByID(_ uint) (*entity.Question, error) { return nil, nil }
func (m *mockQuestionRepo) FindByIDs(_ []uint) ([]*entity.Question, error) { return nil, nil }
func (m *mockQuestionRepo) ListByBank(_ uint, _ pagination.Params) ([]*entity.Question, int64, error) {
	return nil, 0, nil
}
func (m *mockQuestionRepo) BatchCreate(_ []*entity.Question) error { return nil }
func (m *mockQuestionRepo) CountByBank(_ uint) (int64, error)      { return 0, nil }
func (m *mockQuestionRepo) FindByBankAndType(_ uint, _ string) ([]*entity.Question, error) {
	return nil, nil
}
func (m *mockQuestionRepo) RandomByBank(_ uint, _ int) ([]*entity.Question, error) {
	return nil, nil
}
func (m *mockQuestionRepo) BulkDelete(_ []uint) error              { return nil }
func (m *mockQuestionRepo) MoveToBank(_ []uint, _ uint) error      { return nil }
func (m *mockQuestionRepo) CopyToBank(_ []uint, _ uint) error      { return nil }
func (m *mockQuestionRepo) BulkCreate(_ []*entity.Question) error  { return nil }
func (m *mockQuestionRepo) Reorder(_ uint, _ []dto.ReorderItem) error { return nil }
func (m *mockQuestionRepo) CreateStimulus(_ *entity.Stimulus) error   { return nil }
func (m *mockQuestionRepo) FindStimulusByID(_ uint) (*entity.Stimulus, error) { return nil, nil }
func (m *mockQuestionRepo) ListStimuliByBank(_ uint) ([]*entity.Stimulus, error) { return nil, nil }
func (m *mockQuestionRepo) UpdateStimulus(_ *entity.Stimulus) error { return nil }
func (m *mockQuestionRepo) DeleteStimulus(_ uint) error             { return nil }
func (m *mockQuestionRepo) ListAllByBank(_ uint) ([]*entity.Question, error) { return nil, nil }
func (m *mockQuestionRepo) ListIDsByBank(_ uint, _ string) ([]uint, error)   { return nil, nil }

var _ repository.QuestionRepository = (*mockQuestionRepo)(nil)

// ─── Test Setup ────────────────────────────────────────────────

func setupTestUseCase(t *testing.T, sessionRepo *mockSessionRepo) *ExamSessionUseCase {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	examCache := cache.NewExamCache(rdb)
	flusher := cache.NewAnswerFlusher(examCache, nil, 10*time.Minute) // nil repo — we won't actually flush

	return &ExamSessionUseCase{
		sessionRepo:  sessionRepo,
		scheduleRepo: &mockScheduleRepo{},
		questionRepo: &mockQuestionRepo{},
		calculator:   service.NewScoreCalculator(),
		examCache:    examCache,
		flusher:      flusher,
	}
}

func makeOngoingSession(id, userID uint) *entity.ExamSession {
	future := time.Now().Add(2 * time.Hour)
	return &entity.ExamSession{
		ID:             id,
		UserID:         userID,
		Status:         entity.SessionStatusOngoing,
		EndTime:        &future,
		ExamScheduleID: 1,
		QuestionOrder:  types.JSON(`[1,2,3]`),
	}
}

// ─── Tests ─────────────────────────────────────────────────────

func TestSaveAnswer_SessionNotFound(t *testing.T) {
	repo := &mockSessionRepo{
		findByIDBasicFn: func(id uint) (*entity.ExamSession, error) {
			return nil, errors.New("record not found")
		},
	}
	uc := setupTestUseCase(t, repo)

	req := dto.SaveAnswerRequest{
		QuestionID: 1,
		Answer:     json.RawMessage(`{"option_index":0}`),
	}

	_, _, _, err := uc.SaveAnswer(999, 1, req)
	if err == nil {
		t.Fatal("expected error for non-existent session")
	}
	if err.Error() != "sesi tidak valid" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSaveAnswer_WrongUser(t *testing.T) {
	session := makeOngoingSession(1, 42)
	repo := &mockSessionRepo{
		findByIDBasicFn: func(id uint) (*entity.ExamSession, error) {
			return session, nil
		},
	}
	uc := setupTestUseCase(t, repo)

	req := dto.SaveAnswerRequest{
		QuestionID: 1,
		Answer:     json.RawMessage(`{"option_index":0}`),
	}

	// UserID 99 != session.UserID 42
	_, _, _, err := uc.SaveAnswer(1, 99, req)
	if err == nil {
		t.Fatal("expected error for wrong user")
	}
	if err.Error() != "sesi tidak valid" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSaveAnswer_NotOngoing(t *testing.T) {
	session := makeOngoingSession(1, 42)
	session.Status = entity.SessionStatusFinished

	repo := &mockSessionRepo{
		findByIDBasicFn: func(id uint) (*entity.ExamSession, error) {
			return session, nil
		},
	}
	uc := setupTestUseCase(t, repo)

	req := dto.SaveAnswerRequest{
		QuestionID: 1,
		Answer:     json.RawMessage(`{"option_index":0}`),
	}

	_, _, _, err := uc.SaveAnswer(1, 42, req)
	if err == nil {
		t.Fatal("expected error for finished session")
	}
	if err.Error() != "sesi ujian tidak aktif" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSaveAnswer_TimeExpired(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	session := &entity.ExamSession{
		ID:             1,
		UserID:         42,
		Status:         entity.SessionStatusOngoing,
		EndTime:        &past,
		ExamScheduleID: 1,
		QuestionOrder:  types.JSON(`[1,2,3]`),
	}

	updateCalled := false
	repo := &mockSessionRepo{
		findByIDBasicFn: func(id uint) (*entity.ExamSession, error) {
			return session, nil
		},
		findByIDFn: func(id uint) (*entity.ExamSession, error) {
			return session, nil
		},
		getAllAnswersFn: func(sessionID uint) ([]entity.ExamAnswer, error) {
			return nil, nil
		},
		updateFn: func(s *entity.ExamSession) error {
			updateCalled = true
			return nil
		},
	}
	uc := setupTestUseCase(t, repo)

	// Also set up schedule repo to return something for finishAndScore
	uc.scheduleRepo = &mockScheduleRepo{
		findByIDFn: func(id uint) (*entity.ExamSchedule, error) {
			return &entity.ExamSchedule{ID: 1}, nil
		},
	}

	req := dto.SaveAnswerRequest{
		QuestionID: 1,
		Answer:     json.RawMessage(`{"option_index":0}`),
	}

	_, _, _, err := uc.SaveAnswer(1, 42, req)
	if err == nil {
		t.Fatal("expected error for expired time")
	}
	if err.Error() != "waktu ujian telah habis" {
		t.Errorf("unexpected error: %v", err)
	}
	if !updateCalled {
		t.Error("expected forceFinish to call Update (session should be finished)")
	}
}

func TestSaveAnswer_Success(t *testing.T) {
	session := makeOngoingSession(1, 42)
	repo := &mockSessionRepo{
		findByIDBasicFn: func(id uint) (*entity.ExamSession, error) {
			return session, nil
		},
	}
	uc := setupTestUseCase(t, repo)

	req := dto.SaveAnswerRequest{
		QuestionID: 1,
		Answer:     json.RawMessage(`{"option_index":0}`),
	}

	result, answered, total, err := uc.SaveAnswer(1, 42, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != 1 {
		t.Errorf("session ID: got %d, want 1", result.ID)
	}
	if answered != 1 {
		t.Errorf("answered: got %d, want 1", answered)
	}
	if total != 3 {
		t.Errorf("total: got %d, want 3", total)
	}
}
