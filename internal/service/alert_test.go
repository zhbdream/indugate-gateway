package service

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/indugate/gateway/internal/model"
	"gorm.io/gorm"
)

func setupAlertTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AlertRule{}, &model.AlertEvent{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAlertEvaluate(t *testing.T) {
	svc := NewAlertService(setupAlertTestDB(t), nil)
	if err := svc.CreateRule(context.Background(), &model.AlertRule{
		DeviceID:  1,
		NodeID:    "holding:0",
		Name:      "high temp",
		Condition: model.AlertCondGT,
		Threshold: 80,
		Level:     model.AlertLevelWarn,
	}); err != nil {
		t.Fatal(err)
	}

	triggered, err := svc.Evaluate(context.Background(), 1, "holding:0", 95.0)
	if err != nil {
		t.Fatal(err)
	}
	if len(triggered) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(triggered))
	}

	none, err := svc.Evaluate(context.Background(), 1, "holding:0", 50.0)
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Fatalf("expected no alert, got %d", len(none))
	}
}

func TestAlertEvaluateDedup(t *testing.T) {
	svc := NewAlertService(setupAlertTestDB(t), nil)
	if err := svc.CreateRule(context.Background(), &model.AlertRule{
		DeviceID:  1,
		NodeID:    "holding:0",
		Name:      "high temp",
		Condition: model.AlertCondGT,
		Threshold: 80,
		Level:     model.AlertLevelWarn,
	}); err != nil {
		t.Fatal(err)
	}

	first, err := svc.Evaluate(context.Background(), 1, "holding:0", 95.0)
	if err != nil || len(first) != 1 {
		t.Fatalf("expected first alert, got %v err=%v", first, err)
	}

	second, err := svc.Evaluate(context.Background(), 1, "holding:0", 96.0)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 0 {
		t.Fatalf("expected deduplicated alert, got %d", len(second))
	}
}

func TestAlertChangeRate(t *testing.T) {
	svc := NewAlertService(setupAlertTestDB(t), nil)
	if err := svc.CreateRule(context.Background(), &model.AlertRule{
		DeviceID:  1,
		NodeID:    "sensor/temp",
		Name:      "rapid change",
		Condition: model.AlertCondChangeRate,
		Threshold: 5,
		Level:     model.AlertLevelError,
	}); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if _, err := svc.Evaluate(ctx, 1, "sensor/temp", 10.0); err != nil {
		t.Fatal(err)
	}

	svc.setSample(1, "sensor/temp", valueSample{value: 10, at: time.Now().Add(-1 * time.Second)})

	triggered, err := svc.Evaluate(ctx, 1, "sensor/temp", 20.0)
	if err != nil {
		t.Fatal(err)
	}
	if len(triggered) != 1 {
		t.Fatalf("expected change_rate alert, got %d", len(triggered))
	}
}

func TestHistoryPurge(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.DataHistory{}); err != nil {
		t.Fatal(err)
	}

	svc := NewHistoryService(db)
	old := time.Now().AddDate(0, 0, -60)
	recent := time.Now()
	db.Create(&model.DataHistory{DeviceID: 1, NodeID: "a", Value: "1", Timestamp: old})
	db.Create(&model.DataHistory{DeviceID: 1, NodeID: "b", Value: "2", Timestamp: recent})

	deleted, err := svc.PurgeBefore(context.Background(), time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleted)
	}
}
