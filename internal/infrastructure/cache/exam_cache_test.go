package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/types"
)

func init() {
	// Initialize logger so calls in exam_cache.go don't panic.
	logger.Init("test")
}

func setupTestCache(t *testing.T) (*ExamCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewExamCache(rdb), mr
}

func makeTestSession(id, userID uint) *entity.ExamSession {
	now := time.Now().Add(2 * time.Hour)
	return &entity.ExamSession{
		ID:             id,
		UserID:         userID,
		Status:         entity.SessionStatusOngoing,
		EndTime:        &now,
		ExamScheduleID: 100,
		QuestionOrder:  types.JSON(`[1,2,3]`),
	}
}

// ─── Session Cache ─────────────────────────────────────────────

func TestCacheSession(t *testing.T) {
	ec, _ := setupTestCache(t)
	session := makeTestSession(1, 42)

	if err := ec.CacheSession(session, 10*time.Minute); err != nil {
		t.Fatalf("CacheSession: %v", err)
	}

	got, err := ec.GetSession(1)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.ID != 1 {
		t.Errorf("ID: got %d, want 1", got.ID)
	}
	if got.UserID != 42 {
		t.Errorf("UserID: got %d, want 42", got.UserID)
	}
	if got.Status != entity.SessionStatusOngoing {
		t.Errorf("Status: got %q, want %q", got.Status, entity.SessionStatusOngoing)
	}
	if got.ExamScheduleID != 100 {
		t.Errorf("ExamScheduleID: got %d, want 100", got.ExamScheduleID)
	}
}

func TestCacheSessionExpiry(t *testing.T) {
	ec, mr := setupTestCache(t)
	session := makeTestSession(2, 42)

	if err := ec.CacheSession(session, 1*time.Second); err != nil {
		t.Fatalf("CacheSession: %v", err)
	}

	// Should exist now
	if _, err := ec.GetSession(2); err != nil {
		t.Fatalf("GetSession before expiry: %v", err)
	}

	// Fast-forward time in miniredis
	mr.FastForward(2 * time.Second)

	// Should be gone
	_, err := ec.GetSession(2)
	if err == nil {
		t.Error("expected error after TTL expiry, got nil")
	}
}

func TestInvalidateSession(t *testing.T) {
	ec, _ := setupTestCache(t)
	session := makeTestSession(3, 42)

	if err := ec.CacheSession(session, 10*time.Minute); err != nil {
		t.Fatalf("CacheSession: %v", err)
	}

	ec.InvalidateSession(3)

	_, err := ec.GetSession(3)
	if err == nil {
		t.Error("expected error after invalidation, got nil")
	}
}

// ─── Answer Buffer ─────────────────────────────────────────────

func TestSaveAnswer(t *testing.T) {
	ec, _ := setupTestCache(t)
	sessionID := uint(10)
	answer := json.RawMessage(`{"option_index": 2}`)

	if err := ec.SaveAnswer(sessionID, 100, answer, false); err != nil {
		t.Fatalf("SaveAnswer: %v", err)
	}

	answers, err := ec.GetAllBufferedAnswers(sessionID)
	if err != nil {
		t.Fatalf("GetAllBufferedAnswers: %v", err)
	}
	if len(answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(answers))
	}
	if answers[0].QuestionID != 100 {
		t.Errorf("QuestionID: got %d, want 100", answers[0].QuestionID)
	}
	if string(answers[0].Answer) != `{"option_index":2}` {
		t.Errorf("Answer: got %s", string(answers[0].Answer))
	}
}

func TestGetAnswerCount(t *testing.T) {
	ec, _ := setupTestCache(t)
	sessionID := uint(20)

	// Save 3 answers for different questions
	for _, qID := range []uint{1, 2, 3} {
		ans := json.RawMessage(`{"option_index": 0}`)
		if err := ec.SaveAnswer(sessionID, qID, ans, false); err != nil {
			t.Fatalf("SaveAnswer q%d: %v", qID, err)
		}
	}

	count, err := ec.GetAnswerCount(sessionID)
	if err != nil {
		t.Fatalf("GetAnswerCount: %v", err)
	}
	if count != 3 {
		t.Errorf("count: got %d, want 3", count)
	}

	// Overwrite question 1 — count should stay 3
	if err := ec.SaveAnswer(sessionID, 1, json.RawMessage(`{"option_index": 1}`), false); err != nil {
		t.Fatalf("SaveAnswer overwrite: %v", err)
	}
	count, _ = ec.GetAnswerCount(sessionID)
	if count != 3 {
		t.Errorf("count after overwrite: got %d, want 3", count)
	}
}

func TestClearAnswerBuffer(t *testing.T) {
	ec, _ := setupTestCache(t)
	sessionID := uint(30)

	if err := ec.SaveAnswer(sessionID, 1, json.RawMessage(`{}`), false); err != nil {
		t.Fatalf("SaveAnswer: %v", err)
	}

	ec.ClearAnswerBuffer(sessionID)

	count, _ := ec.GetAnswerCount(sessionID)
	if count != 0 {
		t.Errorf("count after clear: got %d, want 0", count)
	}

	answers, _ := ec.GetAllBufferedAnswers(sessionID)
	if len(answers) != 0 {
		t.Errorf("answers after clear: got %d, want 0", len(answers))
	}
}

func TestGetDirtySessionIDs(t *testing.T) {
	ec, _ := setupTestCache(t)

	// Create buffered answers for sessions 100, 200, 300
	for _, sid := range []uint{100, 200, 300} {
		if err := ec.SaveAnswer(sid, 1, json.RawMessage(`{}`), false); err != nil {
			t.Fatalf("SaveAnswer session %d: %v", sid, err)
		}
	}

	ids, err := ec.GetDirtySessionIDs(context.Background())
	if err != nil {
		t.Fatalf("GetDirtySessionIDs: %v", err)
	}

	if len(ids) != 3 {
		t.Fatalf("expected 3 dirty sessions, got %d", len(ids))
	}

	// Check all IDs are present
	idSet := make(map[uint]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	for _, want := range []uint{100, 200, 300} {
		if !idSet[want] {
			t.Errorf("missing dirty session ID %d", want)
		}
	}
}
