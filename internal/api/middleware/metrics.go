package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/metrics"
)

func Prometheus() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		metrics.ObserveHTTPRequest()
	}
}
