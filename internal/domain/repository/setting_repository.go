package repository

import "github.com/omanjaya/patra/internal/domain/entity"

type SettingRepository interface {
	GetAll() ([]*entity.Setting, error)
	GetByKey(key string) (*entity.Setting, error)
	Set(key, value string) error
	SetMultiple(pairs map[string]string) error
}
