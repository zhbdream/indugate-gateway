package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/model"
)

func TestAllowedRole(t *testing.T) {
	if !allowedRole(model.RoleAdmin, http.MethodDelete, "/api/v1/users/1") {
		t.Fatal("admin should access users")
	}
	if allowedRole(model.RoleViewer, http.MethodPost, "/api/v1/devices/1/connect") {
		t.Fatal("viewer should not connect devices")
	}
	if !allowedRole(model.RoleViewer, http.MethodGet, "/api/v1/devices") {
		t.Fatal("viewer should list devices")
	}
	if allowedRole(model.RoleOperator, http.MethodPost, "/api/v1/users") {
		t.Fatal("operator should not manage users")
	}
	if !allowedRole(model.RoleOperator, http.MethodPost, "/api/v1/devices/1/connect") {
		t.Fatal("operator should connect devices")
	}
}

func TestRequireAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("role", model.RoleOperator)
	if RequireAdmin(c) {
		t.Fatal("operator should not pass admin check")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}
