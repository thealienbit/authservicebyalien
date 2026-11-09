package http

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	limiterCleanupInterval = 5 * time.Minute
	limiterTTL             = 10 * time.Minute
)

// KeyedRateLimiter keeps one bucket per key. Keys are derived from the request
// (client IP, or app + client IP) by RateLimitMiddleware.
type KeyedRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
	rate     rate.Limit
	burst    int
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewKeyedRateLimiter(r rate.Limit, b int) *KeyedRateLimiter {
	l := &KeyedRateLimiter{
		limiters: make(map[string]*limiterEntry),
		rate:     r,
		burst:    b,
	}

	go l.cleanup()

	return l
}

func (l *KeyedRateLimiter) cleanup() {
	ticker := time.NewTicker(limiterCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-limiterTTL)

		l.mu.Lock()
		for key, entry := range l.limiters {
			if entry.lastSeen.Before(cutoff) {
				delete(l.limiters, key)
			}
		}
		l.mu.Unlock()
	}
}

func (l *KeyedRateLimiter) limiter(key string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.limiters[key]
	if !ok {
		entry = &limiterEntry{limiter: rate.NewLimiter(l.rate, l.burst)}
		l.limiters[key] = entry
	}
	entry.lastSeen = time.Now()

	return entry.limiter
}

func RateLimitMiddleware(limiter *KeyedRateLimiter, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.limiter(keyFn(r)).Allow() {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
