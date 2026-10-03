/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package terminal

import (
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting for terminal input
type RateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate int // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// maxTokens: maximum number of tokens (burst capacity)
// refillRate: number of tokens added per second
func NewRateLimiter(maxTokens int, refillRate int) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if an action is allowed and consumes a token if available
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens based on time passed
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed.Seconds() * float64(rl.refillRate))

	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}

	// Check if we have tokens available
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// Wait waits until an action is allowed (blocking)
func (rl *RateLimiter) Wait() {
	for !rl.Allow() {
		time.Sleep(100 * time.Millisecond)
	}
}

// Tokens returns the current number of available tokens
func (rl *RateLimiter) Tokens() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.tokens
}
