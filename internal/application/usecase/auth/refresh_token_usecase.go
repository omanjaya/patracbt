package auth

import (
	"errors"

	"github.com/omanjaya/patra/config"
	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/jwt"
)

type RefreshTokenUseCase struct {
	userRepo repository.UserRepository
	cfg      *config.Config
}

func NewRefreshTokenUseCase(userRepo repository.UserRepository, cfg *config.Config) *RefreshTokenUseCase {
	return &RefreshTokenUseCase{userRepo: userRepo, cfg: cfg}
}

func (uc *RefreshTokenUseCase) Execute(req dto.RefreshRequest) (*dto.RefreshResponse, error) {
	claims, err := jwt.ValidateToken(req.RefreshToken, uc.cfg.JWT.RefreshSecret)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := uc.userRepo.FindByID(claims.UserID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	// Validate that the refresh token's LoginToken matches the user's current LoginToken.
	// If they don't match, this session was invalidated by a newer login.
	if claims.LoginToken != "" {
		if user.LoginToken == nil || *user.LoginToken != claims.LoginToken {
			return nil, errors.New("sesi Anda telah berakhir")
		}
	}

	// Carry forward the current login_token so single-session middleware works
	var currentLoginToken string
	if user.LoginToken != nil {
		currentLoginToken = *user.LoginToken
	}

	tokenPair, err := jwt.GenerateTokenPair(
		user.ID, user.Username, user.Role,
		uc.cfg.JWT.AccessSecret, uc.cfg.JWT.RefreshSecret,
		uc.cfg.JWT.AccessTTL, uc.cfg.JWT.RefreshTTL,
		currentLoginToken,
	)
	if err != nil {
		return nil, err
	}

	return &dto.RefreshResponse{
		AccessToken: tokenPair.AccessToken,
		ExpiresIn:   tokenPair.ExpiresIn,
	}, nil
}
