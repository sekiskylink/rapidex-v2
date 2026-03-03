package middleware

import (
	"log/slog"
	"time"

	"basepro/backend/internal/logging"
	"github.com/gin-gonic/gin"
)

func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		duration := time.Since(start)
		requestID := RequestIDFromContext(c)
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		logging.ForContext(c.Request.Context()).Info("http_request",
			slog.String("request_id", requestID),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", c.Writer.Status()),
			slog.Int64("duration_ms", duration.Milliseconds()),
		)
	}
}
