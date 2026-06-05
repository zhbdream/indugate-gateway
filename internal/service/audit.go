package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

type AuditQuery struct {
	Username string
	Action   string
	Limit    int
	Offset   int
	Since    *time.Time
}

func (s *AuditService) Record(ctx context.Context, entry *model.AuditLog) error {
	if entry == nil {
		return nil
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	return s.db.WithContext(ctx).Create(entry).Error
}

func (s *AuditService) List(ctx context.Context, q AuditQuery) ([]model.AuditLog, int64, error) {
	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.Limit > 200 {
		q.Limit = 200
	}

	query := s.db.WithContext(ctx).Model(&model.AuditLog{})
	if q.Username != "" {
		query = query.Where("username = ?", q.Username)
	}
	if q.Action != "" {
		query = query.Where("action LIKE ?", "%"+q.Action+"%")
	}
	if q.Since != nil {
		query = query.Where("created_at >= ?", *q.Since)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	var rows []model.AuditLog
	if err := query.Order("created_at desc").Offset(q.Offset).Limit(q.Limit).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	return rows, total, nil
}

func (s *AuditService) PurgeBefore(ctx context.Context, before time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Where("created_at < ?", before).Delete(&model.AuditLog{})
	return result.RowsAffected, result.Error
}

func (s *AuditService) StartRetentionJob(ctx context.Context, log *zap.Logger, cfg config.AuditConfig) {
	if cfg.RetentionDays <= 0 {
		return
	}
	interval := time.Duration(cfg.CleanupIntervalHours) * time.Hour
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	go func() {
		runAuditPurge(ctx, s, log, cfg.RetentionDays)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runAuditPurge(ctx, s, log, cfg.RetentionDays)
			}
		}
	}()
}

func runAuditPurge(ctx context.Context, s *AuditService, log *zap.Logger, retentionDays int) {
	before := time.Now().AddDate(0, 0, -retentionDays)
	deleted, err := s.PurgeBefore(ctx, before)
	if err != nil {
		log.Warn("audit retention purge failed", zap.Error(err))
		return
	}
	if deleted > 0 {
		log.Info("audit retention purge completed", zap.Int64("deleted", deleted), zap.Int("retention_days", retentionDays))
	}
}

func DeriveAuditAction(method, path string) string {
	method = strings.ToUpper(method)
	path = strings.TrimSuffix(path, "/")

	switch {
	case path == "/api/v1/auth/login" && method == "POST":
		return "auth.login"
	case path == "/api/v1/users" && method == "POST":
		return "user.create"
	case strings.HasPrefix(path, "/api/v1/users/") && strings.HasSuffix(path, "/password") && method == "PUT":
		return "user.change_password"
	case strings.HasPrefix(path, "/api/v1/users/") && method == "PUT":
		return "user.update"
	case strings.HasPrefix(path, "/api/v1/users/") && method == "DELETE":
		return "user.delete"
	case path == "/api/v1/devices" && method == "POST":
		return "device.create"
	case strings.HasPrefix(path, "/api/v1/devices/") && strings.HasSuffix(path, "/connect") && method == "POST":
		return "device.connect"
	case strings.HasPrefix(path, "/api/v1/devices/") && strings.HasSuffix(path, "/disconnect") && method == "POST":
		return "device.disconnect"
	case strings.HasPrefix(path, "/api/v1/devices/") && strings.Contains(path, "/data/") && method == "POST":
		return "device.write"
	case strings.HasPrefix(path, "/api/v1/devices/") && strings.HasSuffix(path, "/subscribe") && method == "POST":
		return "device.subscribe"
	case strings.HasPrefix(path, "/api/v1/devices/") && strings.Contains(path, "/subscriptions/") && method == "DELETE":
		return "device.unsubscribe"
	case strings.HasPrefix(path, "/api/v1/devices/") && method == "DELETE":
		return "device.delete"
	case strings.HasPrefix(path, "/api/v1/devices/") && method == "PUT":
		return "device.update"
	case path == "/api/v1/alerts/rules" && method == "POST":
		return "alert.rule.create"
	case strings.HasPrefix(path, "/api/v1/alerts/rules/") && method == "PUT":
		return "alert.rule.update"
	case strings.HasPrefix(path, "/api/v1/alerts/rules/") && method == "DELETE":
		return "alert.rule.delete"
	case strings.HasPrefix(path, "/api/v1/alerts/events/") && strings.HasSuffix(path, "/acknowledge") && method == "POST":
		return "alert.event.acknowledge"
	case strings.HasPrefix(path, "/api/v1/simulators/") && strings.HasSuffix(path, "/start") && method == "POST":
		return "simulator.start"
	case strings.HasPrefix(path, "/api/v1/simulators/") && strings.HasSuffix(path, "/stop") && method == "POST":
		return "simulator.stop"
	case strings.HasPrefix(path, "/api/v1/simulators/") && strings.HasSuffix(path, "/config") && method == "PUT":
		return "simulator.config"
	default:
		return strings.ToLower(method) + " " + path
	}
}
