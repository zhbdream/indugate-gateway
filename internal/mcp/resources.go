package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const deviceResourcePrefix = "indugate://device/"

func (s *Server) handleResourcesList(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	var params ResourcesListParams
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params, &params)
	}
	if params.Cursor != "" {
		return successResponse(req.ID, ResourcesListResult{Resources: []Resource{}})
	}

	devices, err := s.devices.List(ctx)
	if err != nil {
		return errorResponse(req.ID, -32603, err.Error())
	}
	devices = filterDevices(devices, DeviceFilterFromContext(ctx))

	resources := []Resource{
		{
			URI:         "indugate://devices",
			Name:        "All Devices",
			Description: "List of all configured industrial devices",
			MimeType:    "application/json",
		},
	}
	for _, d := range devices {
		resources = append(resources, Resource{
			URI:         fmt.Sprintf("%s%d", deviceResourcePrefix, d.ID),
			Name:        d.Name,
			Description: fmt.Sprintf("%s device at %s (status: %s)", d.Protocol, d.Address, d.Status),
			MimeType:    "application/json",
		})
	}
	return successResponse(req.ID, ResourcesListResult{Resources: resources})
}

func (s *Server) handleResourcesRead(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	var params ResourcesReadParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, -32602, "invalid resources/read params")
	}
	if params.URI == "" {
		return errorResponse(req.ID, -32602, "uri is required")
	}

	content, err := s.readResource(ctx, params.URI)
	if err != nil {
		return errorResponse(req.ID, -32602, err.Error())
	}
	return successResponse(req.ID, ResourcesReadResult{Contents: []ResourceContent{content}})
}

func (s *Server) readResource(ctx context.Context, uri string) (ResourceContent, error) {
	if uri == "indugate://devices" {
		devices, err := s.devices.List(ctx)
		if err != nil {
			return ResourceContent{}, err
		}
		devices = filterDevices(devices, DeviceFilterFromContext(ctx))
		text, err := json.MarshalIndent(devices, "", "  ")
		if err != nil {
			return ResourceContent{}, err
		}
		return ResourceContent{URI: uri, MimeType: "application/json", Text: string(text)}, nil
	}

	if strings.HasPrefix(uri, deviceResourcePrefix) {
		idStr := strings.TrimPrefix(uri, deviceResourcePrefix)
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			return ResourceContent{}, fmt.Errorf("invalid device uri: %s", uri)
		}
		if !canAccessDevice(uint(id), DeviceFilterFromContext(ctx)) {
			return ResourceContent{}, deviceAccessError(uint(id))
		}
		device, err := s.devices.Get(ctx, uint(id))
		if err != nil {
			return ResourceContent{}, err
		}
		text, err := json.MarshalIndent(device, "", "  ")
		if err != nil {
			return ResourceContent{}, err
		}
		return ResourceContent{URI: uri, MimeType: "application/json", Text: string(text)}, nil
	}

	return ResourceContent{}, fmt.Errorf("resource not found: %s", uri)
}

func (s *Server) handlePromptsList(req *JSONRPCRequest) *JSONRPCResponse {
	var params PromptsListParams
	if len(req.Params) > 0 {
		_ = json.Unmarshal(req.Params, &params)
	}
	if params.Cursor != "" {
		return successResponse(req.ID, PromptsListResult{Prompts: []Prompt{}})
	}
	return successResponse(req.ID, PromptsListResult{Prompts: promptDefinitions()})
}

func (s *Server) handlePromptsGet(req *JSONRPCRequest) *JSONRPCResponse {
	var params PromptsGetParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, -32602, "invalid prompts/get params")
	}
	if params.Name == "" {
		return errorResponse(req.ID, -32602, "name is required")
	}

	for _, p := range promptDefinitions() {
		if p.Name == params.Name {
			result := buildPromptMessages(params.Name, params.Arguments)
			return successResponse(req.ID, result)
		}
	}
	return errorResponse(req.ID, -32602, fmt.Sprintf("prompt not found: %s", params.Name))
}
