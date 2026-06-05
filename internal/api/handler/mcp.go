package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/mcp"
	"github.com/indugate/gateway/internal/service"
)

type MCPHandler struct {
	basePath string
	server   *mcp.Server
	perm     *service.DevicePermissionService
}

func NewMCPHandler(cfg config.MCPConfig, devices *service.DeviceService, perm *service.DevicePermissionService) *MCPHandler {
	return &MCPHandler{
		basePath: cfg.BasePath,
		server:   mcp.NewServer(devices),
		perm:     perm,
	}
}

func (h *MCPHandler) mcpContext(c *gin.Context) context.Context {
	ctx := c.Request.Context()
	if h.perm == nil {
		return ctx
	}
	filter, err := h.perm.ResolveFilter(ctx, accessPrincipal(c))
	if err != nil {
		return ctx
	}
	return mcp.WithDeviceFilter(ctx, filter)
}

func (h *MCPHandler) Discovery(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":            mcp.ServerName,
		"version":         mcp.ServerVersion,
		"protocolVersion": mcp.ProtocolVersion20241105,
		"description":     "Industrial Agent Protocol Gateway MCP Server",
		"transport":       "streamable-http",
		"endpoints": gin.H{
			"message": h.basePath + "/message",
			"sse":     h.basePath + "/sse",
		},
		"capabilities": gin.H{
			"tools":     true,
			"resources": true,
			"prompts":   true,
		},
	})
}

func (h *MCPHandler) SSEStream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("MCP-Protocol-Version", negotiatedVersion(c))

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "streaming not supported")
		return
	}

	endpoint := fmt.Sprintf("%s/message", h.basePath)
	fmt.Fprintf(c.Writer, "event: endpoint\ndata: %s\n\n", endpoint)
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (h *MCPHandler) Message(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeJSONRPCParseError(c, nil, "failed to read request body")
		return
	}
	if len(body) == 0 {
		writeJSONRPCParseError(c, nil, "empty request body")
		return
	}

	var req mcp.JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONRPCParseError(c, nil, "invalid JSON")
		return
	}

	resp, isNotification := h.server.Handle(h.mcpContext(c), &req)
	if isNotification {
		c.Status(http.StatusAccepted)
		return
	}

	acceptSSE := strings.Contains(c.GetHeader("Accept"), "text/event-stream")
	if acceptSSE {
		c.Header("Content-Type", "text/event-stream")
		c.Header("MCP-Protocol-Version", negotiatedVersion(c))
		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			h.writeJSONResponse(c, resp)
			return
		}
		data, _ := json.Marshal(resp)
		fmt.Fprintf(c.Writer, "event: message\ndata: %s\n\n", data)
		flusher.Flush()
		return
	}

	h.writeJSONResponse(c, resp)
}

func (h *MCPHandler) writeJSONResponse(c *gin.Context, resp *mcp.JSONRPCResponse) {
	c.Header("Content-Type", "application/json")
	c.Header("MCP-Protocol-Version", negotiatedVersion(c))
	c.JSON(http.StatusOK, resp)
}

func writeJSONRPCParseError(c *gin.Context, id json.RawMessage, message string) {
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, &mcp.JSONRPCResponse{
		JSONRPC: mcp.JSONRPCVersion,
		ID:      id,
		Error: &mcp.JSONRPCError{
			Code:    -32700,
			Message: message,
		},
	})
}

func negotiatedVersion(c *gin.Context) string {
	if v := c.GetHeader("MCP-Protocol-Version"); v != "" {
		return v
	}
	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "2025") {
		return mcp.ProtocolVersion20250618
	}
	return mcp.ProtocolVersion20241105
}
