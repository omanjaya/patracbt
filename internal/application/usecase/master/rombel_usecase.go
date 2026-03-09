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
	ErrRombelNotFound    = errors.New("rombel tidak ditemukan")
	ErrRombelHasStudents = errors.New("rombel masih memiliki siswa, tidak bisa dihapus")
)

type RombelUseCase struct {
	repo repository.RombelRepository
}

func NewRombelUseCase(repo repository.RombelRepository) *RombelUseCase {
	return &RombelUseCase{repo: repo}
}

func (uc *RombelUseCase) List(search, gradeLevel string, p pagination.Params) ([]*entity.RombelWithCount, int64, error) {
	return uc.repo.List(search, gradeLevel, p)
}

func (uc *RombelUseCase) Create(req dto.CreateRombelRequest) (*entity.Rombel, error) {
	rombel := &entity.Rombel{
		Name:        req.Name,
		GradeLevel:  req.GradeLevel,
		Description: req.Description,
	}
	return rombel, uc.repo.Create(rombel)
}

func (uc *RombelUseCase) Update(id uint, req dto.UpdateRombelRequest) (*entity.Rombel, error) {
	rombel, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if rombel == nil {
		return nil, ErrRombelNotFound
	}

	rombel.Name = req.Name
	rombel.GradeLevel = req.GradeLevel
	rombel.Description = req.Description
	return rombel, uc.repo.Update(rombel)
}

func (uc *RombelUseCase) Delete(id uint) error {
	rombel, err := uc.repo.FindByID(id)
	if err != nil {
		return err
	}
	if rombel == nil {
		return ErrRombelNotFound
	}

	count, err := uc.repo.CountStudents(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrRombelHasStudents
	}

	return uc.repo.Delete(id)
}

// BulkDeleteResult holds the result of a bulk delete operation.
type BulkDeleteResult struct {
	Deleted int      `json:"deleted"`
	Skipped []string `json:"skipped,omitempty"`
}

func (uc *RombelUseCase) BulkDelete(ids []uint) (*BulkDeleteResult, error) {
	result := &BulkDeleteResult{}
	var toDelete []uint

	for _, id := range ids {
		count, err := uc.repo.CountStudents(id)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			rombel, _ := uc.repo.FindByID(id)
			name := fmt.Sprintf("ID %d", id)
			if rombel != nil {
				name = rombel.Name
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

func (uc *RombelUseCase) AssignUsers(rombelID uint, req dto.AssignUsersRequest) error {
	rombel, err := uc.repo.FindByID(rombelID)
	if err != nil {
		return err
	}
	if rombel == nil {
		return ErrRombelNotFound
	}
	return uc.repo.AssignUsers(rombelID, req.UserIDs)
}

func (uc *RombelUseCase) RemoveUsers(rombelID uint, req dto.AssignUsersRequest) error {
	return uc.repo.RemoveUsers(rombelID, req.UserIDs)
}
