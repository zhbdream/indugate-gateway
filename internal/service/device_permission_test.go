package service

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/model"
	"gorm.io/gorm"
)

func setupPermissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Device{}, &model.UserDevice{}); err != nil {
		t.Fatal(err)
	}
	devices := []model.Device{{Name: "A"}, {Name: "B"}, {Name: "C"}}
	if err := db.Create(&devices).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestDevicePermissionACL(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := NewDevicePermissionService(db, config.AuthConfig{Enabled: true, DeviceACLEnabled: true})
	ctx := context.Background()

	operator := AccessPrincipal{UserID: 1, Role: model.RoleOperator}
	if err := svc.SetUserDevices(ctx, 1, []uint{1, 2}); err != nil {
		t.Fatal(err)
	}

	ok, err := svc.CanAccess(ctx, operator, 1)
	if err != nil || !ok {
		t.Fatal("operator should access device 1")
	}
	ok, err = svc.CanAccess(ctx, operator, 3)
	if err != nil || ok {
		t.Fatal("operator should not access device 3")
	}

	admin := AccessPrincipal{UserID: 2, Role: model.RoleAdmin}
	ok, err = svc.CanAccess(ctx, admin, 3)
	if err != nil || !ok {
		t.Fatal("admin should access all devices")
	}

	filter, err := svc.ResolveFilter(ctx, operator)
	if err != nil || filter == nil || len(*filter) != 2 {
		t.Fatalf("unexpected filter: %v err=%v", filter, err)
	}

	all := []model.Device{{ID: 1}, {ID: 2}, {ID: 3}}
	filtered := svc.FilterDevices(ctx, filter, all)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(filtered))
	}
}

func TestDevicePermissionDisabled(t *testing.T) {
	db := setupPermissionTestDB(t)
	svc := NewDevicePermissionService(db, config.AuthConfig{Enabled: false, DeviceACLEnabled: true})
	ctx := context.Background()
	operator := AccessPrincipal{UserID: 1, Role: model.RoleOperator}

	ok, err := svc.CanAccess(ctx, operator, 99)
	if err != nil || !ok {
		t.Fatal("ACL disabled should allow all devices")
	}
}
