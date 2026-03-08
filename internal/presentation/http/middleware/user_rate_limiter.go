package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/pkg/response"
	"golang.org/x/time/rate"
)

// userLimiterStore stores per-user rate limiters keyed by user ID.
type userLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*userLimiterEntry
}

type userLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newUserLimiterStore() *userLimiterStore {
	s := &userLimiterStore{
		limiters: make(map[string]*userLimiterEntry),
	}
	// Cleanup stale entries every 5 minutes.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			s.mu.Lock()
			for uid, entry := range s.limiters {
				if time.Since(entry.lastSeen) > 5*time.Minute {
					delete(s.limiters, uid)
				}
			}
			s.mu.Unlock()
		}
	}()
	return s
}

func (s *userLimiterStore) get(userID string, r rate.Limit, burst int) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Guard against unbounded map growth: evict stale entries when limit exceeded.
	if len(s.limiters) > 5000 {
		now := time.Now()
		for k, e := range s.limiters {
			if now.Sub(e.lastSeen) > 5*time.Minute {
				delete(s.limiters, k)
			}
		}
	}

	entry, ok := s.limiters[userID]
	if !ok {
		entry = &userLimiterEntry{limiter: rate.NewLimiter(r, burst)}
		s.limiters[userID] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

// UserRateLimiter returns a middleware that limits requests per authenticated user.
// It extracts "user_id" from the gin context (set by AuthMiddleware).
// rps = requests per second, burst = max burst size.
func UserRateLimiter(rps float64, burst int) gin.HandlerFunc {
	store := newUserLimiterStore()

	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			// If no user_id in context, fall back to IP-based limiting.
			userID = c.ClientIP()
		}

		var key string
		switch v := userID.(type) {
		case string:
			key = v
		case uint:
			key = uintToString(v)
		case int:
			key = intToString(v)
		default:
			key = c.ClientIP()
		}

		limiter := store.get(key, rate.Limit(rps), burst)
		if !limiter.Allow() {
			response.Error(c, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "Terlalu banyak permintaan, coba lagi nanti")
			c.Abort()
			return
		}
		c.Next()
	}
}

// uintToString converts uint to string without importing strconv.
func uintToString(n uint) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// intToString converts int to string without importing strconv.
func intToString(n int) string {
	if n < 0 {
		return "-" + uintToString(uint(-n))
	}
	return uintToString(uint(n))
}
