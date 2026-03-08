package entity

import "time"

// Permission represents a named permission/group that can be assigned to users.
// It mirrors the Spatie-like permission concept but is lightweight and stored in DB.
type Permission struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"uniqueIndex;not null" json:"name"`
	GroupName   string     `gorm:"not null;default:'General'" json:"group_name"`
	Description *string    `json:"description,omitempty"`
	DeletedAt   *time.Time `gorm:"index" json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	Users []User `gorm:"many2many:user_permissions;" json:"users,omitempty"`
}

func (Permission) TableName() string { return "permissions" }

type UserPermission struct {
	UserID       uint `gorm:"primaryKey" json:"user_id"`
	PermissionID uint `gorm:"primaryKey" json:"permission_id"`
}

func (UserPermission) TableName() string { return "user_permissions" }
