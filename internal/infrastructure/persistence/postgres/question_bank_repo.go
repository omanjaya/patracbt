package postgres

import (
	"time"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
	"gorm.io/gorm"
)

type QuestionBankRepo struct {
	db *gorm.DB
}

func NewQuestionBankRepository(db *gorm.DB) *QuestionBankRepo {
	return &QuestionBankRepo{db: db}
}

func (r *QuestionBankRepo) Create(bank *entity.QuestionBank) error {
	return r.db.Create(bank).Error
}

func (r *QuestionBankRepo) FindByID(id uint) (*entity.QuestionBank, error) {
	var bank entity.QuestionBank
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).Preload("Subject").First(&bank).Error
	if err != nil {
		return nil, err
	}
	return &bank, nil
}

func (r *QuestionBankRepo) Update(bank *entity.QuestionBank) error {
	return r.db.Model(bank).Updates(map[string]interface{}{
		"name":        bank.Name,
		"subject_id":  bank.SubjectID,
		"description": bank.Description,
		"status":      bank.Status,
		"updated_at":  bank.UpdatedAt,
	}).Error
}

func (r *QuestionBankRepo) Delete(id uint) error {
	now := time.Now()
	return r.db.Model(&entity.QuestionBank{}).Where("id = ?", id).Update("deleted_at", now).Error
}

func (r *QuestionBankRepo) BulkDelete(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now()
	return r.db.Model(&entity.QuestionBank{}).Where("id IN ?", ids).Update("deleted_at", now).Error
}

func (r *QuestionBankRepo) ToggleStatus(id uint) error {
	return r.db.Exec(`
		UPDATE question_banks
		SET status = CASE WHEN status = 'active' THEN 'inactive' ELSE 'active' END
		WHERE id = ?
	`, id).Error
}

func (r *QuestionBankRepo) IsBankUsedInSchedule(bankID uint) bool {
	var count int64
	r.db.Model(&entity.ExamScheduleQuestionBank{}).
		Where("question_bank_id = ?", bankID).
		Count(&count)
	return count > 0
}

func (r *QuestionBankRepo) Clone(bankID uint, newName string, createdBy uint) (*entity.QuestionBank, error) {
	var original entity.QuestionBank
	if err := r.db.Preload("Subject").First(&original, bankID).Error; err != nil {
		return nil, err
	}

	newBank := &entity.QuestionBank{
		Name:        newName,
		SubjectID:   original.SubjectID,
		Description: original.Description,
		Status:      "active",
		CreatedBy:   createdBy,
	}
	if err := r.db.Create(newBank).Error; err != nil {
		return nil, err
	}

	// Clone all questions using batch insert
	var questions []entity.Question
	r.db.Where("question_bank_id = ? AND deleted_at IS NULL", bankID).Find(&questions)

	if len(questions) > 0 {
		newQuestions := make([]entity.Question, len(questions))
		for i := range questions {
			newQuestions[i] = entity.Question{
				QuestionBankID: newBank.ID,
				StimulusID:     nil, // Don't clone stimulus reference
				QuestionType:   questions[i].QuestionType,
				Body:           questions[i].Body,
				Score:          questions[i].Score,
				Difficulty:     questions[i].Difficulty,
				BloomLevel:     questions[i].BloomLevel,
				TopicCode:      questions[i].TopicCode,
				Options:        questions[i].Options,
				CorrectAnswer:  questions[i].CorrectAnswer,
				AudioPath:      nil, // Don't clone audio files
				AudioLimit:     questions[i].AudioLimit,
				OrderIndex:     questions[i].OrderIndex,
			}
		}
		if err := r.db.CreateInBatches(newQuestions, 50).Error; err != nil {
			return nil, err
		}
	}

	return newBank, nil
}

func (r *QuestionBankRepo) List(filter repository.QuestionBankFilter, p pagination.Params) ([]*entity.QuestionBank, int64, error) {
	q := r.db.Model(&entity.QuestionBank{}).Where("deleted_at IS NULL").Preload("Subject")

	if filter.Search != "" {
		q = q.Where("name ILIKE ?", "%"+filter.Search+"%")
	}
	if filter.SubjectID != nil {
		q = q.Where("subject_id = ?", *filter.SubjectID)
	}
	if filter.CreatedBy != nil {
		q = q.Where("created_by = ?", *filter.CreatedBy)
	}

	var total int64
	q.Count(&total)

	var banks []*entity.QuestionBank
	err := q.Order("created_at DESC").Offset(p.Offset()).Limit(p.PerPage).Find(&banks).Error
	if err != nil || len(banks) == 0 {
		return banks, total, err
	}

	// Batch fetch question counts and schedule usage to avoid N+1 queries
	bankIDs := make([]uint, len(banks))
	bankMap := make(map[uint]*entity.QuestionBank, len(banks))
	for i, b := range banks {
		bankIDs[i] = b.ID
		bankMap[b.ID] = b
	}

	// Batch query: question counts per bank
	type countResult struct {
		QuestionBankID uint
		Count          int
	}
	var counts []countResult
	r.db.Model(&entity.Question{}).
		Select("question_bank_id, COUNT(*) as count").
		Where("question_bank_id IN ? AND deleted_at IS NULL", bankIDs).
		Group("question_bank_id").
		Scan(&counts)
	for _, c := range counts {
		if b, ok := bankMap[c.QuestionBankID]; ok {
			b.QuestionCount = c.Count
		}
	}

	// Batch query: which banks are used in schedules
	var usedBankIDs []uint
	r.db.Model(&entity.ExamScheduleQuestionBank{}).
		Select("DISTINCT question_bank_id").
		Where("question_bank_id IN ?", bankIDs).
		Pluck("question_bank_id", &usedBankIDs)
	for _, id := range usedBankIDs {
		if b, ok := bankMap[id]; ok {
			b.IsLocked = true
		}
	}

	return banks, total, nil
}
