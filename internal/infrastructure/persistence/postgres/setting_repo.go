package postgres

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettingRepo struct{ db *gorm.DB }

func NewSettingRepository(db *gorm.DB) *SettingRepo { return &SettingRepo{db: db} }

func (r *SettingRepo) GetAll() ([]*entity.Setting, error) {
	var settings []*entity.Setting
	err := r.db.Find(&settings).Error
	return settings, err
}

func (r *SettingRepo) GetByKey(key string) (*entity.Setting, error) {
	var setting entity.Setting
	err := r.db.Where("key = ?", key).First(&setting).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &setting, err
}

func (r *SettingRepo) Set(key, value string) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&entity.Setting{Key: key, Value: &value}).Error
}

func (r *SettingRepo) SetMultiple(pairs map[string]string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for k, v := range pairs {
			val := v
			err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}},
				DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
			}).Create(&entity.Setting{Key: k, Value: &val}).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}
