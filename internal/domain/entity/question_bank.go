package entity

import "time"

type QuestionBank struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Name          string     `gorm:"not null" json:"name"`
	SubjectID     *uint      `gorm:"index" json:"subject_id"`
	Subject       *Subject   `gorm:"foreignKey:SubjectID" json:"subject,omitempty"`
	Description   string     `json:"description"`
	Status        string     `gorm:"default:'active'" json:"status"`
	CreatedBy     uint       `gorm:"index" json:"created_by"`
	QuestionCount int        `gorm:"-" json:"question_count"`
	IsLocked      bool       `gorm:"-" json:"is_locked"`
	DeletedAt     *time.Time `gorm:"index" json:"-"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Stimulus struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	QuestionBankID uint      `gorm:"not null;index" json:"question_bank_id"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
