package repository

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
)

type RombelRepository interface {
	Create(rombel *entity.Rombel) error
	FindByID(id uint) (*entity.Rombel, error)
	Update(rombel *entity.Rombel) error
	Delete(id uint) error
	BulkDelete(ids []uint) error
	List(search string, p pagination.Params) ([]*entity.Rombel, int64, error)
	AssignUsers(rombelID uint, userIDs []uint) error
	RemoveUsers(rombelID uint, userIDs []uint) error
}
