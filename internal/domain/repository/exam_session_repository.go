package repository

import (
	"time"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
)

type ExamSessionRepository interface {
	Create(s *entity.ExamSession) error
	FindByID(id uint) (*entity.ExamSession, error)
	FindByIDBasic(id uint) (*entity.ExamSession, error)
	FindByUserAndSchedule(userID, scheduleID uint) (*entity.ExamSession, error)
	Update(s *entity.ExamSession) error

	// Targeted updates (avoid locking entire row)
	AtomicFinish(id uint) (int64, error) // returns rows affected; 0 = already finished
	UpdateStatus(id uint, status string, finishedAt *time.Time) error
	UpdateScore(id uint, score, maxScore float64) error
	IncrementViolation(id uint) error
	UpdateExtraTime(id uint, extraTime int) error
	ListBySchedule(scheduleID uint, p pagination.Params) ([]*entity.ExamSession, int64, error)
	ListByUser(userID uint) ([]*entity.ExamSession, error)

	// Answers
	UpsertAnswer(answer *entity.ExamAnswer) error
	BatchUpsertAnswers(answers []*entity.ExamAnswer) error
	GetAnswer(sessionID, questionID uint) (*entity.ExamAnswer, error)
	GetAllAnswers(sessionID uint) ([]entity.ExamAnswer, error)
	GetAllAnswersBySchedule(scheduleID uint) (map[uint][]entity.ExamAnswer, error)
	// BUG-02 fix: hapus semua jawaban saat reset session
	DeleteAnswersBySession(sessionID uint) error
	// BUG-03 fix: hitung hanya jawaban yang benar-benar terisi
	CountNonEmptyAnswers(sessionID uint) (int, error)

	// Violations
	LogViolation(log *entity.ViolationLog) error
	CountViolations(sessionID uint) (int, error)

	// Eligibility helpers
	UserInRombels(userID uint, rombelIDs []uint) (bool, error)
	UserHasTags(userID uint, tagIDs []uint) (bool, error)
	GetUserRombelIDs(userID uint) ([]uint, error)
	GetUserTagIDs(userID uint) ([]uint, error)

	// Supervision
	FindOngoingByUser(userID uint) (*entity.ExamSession, error)
	FindExpiredOngoing() ([]*entity.ExamSession, error)
	ListFinishedBySchedule(scheduleID uint) ([]*entity.ExamSession, error)
	ListOngoingBySchedule(scheduleID uint) ([]*entity.ExamSession, error)
	ListNotStartedBySchedule(scheduleID uint) ([]*entity.ExamSession, error)
	Delete(id uint) error
	CountByScheduleAndStatus(scheduleID uint, status string) (int64, error)

	// Regrade
	CreateRegradeLog(log *entity.RegradeLog) error
	ListRegradeLogs(scheduleID uint) ([]entity.RegradeLog, error)

	// User Rombel mapping: returns map[userID][]rombelName
	GetUserRombelNames(userIDs []uint) (map[uint][]string, error)
}
