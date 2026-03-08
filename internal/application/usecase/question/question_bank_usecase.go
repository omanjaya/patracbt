package question

import (
	"errors"
	"fmt"

	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
)

type QuestionBankUseCase struct {
	bankRepo     repository.QuestionBankRepository
	questionRepo repository.QuestionRepository
}

func NewQuestionBankUseCase(bankRepo repository.QuestionBankRepository, questionRepo repository.QuestionRepository) *QuestionBankUseCase {
	return &QuestionBankUseCase{bankRepo: bankRepo, questionRepo: questionRepo}
}

func (uc *QuestionBankUseCase) List(filter repository.QuestionBankFilter, p pagination.Params) ([]*entity.QuestionBank, int64, error) {
	// question_count and is_locked are now populated via subqueries in the repository List method
	banks, total, err := uc.bankRepo.List(filter, p)
	if err != nil {
		return nil, 0, err
	}
	return banks, total, nil
}

func (uc *QuestionBankUseCase) IsLocked(bankID uint) bool {
	return uc.bankRepo.IsBankUsedInSchedule(bankID)
}

func (uc *QuestionBankUseCase) GetByID(id uint) (*entity.QuestionBank, error) {
	bank, err := uc.bankRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("bank soal tidak ditemukan")
	}
	count, _ := uc.questionRepo.CountByBank(id)
	bank.QuestionCount = int(count)
	bank.IsLocked = uc.bankRepo.IsBankUsedInSchedule(id)
	return bank, nil
}

func (uc *QuestionBankUseCase) Create(req dto.CreateQuestionBankRequest, createdBy uint) (*entity.QuestionBank, error) {
	bank := &entity.QuestionBank{
		Name:        req.Name,
		SubjectID:   req.SubjectID,
		Description: req.Description,
		CreatedBy:   createdBy,
	}
	if err := uc.bankRepo.Create(bank); err != nil {
		return nil, err
	}
	return bank, nil
}

func (uc *QuestionBankUseCase) Update(id uint, req dto.UpdateQuestionBankRequest) (*entity.QuestionBank, error) {
	bank, err := uc.bankRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("bank soal tidak ditemukan")
	}
	bank.Name = req.Name
	bank.SubjectID = req.SubjectID
	bank.Description = req.Description
	if err := uc.bankRepo.Update(bank); err != nil {
		return nil, err
	}
	return bank, nil
}

func (uc *QuestionBankUseCase) Delete(id uint) error {
	if _, err := uc.bankRepo.FindByID(id); err != nil {
		return errors.New("bank soal tidak ditemukan")
	}
	if uc.bankRepo.IsBankUsedInSchedule(id) {
		return errors.New("bank soal sedang digunakan dalam jadwal ujian")
	}
	return uc.bankRepo.Delete(id)
}

func (uc *QuestionBankUseCase) BulkDelete(ids []uint) error {
	// Check if any bank is used in a schedule before bulk deleting
	for _, id := range ids {
		if uc.bankRepo.IsBankUsedInSchedule(id) {
			return fmt.Errorf("bank soal ID %d tidak dapat dihapus karena sedang digunakan dalam jadwal ujian", id)
		}
	}
	return uc.bankRepo.BulkDelete(ids)
}

func (uc *QuestionBankUseCase) Clone(bankID uint, createdBy uint) (*entity.QuestionBank, error) {
	original, err := uc.bankRepo.FindByID(bankID)
	if err != nil {
		return nil, errors.New("bank soal tidak ditemukan")
	}
	newName := original.Name + " (Salinan)"
	bank, err := uc.bankRepo.Clone(bankID, newName, createdBy)
	if err != nil {
		return nil, err
	}
	count, _ := uc.questionRepo.CountByBank(bank.ID)
	bank.QuestionCount = int(count)
	return bank, nil
}

func (uc *QuestionBankUseCase) ToggleStatus(id uint) error {
	if _, err := uc.bankRepo.FindByID(id); err != nil {
		return errors.New("bank soal tidak ditemukan")
	}
	return uc.bankRepo.ToggleStatus(id)
}
