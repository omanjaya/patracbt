package postgres

import (
	"errors"
	"time"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(user *entity.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepo) CreateInTx(tx interface{}, user *entity.User) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok {
		return errors.New("invalid transaction type")
	}
	return gormTx.Create(user).Error
}

func (r *UserRepo) BeginTx() (interface{}, error) {
	tx := r.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tx, nil
}

func (r *UserRepo) CommitTx(tx interface{}) error {
	gormTx, ok := tx.(*gorm.DB)
	if !ok {
		return errors.New("invalid transaction type")
	}
	return gormTx.Commit().Error
}

func (r *UserRepo) RollbackTx(tx interface{}) {
	if gormTx, ok := tx.(*gorm.DB); ok {
		gormTx.Rollback()
	}
}

func (r *UserRepo) FindByID(id uint) (*entity.User, error) {
	var user entity.User
	err := r.db.Preload("Profile").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepo) FindByUsername(username string) (*entity.User, error) {
	var user entity.User
	err := r.db.Preload("Profile").
		Where("username = ? AND deleted_at IS NULL", username).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepo) FindByUsernameOrEmail(login string) (*entity.User, error) {
	var user entity.User
	err := r.db.Preload("Profile").
		Where("(username = ? OR email = ?) AND deleted_at IS NULL", login, login).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepo) Update(user *entity.User) error {
	return r.db.Model(user).Updates(map[string]interface{}{
		"name":       user.Name,
		"username":   user.Username,
		"email":      user.Email,
		"password":   user.Password,
		"role":       user.Role,
		"avatar_path": user.AvatarPath,
		"updated_at": user.UpdatedAt,
	}).Error
}

func (r *UserRepo) UpdateLastLogin(id uint) error {
	now := time.Now()
	return r.db.Model(&entity.User{}).Where("id = ?", id).Update("last_login_at", now).Error
}

func (r *UserRepo) UpdateLoginToken(id uint, token string) error {
	return r.db.Model(&entity.User{}).Where("id = ?", id).Update("login_token", token).Error
}

func (r *UserRepo) Delete(id uint) error {
	return r.db.Model(&entity.User{}).Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *UserRepo) Restore(id uint) error {
	return r.db.Model(&entity.User{}).Where("id = ?", id).Update("deleted_at", nil).Error
}

func (r *UserRepo) ForceDelete(id uint) error {
	return r.db.Unscoped().Delete(&entity.User{}, id).Error
}

func (r *UserRepo) UpdateAvatar(id uint, path string) error {
	return r.db.Model(&entity.User{}).Where("id = ?", id).Update("avatar_path", path).Error
}

func (r *UserRepo) ListTrashed(filter repository.UserListFilter, p pagination.Params) ([]*entity.User, int64, error) {
	var users []*entity.User
	var total int64
	q := r.db.Model(&entity.User{}).Preload("Profile").Where("deleted_at IS NOT NULL")
	if filter.Search != "" {
		q = q.Where("name ILIKE ? OR username ILIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}
	if filter.Role != "" {
		q = q.Where("role = ?", filter.Role)
	}
	q.Count(&total)
	err := q.Offset(p.Offset()).Limit(p.PerPage).Order("deleted_at DESC").Find(&users).Error
	return users, total, err
}

func (r *UserRepo) BulkDelete(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Model(&entity.User{}).Where("id IN ?", ids).Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *UserRepo) BulkRestore(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Model(&entity.User{}).Where("id IN ?", ids).Update("deleted_at", nil).Error
}

func (r *UserRepo) BulkForceDelete(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Unscoped().Delete(&entity.User{}, ids).Error
}

func (r *UserRepo) List(filter repository.UserListFilter, p pagination.Params) ([]*entity.User, int64, error) {
	var users []*entity.User
	var total int64

	q := r.db.Model(&entity.User{}).Preload("Profile").Where("deleted_at IS NULL")
	if filter.Search != "" {
		q = q.Where("name ILIKE ? OR username ILIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}
	if filter.Role != "" {
		q = q.Where("role = ?", filter.Role)
	}
	if filter.RombelID != nil {
		q = q.Joins("JOIN user_rombels ON user_rombels.user_id = users.id").
			Where("user_rombels.rombel_id = ?", *filter.RombelID)
	}

	q.Count(&total)
	err := q.Offset(p.Offset()).Limit(p.PerPage).Order("name ASC").Find(&users).Error
	return users, total, err
}

func (r *UserRepo) FindByEmail(email string) (*entity.User, error) {
	var user entity.User
	err := r.db.Preload("Profile").
		Where("email = ? AND deleted_at IS NULL", email).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepo) FindExistingUsernames(usernames []string) ([]string, error) {
	if len(usernames) == 0 {
		return nil, nil
	}
	var existing []string
	// Check ALL usernames including soft-deleted to prevent conflicts on restore
	err := r.db.Model(&entity.User{}).
		Where("username IN ?", usernames).
		Pluck("username", &existing).Error
	return existing, err
}

func (r *UserRepo) FindExistingEmails(emails []string) ([]string, error) {
	if len(emails) == 0 {
		return nil, nil
	}
	var existing []string
	// Check ALL emails including soft-deleted to prevent conflicts on restore
	err := r.db.Model(&entity.User{}).
		Where("email IN ?", emails).
		Pluck("email", &existing).Error
	return existing, err
}

func (r *UserRepo) FindExistingNIS(nisList []string) ([]string, error) {
	if len(nisList) == 0 {
		return nil, nil
	}
	var existing []string
	err := r.db.Model(&entity.UserProfile{}).
		Where("nis IN ?", nisList).
		Pluck("nis", &existing).Error
	return existing, err
}

func (r *UserRepo) FindExistingNIP(nipList []string) ([]string, error) {
	if len(nipList) == 0 {
		return nil, nil
	}
	var existing []string
	err := r.db.Model(&entity.UserProfile{}).
		Where("nip IN ?", nipList).
		Pluck("nip", &existing).Error
	return existing, err
}

func (r *UserRepo) BulkCreate(users []*entity.User) error {
	return r.db.CreateInBatches(users, 100).Error
}
