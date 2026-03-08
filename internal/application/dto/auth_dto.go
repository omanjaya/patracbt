package dto

type LoginRequest struct {
	Login      string `json:"login" binding:"required,min=1,max=255"`
	Password   string `json:"password" binding:"required,min=1"`
	ForceLogin bool   `json:"force_login"`
}

type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
	User         UserResponse `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

type UserResponse struct {
	ID        uint            `json:"id"`
	Name      string          `json:"name"`
	Username  string          `json:"username"`
	Role      string          `json:"role"`
	AvatarURL *string         `json:"avatar_url"`
	Profile   *ProfileResponse `json:"profile,omitempty"`
}

type ProfileResponse struct {
	NIS   *string `json:"nis"`
	Class *string `json:"class"`
	Major *string `json:"major"`
}
