package master

import (
	"errors"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
)

var ErrPermissionNotFound = errors.New("permission tidak ditemukan")

type PermissionUseCase struct {
	repo repository.PermissionRepository
}

func NewPermissionUseCase(repo repository.PermissionRepository) *PermissionUseCase {
	return &PermissionUseCase{repo: repo}
}

func (uc *PermissionUseCase) List(filter repository.PermissionListFilter, p pagination.Params) ([]*entity.Permission, int64, error) {
	return uc.repo.List(filter, p)
}

func (uc *PermissionUseCase) ListAll() ([]*entity.Permission, error) {
	return uc.repo.ListAll()
}

func (uc *PermissionUseCase) ListGroups() ([]string, error) {
	return uc.repo.ListGroups()
}

func (uc *PermissionUseCase) Create(name, groupName string, description *string) (*entity.Permission, error) {
	p := &entity.Permission{
		Name:        name,
		GroupName:   groupName,
		Description: description,
	}
	return p, uc.repo.Create(p)
}

func (uc *PermissionUseCase) Update(id uint, name, groupName string, description *string) (*entity.Permission, error) {
	p, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPermissionNotFound
	}
	p.Name = name
	p.GroupName = groupName
	p.Description = description
	return p, uc.repo.Update(p)
}

func (uc *PermissionUseCase) Delete(id uint) error {
	p, err := uc.repo.FindByID(id)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrPermissionNotFound
	}
	return uc.repo.Delete(id)
}

func (uc *PermissionUseCase) AssignToUsers(permissionID uint, userIDs []uint) error {
	return uc.repo.AssignToUsers(permissionID, userIDs)
}

func (uc *PermissionUseCase) RemoveFromUsers(permissionID uint, userIDs []uint) error {
	return uc.repo.RemoveFromUsers(permissionID, userIDs)
}

func (uc *PermissionUseCase) ListUsersWithPermissions(filter repository.UserPermissionListFilter, p pagination.Params) ([]*repository.UserWithPermissions, int64, error) {
	return uc.repo.ListUsersWithPermissions(filter, p)
}
