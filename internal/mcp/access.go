package mcp

import (
	"context"
	"fmt"

	"github.com/indugate/gateway/internal/model"
)

var viewerReadOnlyTools = map[string]struct{}{
	"list_devices":   {},
	"read_data":      {},
	"subscribe_data": {},
	"get_device_info": {},
}

type deviceFilterKey struct{}
type roleKey struct{}

func WithDeviceFilter(ctx context.Context, filter *[]uint) context.Context {
	return context.WithValue(ctx, deviceFilterKey{}, filter)
}

func DeviceFilterFromContext(ctx context.Context) *[]uint {
	if ctx == nil {
		return nil
	}
	filter, _ := ctx.Value(deviceFilterKey{}).(*[]uint)
	return filter
}

func filterDevices(devices []model.Device, filter *[]uint) []model.Device {
	if filter == nil {
		return devices
	}
	if len(*filter) == 0 {
		return []model.Device{}
	}
	allowed := make(map[uint]struct{}, len(*filter))
	for _, id := range *filter {
		allowed[id] = struct{}{}
	}
	result := make([]model.Device, 0, len(devices))
	for _, d := range devices {
		if _, ok := allowed[d.ID]; ok {
			result = append(result, d)
		}
	}
	return result
}

func canAccessDevice(deviceID uint, filter *[]uint) bool {
	if filter == nil {
		return true
	}
	if len(*filter) == 0 {
		return false
	}
	for _, id := range *filter {
		if id == deviceID {
			return true
		}
	}
	return false
}

func deviceAccessError(deviceID uint) error {
	return fmt.Errorf("device access denied: %d", deviceID)
}

func WithRole(ctx context.Context, role model.UserRole) context.Context {
	return context.WithValue(ctx, roleKey{}, role)
}

func RoleFromContext(ctx context.Context) model.UserRole {
	if ctx == nil {
		return ""
	}
	role, _ := ctx.Value(roleKey{}).(model.UserRole)
	return role
}

func canCallTool(ctx context.Context, toolName string) error {
	role := RoleFromContext(ctx)
	if role != model.RoleViewer {
		return nil
	}
	if _, ok := viewerReadOnlyTools[toolName]; ok {
		return nil
	}
	return fmt.Errorf("tool %q is not allowed for viewer role", toolName)
}

func toolsForRole(role model.UserRole, tools []Tool) []Tool {
	if role != model.RoleViewer {
		return tools
	}
	filtered := make([]Tool, 0, len(tools))
	for _, tool := range tools {
		if _, ok := viewerReadOnlyTools[tool.Name]; ok {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}
