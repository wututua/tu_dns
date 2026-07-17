package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type ipBucket struct {
	count    int
	resetAt  time.Time
	interval time.Duration
	limit    int
}

type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*ipBucket
	limit    int
	interval time.Duration
}

func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*ipBucket),
		limit:    limit,
		interval: interval,
	}
	go rl.cleanup(5 * interval)
	return rl
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		rl.mu.Lock()
		b, ok := rl.buckets[ip]
		now := time.Now()
		if !ok || now.After(b.resetAt) {
			rl.buckets[ip] = &ipBucket{
				count:    1,
				resetAt:  now.Add(rl.interval),
				interval: rl.interval,
				limit:    rl.limit,
			}
			rl.mu.Unlock()
			c.Next()
			return
		}
		if b.count >= b.limit {
			rl.mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, Body{
				Code:    429,
				Message: "请求过于频繁，请稍后重试",
			})
			return
		}
		b.count++
		rl.mu.Unlock()
		c.Next()
	}
}

func (rl *RateLimiter) cleanup(maxAge time.Duration) {
	ticker := time.NewTicker(maxAge)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, b := range rl.buckets {
			if now.After(b.resetAt) {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}
