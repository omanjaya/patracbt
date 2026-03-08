package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/pkg/logger"
)

// ActivityLogger logs user actions for audit trail.
// Only mutating actions (POST, PUT, DELETE, PATCH) are logged.
func ActivityLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Before request
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		// After request — log important actions
		status := c.Writer.Status()
		userID, _ := c.Get("user_id")
		duration := time.Since(start)

		// Only log mutating actions
		if method != "GET" && method != "OPTIONS" {
			logger.Log.Infow("activity",
				"request_id", c.GetString("request_id"),
				"user_id", userID,
				"method", method,
				"path", path,
				"status", status,
				"duration_ms", duration.Milliseconds(),
				"ip", c.ClientIP(),
			)
		}
	}
}
