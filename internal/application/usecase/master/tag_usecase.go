package master

import (
	"errors"

	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
)

var ErrTagNotFound = errors.New("tag tidak ditemukan")

type TagUseCase struct {
	repo repository.TagRepository
}

func NewTagUseCase(repo repository.TagRepository) *TagUseCase {
	return &TagUseCase{repo: repo}
}

func (uc *TagUseCase) List(search string, p pagination.Params) ([]*entity.Tag, int64, error) {
	return uc.repo.List(search, p)
}

func (uc *TagUseCase) ListAll() ([]*entity.Tag, error) {
	return uc.repo.ListAll()
}

func (uc *TagUseCase) Create(req dto.CreateTagRequest) (*entity.Tag, error) {
	color := req.Color
	if color == "" {
		color = "#6B7280"
	}
	tag := &entity.Tag{Name: req.Name, Color: color}
	return tag, uc.repo.Create(tag)
}

func (uc *TagUseCase) Update(id uint, req dto.UpdateTagRequest) (*entity.Tag, error) {
	tag, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if tag == nil {
		return nil, ErrTagNotFound
	}
	tag.Name = req.Name
	if req.Color != "" {
		tag.Color = req.Color
	}
	return tag, uc.repo.Update(tag)
}

func (uc *TagUseCase) Delete(id uint) error {
	tag, err := uc.repo.FindByID(id)
	if err != nil {
		return err
	}
	if tag == nil {
		return ErrTagNotFound
	}
	return uc.repo.Delete(id)
}

func (uc *TagUseCase) BulkDelete(ids []uint) error {
	return uc.repo.BulkDelete(ids)
}

func (uc *TagUseCase) AssignUsers(tagID uint, req dto.AssignUsersRequest) error {
	return uc.repo.AssignUsers(tagID, req.UserIDs)
}

func (uc *TagUseCase) RemoveUsers(tagID uint, req dto.AssignUsersRequest) error {
	return uc.repo.RemoveUsers(tagID, req.UserIDs)
}
