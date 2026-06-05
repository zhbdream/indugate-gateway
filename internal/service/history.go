package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/model"
	"github.com/indugate/gateway/internal/protocol"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type HistoryService struct {
	db *gorm.DB
}

func NewHistoryService(db *gorm.DB) *HistoryService {
	return &HistoryService{db: db}
}

func (s *HistoryService) Record(ctx context.Context, deviceID uint, value *protocol.DataValue) error {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value.Value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}
	entry := model.DataHistory{
		DeviceID:  deviceID,
		NodeID:    value.NodeID,
		Value:     string(raw),
		DataType:  value.DataType,
		Status:    value.Status,
		Timestamp: value.Timestamp,
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	return s.db.WithContext(ctx).Create(&entry).Error
}

type HistoryQuery struct {
	DeviceID uint
	NodeID   string
	Limit    int
	Since    *time.Time
}

func (s *HistoryService) Query(ctx context.Context, q HistoryQuery) ([]model.DataHistory, error) {
	if q.Limit <= 0 {
		q.Limit = 100
	}
	if q.Limit > 1000 {
		q.Limit = 1000
	}

	query := s.db.WithContext(ctx).Where("device_id = ?", q.DeviceID)
	if q.NodeID != "" {
		query = query.Where("node_id = ?", q.NodeID)
	}
	if q.Since != nil {
		query = query.Where("timestamp >= ?", *q.Since)
	}

	var rows []model.DataHistory
	if err := query.Order("timestamp desc").Limit(q.Limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	return rows, nil
}

func (s *HistoryService) PurgeBefore(ctx context.Context, before time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Where("timestamp < ?", before).Delete(&model.DataHistory{})
	return result.RowsAffected, result.Error
}

func (s *HistoryService) StartRetentionJob(ctx context.Context, log *zap.Logger, cfg config.HistoryConfig) {
	if cfg.RetentionDays <= 0 {
		return
	}
	interval := time.Duration(cfg.CleanupIntervalHours) * time.Hour
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	go func() {
		runHistoryPurge(ctx, s, log, cfg.RetentionDays)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runHistoryPurge(ctx, s, log, cfg.RetentionDays)
			}
		}
	}()
}

func runHistoryPurge(ctx context.Context, s *HistoryService, log *zap.Logger, retentionDays int) {
	before := time.Now().AddDate(0, 0, -retentionDays)
	deleted, err := s.PurgeBefore(ctx, before)
	if err != nil {
		log.Warn("history retention purge failed", zap.Error(err))
		return
	}
	if deleted > 0 {
		log.Info("history retention purge completed", zap.Int64("deleted", deleted), zap.Int("retention_days", retentionDays))
	}
}
