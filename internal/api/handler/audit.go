package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/api/middleware"
	"github.com/indugate/gateway/internal/api/response"
	"github.com/indugate/gateway/internal/service"
)

type AuditHandler struct {
	audit *service.AuditService
}

func NewAuditHandler(audit *service.AuditService) *AuditHandler {
	return &AuditHandler{audit: audit}
}

func (h *AuditHandler) List(c *gin.Context) {
	if !middleware.RequireAdmin(c) {
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	var since *time.Time
	if raw := c.Query("since"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			since = &t
		}
	}

	rows, total, err := h.audit.List(c.Request.Context(), service.AuditQuery{
		Username: c.Query("username"),
		Action:   c.Query("action"),
		Limit:    limit,
		Offset:   offset,
		Since:    since,
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c, gin.H{
		"items": rows,
		"total": total,
	})
}
