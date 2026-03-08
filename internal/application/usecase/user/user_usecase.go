package user

import (
	"errors"

	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	pkgbcrypt "github.com/omanjaya/patra/pkg/bcrypt"
	"github.com/omanjaya/patra/pkg/pagination"
)

var (
	ErrUserNotFound       = errors.New("user tidak ditemukan")
	ErrUsernameTaken      = errors.New("username sudah digunakan")
	ErrUserHasOngoingExam = errors.New("user tidak dapat dihapus karena memiliki ujian yang sedang berlangsung")
)

type UserUseCase struct {
	repo        repository.UserRepository
	sessionRepo repository.ExamSessionRepository
}

func NewUserUseCase(repo repository.UserRepository, sessionRepo repository.ExamSessionRepository) *UserUseCase {
	return &UserUseCase{repo: repo, sessionRepo: sessionRepo}
}

func (uc *UserUseCase) List(filter repository.UserListFilter, p pagination.Params) ([]*entity.User, int64, error) {
	return uc.repo.List(filter, p)
}

func (uc *UserUseCase) Create(req dto.CreateUserRequest) (*entity.User, error) {
	existing, err := uc.repo.FindByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameTaken
	}

	hashed, err := pkgbcrypt.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &entity.User{
		Name:     req.Name,
		Username: req.Username,
		Email:    req.Email,
		Password: hashed,
		Role:     req.Role,
	}

	if req.Profile != nil {
		user.Profile = &entity.UserProfile{
			NIS:   req.Profile.NIS,
			NIP:   req.Profile.NIP,
			Class: req.Profile.Class,
			Major: req.Profile.Major,
			Year:  req.Profile.Year,
			Phone: req.Profile.Phone,
		}
	}

	return user, uc.repo.Create(user)
}

func (uc *UserUseCase) Update(id uint, req dto.UpdateUserRequest) (*entity.User, error) {
	user, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	user.Name = req.Name
	user.Email = req.Email
	user.Role = req.Role

	if req.Password != nil && *req.Password != "" {
		hashed, err := pkgbcrypt.HashPassword(*req.Password)
		if err != nil {
			return nil, err
		}
		user.Password = hashed
	}

	if req.Profile != nil && user.Profile != nil {
		user.Profile.NIS = req.Profile.NIS
		user.Profile.NIP = req.Profile.NIP
		user.Profile.Class = req.Profile.Class
		user.Profile.Major = req.Profile.Major
		user.Profile.Year = req.Profile.Year
		user.Profile.Phone = req.Profile.Phone
	}

	return user, uc.repo.Update(user)
}

func (uc *UserUseCase) Delete(id uint) error {
	user, err := uc.repo.FindByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	// Check if user has an ongoing exam session
	if uc.sessionRepo != nil {
		ongoing, err := uc.sessionRepo.FindOngoingByUser(id)
		if err == nil && ongoing != nil {
			return ErrUserHasOngoingExam
		}
	}
	return uc.repo.Delete(id)
}

func (uc *UserUseCase) Restore(id uint) error {
	return uc.repo.Restore(id)
}

func (uc *UserUseCase) ForceDelete(id uint) error {
	return uc.repo.ForceDelete(id)
}

func (uc *UserUseCase) ListTrashed(filter repository.UserListFilter, p pagination.Params) ([]*entity.User, int64, error) {
	return uc.repo.ListTrashed(filter, p)
}

func (uc *UserUseCase) BulkDelete(ids []uint) error {
	return uc.repo.BulkDelete(ids)
}

func (uc *UserUseCase) BulkRestore(ids []uint) error {
	return uc.repo.BulkRestore(ids)
}

func (uc *UserUseCase) BulkForceDelete(ids []uint) error {
	return uc.repo.BulkForceDelete(ids)
}

func (uc *UserUseCase) ImportExcel(data []byte) (*ImportResult, error) {
	return ImportUsersFromExcel(data, uc.repo)
}
