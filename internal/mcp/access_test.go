package mcp

import (
	"encoding/json"
	"testing"

	"github.com/indugate/gateway/internal/model"
)

func TestFilterDevices(t *testing.T) {
	devices := []model.Device{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
		{ID: 3, Name: "C"},
	}
	filter := []uint{1, 3}
	result := filterDevices(devices, &filter)
	if len(result) != 2 || result[0].ID != 1 || result[1].ID != 3 {
		t.Fatalf("unexpected filter result: %+v", result)
	}

	empty := []uint{}
	result = filterDevices(devices, &empty)
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %d", len(result))
	}

	all := filterDevices(devices, nil)
	if len(all) != 3 {
		t.Fatalf("expected all devices, got %d", len(all))
	}
}

func TestCanAccessDevice(t *testing.T) {
	filter := []uint{2}
	if !canAccessDevice(2, &filter) {
		t.Fatal("should allow device 2")
	}
	if canAccessDevice(1, &filter) {
		t.Fatal("should deny device 1")
	}
	if !canAccessDevice(99, nil) {
		t.Fatal("nil filter should allow all")
	}
}

func TestToolReadDataAccessDenied(t *testing.T) {
	s := setupTestServer(t)
	filter := []uint{2}
	ctx := WithDeviceFilter(t.Context(), &filter)

	req := &JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      json.RawMessage(`6`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"read_data","arguments":{"device_id":1,"node_id":"holding:0"}}`),
	}

	resp, _ := s.Handle(ctx, req)
	resultBytes, _ := json.Marshal(resp.Result)
	var result ToolsCallResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected access denied error")
	}
}
