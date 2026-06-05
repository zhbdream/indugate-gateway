package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/model"
	"github.com/indugate/gateway/internal/service"
)

func Auth(cfg config.AuthConfig, authSvc *service.AuthService) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if isPublicPath(path) || isAuthPublicPath(path) {
			c.Next()
			return
		}

		token := extractBearerToken(c.GetHeader("Authorization"))
		if token == "" {
			abortUnauthorized(c)
			return
		}

		if cfg.APIToken != "" && token == cfg.APIToken {
			c.Set("role", model.RoleAdmin)
			c.Next()
			return
		}

		if authSvc != nil && authSvc.JWTEnabled() {
			if claims, err := authSvc.ValidateToken(token); err == nil {
				c.Set("user_id", claims.UserID)
				c.Set("username", claims.Username)
				c.Set("role", claims.Role)
				c.Next()
				return
			}
		}

		if cfg.APIToken != "" {
			abortUnauthorized(c)
			return
		}

		if authSvc != nil && authSvc.JWTEnabled() {
			abortUnauthorized(c)
			return
		}

		abortUnauthorized(c)
	}
}

func isAuthPublicPath(path string) bool {
	return path == "/api/v1/auth/login" || path == "/api/v1/auth/config"
}

func abortUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(401, gin.H{
		"code":    401,
		"message": "unauthorized",
	})
}

func isPublicPath(path string) bool {
	if path == "/health" || path == "/metrics" {
		return true
	}
	if strings.HasPrefix(path, "/swagger") {
		return true
	}
	if strings.HasPrefix(path, "/assets/") || path == "/favicon.svg" {
		return true
	}
	return false
}

func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return strings.TrimSpace(header)
}
