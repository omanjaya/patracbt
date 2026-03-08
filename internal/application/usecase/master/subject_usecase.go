package master

import (
	"errors"

	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
)

var ErrSubjectNotFound = errors.New("mata pelajaran tidak ditemukan")

type SubjectUseCase struct {
	repo repository.SubjectRepository
}

func NewSubjectUseCase(repo repository.SubjectRepository) *SubjectUseCase {
	return &SubjectUseCase{repo: repo}
}

func (uc *SubjectUseCase) List(search string, p pagination.Params) ([]*entity.Subject, int64, error) {
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
	return uc.repo.Delete(id)
}

func (uc *SubjectUseCase) BulkDelete(ids []uint) error {
	return uc.repo.BulkDelete(ids)
}
