package entity

import "time"

type Subject struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Name      string     `gorm:"not null" json:"name"`
	Code      *string    `json:"code,omitempty"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (Subject) TableName() string { return "subjects" }

// SubjectWithCount embeds Subject and adds usage count fields.
type SubjectWithCount struct {
	Subject
	QuestionBanksCount int64 `json:"question_banks_count"`
}
