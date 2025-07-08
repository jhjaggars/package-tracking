package parser

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter provides rate limiting functionality for LLM requests
type RateLimiter struct {
	// Maximum number of requests per time window
	maxRequests int
	// Time window duration
	window time.Duration
	// Storage for request timestamps
	requests []time.Time
	// Mutex for thread safety
	mutex sync.Mutex
	// Minimum interval between requests (for burst protection)
	minInterval time.Duration
	// Last request time
	lastRequest time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, window time.Duration, minInterval time.Duration) *RateLimiter {
	return &RateLimiter{
		maxRequests: maxRequests,
		window:      window,
		requests:    make([]time.Time, 0, maxRequests),
		minInterval: minInterval,
	}
}

// DefaultLLMRateLimiter creates a rate limiter with sensible defaults for LLM requests
func DefaultLLMRateLimiter() *RateLimiter {
	// Allow 60 requests per minute with minimum 1 second between requests
	return NewRateLimiter(60, time.Minute, time.Second)
}

// Allow checks if a request should be allowed and returns wait time if not
func (rl *RateLimiter) Allow() (bool, time.Duration) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Check minimum interval (burst protection)
	if !rl.lastRequest.IsZero() {
		timeSinceLastRequest := now.Sub(rl.lastRequest)
		if timeSinceLastRequest < rl.minInterval {
			waitTime := rl.minInterval - timeSinceLastRequest
			return false, waitTime
		}
	}

	// Clean up old requests outside the window
	rl.cleanupOldRequests(now)

	// Check if we're at the limit
	if len(rl.requests) >= rl.maxRequests {
		// Calculate wait time until the oldest request expires
		oldestRequest := rl.requests[0]
		waitTime := rl.window - now.Sub(oldestRequest)
		if waitTime > 0 {
			return false, waitTime
		}
	}

	// Allow the request
	rl.requests = append(rl.requests, now)
	rl.lastRequest = now
	return true, 0
}

// Wait blocks until a request is allowed, with context cancellation support
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		allowed, waitTime := rl.Allow()
		if allowed {
			return nil
		}

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Wait for the specified time or until context is cancelled
		timer := time.NewTimer(waitTime)
		select {
		case <-timer.C:
			// Continue to next iteration
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	}
}

// cleanupOldRequests removes requests that are outside the time window
func (rl *RateLimiter) cleanupOldRequests(now time.Time) {
	cutoff := now.Add(-rl.window)
	
	// Find the first request that's still within the window
	start := 0
	for i, requestTime := range rl.requests {
		if requestTime.After(cutoff) {
			start = i
			break
		}
		start = i + 1
	}

	// Keep only requests within the window
	if start > 0 {
		rl.requests = rl.requests[start:]
	}
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats() RateLimiterStats {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	rl.cleanupOldRequests(now)

	return RateLimiterStats{
		CurrentRequests: len(rl.requests),
		MaxRequests:     rl.maxRequests,
		Window:          rl.window,
		MinInterval:     rl.minInterval,
		LastRequest:     rl.lastRequest,
	}
}

// RateLimiterStats contains statistics about the rate limiter
type RateLimiterStats struct {
	CurrentRequests int
	MaxRequests     int
	Window          time.Duration
	MinInterval     time.Duration
	LastRequest     time.Time
}

// IsNearLimit returns true if we're close to the rate limit (80% or more)
func (s RateLimiterStats) IsNearLimit() bool {
	threshold := float64(s.MaxRequests) * 0.8
	return float64(s.CurrentRequests) >= threshold
}

// RemainingRequests returns the number of requests remaining in the current window
func (s RateLimiterStats) RemainingRequests() int {
	remaining := s.MaxRequests - s.CurrentRequests
	if remaining < 0 {
		return 0
	}
	return remaining
}

// String returns a human-readable representation of the stats
func (s RateLimiterStats) String() string {
	return fmt.Sprintf("Rate Limiter: %d/%d requests in %v window, min interval %v",
		s.CurrentRequests, s.MaxRequests, s.Window, s.MinInterval)
}