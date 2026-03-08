package exam

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/logger"
)

func (uc *ExamSessionUseCase) FinishExam(sessionID, userID uint) (*entity.ExamSession, error) {
	session, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil || session.UserID != userID {
		return nil, errors.New("sesi tidak valid")
	}
	if session.Status == entity.SessionStatusFinished {
		return session, nil
	}

	// Atomic status transition: only proceed if status is still "ongoing"
	// This prevents concurrent finish calls from double-scoring
	rowsAffected, err := uc.sessionRepo.AtomicFinish(sessionID)
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		// Another goroutine already finished this session — return the current state
		refreshed, err := uc.sessionRepo.FindByID(sessionID)
		if err != nil {
			return nil, err
		}
		return refreshed, nil
	}

	// We won the race — proceed with scoring
	session.Status = entity.SessionStatusFinished
	return uc.finishAndScore(session)
}

func (uc *ExamSessionUseCase) finishAndScore(session *entity.ExamSession) (*entity.ExamSession, error) {
	// Flush Redis answer buffer to PostgreSQL before scoring
	if err := uc.flusher.FlushSession(session.ID); err != nil {
		logger.Log.Errorf("finishAndScore: flush session #%d failed: %v", session.ID, err)
		// Continue with scoring — answers already in DB from periodic flush are still valid
	}
	uc.examCache.InvalidateSession(session.ID)

	// Load questions
	var order []uint
	if err := json.Unmarshal(session.QuestionOrder, &order); err != nil {
		return nil, err
	}

	answers, err := uc.sessionRepo.GetAllAnswers(session.ID)
	if err != nil {
		logger.Log.Errorf("finishAndScore: failed to get answers for session %d: %v", session.ID, err)
		return nil, fmt.Errorf("gagal mengambil jawaban: %w", err)
	}
	answerMap := make(map[uint]entity.ExamAnswer)
	for _, a := range answers {
		answerMap[a.QuestionID] = a
	}

	// Batch-load all questions in a single query to avoid N+1
	fetched, err := uc.questionRepo.FindByIDs(order)
	if err != nil {
		return nil, err
	}
	qMap := make(map[uint]*entity.Question, len(fetched))
	for _, q := range fetched {
		qMap[q.ID] = q
	}

	// Load schedule to check for weighted question banks
	schedule, schedErr := uc.scheduleRepo.FindByID(session.ExamScheduleID)

	// Check if we have multiple banks with different weights
	hasWeights := false
	if schedErr == nil && len(schedule.QuestionBanks) > 1 {
		for _, b := range schedule.QuestionBanks {
			if b.Weight != 1 {
				hasWeights = true
				break
			}
		}
	}

	var score, maxScore float64

	if len(qMap) == 0 {
		// No questions found — set score to 0 and finish
		now := time.Now()
		session.Status = entity.SessionStatusFinished
		session.FinishedAt = &now
		session.Score = 0
		session.MaxScore = 0
		if err := uc.sessionRepo.Update(session); err != nil {
			return nil, err
		}
		return session, nil
	}

	if hasWeights && schedErr == nil {
		// Weighted scoring: calculate per-bank score, then apply weights
		// Build bank weight map
		bankWeight := make(map[uint]float64)
		for _, b := range schedule.QuestionBanks {
			w := b.Weight
			if w <= 0 {
				w = 1
			}
			bankWeight[b.QuestionBankID] = w
		}

		// Per-bank scoring
		type bankScore struct {
			earned float64
			max    float64
			weight float64
		}
		bankScores := make(map[uint]*bankScore)

		for _, qID := range order {
			q, ok := qMap[qID]
			if !ok {
				continue
			}
			bs, exists := bankScores[q.QuestionBankID]
			if !exists {
				w := bankWeight[q.QuestionBankID]
				if w <= 0 {
					w = 1
				}
				bs = &bankScore{weight: w}
				bankScores[q.QuestionBankID] = bs
			}
			bs.max += q.Score
			if a, ok := answerMap[qID]; ok {
				bs.earned += uc.calculator.Calculate(q, a.Answer)
			}
		}

		// Final score = sum(bank_percentage * weight) / sum(weight) * 100
		var weightedSum, totalWeight float64
		for _, bs := range bankScores {
			if bs.max > 0 {
				bankPct := bs.earned / bs.max
				weightedSum += bankPct * bs.weight
			}
			totalWeight += bs.weight
		}

		// Calculate total max score (unweighted) for reference
		for _, qID := range order {
			if q, ok := qMap[qID]; ok {
				maxScore += q.Score
			}
		}

		if totalWeight > 0 {
			// Scale weighted percentage to maxScore
			score = (weightedSum / totalWeight) * maxScore
		} else {
			logger.Log.Warnf("finishAndScore: session #%d has totalWeight=0, defaulting score to 0", session.ID)
			score = 0
		}

		// Clamp score to not exceed maxScore
		if maxScore > 0 {
			score = math.Min(score, maxScore)
		}
	} else {
		// Simple scoring (no weights or single bank)
		for _, qID := range order {
			q, ok := qMap[qID]
			if !ok {
				continue
			}
			maxScore += q.Score
			if a, ok := answerMap[qID]; ok {
				score += uc.calculator.Calculate(q, a.Answer)
			}
		}
	}

	now := time.Now()
	session.Status = entity.SessionStatusFinished
	session.FinishedAt = &now
	session.Score = score
	session.MaxScore = maxScore

	if err := uc.sessionRepo.Update(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (uc *ExamSessionUseCase) forceFinish(session *entity.ExamSession) error {
	_, err := uc.finishAndScore(session)
	return err
}

// FinishAllOngoing — force finish all ongoing sessions for a schedule.
func (uc *ExamSessionUseCase) FinishAllOngoing(scheduleID uint) ([]*entity.ExamSession, error) {
	ongoing, err := uc.sessionRepo.ListOngoingBySchedule(scheduleID)
	if err != nil {
		return nil, err
	}
	var finished []*entity.ExamSession
	for _, s := range ongoing {
		if result, err := uc.finishAndScore(s); err == nil {
			finished = append(finished, result)
		}
	}
	return finished, nil
}
