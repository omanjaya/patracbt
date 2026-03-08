package postgres

import (
	"errors"
	"time"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
	"gorm.io/gorm"
)

type ExamScheduleRepo struct {
	db *gorm.DB
}

func NewExamScheduleRepository(db *gorm.DB) *ExamScheduleRepo {
	return &ExamScheduleRepo{db: db}
}

func (r *ExamScheduleRepo) Create(s *entity.ExamSchedule) error {
	return r.db.Create(s).Error
}

func (r *ExamScheduleRepo) FindByID(id uint) (*entity.ExamSchedule, error) {
	var s entity.ExamSchedule
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).
		Preload("QuestionBanks").
		Preload("QuestionBanks.QuestionBank").
		Preload("Rombels").
		Preload("Rombels.Rombel").
		Preload("Tags").
		Preload("Tags.Tag").
		Preload("ExamRooms").
		Preload("ExamRooms.Room").
		Preload("Users").
		Preload("Users.User").
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ExamScheduleRepo) FindByToken(token string) (*entity.ExamSchedule, error) {
	var s entity.ExamSchedule
	err := r.db.Where("token = ? AND deleted_at IS NULL", token).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ExamScheduleRepo) Update(s *entity.ExamSchedule) error {
	return r.db.Model(s).Updates(map[string]interface{}{
		"name":                  s.Name,
		"token":                 s.Token,
		"start_time":            s.StartTime,
		"end_time":              s.EndTime,
		"duration_minutes":      s.DurationMinutes,
		"status":                s.Status,
		"allow_see_result":      s.AllowSeeResult,
		"max_violations":        s.MaxViolations,
		"randomize_questions":   s.RandomizeQuestions,
		"randomize_options":     s.RandomizeOptions,
		"next_exam_schedule_id": s.NextExamScheduleID,
		"updated_at":            s.UpdatedAt,
	}).Error
}

func (r *ExamScheduleRepo) UpdateSupervisionToken(id uint, token string) error {
	return r.db.Model(&entity.ExamSchedule{}).Where("id = ?", id).Update("supervision_token", token).Error
}

func (r *ExamScheduleRepo) Delete(id uint) error {
	now := time.Now()
	return r.db.Model(&entity.ExamSchedule{}).Where("id = ?", id).Update("deleted_at", now).Error
}

func (r *ExamScheduleRepo) Restore(id uint) error {
	return r.db.Model(&entity.ExamSchedule{}).Where("id = ?", id).Update("deleted_at", nil).Error
}

func (r *ExamScheduleRepo) ForceDelete(id uint) error {
	return r.db.Unscoped().Delete(&entity.ExamSchedule{}, id).Error
}

func (r *ExamScheduleRepo) ListTrashed(filter repository.ExamScheduleFilter, p pagination.Params) ([]*entity.ExamSchedule, int64, error) {
	q := r.db.Model(&entity.ExamSchedule{}).Where("deleted_at IS NOT NULL")
	if filter.Search != "" {
		q = q.Where("name ILIKE ?", "%"+filter.Search+"%")
	}
	if filter.CreatedBy != nil {
		q = q.Where("created_by = ?", *filter.CreatedBy)
	}
	var total int64
	q.Count(&total)
	var schedules []*entity.ExamSchedule
	err := q.Order("deleted_at DESC").Offset(p.Offset()).Limit(p.PerPage).Find(&schedules).Error
	return schedules, total, err
}

func (r *ExamScheduleRepo) List(filter repository.ExamScheduleFilter, p pagination.Params) ([]*entity.ExamSchedule, int64, error) {
	q := r.db.Model(&entity.ExamSchedule{}).Where("deleted_at IS NULL")

	if filter.Search != "" {
		q = q.Where("name ILIKE ?", "%"+filter.Search+"%")
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.CreatedBy != nil {
		q = q.Where("created_by = ?", *filter.CreatedBy)
	}

	var total int64
	q.Count(&total)

	var schedules []*entity.ExamSchedule
	err := q.Preload("Rombels").Preload("Rombels.Rombel").
		Preload("Tags").Preload("Tags.Tag").
		Preload("Users").Preload("Users.User").
		Order("start_time DESC").Offset(p.Offset()).Limit(p.PerPage).Find(&schedules).Error
	return schedules, total, err
}

func (r *ExamScheduleRepo) SetQuestionBanks(scheduleID uint, banks []entity.ExamScheduleQuestionBank) error {
	if err := r.db.Where("exam_schedule_id = ?", scheduleID).Delete(&entity.ExamScheduleQuestionBank{}).Error; err != nil {
		return err
	}
	if len(banks) == 0 {
		return nil
	}
	return r.db.Create(&banks).Error
}

func (r *ExamScheduleRepo) SetRombels(scheduleID uint, rombelIDs []uint) error {
	if err := r.db.Where("exam_schedule_id = ?", scheduleID).Delete(&entity.ExamScheduleRombel{}).Error; err != nil {
		return err
	}
	if len(rombelIDs) == 0 {
		return nil
	}
	// Validate that none of the rombels are soft-deleted
	var activeCount int64
	r.db.Model(&entity.Rombel{}).Where("id IN ? AND deleted_at IS NULL", rombelIDs).Count(&activeCount)
	if int(activeCount) != len(rombelIDs) {
		return errors.New("satu atau lebih rombel tidak valid atau sudah dihapus")
	}
	rows := make([]entity.ExamScheduleRombel, len(rombelIDs))
	for i, id := range rombelIDs {
		rows[i] = entity.ExamScheduleRombel{ExamScheduleID: scheduleID, RombelID: id}
	}
	return r.db.Create(&rows).Error
}

func (r *ExamScheduleRepo) SetTags(scheduleID uint, tagIDs []uint) error {
	if err := r.db.Where("exam_schedule_id = ?", scheduleID).Delete(&entity.ExamScheduleTag{}).Error; err != nil {
		return err
	}
	if len(tagIDs) == 0 {
		return nil
	}
	rows := make([]entity.ExamScheduleTag, len(tagIDs))
	for i, id := range tagIDs {
		rows[i] = entity.ExamScheduleTag{ExamScheduleID: scheduleID, TagID: id}
	}
	return r.db.Create(&rows).Error
}

func (r *ExamScheduleRepo) SetExamRooms(scheduleID uint, rooms []entity.ExamScheduleRoom) error {
	if err := r.db.Where("exam_schedule_id = ?", scheduleID).Delete(&entity.ExamScheduleRoom{}).Error; err != nil {
		return err
	}
	if len(rooms) == 0 {
		return nil
	}
	return r.db.Create(&rooms).Error
}

func (r *ExamScheduleRepo) SetUsers(scheduleID uint, users []entity.ExamScheduleUser) error {
	if err := r.db.Where("exam_schedule_id = ?", scheduleID).Delete(&entity.ExamScheduleUser{}).Error; err != nil {
		return err
	}
	if len(users) == 0 {
		return nil
	}
	return r.db.Create(&users).Error
}

func (r *ExamScheduleRepo) GetUsersBySchedule(scheduleID uint) ([]entity.ExamScheduleUser, error) {
	var users []entity.ExamScheduleUser
	err := r.db.Where("exam_schedule_id = ?", scheduleID).Preload("User").Find(&users).Error
	return users, err
}
