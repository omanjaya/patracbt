package master

import (
	"errors"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
)

var systemRoles = []string{"Super Admin", "Operator", "Guru", "Peserta", "Pengawas"}

var (
	ErrSystemRole  = errors.New("role sistem tidak bisa diubah atau dihapus")
	ErrRoleHasUsers = errors.New("role masih memiliki pengguna")
)

type RoleUseCase struct {
	repo repository.RoleRepository
}

func NewRoleUseCase(repo repository.RoleRepository) *RoleUseCase {
	return &RoleUseCase{repo: repo}
}

func isSystemRole(name string) bool {
	for _, sr := range systemRoles {
		if sr == name {
			return true
		}
	}
	return false
}

func (uc *RoleUseCase) List(search string, p pagination.Params) ([]*entity.RoleWithCount, int64, error) {
	return uc.repo.List(search, p)
}

func (uc *RoleUseCase) Create(name, guardName string) (*entity.Role, error) {
	if name == "" {
		return nil, errors.New("nama role tidak boleh kosong")
	}
	role := &entity.Role{
		Name:      name,
		GuardName: guardName,
	}
	if err := uc.repo.Create(role); err != nil {
		return nil, err
	}
	return role, nil
}

func (uc *RoleUseCase) Update(id uint, name, guardName string) (*entity.Role, error) {
	role, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Prevent renaming system roles
	if isSystemRole(role.Name) {
		return nil, ErrSystemRole
	}

	if name != "" {
		role.Name = name
	}
	if guardName != "" {
		role.GuardName = guardName
	}
	if err := uc.repo.Update(role); err != nil {
		return nil, err
	}
	return role, nil
}

func (uc *RoleUseCase) Delete(id uint) error {
	role, err := uc.repo.GetByID(id)
	if err != nil {
		return err
	}

	// Prevent deleting system roles
	if isSystemRole(role.Name) {
		return ErrSystemRole
	}

	// Check if role has users
	count, err := uc.repo.CountUsers(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrRoleHasUsers
	}

	return uc.repo.Delete(id)
}

func (uc *RoleUseCase) GetRolePermissions(roleID uint) ([]entity.Permission, error) {
	return uc.repo.GetPermissionsByRole(roleID)
}

func (uc *RoleUseCase) AssignPermissions(roleID uint, permissionIDs []uint) error {
	return uc.repo.AssignPermissions(roleID, permissionIDs)
}
