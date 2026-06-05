package storage

import (
	"fmt"
	"time"

	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/model"
	"go.uber.org/zap"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func NewDB(cfg config.DatabaseConfig, log *zap.Logger) (*gorm.DB, error) {
	gormLog := gormlogger.Default.LogMode(gormlogger.Silent)
	if log.Core().Enabled(zap.DebugLevel) {
		gormLog = gormlogger.Default.LogMode(gormlogger.Info)
	}

	var dialector gorm.Dialector
	switch cfg.Driver {
	case "sqlite":
		dialector = sqlite.Open(cfg.DSN)
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{Logger: gormLog})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}

	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	}

	if err := db.AutoMigrate(&model.Device{}); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	log.Info("database connected", zap.String("driver", cfg.Driver))
	return db, nil
}
