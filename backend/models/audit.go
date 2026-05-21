package models

import "time"

type AuditLog struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint64    `gorm:"index;not null" json:"user_id"`
	Username   string    `gorm:"size:128" json:"username"`
	Action     string    `gorm:"size:64;not null" json:"action"`
	Resource   string    `gorm:"size:256" json:"resource"`
	ResourceID string    `gorm:"size:64" json:"resource_id"`
	IP         string    `gorm:"size:64" json:"ip"`
	CreatedAt  time.Time `json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_log"
}
