package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/types"
)

// ─── Benchmark Helpers ──────────────────────────────────────────

func setupBenchCache(b *testing.B) (*ExamCache, *miniredis.Miniredis) {
	b.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("miniredis.Run: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewExamCache(rdb), mr
}

func benchSession(id, userID uint) *entity.ExamSession {
	now := time.Now().Add(2 * time.Hour)
	return &entity.ExamSession{
		ID:             id,
		UserID:         userID,
		Status:         entity.SessionStatusOngoing,
		EndTime:        &now,
		ExamScheduleID: 100,
		QuestionOrder:  types.JSON(`[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50]`),
	}
}

// ─── BenchmarkSaveAnswer ────────────────────────────────────────

func BenchmarkSaveAnswer(b *testing.B) {
	cache, mr := setupBenchCache(b)
	defer mr.Close()

	answer := json.RawMessage(`{"option_index": 1}`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.SaveAnswer(1, uint(i%50), answer, false)
	}
}

func BenchmarkSaveAnswer_Parallel(b *testing.B) {
	cache, mr := setupBenchCache(b)
	defer mr.Close()

	answer := json.RawMessage(`{"option_index": 2}`)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Spread across 100 sessions, 50 questions each
			sessionID := uint(i%100) + 1
			questionID := uint(i%50) + 1
			cache.SaveAnswer(sessionID, questionID, answer, i%10 == 0)
			i++
		}
	})
}

// ─── BenchmarkSaveAnswerBatch ───────────────────────────────────

func BenchmarkSaveAnswerBatch_5(b *testing.B) {
	benchSaveAnswerBatch(b, 5)
}

func BenchmarkSaveAnswerBatch_10(b *testing.B) {
	benchSaveAnswerBatch(b, 10)
}

func BenchmarkSaveAnswerBatch_25(b *testing.B) {
	benchSaveAnswerBatch(b, 25)
}

func benchSaveAnswerBatch(b *testing.B, batchSize int) {
	cache, mr := setupBenchCache(b)
	defer mr.Close()

	answers := make([]CachedAnswer, batchSize)
	for i := range answers {
		answers[i] = CachedAnswer{
			QuestionID: uint(i + 1),
			Answer:     json.RawMessage(fmt.Sprintf(`{"option_index": %d}`, i%4)),
			IsFlagged:  i%10 == 0,
			AnsweredAt: time.Now(),
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.SaveAnswerBatch(uint(i%100)+1, answers)
	}
}

// ─── BenchmarkGetAnswerCount ────────────────────────────────────

func BenchmarkGetAnswerCount(b *testing.B) {
	cache, mr := setupBenchCache(b)
	defer mr.Close()

	// Pre-populate 50 answers
	sessionID := uint(1)
	for q := 0; q < 50; q++ {
		cache.SaveAnswer(sessionID, uint(q+1), json.RawMessage(`{"option_index":0}`), false)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetAnswerCount(sessionID)
	}
}

func BenchmarkGetAnswerCount_Parallel(b *testing.B) {
	cache, mr := setupBenchCache(b)
	defer mr.Close()

	// Pre-populate answers for multiple sessions
	for s := uint(1); s <= 100; s++ {
		for q := uint(1); q <= 50; q++ {
			cache.SaveAnswer(s, q, json.RawMessage(`{"option_index":0}`), false)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.GetAnswerCount(uint(i%100) + 1)
			i++
		}
	})
}

// ─── BenchmarkGetSession ────────────────────────────────────────

func BenchmarkGetSession(b *testing.B) {
	cache, mr := setupBenchCache(b)
	defer mr.Close()

	session := benchSession(1, 42)
	cache.CacheSession(session, 30*time.Minute)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetSession(1)
	}
}

func BenchmarkGetSession_Parallel(b *testing.B) {
	cache, mr := setupBenchCache(b)
	defer mr.Close()

	// Pre-populate 100 sessions
	for s := uint(1); s <= 100; s++ {
		session := benchSession(s, s+1000)
		cache.CacheSession(session, 30*time.Minute)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.GetSession(uint(i%100) + 1)
			i++
		}
	})
}

// ─── BenchmarkCacheSession ──────────────────────────────────────

func BenchmarkCacheSession(b *testing.B) {
	cache, mr := setupBenchCache(b)
	defer mr.Close()

	session := benchSession(1, 42)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.ID = uint(i%100) + 1
		cache.CacheSession(session, 30*time.Minute)
	}
}

func BenchmarkCacheSession_Parallel(b *testing.B) {
	cache, mr := setupBenchCache(b)
	defer mr.Close()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			session := benchSession(uint(i%100)+1, uint(i)+1000)
			cache.CacheSession(session, 30*time.Minute)
			i++
		}
	})
}

// ─── BenchmarkGetAllBufferedAnswers ─────────────────────────────

func BenchmarkGetAllBufferedAnswers(b *testing.B) {
	cache, mr := setupBenchCache(b)
	defer mr.Close()

	sessionID := uint(1)
	for q := uint(1); q <= 50; q++ {
		cache.SaveAnswer(sessionID, q, json.RawMessage(fmt.Sprintf(`{"option_index":%d}`, q%4)), false)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetAllBufferedAnswers(sessionID)
	}
}

// ─── BenchmarkGetDirtySessionIDs ────────────────────────────────

func BenchmarkGetDirtySessionIDs(b *testing.B) {
	cache, mr := setupBenchCache(b)
	defer mr.Close()

	// Simulate 100 active sessions with buffered answers
	for s := uint(1); s <= 100; s++ {
		cache.SaveAnswer(s, 1, json.RawMessage(`{}`), false)
	}

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetDirtySessionIDs(ctx)
	}
}

// ─── BenchmarkFullAnswerFlow ────────────────────────────────────
// Simulates the complete answer submission: save answer + get count + get session

func BenchmarkFullAnswerFlow(b *testing.B) {
	cache, mr := setupBenchCache(b)
	defer mr.Close()

	session := benchSession(1, 42)
	cache.CacheSession(session, 30*time.Minute)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qID := uint(i%50) + 1
		answer := json.RawMessage(fmt.Sprintf(`{"option_index":%d}`, i%4))

		// Save the answer
		cache.SaveAnswer(1, qID, answer, false)
		// Check progress
		cache.GetAnswerCount(1)
		// Validate session
		cache.GetSession(1)
	}
}

func BenchmarkFullAnswerFlow_Parallel(b *testing.B) {
	cache, mr := setupBenchCache(b)
	defer mr.Close()

	// Pre-populate 100 sessions
	for s := uint(1); s <= 100; s++ {
		session := benchSession(s, s+1000)
		cache.CacheSession(session, 30*time.Minute)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			sessionID := uint(i%100) + 1
			qID := uint(i%50) + 1
			answer := json.RawMessage(fmt.Sprintf(`{"option_index":%d}`, i%4))

			cache.SaveAnswer(sessionID, qID, answer, false)
			cache.GetAnswerCount(sessionID)
			cache.GetSession(sessionID)
			i++
		}
	})
}
