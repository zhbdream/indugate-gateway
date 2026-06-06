package handler

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/api/response"
	"github.com/indugate/gateway/internal/model"
	bacnetdriver "github.com/indugate/gateway/internal/protocol/bacnet"
	modbusdriver "github.com/indugate/gateway/internal/protocol/modbus"
	mqttdriver "github.com/indugate/gateway/internal/protocol/mqtt"
	opcuadriver "github.com/indugate/gateway/internal/protocol/opcua"
	s7driver "github.com/indugate/gateway/internal/protocol/s7"
	"github.com/indugate/gateway/internal/service"
)

type DeviceDataHandler struct {
	devices  *service.DeviceService
	recorder *service.HistoryRecorder
	perm     *service.DevicePermissionService
}

func NewDeviceDataHandler(devices *service.DeviceService, recorder *service.HistoryRecorder, perm *service.DevicePermissionService) *DeviceDataHandler {
	return &DeviceDataHandler{devices: devices, recorder: recorder, perm: perm}
}

func (h *DeviceDataHandler) BrowseNodes(c *gin.Context) {
	device, err := h.loadDevice(c)
	if err != nil {
		return
	}

	nodeID := c.DefaultQuery("node", "i=85")
	depth, _ := strconv.Atoi(c.DefaultQuery("depth", "3"))
	childrenOnly := c.Query("children_only") == "true"

	nodes, err := h.devices.Drivers().Browse(c.Request.Context(), device, nodeID, depth, childrenOnly)
	if err != nil {
		handleDriverError(c, err)
		return
	}
	response.OK(c, nodes)
}

func (h *DeviceDataHandler) ReadData(c *gin.Context) {
	device, err := h.loadDevice(c)
	if err != nil {
		return
	}

	nodeID := nodeIDFromRequest(c)
	if nodeID == "" {
		response.BadRequest(c, "node id is required")
		return
	}

	value, err := h.devices.Drivers().Read(c.Request.Context(), device, nodeID)
	if err != nil {
		handleDriverError(c, err)
		return
	}
	if h.recorder != nil {
		h.recorder.RecordRead(c.Request.Context(), device.ID, value)
	}
	response.OK(c, value)
}

type writeDataRequest struct {
	Value any `json:"value" binding:"required"`
}

func (h *DeviceDataHandler) WriteData(c *gin.Context) {
	device, err := h.loadDevice(c)
	if err != nil {
		return
	}

	nodeID := nodeIDFromRequest(c)
	if nodeID == "" {
		response.BadRequest(c, "node id is required")
		return
	}

	var req writeDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.devices.Drivers().Write(c.Request.Context(), device, nodeID, req.Value); err != nil {
		handleDriverError(c, err)
		return
	}
	response.OK(c, gin.H{"node_id": nodeID, "value": req.Value})
}

type subscribeRequest struct {
	NodeIDs    []string `json:"node_ids" binding:"required"`
	IntervalMS int      `json:"interval_ms"`
}

func (h *DeviceDataHandler) Subscribe(c *gin.Context) {
	device, err := h.loadDevice(c)
	if err != nil {
		return
	}
	var req subscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	interval := time.Duration(req.IntervalMS) * time.Millisecond
	info, err := h.devices.Drivers().Subscribe(c.Request.Context(), device, req.NodeIDs, interval)
	if err != nil {
		handleDriverError(c, err)
		return
	}
	response.Created(c, info)
}

func (h *DeviceDataHandler) PollSubscription(c *gin.Context) {
	deviceID, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "invalid device id")
		return
	}

	subID := c.Param("subId")
	clear := c.DefaultQuery("clear", "true") == "true"

	events, err := h.devices.Drivers().PollSubscription(deviceID, subID, clear)
	if err != nil {
		handleDriverError(c, err)
		return
	}
	if h.recorder != nil {
		ctx := c.Request.Context()
		for _, evt := range events {
			h.recorder.RecordEvent(ctx, deviceID, evt)
		}
	}
	response.OK(c, events)
}

func (h *DeviceDataHandler) Unsubscribe(c *gin.Context) {
	deviceID, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "invalid device id")
		return
	}

	subID := c.Param("subId")
	if err := h.devices.Drivers().Unsubscribe(deviceID, subID); err != nil {
		handleDriverError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *DeviceDataHandler) ListSubscriptions(c *gin.Context) {
	deviceID, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "invalid device id")
		return
	}
	if !requireDeviceAccess(c, h.perm, deviceID) {
		return
	}

	subs, err := h.devices.Drivers().ListSubscriptions(deviceID)
	if err != nil {
		handleDriverError(c, err)
		return
	}
	response.OK(c, subs)
}

func (h *DeviceDataHandler) loadDevice(c *gin.Context) (*model.Device, error) {
	id, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "invalid device id")
		return nil, err
	}

	device, err := h.devices.Get(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, err.Error())
		return nil, err
	}
	if !requireDeviceAccess(c, h.perm, device.ID) {
		return nil, service.ErrDeviceAccessDenied
	}
	return device, nil
}

func nodeIDFromRequest(c *gin.Context) string {
	if node := c.Query("node"); node != "" {
		return node
	}
	nodeID := c.Param("nodeId")
	return strings.TrimPrefix(nodeID, "/")
}

func handleDriverError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDeviceNotConnected),
		errors.Is(err, opcuadriver.ErrNotConnected),
		errors.Is(err, modbusdriver.ErrNotConnected),
		errors.Is(err, mqttdriver.ErrNotConnected),
		errors.Is(err, s7driver.ErrNotConnected),
		errors.Is(err, bacnetdriver.ErrNotConnected):
		response.Fail(c, 409, 409, err.Error())
	case errors.Is(err, service.ErrUnsupportedProtocol):
		response.BadRequest(c, err.Error())
	case errors.Is(err, opcuadriver.ErrSubscriptionNotFound),
		errors.Is(err, modbusdriver.ErrSubscriptionNotFound),
		errors.Is(err, mqttdriver.ErrSubscriptionNotFound),
		errors.Is(err, s7driver.ErrSubscriptionNotFound),
		errors.Is(err, bacnetdriver.ErrSubscriptionNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, mqttdriver.ErrTopicNotFound):
		response.NotFound(c, err.Error())
	case strings.Contains(err.Error(), "read failed:"),
		strings.Contains(err.Error(), "write failed:"):
		response.BadRequest(c, err.Error())
	default:
		response.InternalError(c, err.Error())
	}
}
