/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
