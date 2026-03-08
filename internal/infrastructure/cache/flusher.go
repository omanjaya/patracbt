package cache

import (
	"context"
	"sync"
	"time"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/types"
)

// AnswerFlusher periodically syncs buffered Redis answers to PostgreSQL.
type AnswerFlusher struct {
	cache       *ExamCache
	sessionRepo repository.ExamSessionRepository
	interval    time.Duration
	stopCh      chan struct{}
	stopOnce    sync.Once
	done        chan struct{} // signals that the flush loop has exited
	flushingMu  sync.Mutex   // prevents concurrent flushes of the same session
}

// NewAnswerFlusher creates a new background flusher.
func NewAnswerFlusher(cache *ExamCache, sessionRepo repository.ExamSessionRepository, interval time.Duration) *AnswerFlusher {
	return &AnswerFlusher{
		cache:       cache,
		sessionRepo: sessionRepo,
		interval:    interval,
		stopCh:      make(chan struct{}),
		done:        make(chan struct{}),
	}
}

// Start begins the background flush loop.
func (f *AnswerFlusher) Start() {
	go func() {
		defer close(f.done)
		ticker := time.NewTicker(f.interval)
		defer ticker.Stop()
		logger.Log.Infof("AnswerFlusher started (interval: %s)", f.interval)
		for {
			select {
			case <-ticker.C:
				f.flushAll()
			case <-f.stopCh:
				logger.Log.Info("AnswerFlusher stopping, final flush...")
				f.flushAll()
				return
			}
		}
	}()
}

// Stop signals the flusher to perform a final flush and exit.
// Safe to call multiple times.
func (f *AnswerFlusher) Stop() {
	f.stopOnce.Do(func() {
		close(f.stopCh)
	})
	// Wait for the flush loop to complete its final flush.
	<-f.done
}

// FlushSession flushes all buffered answers for a specific session to PostgreSQL.
// This MUST be called before scoring (e.g., on FinishExam).
func (f *AnswerFlusher) FlushSession(sessionID uint) error {
	f.flushingMu.Lock()
	defer f.flushingMu.Unlock()
	return f.flushSessionLocked(sessionID)
}

// flushSessionLocked performs the actual flush. Caller must hold flushingMu.
func (f *AnswerFlusher) flushSessionLocked(sessionID uint) error {
	answers, err := f.cache.GetAllBufferedAnswers(sessionID)
	if err != nil {
		logger.Log.Errorf("FlushSession %d: GetAllBufferedAnswers failed: %v", sessionID, err)
		return err
	}
	if len(answers) == 0 {
		return nil
	}

	batch := make([]*entity.ExamAnswer, 0, len(answers))
	for _, a := range answers {
		batch = append(batch, &entity.ExamAnswer{
			ExamSessionID: sessionID,
			QuestionID:    a.QuestionID,
			Answer:        types.JSON(a.Answer),
			IsFlagged:     a.IsFlagged,
			AnsweredAt:    &a.AnsweredAt,
		})
	}

	if err := f.sessionRepo.BatchUpsertAnswers(batch); err != nil {
		logger.Log.Errorf("FlushSession %d: BatchUpsertAnswers failed: %v", sessionID, err)
		return err
	}

	f.cache.ClearAnswerBuffer(sessionID)
	logger.Log.Debugf("FlushSession %d: flushed %d answers", sessionID, len(answers))
	return nil
}

// flushAll iterates all dirty sessions and flushes each one with retry.
// The flushingMu is acquired only to fetch the list of dirty session IDs,
// then released so that individual FlushSession calls from other goroutines
// are not blocked during the (potentially long) iteration.
func (f *AnswerFlusher) flushAll() {
	f.flushingMu.Lock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ids, err := f.cache.GetDirtySessionIDs(ctx)
	f.flushingMu.Unlock() // release early — don't hold during iteration

	if err != nil {
		logger.Log.Errorf("flushAll: GetDirtySessionIDs failed: %v", err)
		return
	}
	for _, id := range ids {
		if err := f.flushWithRetry(id, 3); err != nil {
			logger.Log.Errorf("flushAll: session %d failed after all retries: %v", id, err)
		}
	}
}

// flushWithRetry attempts to flush a session with exponential backoff.
// It acquires flushingMu per attempt so that the lock is not held across retries.
// Delays: 1s, 2s, 4s (doubling each attempt).
func (f *AnswerFlusher) flushWithRetry(sessionID uint, maxRetries int) error {
	var lastErr error
	delay := 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			logger.Log.Warnf("flushWithRetry: session %d retry %d/%d after %s", sessionID, attempt, maxRetries, delay)
			time.Sleep(delay)
			delay *= 2
		}
		lastErr = f.FlushSession(sessionID)
		if lastErr == nil {
			return nil
		}
	}

	logger.Log.Errorf("flushWithRetry: session %d failed after %d retries: %v", sessionID, maxRetries, lastErr)
	return lastErr
}
