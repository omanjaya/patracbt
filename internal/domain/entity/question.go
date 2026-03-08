package entity

import (
	"time"

	"github.com/omanjaya/patra/pkg/types"
)

// Question types
const (
	QuestionTypePG          = "pg"
	QuestionTypePGK         = "pgk"
	QuestionTypeBenarSalah  = "benar_salah"
	QuestionTypeMenjodohkan = "menjodohkan"
	QuestionTypeIsian       = "isian_singkat"
	QuestionTypeMatrix      = "matrix"
	QuestionTypeEsai        = "esai"
)

// Difficulty levels
const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

type Question struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	QuestionBankID uint       `gorm:"not null;index" json:"question_bank_id"`
	StimulusID     *uint      `gorm:"index" json:"stimulus_id"`
	QuestionType   string     `gorm:"not null" json:"question_type"`
	Body           string     `gorm:"type:text;not null" json:"body"`
	Score          float64    `gorm:"default:1" json:"score"`
	Difficulty     string     `gorm:"default:'medium'" json:"difficulty"`
	Options        types.JSON `gorm:"type:jsonb" json:"options"`
	CorrectAnswer  types.JSON `gorm:"type:jsonb" json:"correct_answer"`
	AudioPath      *string    `gorm:"size:255" json:"audio_path"`
	AudioLimit     int        `gorm:"default:2" json:"audio_limit"`
	BloomLevel     int        `gorm:"default:0" json:"bloom_level"`
	TopicCode      string     `gorm:"size:100" json:"topic_code"`
	OrderIndex     int        `gorm:"default:0" json:"order_index"`
	DeletedAt      *time.Time `gorm:"index" json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
