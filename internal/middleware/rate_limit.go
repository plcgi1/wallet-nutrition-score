package middleware

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"wallet-nutrition-score/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RateLimiter represents a simple in-memory rate limiter
type RateLimiter struct {
	mu          sync.Mutex
	clientCount map[string]struct {
		count     int
		resetTime time.Time
	}
	Requests int
	Window   time.Duration
	log      *logrus.Logger
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(cfg *config.Config, log *logrus.Logger) *RateLimiter {
	return &RateLimiter{
		clientCount: make(map[string]struct {
			count     int
			resetTime time.Time
		}),
		Requests: cfg.App.RateLimit.Requests,
		Window:   time.Duration(cfg.App.RateLimit.Window) * time.Second,
		log:      log,
	}
}

// RateLimitMiddleware returns a Gin middleware that implements rate limiting
func (rl *RateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP address
		clientIP := c.ClientIP()
		if clientIP == "" {
			// Fallback to X-Forwarded-For header
			clientIP = c.Request.Header.Get("X-Forwarded-For")
			if clientIP == "" {
				clientIP = "unknown"
			}
		}

		rl.mu.Lock()
		defer rl.mu.Unlock()

		// Cleanup old entries if window has passed
		now := time.Now()

		// Check if client exists in map
		entry, exists := rl.clientCount[clientIP]

		// Client doesn't exist - create new entry
		if !exists {
			rl.clientCount[clientIP] = struct {
				count     int
				resetTime time.Time
			}{
				count:     1,
				resetTime: now.Add(rl.Window),
			}
			rl.setHeaders(c, clientIP)
			c.Next()
			return
		}

		// Client exists, check if window has expired
		if now.After(entry.resetTime) {
			rl.clientCount[clientIP] = struct {
				count     int
				resetTime time.Time
			}{
				count:     1,
				resetTime: now.Add(rl.Window),
			}
			rl.setHeaders(c, clientIP)
			c.Next()
			return
		}

		// Client exists and window is active, check rate limit
		if entry.count >= rl.Requests {
			rl.log.Warnf("Rate limit exceeded for IP: %s", clientIP)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too Many Requests",
				"message": fmt.Sprintf("Rate limit exceeded. Please try again in %0.0f seconds.", entry.resetTime.Sub(now).Seconds()),
			})
			c.Abort()
			return
		}

		// All checks passed - increment count
		entry.count++
		rl.clientCount[clientIP] = entry
		rl.setHeaders(c, clientIP)
		c.Next()
	}
}

// setHeaders sets the rate limit response headers
func (rl *RateLimiter) setHeaders(c *gin.Context, clientIP string) {
	c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.Requests))
	c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", rl.Requests-rl.clientCount[clientIP].count))
	c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", rl.clientCount[clientIP].resetTime.Unix()))
}

// IPToKey converts an IP address to a string key, handling IPv6 addresses
func IPToKey(ip string) string {
	if ip == "" {
		return "unknown"
	}

	// Handle IPv6 addresses
	parsedIP := net.ParseIP(ip)
	if parsedIP != nil && parsedIP.To4() == nil {
		// IPv6 address, format it properly
		return "[" + ip + "]"
	}

	return ip
}

// ClearExpiredEntries clears expired rate limit entries
func (rl *RateLimiter) ClearExpiredEntries() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, entry := range rl.clientCount {
		if now.After(entry.resetTime) {
			delete(rl.clientCount, ip)
		}
	}
}
