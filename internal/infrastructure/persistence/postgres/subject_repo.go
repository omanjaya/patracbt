package postgres

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
	"gorm.io/gorm"
)

type SubjectRepo struct{ db *gorm.DB }

func NewSubjectRepository(db *gorm.DB) *SubjectRepo { return &SubjectRepo{db: db} }

func (r *SubjectRepo) Create(subject *entity.Subject) error {
	return r.db.Create(subject).Error
}

func (r *SubjectRepo) FindByID(id uint) (*entity.Subject, error) {
	var subject entity.Subject
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&subject).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &subject, err
}

func (r *SubjectRepo) Update(subject *entity.Subject) error {
	return r.db.Save(subject).Error
}

func (r *SubjectRepo) Delete(id uint) error {
	return r.db.Model(&entity.Subject{}).Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *SubjectRepo) List(search string, p pagination.Params) ([]*entity.Subject, int64, error) {
	var subjects []*entity.Subject
	var total int64

	q := r.db.Model(&entity.Subject{}).Where("deleted_at IS NULL")
	if search != "" {
		q = q.Where("name ILIKE ? OR code ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	q.Count(&total)
	err := q.Offset(p.Offset()).Limit(p.PerPage).Order("name ASC").Find(&subjects).Error
	return subjects, total, err
}

func (r *SubjectRepo) ListAll() ([]*entity.Subject, error) {
	var subjects []*entity.Subject
	err := r.db.Where("deleted_at IS NULL").Order("name ASC").Find(&subjects).Error
	return subjects, err
}

func (r *SubjectRepo) BulkDelete(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Model(&entity.Subject{}).Where("id IN ?", ids).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}
