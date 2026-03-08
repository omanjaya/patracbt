package repository

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
)

type ExamScheduleFilter struct {
	Search    string
	Status    string
	CreatedBy *uint
}

type ExamScheduleRepository interface {
	Create(s *entity.ExamSchedule) error
	FindByID(id uint) (*entity.ExamSchedule, error)
	FindByToken(token string) (*entity.ExamSchedule, error)
	Update(s *entity.ExamSchedule) error
	UpdateSupervisionToken(id uint, token string) error
	Delete(id uint) error
	Restore(id uint) error
	ForceDelete(id uint) error
	ListTrashed(filter ExamScheduleFilter, p pagination.Params) ([]*entity.ExamSchedule, int64, error)
	List(filter ExamScheduleFilter, p pagination.Params) ([]*entity.ExamSchedule, int64, error)
	SetQuestionBanks(scheduleID uint, banks []entity.ExamScheduleQuestionBank) error
	SetRombels(scheduleID uint, rombelIDs []uint) error
	SetTags(scheduleID uint, tagIDs []uint) error
	SetExamRooms(scheduleID uint, rooms []entity.ExamScheduleRoom) error
	SetUsers(scheduleID uint, users []entity.ExamScheduleUser) error
	GetUsersBySchedule(scheduleID uint) ([]entity.ExamScheduleUser, error)
}
