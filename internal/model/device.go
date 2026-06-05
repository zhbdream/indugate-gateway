package model

import (
	"time"

	"gorm.io/gorm"
)

type DeviceStatus string

const (
	DeviceStatusDisconnected DeviceStatus = "disconnected"
	DeviceStatusConnected    DeviceStatus = "connected"
	DeviceStatusError        DeviceStatus = "error"
)

type DeviceProtocol string

const (
	ProtocolOPCUA  DeviceProtocol = "opcua"
	ProtocolModbus DeviceProtocol = "modbus"
	ProtocolMQTT   DeviceProtocol = "mqtt"
	ProtocolS7     DeviceProtocol = "s7"
	ProtocolBACnet DeviceProtocol = "bacnet"
)

type Device struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"size:128;not null" json:"name"`
	Protocol    DeviceProtocol `gorm:"size:32;not null;index" json:"protocol"`
	Address     string         `gorm:"size:256;not null" json:"address"`
	Config      string         `gorm:"type:text" json:"config"`
	Status      DeviceStatus   `gorm:"size:32;default:disconnected" json:"status"`
	Description string         `gorm:"size:512" json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Device) TableName() string {
	return "devices"
}
