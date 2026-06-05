package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/metrics"
)

func Metrics(c *gin.Context) {
	metrics.Handler().ServeHTTP(c.Writer, c.Request)
}
