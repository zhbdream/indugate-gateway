package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/indugate/gateway/internal/api/handler"
	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/model"
	"github.com/indugate/gateway/internal/service"
	"gorm.io/gorm"
)

func setupMCPRouter(t *testing.T) *gin.Engine {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Device{}); err != nil {
		t.Fatal(err)
	}

	dm := service.NewDriverManager()
	deviceService := service.NewDeviceService(db, dm)
	mcpHandler := handler.NewMCPHandler(config.MCPConfig{BasePath: "/mcp"}, deviceService, nil)

	r := gin.New()
	mcp := r.Group("/mcp")
	{
		mcp.GET("/.well-known/mcp.json", mcpHandler.Discovery)
		mcp.POST("/message", mcpHandler.Message)
	}
	return r
}

func TestMCPDiscovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := setupMCPRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/mcp/.well-known/mcp.json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["name"] != "InduGate MCP Server" {
		t.Fatalf("unexpected name: %v", body["name"])
	}
}

func TestMCPInitializeAndToolsList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := setupMCPRouter(t)

	initReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0.0"},
		},
	}
	body, _ := json.Marshal(initReq)
	req := httptest.NewRequest(http.MethodPost, "/mcp/message", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("initialize: expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	listReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	}
	body, _ = json.Marshal(listReq)
	req = httptest.NewRequest(http.MethodPost, "/mcp/message", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Result.Tools) != 5 {
		t.Fatalf("expected 5 tools, got %d", len(resp.Result.Tools))
	}
}

func TestMCPListDevicesTool(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Device{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Device{Name: "Test Device", Protocol: model.ProtocolModbus, Address: "127.0.0.1:502"})

	dm := service.NewDriverManager()
	mcpHandler := handler.NewMCPHandler(config.MCPConfig{BasePath: "/mcp"}, service.NewDeviceService(db, dm), nil)

	r := gin.New()
	r.POST("/mcp/message", mcpHandler.Message)

	callReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "list_devices",
			"arguments": map[string]any{},
		},
	}
	body, _ := json.Marshal(callReq)
	req := httptest.NewRequest(http.MethodPost, "/mcp/message", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Result.IsError {
		t.Fatalf("unexpected tool error")
	}
	if !bytes.Contains([]byte(resp.Result.Content[0].Text), []byte("Test Device")) {
		t.Fatalf("expected device in response: %s", resp.Result.Content[0].Text)
	}
}
