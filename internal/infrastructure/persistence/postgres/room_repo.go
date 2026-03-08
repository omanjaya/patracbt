package postgres

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
	"gorm.io/gorm"
)

type RoomRepo struct{ db *gorm.DB }

func NewRoomRepository(db *gorm.DB) *RoomRepo { return &RoomRepo{db: db} }

func (r *RoomRepo) Create(room *entity.Room) error {
	return r.db.Create(room).Error
}

func (r *RoomRepo) FindByID(id uint) (*entity.Room, error) {
	var room entity.Room
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&room).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &room, err
}

func (r *RoomRepo) Update(room *entity.Room) error {
	return r.db.Save(room).Error
}

func (r *RoomRepo) Delete(id uint) error {
	return r.db.Model(&entity.Room{}).Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *RoomRepo) BulkDelete(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Model(&entity.Room{}).Where("id IN ?", ids).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *RoomRepo) List(search string, p pagination.Params) ([]*entity.Room, int64, error) {
	var rooms []*entity.Room
	var total int64

	q := r.db.Model(&entity.Room{}).Where("deleted_at IS NULL")
	if search != "" {
		q = q.Where("name ILIKE ?", "%"+search+"%")
	}

	q.Count(&total)
	err := q.Offset(p.Offset()).Limit(p.PerPage).Order("name ASC").Find(&rooms).Error
	return rooms, total, err
}

func (r *RoomRepo) AssignUsers(roomID uint, userIDs []uint) error {
	if len(userIDs) == 0 {
		return nil
	}
	return r.db.Model(&entity.UserProfile{}).
		Where("user_id IN ?", userIDs).
		Update("room_id", roomID).Error
}

func (r *RoomRepo) RemoveUsers(roomID uint, userIDs []uint) error {
	if len(userIDs) == 0 {
		return nil
	}
	return r.db.Model(&entity.UserProfile{}).
		Where("user_id IN ? AND room_id = ?", userIDs, roomID).
		Update("room_id", nil).Error
}

func (r *RoomRepo) GetUsersByRoom(roomID uint, p pagination.Params) ([]*entity.User, int64, error) {
	var users []*entity.User
	var total int64

	q := r.db.Model(&entity.User{}).
		Joins("JOIN user_profiles ON user_profiles.user_id = users.id").
		Where("user_profiles.room_id = ? AND users.deleted_at IS NULL", roomID)

	q.Count(&total)
	err := q.Preload("Profile").
		Offset(p.Offset()).Limit(p.PerPage).
		Order("users.name ASC").
		Find(&users).Error
	return users, total, err
}
