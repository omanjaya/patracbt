package entity

import (
	"time"
)

type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `json:"user_id"`
	Action     string    `gorm:"size:100;not null" json:"action"`
	TargetID   uint      `json:"target_id"`
	TargetType string    `gorm:"size:50" json:"target_type"`
	IPAddress  string    `gorm:"size:45" json:"ip_address"`
	Details    string    `gorm:"type:text" json:"details"`
	CreatedAt  time.Time `json:"created_at"`

	// Relations (for display)
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (AuditLog) TableName() string { return "audit_logs" }
