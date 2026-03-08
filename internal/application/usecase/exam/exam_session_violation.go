package exam

import (
	"errors"
	"time"

	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/types"
)

func (uc *ExamSessionUseCase) LogViolation(sessionID, userID uint, req dto.LogViolationRequest) error {
	session, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil || session.UserID != userID {
		return errors.New("sesi tidak valid")
	}

	log := &entity.ViolationLog{
		ExamSessionID: sessionID,
		ViolationType: req.ViolationType,
		Description:   req.Description,
	}
	if err := uc.sessionRepo.LogViolation(log); err != nil {
		return err
	}

	// Load schedule for max violations
	schedule, _ := uc.scheduleRepo.FindByID(session.ExamScheduleID)
	count, _ := uc.sessionRepo.CountViolations(sessionID)

	if schedule != nil && count >= schedule.MaxViolations {
		// BUG-07 fix: hitung skor dulu sebelum terminate agar tidak hilang
		finished, err := uc.finishAndScore(session)
		if err != nil {
			logger.Log.Errorf("LogViolation: gagal score sesi #%d sebelum terminate: %v", sessionID, err)
			// Tetap terminate walau scoring gagal
			session.Status = entity.SessionStatusTerminated
			session.ViolationCount = count
			return uc.sessionRepo.Update(session)
		}
		// Override status dari Finished ke Terminated
		finished.Status = entity.SessionStatusTerminated
		finished.ViolationCount = count
		return uc.sessionRepo.Update(finished)
	}
	session.ViolationCount = count
	return uc.sessionRepo.Update(session)
}

// GetViolationCount returns violation count for a session
func (uc *ExamSessionUseCase) GetViolationCount(sessionID uint) (int, error) {
	count, err := uc.sessionRepo.CountViolations(sessionID)
	return count, err
}

// ForceTerminate — pengawas/admin terminates a student's session.
// BUG-11 fix: hitung skor dulu, lalu set status = Terminated (bukan Finished)
func (uc *ExamSessionUseCase) ForceTerminate(sessionID uint) (*entity.ExamSession, error) {
	session, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, errors.New("sesi tidak ditemukan")
	}
	if session.Status == entity.SessionStatusFinished || session.Status == entity.SessionStatusTerminated {
		return session, nil
	}
	scored, err := uc.finishAndScore(session)
	if err != nil {
		return nil, err
	}
	// Override status dari Finished ke Terminated
	scored.Status = entity.SessionStatusTerminated
	if err := uc.sessionRepo.Update(scored); err != nil {
		return nil, err
	}
	return scored, nil
}

// ExtendTime adds extra minutes to an ongoing session.
func (uc *ExamSessionUseCase) ExtendTime(sessionID uint, minutes int) (*entity.ExamSession, error) {
	session, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, errors.New("sesi tidak ditemukan")
	}
	if session.Status != entity.SessionStatusOngoing {
		return nil, errors.New("sesi tidak sedang berlangsung")
	}
	session.ExtraTime += minutes
	if session.EndTime != nil {
		newEnd := session.EndTime.Add(time.Duration(minutes) * time.Minute)
		session.EndTime = &newEnd
	}
	if err := uc.sessionRepo.Update(session); err != nil {
		return nil, err
	}
	return session, nil
}

// UnlockSession re-opens a terminated session so the student can continue.
func (uc *ExamSessionUseCase) UnlockSession(sessionID uint) (*entity.ExamSession, error) {
	session, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, errors.New("sesi tidak ditemukan")
	}
	session.Status = entity.SessionStatusOngoing
	if err := uc.sessionRepo.Update(session); err != nil {
		return nil, err
	}
	return session, nil
}

// ResetSession deletes a student's session answers and resets state so they can re-take the exam.
// BUG-02 fix: hapus semua jawaban lama agar tidak ikut dihitung ulang
func (uc *ExamSessionUseCase) ResetSession(sessionID uint) error {
	session, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil {
		return errors.New("sesi tidak ditemukan")
	}
	// Invalidate Redis cache and clear answer buffer
	uc.examCache.InvalidateSession(sessionID)
	uc.examCache.ClearAnswerBuffer(sessionID)
	// Hapus jawaban lama
	if err := uc.sessionRepo.DeleteAnswersBySession(sessionID); err != nil {
		return err
	}
	session.Status = entity.SessionStatusNotStarted
	session.Score = 0
	session.MaxScore = 0
	session.ViolationCount = 0
	session.ExtraTime = 0
	session.SectionIndex = 0
	session.QuestionOrder = types.JSON("[]")
	session.OptionOrder = types.JSON("[]")
	session.StartTime = nil
	session.EndTime = nil
	session.FinishedAt = nil
	return uc.sessionRepo.Update(session)
}

// ReturnToExam sets a completed/terminated session back to ongoing, clears score but keeps answers.
func (uc *ExamSessionUseCase) ReturnToExam(sessionID uint) (*entity.ExamSession, error) {
	session, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil {
		return nil, errors.New("sesi tidak ditemukan")
	}
	if session.Status != entity.SessionStatusFinished && session.Status != entity.SessionStatusTerminated {
		return nil, errors.New("sesi tidak dalam status selesai atau dihentikan")
	}

	// Reset status to ongoing, clear score but keep answers
	session.Status = entity.SessionStatusOngoing
	session.Score = 0
	session.MaxScore = 0
	session.FinishedAt = nil

	// Extend end time if it has passed
	now := time.Now()
	if session.EndTime != nil && session.EndTime.Before(now) {
		newEnd := now.Add(30 * time.Minute)
		session.EndTime = &newEnd
	}

	if err := uc.sessionRepo.Update(session); err != nil {
		return nil, err
	}
	return session, nil
}
