package repository

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
)

type TagRepository interface {
	Create(tag *entity.Tag) error
	FindByID(id uint) (*entity.Tag, error)
	Update(tag *entity.Tag) error
	Delete(id uint) error
	BulkDelete(ids []uint) error
	List(search string, p pagination.Params) ([]*entity.TagWithCount, int64, error)
	CountUsage(tagID uint) (int64, error)
	ListAll() ([]*entity.Tag, error)
	AssignUsers(tagID uint, userIDs []uint) error
	RemoveUsers(tagID uint, userIDs []uint) error
}
