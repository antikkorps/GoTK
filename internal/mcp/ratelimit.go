package mcp

import (
	"sync"
	"time"
)

// rateLimiter implements a token bucket rate limiter.
// Tokens are added at a fixed rate up to a maximum burst size.
// Each request consumes one token; if no tokens are available, the request is rejected.
type rateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// newRateLimiter creates a rate limiter with the given requests per minute and burst size.
// If requestsPerMinute <= 0, rate limiting is disabled (Allow always returns true).
func newRateLimiter(requestsPerMinute int, burst int) *rateLimiter {
	if requestsPerMinute <= 0 {
		return &rateLimiter{maxTokens: 0}
	}
	b := float64(burst)
	if b <= 0 {
		b = 1
	}
	return &rateLimiter{
		tokens:     b,
		maxTokens:  b,
		refillRate: float64(requestsPerMinute) / 60.0,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed under the rate limit.
// Returns true if allowed, false if the rate limit has been exceeded.
func (rl *rateLimiter) Allow() bool {
	// Disabled
	if rl.maxTokens == 0 {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.lastRefill = now

	// Refill tokens
	rl.tokens += elapsed * rl.refillRate
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}

	if rl.tokens < 1 {
		return false
	}

	rl.tokens--
	return true
}
