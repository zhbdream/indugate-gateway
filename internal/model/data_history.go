package model

import "time"

type DataHistory struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	DeviceID  uint      `gorm:"index;not null" json:"device_id"`
	NodeID    string    `gorm:"size:256;not null;index" json:"node_id"`
	Value     string    `gorm:"type:text" json:"value"`
	DataType  string    `gorm:"size:64" json:"data_type,omitempty"`
	Status    string    `gorm:"size:64" json:"status"`
	Timestamp time.Time `gorm:"index" json:"timestamp"`
	CreatedAt time.Time `json:"created_at"`
}

func (DataHistory) TableName() string {
	return "data_history"
}
