package exam

import (
	"errors"

	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
)

// GetTransition returns the next schedule info if this is a multi-stage exam.
func (uc *ExamSessionUseCase) GetTransition(sessionID, userID uint) (*entity.ExamSchedule, error) {
	session, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil || session.UserID != userID {
		return nil, errors.New("sesi tidak valid")
	}
	schedule, err := uc.scheduleRepo.FindByID(session.ExamScheduleID)
	if err != nil || schedule.NextExamScheduleID == nil {
		return nil, errors.New("tidak ada bagian berikutnya")
	}
	return uc.scheduleRepo.FindByID(*schedule.NextExamScheduleID)
}

// StartSection transitions a session to the next section of a multi-stage exam.
func (uc *ExamSessionUseCase) StartSection(sessionID, userID uint) (*StartExamResult, error) {
	session, err := uc.sessionRepo.FindByID(sessionID)
	if err != nil || session.UserID != userID {
		return nil, errors.New("sesi tidak valid")
	}
	if session.Status != entity.SessionStatusFinished {
		return nil, errors.New("bagian sebelumnya belum selesai")
	}

	schedule, err := uc.scheduleRepo.FindByID(session.ExamScheduleID)
	if err != nil || schedule.NextExamScheduleID == nil {
		return nil, errors.New("tidak ada bagian berikutnya")
	}

	nextSchedule, err := uc.scheduleRepo.FindByID(*schedule.NextExamScheduleID)
	if err != nil {
		return nil, errors.New("jadwal bagian berikutnya tidak ditemukan")
	}

	return uc.StartExam(userID, dto.StartExamRequest{
		ExamScheduleID: nextSchedule.ID,
		Token:          nextSchedule.Token,
	})
}
