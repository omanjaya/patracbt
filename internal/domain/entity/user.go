package entity

import (
	"time"
)

// Role constants to avoid magic strings.
const (
	RoleAdmin    = "admin"
	RoleGuru     = "guru"
	RolePengawas = "pengawas"
	RolePeserta  = "peserta"
)

type User struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"not null" json:"name"`
	Username    string     `gorm:"uniqueIndex:idx_users_username;not null" json:"username"`
	Email       *string    `gorm:"uniqueIndex:idx_users_email" json:"email,omitempty"`
	Password    string     `gorm:"not null" json:"-"`
	Role        string     `gorm:"not null;default:'peserta'" json:"role"`
	IsActive    bool       `gorm:"not null;default:true" json:"is_active"`
	AvatarPath  *string    `json:"avatar_path,omitempty"`
	ForcePasswordChange bool       `gorm:"not null;default:false" json:"force_password_change"`
	LoginToken          *string    `gorm:"index" json:"-"` // single-session enforcement
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	DeletedAt   *time.Time `gorm:"index" json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Relations
	Profile *UserProfile `gorm:"foreignKey:UserID" json:"profile,omitempty"`
}

func (User) TableName() string { return "users" }

func (u *User) IsAdmin() bool    { return u.Role == RoleAdmin }
func (u *User) IsGuru() bool     { return u.Role == RoleGuru }
func (u *User) IsPengawas() bool { return u.Role == RolePengawas }
func (u *User) IsPeserta() bool  { return u.Role == RolePeserta }

type UserProfile struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	NIS       *string   `json:"nis,omitempty"`
	NIP       *string   `json:"nip,omitempty"`
	Class     *string   `json:"class,omitempty"`
	Major     *string   `json:"major,omitempty"`
	Year      *int16    `json:"year,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	Address   *string   `json:"address,omitempty"`
	RombelID  *uint     `json:"rombel_id,omitempty" gorm:"index"`
	RoomID    *uint     `json:"room_id,omitempty" gorm:"index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UserProfile) TableName() string { return "user_profiles" }
