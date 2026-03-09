package repository

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
)

type SubjectRepository interface {
	Create(subject *entity.Subject) error
	FindByID(id uint) (*entity.Subject, error)
	Update(subject *entity.Subject) error
	Delete(id uint) error
	BulkDelete(ids []uint) error
	List(search string, p pagination.Params) ([]*entity.SubjectWithCount, int64, error)
	CountUsage(subjectID uint) (int64, error)
	ListAll() ([]*entity.Subject, error)
}
