package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/api/response"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Check(c *gin.Context) {
	response.OK(c, gin.H{
		"status":  "up",
		"service": "indugate-gateway",
	})
}
