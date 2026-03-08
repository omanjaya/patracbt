package entity

import "time"

type Room struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Name      string     `gorm:"not null" json:"name"`
	Capacity  int        `gorm:"default:30" json:"capacity"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (Room) TableName() string { return "rooms" }
