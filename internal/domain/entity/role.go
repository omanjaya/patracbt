package entity

import "time"

type Role struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Name      string     `gorm:"uniqueIndex;not null" json:"name"`
	GuardName string     `gorm:"default:'web'" json:"guard_name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

// RoleWithCount embeds Role and adds a user count field.
type RoleWithCount struct {
	Role
	UsersCount int64 `json:"users_count"`
}
