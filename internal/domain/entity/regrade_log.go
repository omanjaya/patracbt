package entity

import (
	"time"

	"github.com/omanjaya/patra/pkg/types"
)

type RegradeLog struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	ExamScheduleID uint       `gorm:"not null;index" json:"exam_schedule_id"`
	RequestedBy    uint       `gorm:"not null" json:"requested_by"`
	SessionsCount  int        `gorm:"default:0" json:"sessions_count"`
	ScoreChanges   types.JSON `gorm:"type:jsonb" json:"score_changes"` // []ScoreChange
	CreatedAt      time.Time  `json:"created_at"`
}

// ScoreChange represents a single score change during regrade (serialized to JSON).
type ScoreChange struct {
	SessionID uint    `json:"session_id"`
	OldScore  float64 `json:"old_score"`
	NewScore  float64 `json:"new_score"`
}
