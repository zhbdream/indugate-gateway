package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/indugate/gateway/internal/model"
	"gorm.io/gorm"
)

type valueSample struct {
	value float64
	at    time.Time
}

type AlertService struct {
	db       *gorm.DB
	notifier *AlertNotifier
	samples  sync.Map // key: "deviceID:nodeID" -> valueSample
}

func NewAlertService(db *gorm.DB, notifier *AlertNotifier) *AlertService {
	return &AlertService{db: db, notifier: notifier}
}

func (s *AlertService) ListRules(ctx context.Context, deviceID uint, deviceFilter *[]uint) ([]model.AlertRule, error) {
	query := s.db.WithContext(ctx).Order("id desc")
	if deviceID > 0 {
		query = query.Where("device_id = ?", deviceID)
	}
	query = applyAlertDeviceFilter(query, deviceFilter)
	var rules []model.AlertRule
	if err := query.Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("list alert rules: %w", err)
	}
	return rules, nil
}

func (s *AlertService) CreateRule(ctx context.Context, rule *model.AlertRule) error {
	if rule.Level == "" {
		rule.Level = model.AlertLevelWarn
	}
	rule.Enabled = true
	return s.db.WithContext(ctx).Create(rule).Error
}

func (s *AlertService) GetRule(ctx context.Context, id uint) (*model.AlertRule, error) {
	var rule model.AlertRule
	if err := s.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *AlertService) GetEvent(ctx context.Context, id uint) (*model.AlertEvent, error) {
	var event model.AlertEvent
	if err := s.db.WithContext(ctx).First(&event, id).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *AlertService) UpdateRule(ctx context.Context, id uint, updates *model.AlertRule) (*model.AlertRule, error) {
	var rule model.AlertRule
	if err := s.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		return nil, err
	}
	rule.Name = updates.Name
	rule.NodeID = updates.NodeID
	rule.Enabled = updates.Enabled
	rule.Condition = updates.Condition
	rule.Threshold = updates.Threshold
	rule.ThresholdMax = updates.ThresholdMax
	rule.Level = updates.Level
	rule.Description = updates.Description
	if err := s.db.WithContext(ctx).Save(&rule).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *AlertService) DeleteRule(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&model.AlertRule{}, id).Error
}

func (s *AlertService) ListEvents(ctx context.Context, deviceID uint, status string, limit int, deviceFilter *[]uint) ([]model.AlertEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	query := s.db.WithContext(ctx).Order("triggered_at desc").Limit(limit)
	if deviceID > 0 {
		query = query.Where("device_id = ?", deviceID)
	}
	query = applyAlertDeviceFilter(query, deviceFilter)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var events []model.AlertEvent
	if err := query.Find(&events).Error; err != nil {
		return nil, fmt.Errorf("list alert events: %w", err)
	}
	return events, nil
}

func (s *AlertService) AcknowledgeEvent(ctx context.Context, id uint) (*model.AlertEvent, error) {
	var event model.AlertEvent
	if err := s.db.WithContext(ctx).First(&event, id).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	event.Status = model.AlertStatusResolved
	event.ResolvedAt = &now
	if err := s.db.WithContext(ctx).Save(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *AlertService) Evaluate(ctx context.Context, deviceID uint, nodeID string, value any) ([]model.AlertEvent, error) {
	var rules []model.AlertRule
	if err := s.db.WithContext(ctx).
		Where("device_id = ? AND node_id = ? AND enabled = ?", deviceID, nodeID, true).
		Find(&rules).Error; err != nil {
		return nil, err
	}

	num, ok := toFloat64(value)
	now := time.Now()
	prev, hasPrev := s.getSample(deviceID, nodeID)

	if ok {
		defer s.setSample(deviceID, nodeID, valueSample{value: num, at: now})
	}

	if len(rules) == 0 || !ok {
		return nil, nil
	}

	var triggered []model.AlertEvent
	for _, rule := range rules {
		if !matchCondition(rule, num, prev, hasPrev, now) {
			continue
		}
		if s.hasActiveEvent(ctx, rule.ID) {
			continue
		}

		valStr := fmt.Sprintf("%v", value)
		event := model.AlertEvent{
			RuleID:      rule.ID,
			DeviceID:    deviceID,
			NodeID:      nodeID,
			Level:       rule.Level,
			Message:     buildAlertMessage(rule, value),
			Value:       valStr,
			Status:      model.AlertStatusActive,
			TriggeredAt: now,
		}
		if err := s.db.WithContext(ctx).Create(&event).Error; err != nil {
			return triggered, err
		}
		if s.notifier != nil {
			s.notifier.Notify(ctx, event)
		}
		triggered = append(triggered, event)
	}
	return triggered, nil
}

func buildAlertMessage(rule model.AlertRule, value any) string {
	switch rule.Condition {
	case model.AlertCondChangeRate:
		return fmt.Sprintf("[%s] %s: change rate exceeded (threshold %.4f/s, value %v)", rule.Level, rule.Name, rule.Threshold, value)
	case model.AlertCondRange:
		return fmt.Sprintf("[%s] %s: value %v outside range [%.4f, %.4f]", rule.Level, rule.Name, value, rule.Threshold, rule.ThresholdMax)
	default:
		return fmt.Sprintf("[%s] %s: value %v %s threshold %.4f", rule.Level, rule.Name, value, rule.Condition, rule.Threshold)
	}
}

func (s *AlertService) hasActiveEvent(ctx context.Context, ruleID uint) bool {
	var count int64
	s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("rule_id = ? AND status = ?", ruleID, model.AlertStatusActive).
		Count(&count)
	return count > 0
}

func sampleKey(deviceID uint, nodeID string) string {
	return fmt.Sprintf("%d:%s", deviceID, nodeID)
}

func (s *AlertService) getSample(deviceID uint, nodeID string) (valueSample, bool) {
	v, ok := s.samples.Load(sampleKey(deviceID, nodeID))
	if !ok {
		return valueSample{}, false
	}
	sample, ok := v.(valueSample)
	return sample, ok
}

func (s *AlertService) setSample(deviceID uint, nodeID string, sample valueSample) {
	s.samples.Store(sampleKey(deviceID, nodeID), sample)
}

func matchCondition(rule model.AlertRule, value float64, prev valueSample, hasPrev bool, now time.Time) bool {
	switch rule.Condition {
	case model.AlertCondGT:
		return value > rule.Threshold
	case model.AlertCondLT:
		return value < rule.Threshold
	case model.AlertCondEQ:
		return math.Abs(value-rule.Threshold) < 0.0001
	case model.AlertCondGTE:
		return value >= rule.Threshold
	case model.AlertCondLTE:
		return value <= rule.Threshold
	case model.AlertCondRange:
		min, max := rule.Threshold, rule.ThresholdMax
		if min > max {
			min, max = max, min
		}
		return value < min || value > max
	case model.AlertCondChangeRate:
		if !hasPrev {
			return false
		}
		elapsed := now.Sub(prev.at).Seconds()
		if elapsed <= 0 {
			return false
		}
		rate := math.Abs(value-prev.value) / elapsed
		return rate > rule.Threshold
	default:
		return false
	}
}

func toFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint16:
		return float64(v), true
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

type DashboardStats struct {
	DeviceTotal       int64 `json:"device_total"`
	DeviceConnected   int64 `json:"device_connected"`
	DeviceError       int64 `json:"device_error"`
	ActiveAlerts      int64 `json:"active_alerts"`
	AlertRules        int64 `json:"alert_rules"`
	HistoryRecords24h int64 `json:"history_records_24h"`
}

func (s *AlertService) DashboardStats(ctx context.Context, deviceFilter *[]uint) (*DashboardStats, error) {
	stats := &DashboardStats{}
	deviceQuery := s.db.WithContext(ctx).Model(&model.Device{})
	deviceQuery = ApplyDeviceIDFilter(deviceQuery, deviceFilter)
	deviceQuery.Count(&stats.DeviceTotal)
	ApplyDeviceIDFilter(s.db.WithContext(ctx).Model(&model.Device{}), deviceFilter).Where("status = ?", model.DeviceStatusConnected).Count(&stats.DeviceConnected)
	ApplyDeviceIDFilter(s.db.WithContext(ctx).Model(&model.Device{}), deviceFilter).Where("status = ?", model.DeviceStatusError).Count(&stats.DeviceError)

	alertQuery := s.db.WithContext(ctx).Model(&model.AlertEvent{}).Where("status = ?", model.AlertStatusActive)
	alertQuery = applyAlertDeviceFilter(alertQuery, deviceFilter)
	alertQuery.Count(&stats.ActiveAlerts)

	ruleQuery := s.db.WithContext(ctx).Model(&model.AlertRule{})
	ruleQuery = applyAlertDeviceFilter(ruleQuery, deviceFilter)
	ruleQuery.Count(&stats.AlertRules)

	historyQuery := s.db.WithContext(ctx).Model(&model.DataHistory{}).Where("timestamp >= ?", time.Now().Add(-24*time.Hour))
	historyQuery = applyHistoryDeviceFilter(historyQuery, deviceFilter)
	historyQuery.Count(&stats.HistoryRecords24h)
	return stats, nil
}

func applyAlertDeviceFilter(query *gorm.DB, filter *[]uint) *gorm.DB {
	if filter == nil {
		return query
	}
	if len(*filter) == 0 {
		return query.Where("1 = 0")
	}
	return query.Where("device_id IN ?", *filter)
}

func applyHistoryDeviceFilter(query *gorm.DB, filter *[]uint) *gorm.DB {
	if filter == nil {
		return query
	}
	if len(*filter) == 0 {
		return query.Where("1 = 0")
	}
	return query.Where("device_id IN ?", *filter)
}
