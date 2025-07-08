package parser

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	// Create a rate limiter that allows 3 requests per second with 100ms min interval
	rl := NewRateLimiter(3, time.Second, 100*time.Millisecond)

	t.Run("First requests should be allowed", func(t *testing.T) {
		allowed, waitTime := rl.Allow()
		assert.True(t, allowed)
		assert.Equal(t, time.Duration(0), waitTime)
	})

	t.Run("Requests too close together should be blocked", func(t *testing.T) {
		// Try immediately after first request (violates min interval)
		allowed, waitTime := rl.Allow()
		assert.False(t, allowed)
		assert.Greater(t, waitTime, time.Duration(0))
		assert.LessOrEqual(t, waitTime, 100*time.Millisecond)
	})

	t.Run("Requests after min interval should be allowed", func(t *testing.T) {
		// Wait for min interval to pass
		time.Sleep(110 * time.Millisecond)
		
		allowed, waitTime := rl.Allow()
		assert.True(t, allowed)
		assert.Equal(t, time.Duration(0), waitTime)

		// Another request after min interval
		time.Sleep(110 * time.Millisecond)
		allowed, waitTime = rl.Allow()
		assert.True(t, allowed)
		assert.Equal(t, time.Duration(0), waitTime)
	})

	t.Run("Exceeding max requests should be blocked", func(t *testing.T) {
		// Try one more request (would be 4th in the window)
		time.Sleep(110 * time.Millisecond)
		allowed, waitTime := rl.Allow()
		assert.False(t, allowed)
		assert.Greater(t, waitTime, time.Duration(0))
	})
}

func TestRateLimiter_Wait(t *testing.T) {
	rl := NewRateLimiter(2, 500*time.Millisecond, 50*time.Millisecond)

	t.Run("Wait should allow request when limit not reached", func(t *testing.T) {
		ctx := context.Background()
		err := rl.Wait(ctx)
		assert.NoError(t, err)
	})

	t.Run("Wait should respect context cancellation", func(t *testing.T) {
		// Fill up the rate limiter
		rl.Allow() // This should work
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		
		start := time.Now()
		err := rl.Wait(ctx)
		elapsed := time.Since(start)
		
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		assert.Less(t, elapsed, 50*time.Millisecond) // Should not wait for min interval
	})

	t.Run("Wait should eventually succeed", func(t *testing.T) {
		// Reset by creating new rate limiter
		rl := NewRateLimiter(1, 100*time.Millisecond, 20*time.Millisecond)
		
		// Use up the limit
		allowed, _ := rl.Allow()
		assert.True(t, allowed)
		
		// This should wait and then succeed
		ctx := context.Background()
		start := time.Now()
		err := rl.Wait(ctx)
		elapsed := time.Since(start)
		
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, elapsed, 20*time.Millisecond) // At least min interval
	})
}

func TestRateLimiter_GetStats(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute, time.Second)

	t.Run("Initial stats", func(t *testing.T) {
		stats := rl.GetStats()
		assert.Equal(t, 0, stats.CurrentRequests)
		assert.Equal(t, 5, stats.MaxRequests)
		assert.Equal(t, time.Minute, stats.Window)
		assert.Equal(t, time.Second, stats.MinInterval)
		assert.True(t, stats.LastRequest.IsZero())
	})

	t.Run("Stats after requests", func(t *testing.T) {
		// Make some requests
		time.Sleep(1100 * time.Millisecond) // Ensure min interval
		rl.Allow()
		time.Sleep(1100 * time.Millisecond)
		rl.Allow()

		stats := rl.GetStats()
		assert.Equal(t, 2, stats.CurrentRequests)
		assert.False(t, stats.LastRequest.IsZero())
		assert.Equal(t, 3, stats.RemainingRequests())
		assert.False(t, stats.IsNearLimit()) // 2/5 = 40%, not near 80%
	})

	t.Run("Near limit detection", func(t *testing.T) {
		// Create a small rate limiter to test near limit
		smallRL := NewRateLimiter(5, time.Minute, 10*time.Millisecond)
		
		// Make 4 requests (80% of 5)
		for i := 0; i < 4; i++ {
			time.Sleep(15 * time.Millisecond)
			smallRL.Allow()
		}

		stats := smallRL.GetStats()
		assert.True(t, stats.IsNearLimit()) // 4/5 = 80%
		assert.Equal(t, 1, stats.RemainingRequests())
	})
}

func TestRateLimiter_cleanupOldRequests(t *testing.T) {
	rl := NewRateLimiter(10, 100*time.Millisecond, 10*time.Millisecond)

	// Add some requests
	for i := 0; i < 3; i++ {
		time.Sleep(15 * time.Millisecond)
		rl.Allow()
	}

	// Check that we have 3 requests
	stats := rl.GetStats()
	assert.Equal(t, 3, stats.CurrentRequests)

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Check stats again (should trigger cleanup)
	stats = rl.GetStats()
	assert.Equal(t, 0, stats.CurrentRequests) // Old requests should be cleaned up
}

func TestDefaultLLMRateLimiter(t *testing.T) {
	rl := DefaultLLMRateLimiter()

	stats := rl.GetStats()
	assert.Equal(t, 60, stats.MaxRequests)
	assert.Equal(t, time.Minute, stats.Window)
	assert.Equal(t, time.Second, stats.MinInterval)
}

func TestRateLimiterStats_String(t *testing.T) {
	stats := RateLimiterStats{
		CurrentRequests: 10,
		MaxRequests:     60,
		Window:          time.Minute,
		MinInterval:     time.Second,
	}

	result := stats.String()
	assert.Contains(t, result, "10/60 requests")
	assert.Contains(t, result, "1m0s window")
	assert.Contains(t, result, "1s")
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(100, time.Minute, 10*time.Millisecond)

	// Test concurrent access to ensure thread safety
	const numGoroutines = 10
	const requestsPerGoroutine = 5

	successCount := make(chan int, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			successes := 0
			for j := 0; j < requestsPerGoroutine; j++ {
				time.Sleep(15 * time.Millisecond) // Respect min interval
				if allowed, _ := rl.Allow(); allowed {
					successes++
				}
			}
			successCount <- successes
		}()
	}

	totalSuccesses := 0
	for i := 0; i < numGoroutines; i++ {
		totalSuccesses += <-successCount
	}

	// Should have some successes but may not be all due to rate limiting
	assert.Greater(t, totalSuccesses, 0)
	assert.LessOrEqual(t, totalSuccesses, numGoroutines*requestsPerGoroutine)

	// Verify that rate limiter state is consistent
	stats := rl.GetStats()
	assert.LessOrEqual(t, stats.CurrentRequests, 100)
}