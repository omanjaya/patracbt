package exam

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/infrastructure/cache"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/types"
)

func (uc *ExamSessionUseCase) SaveAnswer(sessionID, userID uint, req dto.SaveAnswerRequest) (session *entity.ExamSession, answered int, total int, err error) {
	// 1. Try Redis cache first (0 DB queries on cache hit)
	cached, cacheErr := uc.examCache.GetSession(sessionID)
	if cacheErr != nil {
		// Cache miss — fall back to DB
		session, err = uc.sessionRepo.FindByIDBasic(sessionID)
		if err != nil {
			return nil, 0, 0, errors.New("sesi tidak valid")
		}
		// Re-cache for next time
		uc.examCache.CacheSession(session, 4*time.Hour)
	} else {
		// Build minimal session from cache
		session = cached.ToEntity()
	}

	if session.UserID != userID {
		return nil, 0, 0, errors.New("sesi tidak valid")
	}
	if session.Status != entity.SessionStatusOngoing {
		return nil, 0, 0, errors.New("sesi ujian tidak aktif")
	}
	// Check time
	if session.EndTime != nil && time.Now().After(*session.EndTime) {
		uc.examCache.InvalidateSession(sessionID)
		// Re-check session status from DB to avoid racing with concurrent forceFinish calls
		dbSession, dbErr := uc.sessionRepo.FindByIDBasic(sessionID)
		if dbErr == nil && dbSession.Status == entity.SessionStatusFinished {
			return nil, 0, 0, errors.New("waktu ujian telah habis")
		}
		if finishErr := uc.forceFinish(session); finishErr != nil {
			logger.Log.Errorf("SaveAnswer: gagal force finish sesi #%d: %v", session.ID, finishErr)
		}
		return nil, 0, 0, errors.New("waktu ujian telah habis")
	}

	// Validate that the questionID exists in the session's QuestionOrder
	var questionOrder []uint
	if unmarshalErr := json.Unmarshal(session.QuestionOrder, &questionOrder); unmarshalErr != nil {
		return nil, 0, 0, fmt.Errorf("invalid question order for session %d: %v", sessionID, unmarshalErr)
	}
	questionAllowed := false
	for _, qID := range questionOrder {
		if qID == req.QuestionID {
			questionAllowed = true
			break
		}
	}
	if !questionAllowed {
		return nil, 0, 0, errors.New("soal tidak termasuk dalam sesi ujian ini")
	}

	// 2. Save to Redis buffer (NOT PostgreSQL) — ~0.1ms
	if err = uc.examCache.SaveAnswer(sessionID, req.QuestionID, req.Answer, req.IsFlagged); err != nil {
		// Redis down: fall back to direct DB write
		logger.Log.Warnf("SaveAnswer: Redis buffer failed, falling back to DB: %v", err)
		answer := &entity.ExamAnswer{
			ExamSessionID: sessionID,
			QuestionID:    req.QuestionID,
			Answer:        types.JSON(req.Answer),
			IsFlagged:     req.IsFlagged,
		}
		if err = uc.sessionRepo.UpsertAnswer(answer); err != nil {
			return nil, 0, 0, err
		}
		// Fall back to DB count as well
		answered, _ = uc.sessionRepo.CountNonEmptyAnswers(sessionID)
		var order []uint
		if unmarshalErr := json.Unmarshal(session.QuestionOrder, &order); unmarshalErr != nil {
			return nil, 0, 0, fmt.Errorf("invalid question order for session %d: %v", sessionID, unmarshalErr)
		}
		total = len(order)
		return session, answered, total, nil
	}

	// 3. Get count from Redis HLEN (NOT PostgreSQL COUNT) — ~0.1ms
	count, _ := uc.examCache.GetAnswerCount(sessionID)
	answered = int(count)
	var order []uint
	if unmarshalErr := json.Unmarshal(session.QuestionOrder, &order); unmarshalErr != nil {
		return nil, 0, 0, fmt.Errorf("invalid question order for session %d: %v", sessionID, unmarshalErr)
	}
	total = len(order)

	return session, answered, total, nil
}

// BatchSaveAnswers — upsert multiple answers at once (offline sync / beacon)
// BUG-13 fix: cek expired time seperti SaveAnswer
func (uc *ExamSessionUseCase) BatchSaveAnswers(sessionID, userID uint, answers []dto.SaveAnswerRequest) error {
	// 1. Try Redis cache first
	cached, cacheErr := uc.examCache.GetSession(sessionID)
	var session *entity.ExamSession
	if cacheErr != nil {
		// Cache miss — fall back to DB (use FindByIDBasic, not FindByID with preloads)
		var err error
		session, err = uc.sessionRepo.FindByIDBasic(sessionID)
		if err != nil {
			return errors.New("sesi tidak valid")
		}
		uc.examCache.CacheSession(session, 4*time.Hour)
	} else {
		session = cached.ToEntity()
	}

	if session.UserID != userID {
		return errors.New("sesi tidak valid")
	}
	if session.Status != entity.SessionStatusOngoing {
		return errors.New("sesi ujian tidak aktif")
	}
	// BUG-13 fix: cek waktu kadaluwarsa
	if session.EndTime != nil && time.Now().After(*session.EndTime) {
		uc.examCache.InvalidateSession(sessionID)
		// Re-check session status from DB to avoid racing with concurrent forceFinish calls
		dbSession, dbErr := uc.sessionRepo.FindByIDBasic(sessionID)
		if dbErr == nil && dbSession.Status == entity.SessionStatusFinished {
			return errors.New("waktu ujian telah habis")
		}
		if finishErr := uc.forceFinish(session); finishErr != nil {
			logger.Log.Errorf("BatchSaveAnswers: gagal force finish sesi #%d: %v", session.ID, finishErr)
		}
		return errors.New("waktu ujian telah habis")
	}

	// 2. Save all answers to Redis buffer
	now := time.Now()
	cachedAnswers := make([]cache.CachedAnswer, 0, len(answers))
	for _, req := range answers {
		cachedAnswers = append(cachedAnswers, cache.CachedAnswer{
			QuestionID: req.QuestionID,
			Answer:     req.Answer,
			IsFlagged:  req.IsFlagged,
			AnsweredAt: now,
		})
	}

	if err := uc.examCache.SaveAnswerBatch(sessionID, cachedAnswers); err != nil {
		// Redis down: fall back to direct DB write
		logger.Log.Warnf("BatchSaveAnswers: Redis buffer failed, falling back to DB: %v", err)
		batch := make([]*entity.ExamAnswer, 0, len(answers))
		for _, req := range answers {
			batch = append(batch, &entity.ExamAnswer{
				ExamSessionID: sessionID,
				QuestionID:    req.QuestionID,
				Answer:        types.JSON(req.Answer),
				IsFlagged:     req.IsFlagged,
			})
		}
		return uc.sessionRepo.BatchUpsertAnswers(batch)
	}

	return nil
}

// ToggleFlag toggles the is_flagged field on an answer record.
// If the answer doesn't exist yet, it creates one with empty answer.
func (uc *ExamSessionUseCase) ToggleFlag(sessionID, userID uint, req dto.ToggleFlagRequest) error {
	// 1. Validate session ownership and status
	cached, cacheErr := uc.examCache.GetSession(sessionID)
	var session *entity.ExamSession
	if cacheErr != nil {
		var err error
		session, err = uc.sessionRepo.FindByIDBasic(sessionID)
		if err != nil {
			return errors.New("sesi tidak valid")
		}
		uc.examCache.CacheSession(session, 4*time.Hour)
	} else {
		session = cached.ToEntity()
	}

	if session.UserID != userID {
		return errors.New("sesi tidak valid")
	}
	if session.Status != entity.SessionStatusOngoing {
		return errors.New("sesi ujian tidak aktif")
	}

	// 2. Save flag to Redis buffer (preserve existing answer data)
	// First try to get existing answer from Redis cache
	existingAnswer, err := uc.examCache.GetBufferedAnswer(sessionID, req.QuestionID)
	if err == nil && existingAnswer != nil {
		// Update flag on existing cached answer
		return uc.examCache.SaveAnswer(sessionID, req.QuestionID, existingAnswer.Answer, req.IsFlagged)
	}

	// No cached answer — try DB
	dbAnswer, dbErr := uc.sessionRepo.GetAnswer(sessionID, req.QuestionID)
	if dbErr != nil || dbAnswer == nil {
		// No answer exists yet — create with empty answer and flag
		return uc.examCache.SaveAnswer(sessionID, req.QuestionID, nil, req.IsFlagged)
	}

	// Update flag on existing DB answer via cache
	return uc.examCache.SaveAnswer(sessionID, req.QuestionID, json.RawMessage(dbAnswer.Answer), req.IsFlagged)
}

// CheckLockStatus returns the current status of a session.
// Used by peserta to poll whether their session has been locked/unlocked by pengawas.
func (uc *ExamSessionUseCase) CheckLockStatus(sessionID, userID uint) (string, error) {
	session, err := uc.sessionRepo.FindByIDBasic(sessionID)
	if err != nil {
		return "", errors.New("sesi tidak ditemukan")
	}
	if session.UserID != userID {
		return "", errors.New("akses ditolak")
	}
	return session.Status, nil
}

// BeaconSync saves answers sent via navigator.sendBeacon() when tab is closing.
// Lightweight — minimal validation, just save answers without heavy processing.
func (uc *ExamSessionUseCase) BeaconSync(sessionID, userID uint, answers []dto.SaveAnswerRequest) (int, error) {
	// 1. Quick session validation
	cached, cacheErr := uc.examCache.GetSession(sessionID)
	var session *entity.ExamSession
	if cacheErr != nil {
		var err error
		session, err = uc.sessionRepo.FindByIDBasic(sessionID)
		if err != nil {
			return 0, errors.New("sesi tidak valid")
		}
		uc.examCache.CacheSession(session, 4*time.Hour)
	} else {
		session = cached.ToEntity()
	}

	if session.UserID != userID {
		return 0, errors.New("sesi tidak valid")
	}
	// Don't process if session is already finished/terminated
	if session.Status != entity.SessionStatusOngoing {
		return 0, nil
	}

	// 2. Save all answers to Redis buffer (lightweight, no progress broadcast)
	now := time.Now()
	cachedAnswers := make([]cache.CachedAnswer, 0, len(answers))
	for _, req := range answers {
		if req.QuestionID == 0 {
			continue
		}
		cachedAnswers = append(cachedAnswers, cache.CachedAnswer{
			QuestionID: req.QuestionID,
			Answer:     req.Answer,
			IsFlagged:  req.IsFlagged,
			AnsweredAt: now,
		})
	}

	if len(cachedAnswers) == 0 {
		return 0, nil
	}

	if err := uc.examCache.SaveAnswerBatch(sessionID, cachedAnswers); err != nil {
		// Redis down: fall back to direct DB write
		logger.Log.Warnf("BeaconSync: Redis buffer failed, falling back to DB: %v", err)
		batch := make([]*entity.ExamAnswer, 0, len(answers))
		for _, req := range answers {
			if req.QuestionID == 0 {
				continue
			}
			batch = append(batch, &entity.ExamAnswer{
				ExamSessionID: sessionID,
				QuestionID:    req.QuestionID,
				Answer:        types.JSON(req.Answer),
				IsFlagged:     req.IsFlagged,
			})
		}
		if err := uc.sessionRepo.BatchUpsertAnswers(batch); err != nil {
			return 0, err
		}
	}

	return len(cachedAnswers), nil
}
