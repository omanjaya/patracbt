package postgres

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"gorm.io/gorm"
)

type AuditLogRepo struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

func (r *AuditLogRepo) Create(log *entity.AuditLog) error {
	return r.db.Create(log).Error
}

func (r *AuditLogRepo) ListByUser(userID uint, page, perPage int) ([]entity.AuditLog, int64, error) {
	var total int64
	r.db.Model(&entity.AuditLog{}).Where("user_id = ?", userID).Count(&total)

	var logs []entity.AuditLog
	offset := (page - 1) * perPage
	err := r.db.Where("user_id = ?", userID).
		Preload("User").
		Order("created_at DESC").
		Offset(offset).Limit(perPage).
		Find(&logs).Error
	return logs, total, err
}

func (r *AuditLogRepo) ListAll(page, perPage int) ([]entity.AuditLog, int64, error) {
	var total int64
	r.db.Model(&entity.AuditLog{}).Count(&total)

	var logs []entity.AuditLog
	offset := (page - 1) * perPage
	err := r.db.
		Preload("User").
		Order("created_at DESC").
		Offset(offset).Limit(perPage).
		Find(&logs).Error
	return logs, total, err
}
