// Package mcp provides MCP (Model Context Protocol) server implementation.
package mcp

// Server represents the MCP protocol server (placeholder).
type Server struct {
	Enabled  bool
	BasePath string
}

// New creates a new MCP server instance.
func New(enabled bool, basePath string) *Server {
	return &Server{
		Enabled:  enabled,
		BasePath: basePath,
	}
}
