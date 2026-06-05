package model

import "time"

type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "INFO"
	AlertLevelWarn     AlertLevel = "WARN"
	AlertLevelError    AlertLevel = "ERROR"
	AlertLevelCritical AlertLevel = "CRITICAL"
)

type AlertCondition string

const (
	AlertCondGT         AlertCondition = "gt"
	AlertCondLT         AlertCondition = "lt"
	AlertCondEQ         AlertCondition = "eq"
	AlertCondGTE        AlertCondition = "gte"
	AlertCondLTE        AlertCondition = "lte"
	AlertCondRange      AlertCondition = "range"
	AlertCondChangeRate AlertCondition = "change_rate"
)

type AlertRule struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	DeviceID     uint           `gorm:"index;not null" json:"device_id"`
	NodeID       string         `gorm:"size:256;not null" json:"node_id"`
	Name         string         `gorm:"size:128;not null" json:"name"`
	Enabled      bool           `gorm:"default:true" json:"enabled"`
	Condition    AlertCondition `gorm:"size:32;not null" json:"condition"`
	Threshold    float64        `json:"threshold"`
	ThresholdMax float64        `json:"threshold_max,omitempty"`
	Level        AlertLevel     `gorm:"size:32;not null" json:"level"`
	Description  string         `gorm:"size:512" json:"description"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func (AlertRule) TableName() string { return "alert_rules" }

type AlertEventStatus string

const (
	AlertStatusActive   AlertEventStatus = "active"
	AlertStatusResolved AlertEventStatus = "resolved"
)

type AlertEvent struct {
	ID          uint             `gorm:"primarykey" json:"id"`
	RuleID      uint             `gorm:"index" json:"rule_id"`
	DeviceID    uint             `gorm:"index;not null" json:"device_id"`
	NodeID      string           `gorm:"size:256;not null" json:"node_id"`
	Level       AlertLevel       `gorm:"size:32;not null" json:"level"`
	Message     string           `gorm:"size:512;not null" json:"message"`
	Value       string           `gorm:"size:256" json:"value"`
	Status      AlertEventStatus `gorm:"size:32;default:active;index" json:"status"`
	TriggeredAt time.Time        `gorm:"index" json:"triggered_at"`
	ResolvedAt  *time.Time       `json:"resolved_at,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
}

func (AlertEvent) TableName() string { return "alert_events" }
