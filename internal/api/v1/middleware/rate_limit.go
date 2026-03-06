package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type ipRecord struct {
	mu         sync.Mutex
	timestamps []time.Time
}

// NewIPRateLimiter returns a Gin middleware that enforces a sliding-window
// rate limit per client IP. Requests exceeding `limit` within `window` receive
// HTTP 429 with a Retry-After header.
func NewIPRateLimiter(limit int, window time.Duration) gin.HandlerFunc {
	var store sync.Map
	var once sync.Once

	return func(c *gin.Context) {
		once.Do(func() {
			go cleanupLoop(&store, window)
		})

		ip := getClientIP(c)
		now := time.Now()
		cutoff := now.Add(-window)

		val, _ := store.LoadOrStore(ip, &ipRecord{})
		rec := val.(*ipRecord)

		rec.mu.Lock()
		filtered := rec.timestamps[:0]
		for _, ts := range rec.timestamps {
			if ts.After(cutoff) {
				filtered = append(filtered, ts)
			}
		}

		if len(filtered) >= limit {
			rec.timestamps = filtered
			rec.mu.Unlock()

			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    1001,
				"message": "请求过于频繁，请稍后再试",
			})
			return
		}

		rec.timestamps = append(filtered, now)
		rec.mu.Unlock()
		c.Next()
	}
}

func getClientIP(c *gin.Context) string {
	if ip := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); ip != "" {
		return strings.TrimSpace(strings.SplitN(ip, ",", 2)[0])
	}
	if ip := strings.TrimSpace(c.GetHeader("X-Real-IP")); ip != "" {
		return ip
	}
	return c.ClientIP()
}

func cleanupLoop(store *sync.Map, window time.Duration) {
	ticker := time.NewTicker(2 * window)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-window)
		store.Range(func(key, value any) bool {
			rec := value.(*ipRecord)
			rec.mu.Lock()
			filtered := rec.timestamps[:0]
			for _, ts := range rec.timestamps {
				if ts.After(cutoff) {
					filtered = append(filtered, ts)
				}
			}
			if len(filtered) == 0 {
				rec.mu.Unlock()
				store.Delete(key)
			} else {
				rec.timestamps = filtered
				rec.mu.Unlock()
			}
			return true
		})
	}
}
