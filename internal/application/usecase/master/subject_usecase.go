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
	ErrSubjectNotFound = errors.New("mata pelajaran tidak ditemukan")
	ErrSubjectInUse    = errors.New("mata pelajaran masih digunakan, tidak bisa dihapus")
)

type SubjectUseCase struct {
	repo repository.SubjectRepository
}

func NewSubjectUseCase(repo repository.SubjectRepository) *SubjectUseCase {
	return &SubjectUseCase{repo: repo}
}

func (uc *SubjectUseCase) List(search string, p pagination.Params) ([]*entity.SubjectWithCount, int64, error) {
	return uc.repo.List(search, p)
}

func (uc *SubjectUseCase) ListAll() ([]*entity.Subject, error) {
	return uc.repo.ListAll()
}

func (uc *SubjectUseCase) Create(req dto.CreateSubjectRequest) (*entity.Subject, error) {
	subject := &entity.Subject{Name: req.Name, Code: req.Code}
	return subject, uc.repo.Create(subject)
}

func (uc *SubjectUseCase) Update(id uint, req dto.UpdateSubjectRequest) (*entity.Subject, error) {
	subject, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if subject == nil {
		return nil, ErrSubjectNotFound
	}
	subject.Name = req.Name
	subject.Code = req.Code
	return subject, uc.repo.Update(subject)
}

func (uc *SubjectUseCase) Delete(id uint) error {
	subject, err := uc.repo.FindByID(id)
	if err != nil {
		return err
	}
	if subject == nil {
		return ErrSubjectNotFound
	}

	count, err := uc.repo.CountUsage(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrSubjectInUse
	}

	return uc.repo.Delete(id)
}

// SubjectBulkDeleteResult holds the result of a bulk delete operation.
type SubjectBulkDeleteResult struct {
	Deleted int      `json:"deleted"`
	Skipped []string `json:"skipped,omitempty"`
}

func (uc *SubjectUseCase) BulkDelete(ids []uint) (*SubjectBulkDeleteResult, error) {
	result := &SubjectBulkDeleteResult{}
	var toDelete []uint

	for _, id := range ids {
		count, err := uc.repo.CountUsage(id)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			subject, _ := uc.repo.FindByID(id)
			name := fmt.Sprintf("ID %d", id)
			if subject != nil {
				name = subject.Name
			}
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s masih memiliki %d bank soal", name, count))
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
