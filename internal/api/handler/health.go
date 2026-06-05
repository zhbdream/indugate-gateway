package handler

import (
	"net/http"

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

type MCPHandler struct {
	basePath string
}

func NewMCPHandler(basePath string) *MCPHandler {
	return &MCPHandler{basePath: basePath}
}

func (h *MCPHandler) Discovery(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":        "InduGate MCP Server",
		"version":     "0.1.0",
		"description": "Industrial Agent Protocol Gateway MCP Server",
		"base_path":   h.basePath,
		"capabilities": gin.H{
			"tools":     true,
			"resources": true,
			"prompts":   false,
		},
	})
}

func (h *MCPHandler) SSE(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "MCP SSE endpoint - implementation pending",
	})
}

func (h *MCPHandler) Message(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "MCP message endpoint - implementation pending",
	})
}
