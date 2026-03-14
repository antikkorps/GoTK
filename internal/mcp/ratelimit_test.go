package mcp

import (
	"testing"
	"time"
)

func TestRateLimiterDisabled(t *testing.T) {
	rl := newRateLimiter(0, 10)
	for i := 0; i < 100; i++ {
		if !rl.Allow() {
			t.Fatalf("disabled limiter should always allow, rejected at request %d", i)
		}
	}
}

func TestRateLimiterNegativeRate(t *testing.T) {
	rl := newRateLimiter(-5, 10)
	for i := 0; i < 100; i++ {
		if !rl.Allow() {
			t.Fatalf("negative rate limiter should always allow, rejected at request %d", i)
		}
	}
}

func TestRateLimiterBurst(t *testing.T) {
	rl := newRateLimiter(60, 5)
	// Should allow burst of 5
	for i := 0; i < 5; i++ {
		if !rl.Allow() {
			t.Fatalf("should allow burst request %d", i)
		}
	}
	// 6th should be rejected
	if rl.Allow() {
		t.Fatal("should reject request exceeding burst")
	}
}

func TestRateLimiterRefill(t *testing.T) {
	rl := newRateLimiter(600, 2) // 10 per second
	// Exhaust burst
	rl.Allow()
	rl.Allow()
	if rl.Allow() {
		t.Fatal("should be exhausted after burst")
	}

	// Simulate time passing by manipulating lastRefill
	rl.mu.Lock()
	rl.lastRefill = time.Now().Add(-1 * time.Second)
	rl.mu.Unlock()

	// Should have refilled ~10 tokens (capped at burst=2)
	if !rl.Allow() {
		t.Fatal("should allow after refill")
	}
	if !rl.Allow() {
		t.Fatal("should allow second request after refill (burst=2)")
	}
	if rl.Allow() {
		t.Fatal("should reject third request (burst cap=2)")
	}
}

func TestRateLimiterDefaultBurst(t *testing.T) {
	rl := newRateLimiter(60, 0) // burst 0 should default to 1
	if !rl.Allow() {
		t.Fatal("should allow first request")
	}
	if rl.Allow() {
		t.Fatal("should reject second request with burst=1")
	}
}
