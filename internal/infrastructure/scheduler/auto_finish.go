package scheduler

import (
	"encoding/json"
	"time"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/internal/domain/service"
	ws "github.com/omanjaya/patra/internal/infrastructure/websocket"
	"github.com/omanjaya/patra/pkg/logger"
)

// StartAutoFinish runs a goroutine that auto-finishes expired exam sessions every minute.
func StartAutoFinish(
	sessionRepo repository.ExamSessionRepository,
	questionRepo repository.QuestionRepository,
	hub *ws.Hub,
) {
	go func() {
		calc := service.NewScoreCalculator()
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			runAutoFinish(sessionRepo, questionRepo, hub, calc)
		}
	}()
}

func runAutoFinish(
	sessionRepo repository.ExamSessionRepository,
	questionRepo repository.QuestionRepository,
	hub *ws.Hub,
	calc *service.ScoreCalculator,
) {
	sessions, err := sessionRepo.FindExpiredOngoing()
	if err != nil {
		logger.Log.Errorf("AutoFinish: gagal query expired sessions: %v", err)
		return
	}
	if len(sessions) == 0 {
		return
	}

	logger.Log.Infof("AutoFinish: memproses %d sesi kadaluwarsa", len(sessions))

	for _, session := range sessions {
		if err := finishSession(session, sessionRepo, questionRepo, hub, calc); err != nil {
			logger.Log.Errorf("AutoFinish: gagal selesaikan sesi #%d: %v", session.ID, err)
		}
	}
}

func finishSession(
	session *entity.ExamSession,
	sessionRepo repository.ExamSessionRepository,
	questionRepo repository.QuestionRepository,
	hub *ws.Hub,
	calc *service.ScoreCalculator,
) error {
	// Parse question order
	var order []uint
	if err := json.Unmarshal(session.QuestionOrder, &order); err != nil || len(order) == 0 {
		// No questions — just mark finished
		now := time.Now().UTC()
		session.Status = entity.SessionStatusFinished
		session.FinishedAt = &now
		return sessionRepo.Update(session)
	}

	answers, _ := sessionRepo.GetAllAnswers(session.ID)
	answerMap := make(map[uint]entity.ExamAnswer)
	for _, a := range answers {
		answerMap[a.QuestionID] = a
	}

	// Batch-load all questions in one query to avoid N+1
	questions, err := questionRepo.FindByIDs(order)
	if err != nil {
		logger.Log.Errorf("AutoFinish: sesi #%d gagal load questions: %v", session.ID, err)
		// Fallback: just mark finished without scoring
		now := time.Now().UTC()
		session.Status = entity.SessionStatusFinished
		session.FinishedAt = &now
		return sessionRepo.Update(session)
	}
	questionMap := make(map[uint]*entity.Question, len(questions))
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	var score, maxScore float64
	for _, qID := range order {
		q, ok := questionMap[qID]
		if !ok {
			continue
		}
		maxScore += q.Score
		if a, aOk := answerMap[qID]; aOk {
			score += calc.Calculate(q, a.Answer)
		}
	}

	now := time.Now().UTC()
	session.Status = entity.SessionStatusFinished
	session.FinishedAt = &now
	session.Score = score
	session.MaxScore = maxScore

	if err := sessionRepo.Update(session); err != nil {
		return err
	}

	// Notify supervisors via WS
	hub.BroadcastToSupervisors(session.ExamScheduleID, ws.Message{
		Event: ws.EventSessionFinished,
		Data: ws.SessionFinishedPayload{
			SessionID: session.ID,
			UserID:    session.UserID,
			Score:     score,
			MaxScore:  maxScore,
		},
	})

	// Notify the student
	hub.SendToUser(session.ExamScheduleID, session.UserID, ws.Message{
		Event: ws.EventForceFinish,
		Data: ws.ForceFinishPayload{
			SessionID: session.ID,
			UserID:    session.UserID,
			Message:   "Waktu ujian Anda telah habis. Ujian diselesaikan otomatis.",
		},
	})

	logger.Log.Infof("AutoFinish: sesi #%d (user %d) → selesai, skor %.1f/%.1f", session.ID, session.UserID, score, maxScore)
	return nil
}
