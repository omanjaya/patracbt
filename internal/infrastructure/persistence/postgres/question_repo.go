package postgres

import (
	"time"

	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
	"gorm.io/gorm"
)

type QuestionRepo struct {
	db *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) *QuestionRepo {
	return &QuestionRepo{db: db}
}

func (r *QuestionRepo) Create(q *entity.Question) error {
	return r.db.Create(q).Error
}

func (r *QuestionRepo) FindByID(id uint) (*entity.Question, error) {
	var q entity.Question
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&q).Error
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *QuestionRepo) FindByIDs(ids []uint) ([]*entity.Question, error) {
	var questions []*entity.Question
	if len(ids) == 0 {
		return questions, nil
	}
	err := r.db.Where("id IN ? AND deleted_at IS NULL", ids).Find(&questions).Error
	return questions, err
}

func (r *QuestionRepo) Update(q *entity.Question) error {
	return r.db.Save(q).Error
}

func (r *QuestionRepo) Delete(id uint) error {
	now := time.Now()
	return r.db.Model(&entity.Question{}).Where("id = ?", id).Update("deleted_at", now).Error
}

func (r *QuestionRepo) ListByBank(bankID uint, p pagination.Params) ([]*entity.Question, int64, error) {
	var total int64
	r.db.Model(&entity.Question{}).Where("question_bank_id = ? AND deleted_at IS NULL", bankID).Count(&total)

	var questions []*entity.Question
	err := r.db.Where("question_bank_id = ? AND deleted_at IS NULL", bankID).
		Order("order_index ASC, created_at ASC").
		Offset(p.Offset()).Limit(p.PerPage).
		Find(&questions).Error
	return questions, total, err
}

func (r *QuestionRepo) CountByBank(bankID uint) (int64, error) {
	var count int64
	err := r.db.Model(&entity.Question{}).Where("question_bank_id = ? AND deleted_at IS NULL", bankID).Count(&count).Error
	return count, err
}

func (r *QuestionRepo) BulkDelete(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now()
	return r.db.Model(&entity.Question{}).Where("id IN ?", ids).Update("deleted_at", now).Error
}

func (r *QuestionRepo) MoveToBank(ids []uint, targetBankID uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Model(&entity.Question{}).Where("id IN ?", ids).Update("question_bank_id", targetBankID).Error
}

func (r *QuestionRepo) CopyToBank(ids []uint, targetBankID uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		var originals []*entity.Question
		if err := tx.Where("id IN ?", ids).Find(&originals).Error; err != nil {
			return err
		}
		copies := make([]*entity.Question, 0, len(originals))
		for _, q := range originals {
			copies = append(copies, &entity.Question{
				QuestionBankID: targetBankID,
				StimulusID:     nil, // don't copy stimulus linkage
				QuestionType:   q.QuestionType,
				Body:           q.Body,
				Score:          q.Score,
				Difficulty:     q.Difficulty,
				Options:        q.Options,
				CorrectAnswer:  q.CorrectAnswer,
				OrderIndex:     q.OrderIndex,
			})
		}
		return tx.CreateInBatches(copies, 50).Error
	})
}

func (r *QuestionRepo) BulkCreate(questions []*entity.Question) error {
	if len(questions) == 0 {
		return nil
	}
	return r.db.CreateInBatches(questions, 50).Error
}

func (r *QuestionRepo) Reorder(bankID uint, items []dto.ReorderItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&entity.Question{}).
				Where("id = ? AND question_bank_id = ?", item.ID, bankID).
				Update("order_index", item.OrderIndex).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *QuestionRepo) ListAllByBank(bankID uint) ([]*entity.Question, error) {
	var questions []*entity.Question
	err := r.db.Where("question_bank_id = ? AND deleted_at IS NULL", bankID).
		Order("order_index ASC, created_at ASC").
		Find(&questions).Error
	return questions, err
}

func (r *QuestionRepo) ListIDsByBank(bankID uint, search string) ([]uint, error) {
	var ids []uint
	q := r.db.Model(&entity.Question{}).Where("question_bank_id = ? AND deleted_at IS NULL", bankID)
	if search != "" {
		q = q.Where("body ILIKE ?", "%"+search+"%")
	}
	err := q.Order("order_index ASC, created_at ASC").Pluck("id", &ids).Error
	return ids, err
}

// Stimulus methods

func (r *QuestionRepo) CreateStimulus(s *entity.Stimulus) error {
	return r.db.Create(s).Error
}

func (r *QuestionRepo) FindStimulusByID(id uint) (*entity.Stimulus, error) {
	var s entity.Stimulus
	err := r.db.First(&s, id).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *QuestionRepo) ListStimuliByBank(bankID uint) ([]*entity.Stimulus, error) {
	var stimuli []*entity.Stimulus
	err := r.db.Where("question_bank_id = ?", bankID).Order("created_at ASC").Limit(500).Find(&stimuli).Error
	return stimuli, err
}

func (r *QuestionRepo) UpdateStimulus(s *entity.Stimulus) error {
	return r.db.Save(s).Error
}

func (r *QuestionRepo) DeleteStimulus(id uint) error {
	return r.db.Delete(&entity.Stimulus{}, id).Error
}
