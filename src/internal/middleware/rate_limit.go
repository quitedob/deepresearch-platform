package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	requestsPerMinute int
	buckets           map[string]*bucket
	mu                sync.RWMutex
	cleanupInterval   time.Duration
}

type bucket struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		requestsPerMinute: requestsPerMinute,
		buckets:           make(map[string]*bucket),
		cleanupInterval:   5 * time.Minute,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// RateLimitMiddleware creates a middleware that limits requests per user
func (rl *RateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user identifier (user ID if authenticated, IP otherwise)
		identifier := c.ClientIP()
		if userID, exists := GetUserID(c); exists {
			identifier = userID
		}

		if !rl.allow(identifier) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// allow checks if a request should be allowed
// 修复：使用更精确的令牌桶算法，避免浮点精度问题
func (rl *RateLimiter) allow(identifier string) bool {
	rl.mu.Lock()
	b, exists := rl.buckets[identifier]
	if !exists {
		b = &bucket{
			tokens:     rl.requestsPerMinute,
			lastRefill: time.Now(),
		}
		rl.buckets[identifier] = b
	}
	rl.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on time elapsed
	// 修复：使用纳秒级精度计算，避免浮点精度问题
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	
	// 计算每纳秒应该添加的令牌数（使用整数运算避免浮点误差）
	// tokensPerMinute / 60秒 / 1e9纳秒 * elapsed纳秒
	// 简化为: elapsed纳秒 * tokensPerMinute / (60 * 1e9)
	elapsedNanos := elapsed.Nanoseconds()
	tokensToAdd := int((elapsedNanos * int64(rl.requestsPerMinute)) / (60 * 1e9))
	
	if tokensToAdd > 0 {
		b.tokens += tokensToAdd
		if b.tokens > rl.requestsPerMinute {
			b.tokens = rl.requestsPerMinute
		}
		b.lastRefill = now
	}

	// Check if we have tokens available
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// cleanup removes old buckets periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for identifier, b := range rl.buckets {
			b.mu.Lock()
			if now.Sub(b.lastRefill) > 10*time.Minute {
				delete(rl.buckets, identifier)
			}
			b.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}
