package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/indugate/gateway/internal/model"
	"github.com/indugate/gateway/internal/service"
	"gorm.io/gorm"
)

func setupTestServer(t *testing.T) *Server {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Device{}); err != nil {
		t.Fatal(err)
	}

	devices := []model.Device{
		{Name: "Modbus Demo", Protocol: model.ProtocolModbus, Address: "127.0.0.1:502", Status: model.DeviceStatusDisconnected},
		{Name: "MQTT Demo", Protocol: model.ProtocolMQTT, Address: "tcp://127.0.0.1:1883", Status: model.DeviceStatusConnected},
	}
	if err := db.Create(&devices).Error; err != nil {
		t.Fatal(err)
	}

	dm := service.NewDriverManager()
	return NewServer(service.NewDeviceService(db, dm))
}

func TestInitialize(t *testing.T) {
	s := NewServer(nil)

	req := &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params: json.RawMessage(`{
			"protocolVersion": "2024-11-05",
			"capabilities": {},
			"clientInfo": {"name": "test", "version": "1.0.0"}
		}`),
	}

	resp, isNotification := s.Handle(t.Context(), req)
	if isNotification {
		t.Fatal("expected response, got notification")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var result InitializeResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatal(err)
	}
	if result.ProtocolVersion != ProtocolVersion20241105 {
		t.Fatalf("expected protocol %s, got %s", ProtocolVersion20241105, result.ProtocolVersion)
	}
	if result.ServerInfo.Name != ServerName {
		t.Fatalf("unexpected server name: %s", result.ServerInfo.Name)
	}
}

func TestToolsList(t *testing.T) {
	s := NewServer(nil)

	req := &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
	}

	resp, _ := s.Handle(t.Context(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var result ToolsListResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Tools) != 5 {
		t.Fatalf("expected 5 tools, got %d", len(result.Tools))
	}

	names := map[string]bool{}
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}
	for _, expected := range []string{"list_devices", "read_data", "write_data", "subscribe_data", "get_device_info"} {
		if !names[expected] {
			t.Fatalf("missing tool %s", expected)
		}
	}
}

func TestInitializedNotification(t *testing.T) {
	s := NewServer(nil)

	req := &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		Method:  "notifications/initialized",
	}

	resp, isNotification := s.Handle(t.Context(), req)
	if !isNotification {
		t.Fatal("expected notification")
	}
	if resp != nil {
		t.Fatal("notification should not produce response")
	}
}

func TestToolsCallUnknownTool(t *testing.T) {
	s := NewServer(nil)

	req := &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      json.RawMessage(`3`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"unknown_tool","arguments":{}}`),
	}

	resp, _ := s.Handle(t.Context(), req)
	resultBytes, _ := json.Marshal(resp.Result)
	var result ToolsCallResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected isError true")
	}
}

func TestToolsCallListDevices(t *testing.T) {
	s := setupTestServer(t)

	req := &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      json.RawMessage(`4`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"list_devices","arguments":{"status":"connected"}}`),
	}

	resp, _ := s.Handle(t.Context(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var result ToolsCallResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "MQTT Demo") {
		t.Fatalf("expected filtered device in result: %s", result.Content[0].Text)
	}
	if strings.Contains(result.Content[0].Text, "Modbus Demo") {
		t.Fatal("disconnected device should be filtered out")
	}
}

func TestToolsCallReadDataNotConnected(t *testing.T) {
	s := setupTestServer(t)

	req := &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      json.RawMessage(`5`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"read_data","arguments":{"device_id":1,"node_id":"holding:0"}}`),
	}

	resp, _ := s.Handle(t.Context(), req)
	resultBytes, _ := json.Marshal(resp.Result)
	var result ToolsCallResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for disconnected device")
	}
}
