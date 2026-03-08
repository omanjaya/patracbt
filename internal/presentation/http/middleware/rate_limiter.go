package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/pkg/response"
	"golang.org/x/time/rate"
)

// ipLimiter stores per-IP rate limiters.
type ipLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimiterEntry
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var globalLimiter = &ipLimiter{
	limiters: make(map[string]*rateLimiterEntry),
}

func init() {
	// Cleanup stale entries every 5 minutes.
	// Uses time.NewTicker (not time.Tick) so the ticker can be GC'd if needed.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			globalLimiter.mu.Lock()
			for ip, entry := range globalLimiter.limiters {
				if time.Since(entry.lastSeen) > 5*time.Minute {
					delete(globalLimiter.limiters, ip)
				}
			}
			globalLimiter.mu.Unlock()
		}
	}()
}

func (il *ipLimiter) get(ip string, r rate.Limit, burst int) *rate.Limiter {
	il.mu.Lock()
	defer il.mu.Unlock()

	// Guard against unbounded map growth: evict stale entries when limit exceeded.
	if len(il.limiters) > 5000 {
		now := time.Now()
		for k, e := range il.limiters {
			if now.Sub(e.lastSeen) > 5*time.Minute {
				delete(il.limiters, k)
			}
		}
	}

	entry, ok := il.limiters[ip]
	if !ok {
		entry = &rateLimiterEntry{limiter: rate.NewLimiter(r, burst)}
		il.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

// RateLimit returns a middleware that limits requests per IP.
// r = requests per second, burst = max burst.
func RateLimit(r rate.Limit, burst int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := globalLimiter.get(ip, r, burst)
		if !limiter.Allow() {
			response.Error(c, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "Terlalu banyak permintaan, coba lagi nanti")
			c.Abort()
			return
		}
		c.Next()
	}
}
