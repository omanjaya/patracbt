package entity

import "time"

// ExamSupervision tracks supervision tokens per room per schedule.
type ExamSupervision struct {
	ID             uint      `gorm:"primaryKey"`
	ExamScheduleID uint      `gorm:"not null;uniqueIndex:idx_supervision_schedule_room"`
	RoomID         uint      `gorm:"not null;uniqueIndex:idx_supervision_schedule_room"`
	Token          string    `gorm:"size:10;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time

	ExamSchedule ExamSchedule `gorm:"foreignKey:ExamScheduleID"`
	Room         Room         `gorm:"foreignKey:RoomID"`
}
