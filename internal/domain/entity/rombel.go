package entity

import "time"

type Rombel struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"not null" json:"name"`
	GradeLevel  *string    `json:"grade_level,omitempty"`
	Description *string    `json:"description,omitempty"`
	DeletedAt   *time.Time `gorm:"index" json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	Users []User `gorm:"many2many:user_rombels;" json:"users,omitempty"`
}

func (Rombel) TableName() string { return "rombels" }

type UserRombel struct {
	UserID   uint `gorm:"primaryKey" json:"user_id"`
	RombelID uint `gorm:"primaryKey" json:"rombel_id"`
}

func (UserRombel) TableName() string { return "user_rombels" }
