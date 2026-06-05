package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/api/response"
	"github.com/indugate/gateway/internal/service"
)

type HistoryHandler struct {
	history *service.HistoryService
	devices *service.DeviceService
	perm    *service.DevicePermissionService
}

func NewHistoryHandler(history *service.HistoryService, devices *service.DeviceService, perm *service.DevicePermissionService) *HistoryHandler {
	return &HistoryHandler{history: history, devices: devices, perm: perm}
}

func (h *HistoryHandler) QueryHistory(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "invalid device id")
		return
	}

	if _, err := h.devices.Get(c.Request.Context(), id); err != nil {
		response.NotFound(c, err.Error())
		return
	}
	if !requireDeviceAccess(c, h.perm, id) {
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	nodeID := c.Query("node_id")

	var since *time.Time
	if sinceStr := c.Query("since"); sinceStr != "" {
		t, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			response.BadRequest(c, "invalid since timestamp, use RFC3339")
			return
		}
		since = &t
	}

	rows, err := h.history.Query(c.Request.Context(), service.HistoryQuery{
		DeviceID: id,
		NodeID:   nodeID,
		Limit:    limit,
		Since:    since,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, rows)
}

func (h *HistoryHandler) ExportCSV(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.BadRequest(c, "invalid device id")
		return
	}

	if _, err := h.devices.Get(c.Request.Context(), id); err != nil {
		response.NotFound(c, err.Error())
		return
	}
	if !requireDeviceAccess(c, h.perm, id) {
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1000"))
	nodeID := c.Query("node_id")

	var since *time.Time
	if sinceStr := c.Query("since"); sinceStr != "" {
		t, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			response.BadRequest(c, "invalid since timestamp, use RFC3339")
			return
		}
		since = &t
	}

	rows, err := h.history.Query(c.Request.Context(), service.HistoryQuery{
		DeviceID: id,
		NodeID:   nodeID,
		Limit:    limit,
		Since:    since,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=history.csv")
	c.Writer.WriteString("timestamp,node_id,value,data_type,status\n")
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		line := fmt.Sprintf("%s,%s,%s,%s,%s\n",
			row.Timestamp.Format(time.RFC3339),
			escapeCSV(row.NodeID),
			escapeCSV(row.Value),
			escapeCSV(row.DataType),
			escapeCSV(row.Status),
		)
		c.Writer.WriteString(line)
	}
}

func escapeCSV(s string) string {
	if s == "" {
		return ""
	}
	for _, ch := range s {
		if ch == ',' || ch == '"' || ch == '\n' {
			return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
		}
	}
	return s
}
