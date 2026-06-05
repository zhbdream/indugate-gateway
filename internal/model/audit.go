package model

import "time"

type AuditLog struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	Username   string    `gorm:"size:64;index" json:"username"`
	Role       UserRole  `gorm:"size:32" json:"role"`
	Method     string    `gorm:"size:16;not null" json:"method"`
	Path       string    `gorm:"size:512;not null" json:"path"`
	Action     string    `gorm:"size:128;index" json:"action"`
	Detail     string    `gorm:"size:512" json:"detail"`
	ClientIP   string    `gorm:"size:64" json:"client_ip"`
	StatusCode int       `json:"status_code"`
	Success    bool      `json:"success"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }
