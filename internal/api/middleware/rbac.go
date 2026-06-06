package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/model"
)

func RBAC(cfg config.AuthConfig) gin.HandlerFunc {
	if !cfg.Enabled || !cfg.RBACEnabled {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if isPublicPath(path) || isAuthPublicPath(path) || path == "/metrics" {
			c.Next()
			return
		}

		role := roleFromContext(c)
		if role == "" {
			c.Next()
			return
		}

		if allowedRole(role, c.Request.Method, path) {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "forbidden",
		})
	}
}

func roleFromContext(c *gin.Context) model.UserRole {
	if v, ok := c.Get("role"); ok {
		if role, ok := v.(model.UserRole); ok {
			return role
		}
	}
	return ""
}

func allowedRole(role model.UserRole, method, path string) bool {
	if role == model.RoleAdmin {
		return true
	}

	if strings.HasPrefix(path, "/api/v1/users") {
		return false
	}

	// MCP uses POST for JSON-RPC; viewers need read-only tool access.
	if strings.HasPrefix(path, "/mcp") {
		switch role {
		case model.RoleOperator:
			return true
		case model.RoleViewer:
			return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions || method == http.MethodPost
		default:
			return false
		}
	}

	switch role {
	case model.RoleOperator:
		return true
	case model.RoleViewer:
		return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
	default:
		return false
	}
}

func RequireAdmin(c *gin.Context) bool {
	role := roleFromContext(c)
	if role == model.RoleAdmin {
		return true
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"code":    403,
		"message": "admin role required",
	})
	return false
}
