package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/api/handler"
)

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handler.NewHealthHandler()
	r := gin.New()
	r.GET("/health", h.Check)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if body["code"].(float64) != 0 {
		t.Fatalf("expected code 0, got %v", body["code"])
	}
}
