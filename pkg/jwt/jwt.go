package jwt

import (
	"errors"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenExpired = errors.New("token sudah expired")
	ErrTokenInvalid = errors.New("token tidak valid")
)

type Claims struct {
	UserID     uint   `json:"user_id"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	LoginToken string `json:"login_token,omitempty"`
	jwtgo.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func GenerateTokenPair(userID uint, username, role, accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration, opts ...string) (*TokenPair, error) {
	now := time.Now()

	// Optional login token for single-session enforcement
	var loginToken string
	if len(opts) > 0 {
		loginToken = opts[0]
	}

	accessClaims := &Claims{
		UserID:     userID,
		Username:   username,
		Role:       role,
		LoginToken: loginToken,
		RegisteredClaims: jwtgo.RegisteredClaims{
			ExpiresAt: jwtgo.NewNumericDate(now.Add(accessTTL)),
			IssuedAt:  jwtgo.NewNumericDate(now),
		},
	}

	accessToken, err := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, accessClaims).SignedString([]byte(accessSecret))
	if err != nil {
		return nil, err
	}

	refreshClaims := &Claims{
		UserID:     userID,
		Username:   username,
		Role:       role,
		LoginToken: loginToken,
		RegisteredClaims: jwtgo.RegisteredClaims{
			ExpiresAt: jwtgo.NewNumericDate(now.Add(refreshTTL)),
			IssuedAt:  jwtgo.NewNumericDate(now),
		},
	}

	refreshToken, err := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, refreshClaims).SignedString([]byte(refreshSecret))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(accessTTL.Seconds()),
	}, nil
}

func ValidateToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwtgo.ParseWithClaims(tokenStr, &Claims{}, func(t *jwtgo.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtgo.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwtgo.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
