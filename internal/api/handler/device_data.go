package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/api/response"
	"github.com/indugate/gateway/internal/model"
	opcuadriver "github.com/indugate/gateway/internal/protocol/opcua"
	"github.com/indugate/gateway/internal/service"
)

type DeviceDataHandler struct {
	devices *service.DeviceService
}

func NewDeviceDataHandler(devices *service.DeviceService) *DeviceDataHandler {
	return &DeviceDataHandler{devices: devices}
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

	nodeID := c.Param("nodeId")
	if nodeID == "" {
		response.BadRequest(c, "node id is required")
		return
	}

	value, err := h.devices.Drivers().Read(c.Request.Context(), device, nodeID)
	if err != nil {
		handleDriverError(c, err)
		return
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

	nodeID := c.Param("nodeId")
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
	NodeIDs      []string `json:"node_ids" binding:"required"`
	IntervalMS   int      `json:"interval_ms"`
}

func (h *DeviceDataHandler) Subscribe(c *gin.Context) {
	device, err := h.loadDevice(c)
	if err != nil {
		return
	}
	if device.Protocol != model.ProtocolOPCUA {
		response.BadRequest(c, "subscription is only supported for opcua devices")
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
	return device, nil
}

func handleDriverError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDeviceNotConnected), errors.Is(err, opcuadriver.ErrNotConnected):
		response.Fail(c, 409, 409, err.Error())
	case errors.Is(err, service.ErrUnsupportedProtocol):
		response.BadRequest(c, err.Error())
	case errors.Is(err, opcuadriver.ErrSubscriptionNotFound):
		response.NotFound(c, err.Error())
	default:
		response.InternalError(c, err.Error())
	}
}
