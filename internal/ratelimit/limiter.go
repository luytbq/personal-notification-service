package ratelimit

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter implements a token bucket rate limiter per API key and channel
type Limiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit // requests per second
	burst    int        // bucket size
}

// NewLimiter creates a new rate limiter
// requestsPerMinute: maximum requests allowed per minute per key+channel combination
func NewLimiter(requestsPerMinute int) *Limiter {
	return &Limiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(float64(requestsPerMinute) / 60.0), // convert to per-second
		burst:    requestsPerMinute,                             // allow burst up to the full minute limit
	}
}

// key generates a unique key for the API key and channel combination
func (l *Limiter) key(apiKey, channel string) string {
	return fmt.Sprintf("%s:%s", apiKey, channel)
}

// getLimiter returns the rate limiter for a given API key and channel, creating one if needed
func (l *Limiter) getLimiter(apiKey, channel string) *rate.Limiter {
	key := l.key(apiKey, channel)

	// Try read lock first for better performance
	l.mu.RLock()
	limiter, exists := l.limiters[key]
	l.mu.RUnlock()

	if exists {
		return limiter
	}

	// Create new limiter with write lock
	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists = l.limiters[key]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(l.rate, l.burst)
	l.limiters[key] = limiter
	return limiter
}

// Allow checks if a request is allowed for the given API key and channel
// Returns true if the request is allowed, false if rate limited
func (l *Limiter) Allow(apiKey, channel string) bool {
	return l.getLimiter(apiKey, channel).Allow()
}

// AllowN checks if n requests are allowed for the given API key and channel
func (l *Limiter) AllowN(apiKey, channel string, n int) bool {
	return l.getLimiter(apiKey, channel).AllowN(time.Now(), n)
}
