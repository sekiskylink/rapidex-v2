package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"basepro/backend/internal/logging"
	"github.com/gin-gonic/gin"
)

const (
	RequestIDHeader   = "X-Request-Id"
	RequestIDContext  = "request_id"
	requestIDByteSize = 16
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(RequestIDHeader))
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set(RequestIDContext, requestID)
		c.Writer.Header().Set(RequestIDHeader, requestID)

		ctxWithID := logging.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctxWithID)
		c.Next()
	}
}

func RequestIDFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if value, ok := c.Get(RequestIDContext); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return logging.RequestIDFromContext(c.Request.Context())
}

func generateRequestID() string {
	buf := make([]byte, requestIDByteSize)
	if _, err := rand.Read(buf); err != nil {
		return "request-id-unavailable"
	}
	return hex.EncodeToString(buf)
}
