package postgres

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
	"gorm.io/gorm"
)

type TagRepo struct{ db *gorm.DB }

func NewTagRepository(db *gorm.DB) *TagRepo { return &TagRepo{db: db} }

func (r *TagRepo) Create(tag *entity.Tag) error {
	return r.db.Create(tag).Error
}

func (r *TagRepo) FindByID(id uint) (*entity.Tag, error) {
	var tag entity.Tag
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&tag).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &tag, err
}

func (r *TagRepo) Update(tag *entity.Tag) error {
	return r.db.Save(tag).Error
}

func (r *TagRepo) Delete(id uint) error {
	return r.db.Model(&entity.Tag{}).Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *TagRepo) List(search string, p pagination.Params) ([]*entity.TagWithCount, int64, error) {
	var tags []*entity.TagWithCount
	var total int64

	q := r.db.Model(&entity.Tag{}).Where("tags.deleted_at IS NULL")
	if search != "" {
		q = q.Where("name ILIKE ?", "%"+search+"%")
	}

	q.Count(&total)
	err := q.Select(`tags.*,
		(SELECT COUNT(*) FROM user_tags WHERE user_tags.tag_id = tags.id) as users_count,
		(SELECT COUNT(*) FROM exam_schedule_tags WHERE exam_schedule_tags.tag_id = tags.id) as exam_schedules_count`).
		Offset(p.Offset()).Limit(p.PerPage).Order("name ASC").Find(&tags).Error
	return tags, total, err
}

func (r *TagRepo) CountUsage(tagID uint) (int64, error) {
	var count int64
	err := r.db.Raw(`SELECT
		(SELECT COUNT(*) FROM user_tags WHERE tag_id = ?) +
		(SELECT COUNT(*) FROM exam_schedule_tags WHERE tag_id = ?)`, tagID, tagID).Scan(&count).Error
	return count, err
}

func (r *TagRepo) ListAll() ([]*entity.Tag, error) {
	var tags []*entity.Tag
	err := r.db.Where("deleted_at IS NULL").Order("name ASC").Find(&tags).Error
	return tags, err
}

func (r *TagRepo) AssignUsers(tagID uint, userIDs []uint) error {
	tag := entity.Tag{ID: tagID}
	users := make([]entity.User, len(userIDs))
	for i, id := range userIDs {
		users[i] = entity.User{ID: id}
	}
	return r.db.Model(&tag).Association("Users").Append(users)
}

func (r *TagRepo) BulkDelete(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Model(&entity.Tag{}).Where("id IN ?", ids).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *TagRepo) RemoveUsers(tagID uint, userIDs []uint) error {
	tag := entity.Tag{ID: tagID}
	users := make([]entity.User, len(userIDs))
	for i, id := range userIDs {
		users[i] = entity.User{ID: id}
	}
	return r.db.Model(&tag).Association("Users").Delete(users)
}
