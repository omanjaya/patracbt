package repository

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
)

type RoomRepository interface {
	Create(room *entity.Room) error
	FindByID(id uint) (*entity.Room, error)
	Update(room *entity.Room) error
	Delete(id uint) error
	BulkDelete(ids []uint) error
	List(search string, p pagination.Params) ([]*entity.RoomWithCount, int64, error)
	CountStudents(roomID uint) (int64, error)
	AssignUsers(roomID uint, userIDs []uint) error
	RemoveUsers(roomID uint, userIDs []uint) error
	GetUsersByRoom(roomID uint, p pagination.Params) ([]*entity.User, int64, error)
}
