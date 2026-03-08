package master

import (
	"errors"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
)

type RoleUseCase struct {
	repo repository.RoleRepository
}

func NewRoleUseCase(repo repository.RoleRepository) *RoleUseCase {
	return &RoleUseCase{repo: repo}
}

func (uc *RoleUseCase) List(search string, p pagination.Params) ([]entity.Role, int64, error) {
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
	return uc.repo.Delete(id) // will soft/hard delete
}

func (uc *RoleUseCase) GetRolePermissions(roleID uint) ([]entity.Permission, error) {
	return uc.repo.GetPermissionsByRole(roleID)
}

func (uc *RoleUseCase) AssignPermissions(roleID uint, permissionIDs []uint) error {
	return uc.repo.AssignPermissions(roleID, permissionIDs)
}
