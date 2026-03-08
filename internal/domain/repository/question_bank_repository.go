package repository

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
)

type QuestionBankFilter struct {
	Search    string
	SubjectID *uint
	CreatedBy *uint
}

type QuestionBankRepository interface {
	Create(bank *entity.QuestionBank) error
	FindByID(id uint) (*entity.QuestionBank, error)
	Update(bank *entity.QuestionBank) error
	Delete(id uint) error
	BulkDelete(ids []uint) error
	ToggleStatus(id uint) error
	List(filter QuestionBankFilter, p pagination.Params) ([]*entity.QuestionBank, int64, error)
	IsBankUsedInSchedule(bankID uint) bool
	Clone(bankID uint, newName string, createdBy uint) (*entity.QuestionBank, error)
}
