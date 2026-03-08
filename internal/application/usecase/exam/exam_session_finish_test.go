package exam

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/types"
)

// ─── FinishExam Tests ──────────────────────────────────────────

func TestFinishExam_Success(t *testing.T) {
	session := makeOngoingSession(1, 42)

	// Set up questions and answers for scoring
	q1 := &entity.Question{
		ID:           1,
		QuestionType: entity.QuestionTypePG,
		Score:        2.0,
		Options: toJSONHelper([]map[string]interface{}{
			{"text": "A", "is_correct": false},
			{"text": "B", "is_correct": true},
		}),
		QuestionBankID: 10,
	}
	q2 := &entity.Question{
		ID:           2,
		QuestionType: entity.QuestionTypePG,
		Score:        3.0,
		Options: toJSONHelper([]map[string]interface{}{
			{"text": "X", "is_correct": true},
			{"text": "Y", "is_correct": false},
		}),
		QuestionBankID: 10,
	}

	var updatedSession *entity.ExamSession
	repo := &mockSessionRepo{
		findByIDFn: func(id uint) (*entity.ExamSession, error) {
			return session, nil
		},
		atomicFinishFn: func(id uint) (int64, error) {
			return 1, nil // we won the race
		},
		getAllAnswersFn: func(sessionID uint) ([]entity.ExamAnswer, error) {
			return []entity.ExamAnswer{
				{QuestionID: 1, Answer: toJSONHelper(map[string]int{"option_index": 1})}, // correct
				{QuestionID: 2, Answer: toJSONHelper(map[string]int{"option_index": 0})}, // correct
			}, nil
		},
		updateFn: func(s *entity.ExamSession) error {
			updatedSession = s
			return nil
		},
	}

	uc := setupTestUseCase(t, repo)
	uc.questionRepo = &mockQuestionRepoWithData{
		questions: []*entity.Question{q1, q2},
	}
	uc.scheduleRepo = &mockScheduleRepo{
		findByIDFn: func(id uint) (*entity.ExamSchedule, error) {
			return &entity.ExamSchedule{ID: 1}, nil
		},
	}

	result, err := uc.FinishExam(1, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != entity.SessionStatusFinished {
		t.Errorf("expected status '%s', got '%s'", entity.SessionStatusFinished, result.Status)
	}
	if result.FinishedAt == nil {
		t.Error("expected FinishedAt to be set")
	}
	if updatedSession == nil {
		t.Fatal("expected Update to be called")
	}
	// Both answers are correct: 2.0 + 3.0 = 5.0
	if updatedSession.Score != 5.0 {
		t.Errorf("expected score 5.0, got %v", updatedSession.Score)
	}
	if updatedSession.MaxScore != 5.0 {
		t.Errorf("expected max score 5.0, got %v", updatedSession.MaxScore)
	}
}

func TestFinishExam_AlreadyFinished(t *testing.T) {
	session := makeOngoingSession(1, 42)
	session.Status = entity.SessionStatusFinished
	now := time.Now()
	session.FinishedAt = &now
	session.Score = 10.0
	session.MaxScore = 10.0

	repo := &mockSessionRepo{
		findByIDFn: func(id uint) (*entity.ExamSession, error) {
			return session, nil
		},
	}
	uc := setupTestUseCase(t, repo)

	result, err := uc.FinishExam(1, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return the existing session without re-scoring
	if result.Status != entity.SessionStatusFinished {
		t.Errorf("expected status '%s', got '%s'", entity.SessionStatusFinished, result.Status)
	}
	if result.Score != 10.0 {
		t.Errorf("expected score 10.0 (unchanged), got %v", result.Score)
	}
}

func TestFinishExam_SessionNotFound(t *testing.T) {
	repo := &mockSessionRepo{
		findByIDFn: func(id uint) (*entity.ExamSession, error) {
			return nil, errors.New("record not found")
		},
	}
	uc := setupTestUseCase(t, repo)

	_, err := uc.FinishExam(999, 42)
	if err == nil {
		t.Fatal("expected error for non-existent session")
	}
	if err.Error() != "sesi tidak valid" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestFinishExam_WrongUser(t *testing.T) {
	session := makeOngoingSession(1, 42)
	repo := &mockSessionRepo{
		findByIDFn: func(id uint) (*entity.ExamSession, error) {
			return session, nil
		},
	}
	uc := setupTestUseCase(t, repo)

	// UserID 99 != session.UserID 42
	_, err := uc.FinishExam(1, 99)
	if err == nil {
		t.Fatal("expected error for wrong user")
	}
	if err.Error() != "sesi tidak valid" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestFinishExam_AtomicFinishRaceLost(t *testing.T) {
	// Simulate: another goroutine already finished this session
	session := makeOngoingSession(1, 42)

	finishedSession := makeOngoingSession(1, 42)
	finishedSession.Status = entity.SessionStatusFinished
	now := time.Now()
	finishedSession.FinishedAt = &now
	finishedSession.Score = 8.0
	finishedSession.MaxScore = 10.0

	callCount := 0
	repo := &mockSessionRepo{
		findByIDFn: func(id uint) (*entity.ExamSession, error) {
			callCount++
			if callCount == 1 {
				return session, nil // first call: ongoing
			}
			return finishedSession, nil // second call after race lost: already finished
		},
		atomicFinishFn: func(id uint) (int64, error) {
			return 0, nil // 0 rows affected = another goroutine won
		},
	}
	uc := setupTestUseCase(t, repo)

	result, err := uc.FinishExam(1, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return the already-finished session
	if result.Status != entity.SessionStatusFinished {
		t.Errorf("expected status '%s', got '%s'", entity.SessionStatusFinished, result.Status)
	}
	if result.Score != 8.0 {
		t.Errorf("expected score 8.0 from race winner, got %v", result.Score)
	}
}

func TestFinishExam_NoQuestions(t *testing.T) {
	session := makeOngoingSession(1, 42)
	session.QuestionOrder = types.JSON(`[100,200]`) // IDs that won't be found

	var updatedSession *entity.ExamSession
	repo := &mockSessionRepo{
		findByIDFn: func(id uint) (*entity.ExamSession, error) {
			return session, nil
		},
		atomicFinishFn: func(id uint) (int64, error) {
			return 1, nil
		},
		getAllAnswersFn: func(sessionID uint) ([]entity.ExamAnswer, error) {
			return nil, nil
		},
		updateFn: func(s *entity.ExamSession) error {
			updatedSession = s
			return nil
		},
	}

	uc := setupTestUseCase(t, repo)
	// questionRepo returns no questions (default mock returns nil)
	uc.scheduleRepo = &mockScheduleRepo{
		findByIDFn: func(id uint) (*entity.ExamSchedule, error) {
			return &entity.ExamSchedule{ID: 1}, nil
		},
	}

	result, err := uc.FinishExam(1, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedSession == nil {
		t.Fatal("expected Update to be called")
	}
	if result.Score != 0 {
		t.Errorf("expected score 0 for no questions, got %v", result.Score)
	}
	if result.MaxScore != 0 {
		t.Errorf("expected max score 0, got %v", result.MaxScore)
	}
}

// ─── Helper: mockQuestionRepo with data ────────────────────────

type mockQuestionRepoWithData struct {
	mockQuestionRepo // embed the base mock for stubs
	questions        []*entity.Question
}

func (m *mockQuestionRepoWithData) FindByIDs(ids []uint) ([]*entity.Question, error) {
	if m.questions == nil {
		return nil, nil
	}
	idSet := make(map[uint]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var result []*entity.Question
	for _, q := range m.questions {
		if idSet[q.ID] {
			result = append(result, q)
		}
	}
	return result, nil
}

// toJSONHelper marshals a value to types.JSON for test data.
func toJSONHelper(v interface{}) types.JSON {
	b, _ := json.Marshal(v)
	return types.JSON(b)
}
