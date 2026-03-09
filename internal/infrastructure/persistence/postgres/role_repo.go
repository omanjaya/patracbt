package postgres

import (
	"errors"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
	"gorm.io/gorm"
)

type roleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) repository.RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) List(search string, p pagination.Params) ([]*entity.RoleWithCount, int64, error) {
	var roles []*entity.RoleWithCount
	var total int64

	q := r.db.Model(&entity.Role{})
	if search != "" {
		q = q.Where("name ILIKE ?", "%"+search+"%")
	}

	q.Count(&total)

	err := q.Select(`roles.*,
		(SELECT COUNT(*) FROM users WHERE LOWER(users.role) = LOWER(roles.name) AND users.deleted_at IS NULL) as users_count`).
		Order("id desc").
		Offset(p.Offset()).
		Limit(p.PerPage).
		Find(&roles).Error

	return roles, total, err
}

func (r *roleRepository) CountUsers(roleID uint) (int64, error) {
	var role entity.Role
	if err := r.db.First(&role, roleID).Error; err != nil {
		return 0, err
	}
	var count int64
	err := r.db.Model(&entity.User{}).Where("LOWER(role) = LOWER(?) AND deleted_at IS NULL", role.Name).Count(&count).Error
	return count, err
}

func (r *roleRepository) GetByID(id uint) (*entity.Role, error) {
	var role entity.Role
	err := r.db.Preload("Permissions").First(&role, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("role tidak ditemukan")
		}
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) Create(role *entity.Role) error {
	return r.db.Create(role).Error
}

func (r *roleRepository) Update(role *entity.Role) error {
	return r.db.Save(role).Error
}

func (r *roleRepository) Delete(id uint) error {
	return r.db.Delete(&entity.Role{}, id).Error
}

func (r *roleRepository) GetPermissionsByRole(roleID uint) ([]entity.Permission, error) {
	var role entity.Role
	err := r.db.Preload("Permissions").First(&role, roleID).Error
	if err != nil {
		return nil, err
	}
	return role.Permissions, nil
}

func (r *roleRepository) AssignPermissions(roleID uint, permissionIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var role entity.Role
		if err := tx.First(&role, roleID).Error; err != nil {
			return err
		}

		var perms []entity.Permission
		if len(permissionIDs) > 0 {
			if err := tx.Find(&perms, permissionIDs).Error; err != nil {
				return err
			}
		}

		return tx.Model(&role).Association("Permissions").Replace(perms)
	})
}
