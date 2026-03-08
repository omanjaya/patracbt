package repository

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
)

type PermissionListFilter struct {
	Search    string
	GroupName string
}

type PermissionRepository interface {
	Create(p *entity.Permission) error
	FindByID(id uint) (*entity.Permission, error)
	Update(p *entity.Permission) error
	Delete(id uint) error

	List(filter PermissionListFilter, p pagination.Params) ([]*entity.Permission, int64, error)
	ListAll() ([]*entity.Permission, error)
	ListGroups() ([]string, error)

	// User-Permission assignment
	AssignToUsers(permissionID uint, userIDs []uint) error
	RemoveFromUsers(permissionID uint, userIDs []uint) error

	// List users with their permissions (for UserPermissionsPage)
	ListUsersWithPermissions(filter UserPermissionListFilter, p pagination.Params) ([]*UserWithPermissions, int64, error)
}

type UserPermissionListFilter struct {
	Search         string
	PermissionID   *uint
	NoPermissionID *uint
}

type UserWithPermissions struct {
	ID          uint
	Name        string
	Username    string
	NIS         *string
	Rombel      *string
	DeletedAt   interface{}
	Permissions []entity.Permission
}
