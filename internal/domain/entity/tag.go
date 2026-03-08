package entity

import "time"

type Tag struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Name      string     `gorm:"not null" json:"name"`
	Color     string     `gorm:"default:'#6B7280'" json:"color"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	Users []User `gorm:"many2many:user_tags;" json:"users,omitempty"`
}

func (Tag) TableName() string { return "tags" }

type UserTag struct {
	UserID uint `gorm:"primaryKey" json:"user_id"`
	TagID  uint `gorm:"primaryKey" json:"tag_id"`
}

func (UserTag) TableName() string { return "user_tags" }
