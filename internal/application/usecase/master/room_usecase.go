package master

import (
	"errors"
	"fmt"

	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
)

var (
	ErrRoomNotFound    = errors.New("ruangan tidak ditemukan")
	ErrRoomHasStudents = errors.New("ruangan masih memiliki siswa, tidak bisa dihapus")
)

type RoomUseCase struct {
	repo repository.RoomRepository
}

func NewRoomUseCase(repo repository.RoomRepository) *RoomUseCase {
	return &RoomUseCase{repo: repo}
}

func (uc *RoomUseCase) List(search string, p pagination.Params) ([]*entity.RoomWithCount, int64, error) {
	return uc.repo.List(search, p)
}

func (uc *RoomUseCase) Create(req dto.CreateRoomRequest) (*entity.Room, error) {
	cap := req.Capacity
	if cap <= 0 {
		cap = 30
	}
	room := &entity.Room{Name: req.Name, Capacity: cap}
	return room, uc.repo.Create(room)
}

func (uc *RoomUseCase) Update(id uint, req dto.UpdateRoomRequest) (*entity.Room, error) {
	room, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, ErrRoomNotFound
	}
	room.Name = req.Name
	if req.Capacity > 0 {
		room.Capacity = req.Capacity
	}
	return room, uc.repo.Update(room)
}

func (uc *RoomUseCase) Delete(id uint) error {
	room, err := uc.repo.FindByID(id)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrRoomNotFound
	}

	count, err := uc.repo.CountStudents(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrRoomHasStudents
	}

	return uc.repo.Delete(id)
}

// RoomBulkDeleteResult holds the result of a bulk delete operation.
type RoomBulkDeleteResult struct {
	Deleted int      `json:"deleted"`
	Skipped []string `json:"skipped,omitempty"`
}

func (uc *RoomUseCase) BulkDelete(ids []uint) (*RoomBulkDeleteResult, error) {
	result := &RoomBulkDeleteResult{}
	var toDelete []uint

	for _, id := range ids {
		count, err := uc.repo.CountStudents(id)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			room, _ := uc.repo.FindByID(id)
			name := fmt.Sprintf("ID %d", id)
			if room != nil {
				name = room.Name
			}
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s masih memiliki %d siswa", name, count))
		} else {
			toDelete = append(toDelete, id)
		}
	}

	if len(toDelete) > 0 {
		if err := uc.repo.BulkDelete(toDelete); err != nil {
			return nil, err
		}
	}
	result.Deleted = len(toDelete)
	return result, nil
}

func (uc *RoomUseCase) AssignUsers(roomID uint, req dto.AssignUsersRequest) error {
	room, err := uc.repo.FindByID(roomID)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrRoomNotFound
	}
	return uc.repo.AssignUsers(roomID, req.UserIDs)
}

func (uc *RoomUseCase) RemoveUsers(roomID uint, req dto.AssignUsersRequest) error {
	room, err := uc.repo.FindByID(roomID)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrRoomNotFound
	}
	return uc.repo.RemoveUsers(roomID, req.UserIDs)
}

func (uc *RoomUseCase) GetUsers(roomID uint, p pagination.Params) ([]*entity.User, int64, error) {
	room, err := uc.repo.FindByID(roomID)
	if err != nil {
		return nil, 0, err
	}
	if room == nil {
		return nil, 0, ErrRoomNotFound
	}
	return uc.repo.GetUsersByRoom(roomID, p)
}
