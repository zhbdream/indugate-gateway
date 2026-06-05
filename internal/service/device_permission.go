package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/model"
	"gorm.io/gorm"
)

var ErrDeviceAccessDenied = errors.New("device access denied")

type AccessPrincipal struct {
	UserID   uint
	Username string
	Role     model.UserRole
}

type DevicePermissionService struct {
	db  *gorm.DB
	cfg config.AuthConfig
}

func NewDevicePermissionService(db *gorm.DB, cfg config.AuthConfig) *DevicePermissionService {
	return &DevicePermissionService{db: db, cfg: cfg}
}

func (s *DevicePermissionService) ACLActive() bool {
	return s.cfg.Enabled && s.cfg.DeviceACLEnabled
}

func (s *DevicePermissionService) ResolveFilter(ctx context.Context, p AccessPrincipal) (*[]uint, error) {
	if !s.ACLActive() || p.Role == model.RoleAdmin {
		return nil, nil
	}
	if p.UserID == 0 {
		return nil, nil
	}
	ids, err := s.GetUserDeviceIDs(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	return &ids, nil
}

func (s *DevicePermissionService) GetUserDeviceIDs(ctx context.Context, userID uint) ([]uint, error) {
	var rows []model.UserDevice
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list user devices: %w", err)
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.DeviceID)
	}
	return ids, nil
}

func (s *DevicePermissionService) SetUserDevices(ctx context.Context, userID uint, deviceIDs []uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserDevice{}).Error; err != nil {
			return err
		}
		if len(deviceIDs) == 0 {
			return nil
		}
		seen := make(map[uint]struct{}, len(deviceIDs))
		rows := make([]model.UserDevice, 0, len(deviceIDs))
		for _, deviceID := range deviceIDs {
			if deviceID == 0 {
				continue
			}
			if _, ok := seen[deviceID]; ok {
				continue
			}
			seen[deviceID] = struct{}{}
			var count int64
			if err := tx.Model(&model.Device{}).Where("id = ?", deviceID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return fmt.Errorf("device %d not found", deviceID)
			}
			rows = append(rows, model.UserDevice{UserID: userID, DeviceID: deviceID})
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Create(&rows).Error
	})
}

func (s *DevicePermissionService) CanAccess(ctx context.Context, p AccessPrincipal, deviceID uint) (bool, error) {
	if !s.ACLActive() || p.Role == model.RoleAdmin {
		return true, nil
	}
	if p.UserID == 0 {
		return true, nil
	}
	ids, err := s.GetUserDeviceIDs(ctx, p.UserID)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if id == deviceID {
			return true, nil
		}
	}
	return false, nil
}

func (s *DevicePermissionService) RequireAccess(ctx context.Context, p AccessPrincipal, deviceID uint) error {
	ok, err := s.CanAccess(ctx, p, deviceID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrDeviceAccessDenied
	}
	return nil
}

func (s *DevicePermissionService) FilterDevices(_ context.Context, filter *[]uint, devices []model.Device) []model.Device {
	if filter == nil {
		return devices
	}
	if len(*filter) == 0 {
		return []model.Device{}
	}
	allowed := make(map[uint]struct{}, len(*filter))
	for _, id := range *filter {
		allowed[id] = struct{}{}
	}
	result := make([]model.Device, 0, len(devices))
	for _, d := range devices {
		if _, ok := allowed[d.ID]; ok {
			result = append(result, d)
		}
	}
	return result
}

func ApplyDeviceIDFilter(query *gorm.DB, filter *[]uint) *gorm.DB {
	if filter == nil {
		return query
	}
	if len(*filter) == 0 {
		return query.Where("1 = 0")
	}
	return query.Where("id IN ?", *filter)
}
