package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/model"
	"github.com/indugate/gateway/internal/service"
)

func Audit(auditSvc *service.AuditService, cfg config.AuditConfig, authCfg config.AuthConfig) gin.HandlerFunc {
	if !cfg.Enabled || !authCfg.Enabled || auditSvc == nil {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		c.Next()

		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			return
		}

		path := c.Request.URL.Path
		if !strings.HasPrefix(path, "/api/v1/") {
			return
		}

		status := c.Writer.Status()
		entry := &model.AuditLog{
			Username:   actorUsername(c),
			Role:       roleFromContext(c),
			Method:     method,
			Path:       path,
			Action:     service.DeriveAuditAction(method, path),
			Detail:     c.Request.URL.RawQuery,
			ClientIP:   c.ClientIP(),
			StatusCode: status,
			Success:    status >= 200 && status < 400,
		}

		_ = auditSvc.Record(c.Request.Context(), entry)
	}
}

func actorUsername(c *gin.Context) string {
	if v, ok := c.Get("username"); ok {
		if username, ok := v.(string); ok && username != "" {
			return username
		}
	}
	if roleFromContext(c) == model.RoleAdmin {
		return "api_token"
	}
	return "anonymous"
}
