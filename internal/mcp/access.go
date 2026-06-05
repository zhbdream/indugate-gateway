package mcp

import (
	"context"
	"fmt"

	"github.com/indugate/gateway/internal/model"
)

type deviceFilterKey struct{}

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
