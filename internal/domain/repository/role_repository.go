package repository

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
)

type RoleRepository interface {
	List(search string, p pagination.Params) ([]entity.Role, int64, error)
	GetByID(id uint) (*entity.Role, error)
	Create(role *entity.Role) error
	Update(role *entity.Role) error
	Delete(id uint) error

	GetPermissionsByRole(roleID uint) ([]entity.Permission, error)
	AssignPermissions(roleID uint, permissionIDs []uint) error
}
