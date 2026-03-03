package middleware

import (
	"math"
	"strings"
	"sync"
	"time"

	"basepro/backend/internal/apperror"
	"github.com/gin-gonic/gin"
)

type RateLimitConfig struct {
	Enabled           bool
	RequestsPerSecond float64
	Burst             int
	EntryTTL          time.Duration
	CleanupInterval   time.Duration
}

type authRateLimiter struct {
	config      RateLimitConfig
	mu          sync.Mutex
	clients     map[string]*clientLimiter
	lastCleanup time.Time
}

type clientLimiter struct {
	tokens     float64
	lastSeen   time.Time
	lastRefill time.Time
}

func NewAuthRateLimiter(cfg RateLimitConfig) *authRateLimiter {
	if cfg.EntryTTL <= 0 {
		cfg.EntryTTL = 10 * time.Minute
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = 1 * time.Minute
	}
	return &authRateLimiter{
		config:      cfg,
		clients:     make(map[string]*clientLimiter),
		lastCleanup: time.Now().UTC(),
	}
}

func (r *authRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !r.config.Enabled {
			c.Next()
			return
		}
		if !r.allow(c.ClientIP()) {
			apperror.Write(c, apperror.RateLimited("Too many requests"))
			c.Abort()
			return
		}
		c.Next()
	}
}

func (r *authRateLimiter) allow(rawClientIP string) bool {
	clientIP := strings.TrimSpace(rawClientIP)
	if clientIP == "" {
		clientIP = "unknown"
	}
	now := time.Now().UTC()

	r.mu.Lock()
	defer r.mu.Unlock()

	if now.Sub(r.lastCleanup) >= r.config.CleanupInterval {
		for key, limiter := range r.clients {
			if now.Sub(limiter.lastSeen) > r.config.EntryTTL {
				delete(r.clients, key)
			}
		}
		r.lastCleanup = now
	}

	client, ok := r.clients[clientIP]
	if !ok {
		client = &clientLimiter{
			tokens:     float64(r.config.Burst),
			lastRefill: now,
		}
		r.clients[clientIP] = client
	}
	client.lastSeen = now
	elapsed := now.Sub(client.lastRefill).Seconds()
	if elapsed > 0 {
		client.tokens = math.Min(float64(r.config.Burst), client.tokens+elapsed*r.config.RequestsPerSecond)
		client.lastRefill = now
	}
	if client.tokens < 1 {
		return false
	}
	client.tokens--
	return true
}
