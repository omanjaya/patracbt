package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/omanjaya/patra/config"
	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/bcrypt"
	"github.com/omanjaya/patra/pkg/jwt"
	"github.com/omanjaya/patra/pkg/logger"
)

var (
	ErrInvalidCredentials = errors.New("username atau password salah")
	ErrUserNotFound       = errors.New("user tidak ditemukan")
	ErrUserInactive       = errors.New("akun Anda tidak aktif, hubungi administrator")
	ErrSessionExists      = errors.New("SESSION_EXISTS")
	ErrExamInProgress     = errors.New("EXAM_IN_PROGRESS")
)

const authCacheTTL = 1 * time.Hour

type LoginUseCase struct {
	userRepo        repository.UserRepository
	examSessionRepo repository.ExamSessionRepository
	cfg             *config.Config
	rdb             *goredis.Client // optional, nil = skip caching
}

func NewLoginUseCase(userRepo repository.UserRepository, examSessionRepo repository.ExamSessionRepository, cfg *config.Config, rdb *goredis.Client) *LoginUseCase {
	return &LoginUseCase{userRepo: userRepo, examSessionRepo: examSessionRepo, cfg: cfg, rdb: rdb}
}

// authCacheKey returns the Redis key for auth caching.
func authCacheKey(username string) string {
	return fmt.Sprintf("auth_cache:%s", username)
}

// cachedUser is the serialized user data stored in Redis.
type cachedUser struct {
	ID         uint    `json:"id"`
	Name       string  `json:"name"`
	Username   string  `json:"username"`
	Email      *string `json:"email,omitempty"`
	Password   string  `json:"password"`
	Role       string  `json:"role"`
	IsActive   bool    `json:"is_active"`
	AvatarPath *string `json:"avatar_path,omitempty"`
	LoginToken *string `json:"login_token,omitempty"`
}

// getUserFromCache attempts to load user from Redis cache. Returns nil if not found or error.
func (uc *LoginUseCase) getUserFromCache(login string) *entity.User {
	if uc.rdb == nil {
		return nil
	}

	ctx := context.Background()
	data, err := uc.rdb.Get(ctx, authCacheKey(login)).Bytes()
	if err != nil {
		return nil
	}

	var cu cachedUser
	if err := json.Unmarshal(data, &cu); err != nil {
		return nil
	}

	return &entity.User{
		ID:         cu.ID,
		Name:       cu.Name,
		Username:   cu.Username,
		Email:      cu.Email,
		Password:   cu.Password,
		Role:       cu.Role,
		IsActive:   cu.IsActive,
		AvatarPath: cu.AvatarPath,
		LoginToken: cu.LoginToken,
	}
}

// cacheUser stores user data in Redis for faster subsequent logins.
func (uc *LoginUseCase) cacheUser(user *entity.User) {
	if uc.rdb == nil {
		return
	}

	cu := cachedUser{
		ID:         user.ID,
		Name:       user.Name,
		Username:   user.Username,
		Email:      user.Email,
		Password:   user.Password,
		Role:       user.Role,
		IsActive:   user.IsActive,
		AvatarPath: user.AvatarPath,
		LoginToken: user.LoginToken,
	}

	data, err := json.Marshal(cu)
	if err != nil {
		return
	}

	ctx := context.Background()
	if err := uc.rdb.Set(ctx, authCacheKey(user.Username), data, authCacheTTL).Err(); err != nil {
		logger.Log.Warnf("Redis auth cache write failed: %v", err)
	}
}

func (uc *LoginUseCase) Execute(req dto.LoginRequest) (*dto.LoginResponse, error) {
	// Try Redis cache first (graceful: if Redis fails, use DB)
	var user *entity.User
	if cached := uc.getUserFromCache(req.Login); cached != nil {
		// Validate password against cached hash
		if bcrypt.CheckPassword(req.Password, cached.Password) {
			user = cached
		}
		// If password mismatch with cache, fall through to DB
		// (user might have changed password)
	}

	// Fallback to DB if not found in cache or password mismatch
	if user == nil {
		var err error
		user, err = uc.userRepo.FindByUsernameOrEmail(req.Login)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, ErrInvalidCredentials
		}

		if !bcrypt.CheckPassword(req.Password, user.Password) {
			return nil, ErrInvalidCredentials
		}
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// For peserta: check duplicate session
	if user.IsPeserta() && user.LoginToken != nil && *user.LoginToken != "" && *user.LoginToken != "REVOKED" {
		if !req.ForceLogin {
			// Check if user has ongoing exam session
			ongoingSession, _ := uc.examSessionRepo.FindOngoingByUser(user.ID)
			if ongoingSession != nil {
				return nil, ErrExamInProgress
			}
			return nil, ErrSessionExists
		}
	}

	_ = uc.userRepo.UpdateLastLogin(user.ID)

	// Rotate login token — invalidates any previous session (single session enforcement)
	loginToken := uuid.New().String()
	_ = uc.userRepo.UpdateLoginToken(user.ID, loginToken)

	// Update cache with new login token
	user.LoginToken = &loginToken
	uc.cacheUser(user)

	tokenPair, err := jwt.GenerateTokenPair(
		user.ID, user.Username, user.Role,
		uc.cfg.JWT.AccessSecret, uc.cfg.JWT.RefreshSecret,
		uc.cfg.JWT.AccessTTL, uc.cfg.JWT.RefreshTTL,
		loginToken,
	)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		User:         mapUserToResponse(user),
	}, nil
}

func mapUserToResponse(user *entity.User) dto.UserResponse {
	resp := dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Username:  user.Username,
		Role:      user.Role,
		AvatarURL: user.AvatarPath,
	}

	if user.Profile != nil {
		resp.Profile = &dto.ProfileResponse{
			NIS:   user.Profile.NIS,
			Class: user.Profile.Class,
			Major: user.Profile.Major,
		}
	}

	return resp
}
