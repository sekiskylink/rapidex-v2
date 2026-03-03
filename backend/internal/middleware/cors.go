package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

type CORSConfig struct {
	Enabled          bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

// CORS applies configured CORS policy and supports suffix wildcards (e.g.
// "wails://wails.localhost:*") when credentials are disabled.
func CORS(cfg CORSConfig) gin.HandlerFunc {
	normalizedOrigins := make([]string, 0, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			normalizedOrigins = append(normalizedOrigins, trimmed)
		}
	}
	normalizedMethods := cleanList(cfg.AllowedMethods)
	normalizedHeaders := cleanList(cfg.AllowedHeaders)

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allowedOrigin, matched := matchAllowedOrigin(origin, normalizedOrigins)
		if matched {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", allowedOrigin)
			h.Set("Vary", "Origin")
			h.Set("Access-Control-Allow-Methods", strings.Join(normalizedMethods, ","))
			h.Set("Access-Control-Allow-Headers", strings.Join(normalizedHeaders, ","))
			if cfg.AllowCredentials {
				h.Set("Access-Control-Allow-Credentials", "true")
			}
		}

		if c.Request.Method == http.MethodOptions {
			if matched {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Next()
	}
}

func matchAllowedOrigin(origin string, allowedOrigins []string) (string, bool) {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return "*", true
		}

		if strings.HasSuffix(allowed, "*") {
			prefix := strings.TrimSuffix(allowed, "*")
			if strings.HasPrefix(origin, prefix) {
				return origin, true
			}
			continue
		}

		if origin == allowed {
			return origin, true
		}
	}

	return "", false
}

func cleanList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" && !slices.Contains(out, trimmed) {
			out = append(out, trimmed)
		}
	}
	return out
}
