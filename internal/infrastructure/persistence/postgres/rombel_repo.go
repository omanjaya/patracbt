package postgres

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
	"gorm.io/gorm"
)

type RombelRepo struct{ db *gorm.DB }

func NewRombelRepository(db *gorm.DB) *RombelRepo { return &RombelRepo{db: db} }

func (r *RombelRepo) Create(rombel *entity.Rombel) error {
	return r.db.Create(rombel).Error
}

func (r *RombelRepo) FindByID(id uint) (*entity.Rombel, error) {
	var rombel entity.Rombel
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&rombel).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &rombel, err
}

func (r *RombelRepo) Update(rombel *entity.Rombel) error {
	return r.db.Save(rombel).Error
}

func (r *RombelRepo) Delete(id uint) error {
	return r.db.Model(&entity.Rombel{}).Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *RombelRepo) List(search, gradeLevel string, p pagination.Params) ([]*entity.RombelWithCount, int64, error) {
	var rombels []*entity.RombelWithCount
	var total int64

	q := r.db.Model(&entity.Rombel{}).Where("rombels.deleted_at IS NULL")
	if search != "" {
		q = q.Where("name ILIKE ?", "%"+search+"%")
	}
	if gradeLevel != "" {
		q = q.Where("grade_level = ?", gradeLevel)
	}

	q.Count(&total)
	err := q.Select("rombels.*, (SELECT COUNT(*) FROM user_profiles WHERE user_profiles.rombel_id = rombels.id) as students_count").
		Offset(p.Offset()).Limit(p.PerPage).Order("name ASC").Find(&rombels).Error
	return rombels, total, err
}

func (r *RombelRepo) CountStudents(rombelID uint) (int64, error) {
	var count int64
	err := r.db.Model(&entity.UserProfile{}).Where("rombel_id = ?", rombelID).Count(&count).Error
	return count, err
}

func (r *RombelRepo) BulkDelete(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Model(&entity.Rombel{}).Where("id IN ?", ids).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *RombelRepo) AssignUsers(rombelID uint, userIDs []uint) error {
	rombel := entity.Rombel{ID: rombelID}
	users := make([]entity.User, len(userIDs))
	for i, id := range userIDs {
		users[i] = entity.User{ID: id}
	}
	return r.db.Model(&rombel).Association("Users").Append(users)
}

func (r *RombelRepo) RemoveUsers(rombelID uint, userIDs []uint) error {
	rombel := entity.Rombel{ID: rombelID}
	users := make([]entity.User, len(userIDs))
	for i, id := range userIDs {
		users[i] = entity.User{ID: id}
	}
	return r.db.Model(&rombel).Association("Users").Delete(users)
}
