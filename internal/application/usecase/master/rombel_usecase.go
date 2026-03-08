package master

import (
	"errors"

	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
)

var ErrRombelNotFound = errors.New("rombel tidak ditemukan")

type RombelUseCase struct {
	repo repository.RombelRepository
}

func NewRombelUseCase(repo repository.RombelRepository) *RombelUseCase {
	return &RombelUseCase{repo: repo}
}

func (uc *RombelUseCase) List(search string, p pagination.Params) ([]*entity.Rombel, int64, error) {
	return uc.repo.List(search, p)
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
	return uc.repo.Delete(id)
}

func (uc *RombelUseCase) BulkDelete(ids []uint) error {
	return uc.repo.BulkDelete(ids)
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
