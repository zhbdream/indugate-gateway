package service

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/model"
	"gorm.io/gorm"
)

func TestAuthLogin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}

	svc := NewAuthService(db, config.AuthConfig{
		Enabled:   true,
		JWTSecret: "test-secret",
	})
	if _, err := svc.CreateUser(context.Background(), "admin", "secret", model.RoleAdmin); err != nil {
		t.Fatal(err)
	}

	token, user, err := svc.Login(context.Background(), "admin", "secret")
	if err != nil || token == "" || user.Username != "admin" {
		t.Fatalf("login failed: token=%q user=%+v err=%v", token, user, err)
	}

	claims, err := svc.ValidateToken(token)
	if err != nil || claims.Username != "admin" {
		t.Fatalf("validate failed: %+v err=%v", claims, err)
	}

	if _, _, err := svc.Login(context.Background(), "admin", "wrong"); err != ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestDeleteLastAdmin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	svc := NewAuthService(db, config.AuthConfig{JWTSecret: "x"})
	user, err := svc.CreateUser(context.Background(), "admin", "secret", model.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteUser(context.Background(), user.ID); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
}
