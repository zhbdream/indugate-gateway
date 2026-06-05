package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/api/response"
	"github.com/indugate/gateway/internal/model"
	"github.com/indugate/gateway/internal/service"
)

type AlertHandler struct {
	alerts *service.AlertService
	perm   *service.DevicePermissionService
}

func NewAlertHandler(alerts *service.AlertService, perm *service.DevicePermissionService) *AlertHandler {
	return &AlertHandler{alerts: alerts, perm: perm}
}

type alertRuleRequest struct {
	DeviceID     uint                 `json:"device_id" binding:"required"`
	NodeID       string               `json:"node_id" binding:"required"`
	Name         string               `json:"name" binding:"required"`
	Enabled      bool                 `json:"enabled"`
	Condition    model.AlertCondition `json:"condition" binding:"required"`
	Threshold    float64              `json:"threshold" binding:"required"`
	ThresholdMax float64              `json:"threshold_max"`
	Level        model.AlertLevel     `json:"level"`
	Description  string               `json:"description"`
}

func (h *AlertHandler) ListRules(c *gin.Context) {
	deviceID, _ := strconv.ParseUint(c.Query("device_id"), 10, 64)
	filter, err := resolveDeviceFilter(c, h.perm)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if deviceID > 0 && !requireDeviceAccess(c, h.perm, uint(deviceID)) {
		return
	}
	rules, err := h.alerts.ListRules(c.Request.Context(), uint(deviceID), filter)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, rules)
}

func (h *AlertHandler) CreateRule(c *gin.Context) {
	var req alertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if !requireDeviceAccess(c, h.perm, req.DeviceID) {
		return
	}
	rule := &model.AlertRule{
		DeviceID:     req.DeviceID,
		NodeID:       req.NodeID,
		Name:         req.Name,
		Enabled:      req.Enabled,
		Condition:    req.Condition,
		Threshold:    req.Threshold,
		ThresholdMax: req.ThresholdMax,
		Level:        req.Level,
		Description:  req.Description,
	}
	if err := h.alerts.CreateRule(c.Request.Context(), rule); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, rule)
}

func (h *AlertHandler) UpdateRule(c *gin.Context) {
	id, err := parseAlertID(c)
	if err != nil {
		response.BadRequest(c, "invalid rule id")
		return
	}
	existing, err := h.alerts.GetRule(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	if !requireDeviceAccess(c, h.perm, existing.DeviceID) {
		return
	}
	var req alertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	rule, err := h.alerts.UpdateRule(c.Request.Context(), id, &model.AlertRule{
		NodeID:       req.NodeID,
		Name:         req.Name,
		Enabled:      req.Enabled,
		Condition:    req.Condition,
		Threshold:    req.Threshold,
		ThresholdMax: req.ThresholdMax,
		Level:        req.Level,
		Description:  req.Description,
	})
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.OK(c, rule)
}

func (h *AlertHandler) DeleteRule(c *gin.Context) {
	id, err := parseAlertID(c)
	if err != nil {
		response.BadRequest(c, "invalid rule id")
		return
	}
	existing, err := h.alerts.GetRule(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	if !requireDeviceAccess(c, h.perm, existing.DeviceID) {
		return
	}
	if err := h.alerts.DeleteRule(c.Request.Context(), id); err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *AlertHandler) ListEvents(c *gin.Context) {
	deviceID, _ := strconv.ParseUint(c.Query("device_id"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	filter, err := resolveDeviceFilter(c, h.perm)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if deviceID > 0 && !requireDeviceAccess(c, h.perm, uint(deviceID)) {
		return
	}
	events, err := h.alerts.ListEvents(c.Request.Context(), uint(deviceID), c.Query("status"), limit, filter)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, events)
}

func (h *AlertHandler) AcknowledgeEvent(c *gin.Context) {
	id, err := parseAlertID(c)
	if err != nil {
		response.BadRequest(c, "invalid event id")
		return
	}
	existing, err := h.alerts.GetEvent(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	if !requireDeviceAccess(c, h.perm, existing.DeviceID) {
		return
	}
	event, err := h.alerts.AcknowledgeEvent(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.OK(c, event)
}

type DashboardHandler struct {
	alerts *service.AlertService
	perm   *service.DevicePermissionService
}

func NewDashboardHandler(alerts *service.AlertService, perm *service.DevicePermissionService) *DashboardHandler {
	return &DashboardHandler{alerts: alerts, perm: perm}
}

func (h *DashboardHandler) Stats(c *gin.Context) {
	filter, err := resolveDeviceFilter(c, h.perm)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	stats, err := h.alerts.DashboardStats(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, stats)
}

func parseAlertID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	return uint(id), err
}
