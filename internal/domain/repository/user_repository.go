package repository

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/pkg/pagination"
)

type UserListFilter struct {
	Search   string
	Role     string
	RombelID *uint
}

type UserRepository interface {
	Create(user *entity.User) error
	CreateInTx(tx interface{}, user *entity.User) error
	BeginTx() (interface{}, error)
	CommitTx(tx interface{}) error
	RollbackTx(tx interface{})
	FindByID(id uint) (*entity.User, error)
	FindByUsername(username string) (*entity.User, error)
	FindByUsernameOrEmail(login string) (*entity.User, error)
	Update(user *entity.User) error
	Delete(id uint) error
	Restore(id uint) error
	ForceDelete(id uint) error
	UpdateLastLogin(id uint) error
	UpdateLoginToken(id uint, token string) error
	UpdateAvatar(id uint, path string) error
	List(filter UserListFilter, p pagination.Params) ([]*entity.User, int64, error)
	ListTrashed(filter UserListFilter, p pagination.Params) ([]*entity.User, int64, error)
	BulkCreate(users []*entity.User) error
	FindByEmail(email string) (*entity.User, error)
	FindExistingUsernames(usernames []string) ([]string, error)
	FindExistingEmails(emails []string) ([]string, error)
	FindExistingNIS(nisList []string) ([]string, error)
	FindExistingNIP(nipList []string) ([]string, error)
	BulkDelete(ids []uint) error
	BulkRestore(ids []uint) error
	BulkForceDelete(ids []uint) error
}
