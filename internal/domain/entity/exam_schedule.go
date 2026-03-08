package entity

import "time"

// ExamSchedule statuses
const (
	ExamStatusDraft     = "draft"
	ExamStatusPublished = "published"
	ExamStatusActive    = "active"
	ExamStatusFinished  = "finished"
)

type ExamSchedule struct {
	ID                   uint       `gorm:"primaryKey"`
	Name                 string     `gorm:"not null"`
	Token                string     `gorm:"uniqueIndex;not null"`
	SupervisionToken     string     `gorm:"index"`
	StartTime            time.Time
	EndTime              time.Time
	DurationMinutes      int        `gorm:"not null;default:60"`
	Status               string     `gorm:"default:'draft'"`
	AllowSeeResult       bool       `gorm:"default:true"`
	MaxViolations        int        `gorm:"default:3"`
	RandomizeQuestions   bool       `gorm:"default:false"`
	RandomizeOptions     bool       `gorm:"default:false"`
	NextExamScheduleID   *uint      `gorm:"index"` // multi-stage: next section
	LatePolicy           string     `gorm:"default:'allow_full_time'"` // "deduct_time" or "allow_full_time"
	MinWorkingTime       int        `gorm:"default:0"`                 // minimum minutes before finish button appears
	DetectCheating       bool       `gorm:"default:true"`              // enable cheating detection
	CheatingLimit        int        `gorm:"default:0"`                 // max violations before auto-terminate (0 = unlimited)
	ShowScoreAfter       string     `gorm:"default:'immediately'"`     // "immediately", "after_end_time", "manual"
	LastGradedAt         *time.Time `json:"last_graded_at,omitempty"`
	CreatedBy            uint       `gorm:"index"`
	DeletedAt            *time.Time `gorm:"index"`
	CreatedAt            time.Time
	UpdatedAt            time.Time

	QuestionBanks  []ExamScheduleQuestionBank `gorm:"foreignKey:ExamScheduleID"`
	Rombels        []ExamScheduleRombel       `gorm:"foreignKey:ExamScheduleID"`
	Tags           []ExamScheduleTag          `gorm:"foreignKey:ExamScheduleID"`
	ExamRooms      []ExamScheduleRoom         `gorm:"foreignKey:ExamScheduleID"` // NEW: Supervision token per room
	Users          []ExamScheduleUser         `gorm:"foreignKey:ExamScheduleID" json:"users,omitempty"` // Whitelist/Blacklist users
}

type ExamScheduleQuestionBank struct {
	ID             uint         `gorm:"primaryKey"`
	ExamScheduleID uint         `gorm:"not null;index"`
	QuestionBankID uint         `gorm:"not null;index"`
	QuestionBank   QuestionBank `gorm:"foreignKey:QuestionBankID"`
	QuestionCount  int          `gorm:"default:0"` // 0 = all questions
	Weight         float64      `json:"weight" gorm:"default:1"`
}

type ExamScheduleRombel struct {
	ExamScheduleID uint   `gorm:"primaryKey"`
	RombelID       uint   `gorm:"primaryKey"`
	Rombel         Rombel `gorm:"foreignKey:RombelID"`
}

type ExamScheduleTag struct {
	ExamScheduleID uint `gorm:"primaryKey"`
	TagID          uint `gorm:"primaryKey"`
	Tag            Tag  `gorm:"foreignKey:TagID"`
}

// ExamScheduleUser represents individual user whitelist/blacklist for an exam schedule
type ExamScheduleUser struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	ExamScheduleID uint   `gorm:"not null;index" json:"exam_schedule_id"`
	UserID         uint   `gorm:"not null;index" json:"user_id"`
	Type           string `gorm:"size:10;not null;default:include" json:"type"` // "include" or "exclude"
	User           User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// ExamScheduleRoom represents a room assigned to an exam schedule with its specific supervision token
type ExamScheduleRoom struct {
	ExamScheduleID   uint   `gorm:"primaryKey"`
	RoomID           uint   `gorm:"primaryKey"`
	SupervisionToken string `gorm:"size:6;not null" json:"supervision_token"`
	Room             Room   `gorm:"foreignKey:RoomID"`
}
