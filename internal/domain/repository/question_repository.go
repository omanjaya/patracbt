package repository

import (
	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
)

type QuestionRepository interface {
	Create(q *entity.Question) error
	FindByID(id uint) (*entity.Question, error)
	FindByIDs(ids []uint) ([]*entity.Question, error)
	Update(q *entity.Question) error
	Delete(id uint) error
	BulkDelete(ids []uint) error
	MoveToBank(ids []uint, targetBankID uint) error
	CopyToBank(ids []uint, targetBankID uint) error
	ListByBank(bankID uint, p pagination.Params) ([]*entity.Question, int64, error)
	CountByBank(bankID uint) (int64, error)
	BulkCreate(questions []*entity.Question) error
	Reorder(bankID uint, items []dto.ReorderItem) error

	// All questions (no pagination)
	ListAllByBank(bankID uint) ([]*entity.Question, error)
	ListIDsByBank(bankID uint, search string) ([]uint, error)

	// Stimulus
	CreateStimulus(s *entity.Stimulus) error
	FindStimulusByID(id uint) (*entity.Stimulus, error)
	ListStimuliByBank(bankID uint) ([]*entity.Stimulus, error)
	UpdateStimulus(s *entity.Stimulus) error
	DeleteStimulus(id uint) error
}
