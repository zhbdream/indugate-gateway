package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/api/response"
	"github.com/indugate/gateway/internal/model"
	"github.com/indugate/gateway/internal/service"
)

func accessPrincipal(c *gin.Context) service.AccessPrincipal {
	p := service.AccessPrincipal{}
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(uint); ok {
			p.UserID = id
		}
	}
	if v, ok := c.Get("username"); ok {
		if username, ok := v.(string); ok {
			p.Username = username
		}
	}
	if v, ok := c.Get("role"); ok {
		if role, ok := v.(model.UserRole); ok {
			p.Role = role
		}
	}
	return p
}

func resolveDeviceFilter(c *gin.Context, perm *service.DevicePermissionService) (*[]uint, error) {
	if perm == nil {
		return nil, nil
	}
	return perm.ResolveFilter(c.Request.Context(), accessPrincipal(c))
}

func requireDeviceAccess(c *gin.Context, perm *service.DevicePermissionService, deviceID uint) bool {
	if perm == nil {
		return true
	}
	err := perm.RequireAccess(c.Request.Context(), accessPrincipal(c), deviceID)
	if err == nil {
		return true
	}
	if errors.Is(err, service.ErrDeviceAccessDenied) {
		response.Fail(c, 403, 403, err.Error())
		return false
	}
	response.InternalError(c, err.Error())
	return false
}

func filterDevicesByAccess(c *gin.Context, perm *service.DevicePermissionService, devices []model.Device) ([]model.Device, error) {
	filter, err := resolveDeviceFilter(c, perm)
	if err != nil {
		return nil, err
	}
	return perm.FilterDevices(c.Request.Context(), filter, devices), nil
}
