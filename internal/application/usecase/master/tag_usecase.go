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
	ErrTagNotFound = errors.New("tag tidak ditemukan")
	ErrTagInUse    = errors.New("tag masih digunakan, tidak bisa dihapus")
)

type TagUseCase struct {
	repo repository.TagRepository
}

func NewTagUseCase(repo repository.TagRepository) *TagUseCase {
	return &TagUseCase{repo: repo}
}

func (uc *TagUseCase) List(search string, p pagination.Params) ([]*entity.TagWithCount, int64, error) {
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

	count, err := uc.repo.CountUsage(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrTagInUse
	}

	return uc.repo.Delete(id)
}

// TagBulkDeleteResult holds the result of a bulk delete operation.
type TagBulkDeleteResult struct {
	Deleted int      `json:"deleted"`
	Skipped []string `json:"skipped,omitempty"`
}

func (uc *TagUseCase) BulkDelete(ids []uint) (*TagBulkDeleteResult, error) {
	result := &TagBulkDeleteResult{}
	var toDelete []uint

	for _, id := range ids {
		count, err := uc.repo.CountUsage(id)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			tag, _ := uc.repo.FindByID(id)
			name := fmt.Sprintf("ID %d", id)
			if tag != nil {
				name = tag.Name
			}
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s masih digunakan (%d referensi)", name, count))
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

func (uc *TagUseCase) AssignUsers(tagID uint, req dto.AssignUsersRequest) error {
	return uc.repo.AssignUsers(tagID, req.UserIDs)
}

func (uc *TagUseCase) RemoveUsers(tagID uint, req dto.AssignUsersRequest) error {
	return uc.repo.RemoveUsers(tagID, req.UserIDs)
}
