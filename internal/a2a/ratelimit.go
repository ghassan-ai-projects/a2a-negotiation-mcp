package a2a

import (
	"sync"
	"time"
)

// RateLimiter implements a per-key rate limiter using a fixed window.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int           // requests per window
	window   time.Duration // time window
}

type visitor struct {
	count   int
	resetAt time.Time
}

// NewRateLimiter creates a new RateLimiter.
// rate is the maximum number of requests allowed per window.
// window is the duration of the rate limit window.
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}
}

// Allow checks if a key is allowed to make a request.
// Returns (allowed, remaining) where:
// - allowed is true if the request is within the rate limit
// - remaining is the number of requests remaining in the current window
func (rl *RateLimiter) Allow(key string) (bool, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[key]

	if !exists || now.After(v.resetAt) {
		// First request or window has expired — start a new window
		rl.visitors[key] = &visitor{
			count:   1,
			resetAt: now.Add(rl.window),
		}
		return true, rl.rate - 1
	}

	if v.count >= rl.rate {
		// Rate limit exceeded
		return false, 0
	}

	// Within the window and under the limit
	v.count++
	return true, rl.rate - v.count
}
