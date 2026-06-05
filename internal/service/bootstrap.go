package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/indugate/gateway/internal/model"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BootstrapService struct {
	log     *zap.Logger
	db      *gorm.DB
	devices *DeviceService
}

func NewBootstrapService(log *zap.Logger, db *gorm.DB, devices *DeviceService) *BootstrapService {
	return &BootstrapService{log: log, db: db, devices: devices}
}

type devicesYAML struct {
	Devices []deviceTemplate `yaml:"devices"`
}

type deviceTemplate struct {
	Name        string         `yaml:"name"`
	Protocol    string         `yaml:"protocol"`
	Address     string         `yaml:"address"`
	Description string         `yaml:"description"`
	Config      map[string]any `yaml:"config"`
}

func (s *BootstrapService) LoadDevicesFile(ctx context.Context, path string) error {
	if path == "" {
		return nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		s.log.Info("devices bootstrap file not found, skipping", zap.String("path", path))
		return nil
	}

	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read devices file: %w", err)
	}

	var cfg devicesYAML
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("parse devices file: %w", err)
	}

	var existing int64
	if err := s.db.WithContext(ctx).Model(&model.Device{}).Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		s.log.Info("devices already exist, skip bootstrap", zap.Int64("count", existing))
		return nil
	}

	for _, tpl := range cfg.Devices {
		configJSON := "{}"
		if tpl.Config != nil {
			raw, err := json.Marshal(tpl.Config)
			if err != nil {
				return fmt.Errorf("marshal config for %q: %w", tpl.Name, err)
			}
			configJSON = string(raw)
		}
		device := &model.Device{
			Name:        tpl.Name,
			Protocol:    model.DeviceProtocol(tpl.Protocol),
			Address:     tpl.Address,
			Config:      configJSON,
			Description: tpl.Description,
			Status:      model.DeviceStatusDisconnected,
		}
		if err := s.devices.Create(ctx, device); err != nil {
			return fmt.Errorf("bootstrap device %q: %w", tpl.Name, err)
		}
		s.log.Info("bootstrapped device", zap.String("name", tpl.Name), zap.String("protocol", tpl.Protocol))
	}
	return nil
}
