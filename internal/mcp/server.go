package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/indugate/gateway/internal/model"
	"github.com/indugate/gateway/internal/service"
)

const serverInstructions = `InduGate is an industrial protocol gateway. Use list_devices to discover devices, then read_data/write_data/subscribe_data on connected devices.

Protocol-specific node_id formats:
- OPC UA: node ID string (e.g. ns=1;s=Temperature)
- Modbus: holding:0, coil:0, input:0, discrete:0
- MQTT: topic path (e.g. factory/device1/telemetry)
- S7: db1:0, db1:4.real, db1:8.dint, m0.0, i0.0, q0.0
- BACnet: analogInput:1, analogOutput:1, binaryInput:1

Devices must be connected before reading/writing. Use the REST API POST /api/v1/devices/{id}/connect if needed.`

type Server struct {
	devices *service.DeviceService
	tools   []Tool
}

func NewServer(devices *service.DeviceService) *Server {
	return &Server{
		devices: devices,
		tools:   toolDefinitions(),
	}
}

func (s *Server) Handle(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, bool) {
	if req.JSONRPC != JSONRPCVersion {
		return errorResponse(req.ID, -32600, "invalid request: jsonrpc must be 2.0"), false
	}
	if req.Method == "" {
		return errorResponse(req.ID, -32600, "invalid request: method is required"), false
	}

	if isNotification(req.ID) {
		s.handleNotification(ctx, req.Method, req.Params)
		return nil, true
	}

	var resp *JSONRPCResponse
	switch req.Method {
	case "initialize":
		resp = s.handleInitialize(req)
	case "ping":
		resp = successResponse(req.ID, PingResult{})
	case "tools/list":
		resp = s.handleToolsList(req)
	case "tools/call":
		resp = s.handleToolsCall(ctx, req)
	case "resources/list":
		resp = s.handleResourcesList(ctx, req)
	case "resources/read":
		resp = s.handleResourcesRead(ctx, req)
	case "prompts/list":
		resp = s.handlePromptsList(req)
	case "prompts/get":
		resp = s.handlePromptsGet(req)
	default:
		resp = errorResponse(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
	return resp, false
}

func (s *Server) handleNotification(_ context.Context, method string, _ json.RawMessage) {
	switch method {
	case "notifications/initialized", "initialized":
		// Client ready signal; no action required for stateless HTTP transport.
	default:
	}
}

func (s *Server) handleInitialize(req *JSONRPCRequest) *JSONRPCResponse {
	var params InitializeParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(req.ID, -32602, "invalid initialize params")
		}
	}

	protocolVersion := negotiateProtocolVersion(params.ProtocolVersion)

	return successResponse(req.ID, InitializeResult{
		ProtocolVersion: protocolVersion,
		Capabilities: ServerCapabilities{
			Tools:     map[string]any{},
			Resources: map[string]any{},
			Prompts:   map[string]any{},
		},
		ServerInfo: Implementation{
			Name:    ServerName,
			Title:   "InduGate Industrial Gateway",
			Version: ServerVersion,
		},
		Instructions: serverInstructions,
	})
}

func negotiateProtocolVersion(requested string) string {
	for _, v := range SupportedProtocolVersions {
		if v == requested {
			return v
		}
	}
	return ProtocolVersion20241105
}

func (s *Server) handleToolsList(req *JSONRPCRequest) *JSONRPCResponse {
	var params ToolsListParams
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params, &params)
	}
	if params.Cursor != "" {
		return successResponse(req.ID, ToolsListResult{Tools: []Tool{}})
	}
	return successResponse(req.ID, ToolsListResult{Tools: s.tools})
}

func (s *Server) handleToolsCall(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	var params ToolsCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, -32602, "invalid tools/call params")
	}
	if params.Name == "" {
		return errorResponse(req.ID, -32602, "tool name is required")
	}

	result, err := s.callTool(ctx, params.Name, params.Arguments)
	if err != nil {
		return successResponse(req.ID, ToolsCallResult{
			Content: []ContentBlock{{Type: "text", Text: err.Error()}},
			IsError: true,
		})
	}
	return successResponse(req.ID, result)
}

func (s *Server) callTool(ctx context.Context, name string, rawArgs json.RawMessage) (ToolsCallResult, error) {
	switch name {
	case "list_devices":
		return s.toolListDevices(ctx, rawArgs)
	case "read_data":
		return s.toolReadData(ctx, rawArgs)
	case "write_data":
		return s.toolWriteData(ctx, rawArgs)
	case "subscribe_data":
		return s.toolSubscribeData(ctx, rawArgs)
	case "get_device_info":
		return s.toolGetDeviceInfo(ctx, rawArgs)
	default:
		return ToolsCallResult{}, fmt.Errorf("unknown tool: %s", name)
	}
}

type listDevicesArgs struct {
	Protocol string `json:"protocol"`
	Status   string `json:"status"`
}

func (s *Server) toolListDevices(ctx context.Context, rawArgs json.RawMessage) (ToolsCallResult, error) {
	var args listDevicesArgs
	if len(rawArgs) > 0 {
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return ToolsCallResult{}, fmt.Errorf("invalid arguments: %w", err)
		}
	}

	devices, err := s.devices.List(ctx)
	if err != nil {
		return ToolsCallResult{}, err
	}
	devices = filterDevices(devices, DeviceFilterFromContext(ctx))

	filtered := make([]model.Device, 0, len(devices))
	for _, d := range devices {
		if args.Protocol != "" && string(d.Protocol) != args.Protocol {
			continue
		}
		if args.Status != "" && string(d.Status) != args.Status {
			continue
		}
		filtered = append(filtered, d)
	}

	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return ToolsCallResult{}, err
	}
	return textResult(string(data)), nil
}

type readDataArgs struct {
	DeviceID uint   `json:"device_id"`
	NodeID   string `json:"node_id"`
}

func (s *Server) toolReadData(ctx context.Context, rawArgs json.RawMessage) (ToolsCallResult, error) {
	var args readDataArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return ToolsCallResult{}, fmt.Errorf("invalid arguments: %w", err)
	}
	if args.DeviceID == 0 {
		return ToolsCallResult{}, fmt.Errorf("device_id is required")
	}
	if !canAccessDevice(args.DeviceID, DeviceFilterFromContext(ctx)) {
		return ToolsCallResult{}, deviceAccessError(args.DeviceID)
	}
	if args.NodeID == "" {
		return ToolsCallResult{}, fmt.Errorf("node_id is required")
	}

	device, err := s.devices.Get(ctx, args.DeviceID)
	if err != nil {
		return ToolsCallResult{}, err
	}

	value, err := s.devices.Drivers().Read(ctx, device, args.NodeID)
	if err != nil {
		return ToolsCallResult{}, err
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ToolsCallResult{}, err
	}
	return textResult(string(data)), nil
}

type writeDataArgs struct {
	DeviceID uint   `json:"device_id"`
	NodeID   string `json:"node_id"`
	Value    any    `json:"value"`
}

func (s *Server) toolWriteData(ctx context.Context, rawArgs json.RawMessage) (ToolsCallResult, error) {
	var args writeDataArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return ToolsCallResult{}, fmt.Errorf("invalid arguments: %w", err)
	}
	if args.DeviceID == 0 {
		return ToolsCallResult{}, fmt.Errorf("device_id is required")
	}
	if !canAccessDevice(args.DeviceID, DeviceFilterFromContext(ctx)) {
		return ToolsCallResult{}, deviceAccessError(args.DeviceID)
	}
	if args.NodeID == "" {
		return ToolsCallResult{}, fmt.Errorf("node_id is required")
	}
	if args.Value == nil {
		return ToolsCallResult{}, fmt.Errorf("value is required")
	}

	device, err := s.devices.Get(ctx, args.DeviceID)
	if err != nil {
		return ToolsCallResult{}, err
	}

	if err := s.devices.Drivers().Write(ctx, device, args.NodeID, args.Value); err != nil {
		return ToolsCallResult{}, err
	}

	result := map[string]any{
		"device_id": args.DeviceID,
		"node_id":   args.NodeID,
		"value":     args.Value,
		"status":    "OK",
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return ToolsCallResult{}, err
	}
	return textResult(string(data)), nil
}

type subscribeDataArgs struct {
	DeviceID   uint     `json:"device_id"`
	NodeIDs    []string `json:"node_ids"`
	IntervalMS int      `json:"interval_ms"`
}

func (s *Server) toolSubscribeData(ctx context.Context, rawArgs json.RawMessage) (ToolsCallResult, error) {
	var args subscribeDataArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return ToolsCallResult{}, fmt.Errorf("invalid arguments: %w", err)
	}
	if args.DeviceID == 0 {
		return ToolsCallResult{}, fmt.Errorf("device_id is required")
	}
	if !canAccessDevice(args.DeviceID, DeviceFilterFromContext(ctx)) {
		return ToolsCallResult{}, deviceAccessError(args.DeviceID)
	}
	if len(args.NodeIDs) == 0 {
		return ToolsCallResult{}, fmt.Errorf("node_ids is required")
	}

	interval := time.Duration(args.IntervalMS) * time.Millisecond
	if interval <= 0 {
		interval = time.Second
	}

	device, err := s.devices.Get(ctx, args.DeviceID)
	if err != nil {
		return ToolsCallResult{}, err
	}

	info, err := s.devices.Drivers().Subscribe(ctx, device, args.NodeIDs, interval)
	if err != nil {
		return ToolsCallResult{}, err
	}

	result := map[string]any{
		"subscription": info,
		"poll_hint":    fmt.Sprintf("Poll events via GET /api/v1/devices/%d/subscriptions/%s/events", args.DeviceID, info.ID),
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return ToolsCallResult{}, err
	}
	return textResult(string(data)), nil
}

type getDeviceInfoArgs struct {
	DeviceID uint `json:"device_id"`
}

func (s *Server) toolGetDeviceInfo(ctx context.Context, rawArgs json.RawMessage) (ToolsCallResult, error) {
	var args getDeviceInfoArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return ToolsCallResult{}, fmt.Errorf("invalid arguments: %w", err)
	}
	if args.DeviceID == 0 {
		return ToolsCallResult{}, fmt.Errorf("device_id is required")
	}
	if !canAccessDevice(args.DeviceID, DeviceFilterFromContext(ctx)) {
		return ToolsCallResult{}, deviceAccessError(args.DeviceID)
	}

	device, err := s.devices.Get(ctx, args.DeviceID)
	if err != nil {
		return ToolsCallResult{}, err
	}

	info := map[string]any{
		"device":       device,
		"connected":    device.Status == model.DeviceStatusConnected,
		"resource_uri": fmt.Sprintf("%s%d", deviceResourcePrefix, device.ID),
	}
	if device.Status == model.DeviceStatusConnected {
		nodes, err := s.devices.Drivers().Browse(ctx, device, "", 1, false)
		if err == nil && len(nodes) > 0 {
			info["sample_nodes"] = nodes
			if len(nodes) > 10 {
				info["sample_nodes"] = nodes[:10]
			}
		}
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return ToolsCallResult{}, err
	}
	return textResult(string(data)), nil
}

func textResult(text string) ToolsCallResult {
	return ToolsCallResult{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

func isNotification(id json.RawMessage) bool {
	return len(id) == 0
}

func successResponse(id json.RawMessage, result any) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  result,
	}
}

func errorResponse(id json.RawMessage, code int, message string) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
}
