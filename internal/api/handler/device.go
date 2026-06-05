package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/api/response"
	"github.com/indugate/gateway/internal/model"
	"github.com/indugate/gateway/internal/service"
)

type DeviceHandler struct {
	svc  *service.DeviceService
	perm *service.DevicePermissionService
}

func NewDeviceHandler(svc *service.DeviceService, perm *service.DevicePermissionService) *DeviceHandler {
	return &DeviceHandler{svc: svc, perm: perm}
}

type createDeviceRequest struct {
	Name        string               `json:"name" binding:"required"`
	Protocol    model.DeviceProtocol `json:"protocol" binding:"required"`
	Address     string               `json:"address" binding:"required"`
	Config      string               `json:"config"`
	Description string               `json:"description"`
}

func (h *DeviceHandler) List(c *gin.Context) {
	devices, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	devices, err = filterDevicesByAccess(c, h.perm, devices)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, devices)
}

func (h *DeviceHandler) Get(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "invalid device id")
		return
	}

	device, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	if !requireDeviceAccess(c, h.perm, device.ID) {
		return
	}
	response.OK(c, device)
}

func (h *DeviceHandler) Create(c *gin.Context) {
	var req createDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	device := &model.Device{
		Name:        req.Name,
		Protocol:    req.Protocol,
		Address:     req.Address,
		Config:      req.Config,
		Description: req.Description,
		Status:      model.DeviceStatusDisconnected,
	}

	if err := h.svc.Create(c.Request.Context(), device); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, device)
}

func (h *DeviceHandler) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "invalid device id")
		return
	}

	var req createDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if !requireDeviceAccess(c, h.perm, id) {
		return
	}

	device, err := h.svc.Update(c.Request.Context(), id, &model.Device{
		Name:        req.Name,
		Protocol:    req.Protocol,
		Address:     req.Address,
		Config:      req.Config,
		Description: req.Description,
	})
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.OK(c, device)
}

func (h *DeviceHandler) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "invalid device id")
		return
	}

	if !requireDeviceAccess(c, h.perm, id) {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *DeviceHandler) Connect(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "invalid device id")
		return
	}

	if !requireDeviceAccess(c, h.perm, id) {
		return
	}

	device, err := h.svc.Connect(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, device)
}

func (h *DeviceHandler) Disconnect(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "invalid device id")
		return
	}

	if !requireDeviceAccess(c, h.perm, id) {
		return
	}

	device, err := h.svc.Disconnect(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, device)
}

func parseID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	return uint(id), err
}
