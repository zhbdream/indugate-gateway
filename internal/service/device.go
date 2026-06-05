package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/indugate/gateway/internal/model"
	"gorm.io/gorm"
)

var ErrDeviceNotFound = errors.New("device not found")

type DeviceService struct {
	db      *gorm.DB
	drivers *DriverManager
}

func NewDeviceService(db *gorm.DB, drivers *DriverManager) *DeviceService {
	return &DeviceService{db: db, drivers: drivers}
}

func (s *DeviceService) List(ctx context.Context) ([]model.Device, error) {
	var devices []model.Device
	if err := s.db.WithContext(ctx).Order("id desc").Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	return devices, nil
}

func (s *DeviceService) Get(ctx context.Context, id uint) (*model.Device, error) {
	var device model.Device
	if err := s.db.WithContext(ctx).First(&device, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("get device: %w", err)
	}
	return &device, nil
}

func (s *DeviceService) Create(ctx context.Context, device *model.Device) error {
	if err := s.db.WithContext(ctx).Create(device).Error; err != nil {
		return fmt.Errorf("create device: %w", err)
	}
	return nil
}

func (s *DeviceService) Update(ctx context.Context, id uint, updates *model.Device) (*model.Device, error) {
	device, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	device.Name = updates.Name
	device.Protocol = updates.Protocol
	device.Address = updates.Address
	device.Config = updates.Config
	device.Description = updates.Description

	if err := s.db.WithContext(ctx).Save(device).Error; err != nil {
		return nil, fmt.Errorf("update device: %w", err)
	}
	return device, nil
}

func (s *DeviceService) Delete(ctx context.Context, id uint) error {
	_ = s.drivers.Disconnect(ctx, id)

	result := s.db.WithContext(ctx).Delete(&model.Device{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete device: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

func (s *DeviceService) Connect(ctx context.Context, id uint) (*model.Device, error) {
	device, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.drivers.Connect(ctx, device); err != nil {
		device.Status = model.DeviceStatusError
		_ = s.db.WithContext(ctx).Save(device).Error
		return nil, fmt.Errorf("connect device: %w", err)
	}

	device.Status = model.DeviceStatusConnected
	if err := s.db.WithContext(ctx).Save(device).Error; err != nil {
		return nil, fmt.Errorf("update device status: %w", err)
	}
	return device, nil
}

func (s *DeviceService) Disconnect(ctx context.Context, id uint) (*model.Device, error) {
	device, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.drivers.Disconnect(ctx, id); err != nil {
		return nil, fmt.Errorf("disconnect device: %w", err)
	}

	device.Status = model.DeviceStatusDisconnected
	if err := s.db.WithContext(ctx).Save(device).Error; err != nil {
		return nil, fmt.Errorf("update device status: %w", err)
	}
	return device, nil
}

func (s *DeviceService) Drivers() *DriverManager {
	return s.drivers
}
