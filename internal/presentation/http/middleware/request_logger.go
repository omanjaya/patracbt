package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/pkg/logger"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		// Skip health check spam
		if path == "/api/v1/health" {
			return
		}

		logger.Log.Infow("request",
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"ip", c.ClientIP(),
			"request_id", c.GetString("request_id"),
		)
	}
}
