package entity

import (
	"time"

	"github.com/omanjaya/patra/pkg/types"
)

// ExamSession statuses
const (
	SessionStatusNotStarted = "not_started"
	SessionStatusOngoing    = "ongoing"
	SessionStatusFinished   = "finished"
	SessionStatusTerminated = "terminated"
)

type ExamSession struct {
	ID             uint       `gorm:"primaryKey"`
	// BUG-01 fix: composite unique index untuk mencegah double session (race condition)
	// Performance: composite indexes for common query patterns
	ExamScheduleID uint       `gorm:"not null;uniqueIndex:idx_session_user_schedule;index:idx_ses_schedule_status,composite:schedule;index:idx_ses_schedule_id"`
	UserID         uint       `gorm:"not null;uniqueIndex:idx_session_user_schedule;index:idx_ses_user_status,composite:user"`
	Status         string     `gorm:"default:'not_started';index:idx_ses_schedule_status,composite:status;index:idx_ses_user_status,composite:status"`
	StartTime      *time.Time
	EndTime        *time.Time
	FinishedAt     *time.Time
	QuestionOrder  types.JSON `gorm:"type:jsonb"` // []uint — ordered question IDs
	OptionOrder    types.JSON `gorm:"type:jsonb"` // {"question_id": [shuffled_indices], ...}
	Score          float64    `gorm:"default:0"`
	MaxScore       float64    `gorm:"default:0"`
	ViolationCount int        `gorm:"default:0"`
	ExtraTime      int        `gorm:"default:0"` // extra minutes added by supervisor
	SectionIndex   int        `gorm:"default:0"` // multi-stage: current section
	CreatedAt      time.Time
	UpdatedAt      time.Time

	ExamSchedule ExamSchedule `gorm:"foreignKey:ExamScheduleID"`
	User         User         `gorm:"foreignKey:UserID"`
	Answers      []ExamAnswer `gorm:"foreignKey:ExamSessionID"`
}

type ExamAnswer struct {
	ID            uint       `gorm:"primaryKey"`
	// BUG-04 fix: composite unique constraint agar ON CONFLICT (exam_session_id, question_id) bekerja
	// Performance: standalone index on exam_session_id for lookup queries
	ExamSessionID uint       `gorm:"not null;uniqueIndex:idx_answer_session_question;index:idx_exam_answers_session_id"`
	QuestionID    uint       `gorm:"not null;uniqueIndex:idx_answer_session_question"`
	Answer        types.JSON `gorm:"type:jsonb"`
	IsFlagged     bool       `gorm:"default:false"`
	AnsweredAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ViolationLog struct {
	ID            uint      `gorm:"primaryKey"`
	ExamSessionID uint      `gorm:"not null;index"`
	ViolationType string    `gorm:"not null"` // tab_switch, copy_paste, etc.
	Description   string
	CreatedAt     time.Time
}
