package model

import "time"

type UserDevice struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_user_device" json:"user_id"`
	DeviceID  uint      `gorm:"not null;uniqueIndex:idx_user_device" json:"device_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (UserDevice) TableName() string { return "user_devices" }
