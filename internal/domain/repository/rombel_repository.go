package repository

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
)

type RombelRepository interface {
	Create(rombel *entity.Rombel) error
	FindByID(id uint) (*entity.Rombel, error)
	FindByName(name string) (*entity.Rombel, error)
	FindOrCreateByName(name string) (*entity.Rombel, error)
	Update(rombel *entity.Rombel) error
	Delete(id uint) error
	BulkDelete(ids []uint) error
	List(search, gradeLevel string, p pagination.Params) ([]*entity.RombelWithCount, int64, error)
	CountStudents(rombelID uint) (int64, error)
	AssignUsers(rombelID uint, userIDs []uint) error
	RemoveUsers(rombelID uint, userIDs []uint) error
}
