package ratelimit

import (
	"time"
)

// Config interface for rate limiting configuration
type Config interface {
	GetDisableRateLimit() bool
}

// RateLimitResult contains the result of a rate limit check
type RateLimitResult struct {
	ShouldBlock   bool
	RemainingTime time.Duration
	Reason        string
}

// CheckRefreshRateLimit checks if a refresh operation should be rate limited
// This function is used by both manual refresh (handlers) and auto-refresh (workers)
func CheckRefreshRateLimit(cfg Config, lastManualRefresh *time.Time, isForced bool) RateLimitResult {
	// Never rate limit if rate limiting is disabled
	if cfg.GetDisableRateLimit() {
		return RateLimitResult{
			ShouldBlock: false,
			Reason:      "rate_limiting_disabled",
		}
	}

	// Never rate limit forced refreshes
	if isForced {
		return RateLimitResult{
			ShouldBlock: false,
			Reason:      "forced_refresh",
		}
	}

	// Never rate limit if no previous refresh exists
	if lastManualRefresh == nil {
		return RateLimitResult{
			ShouldBlock: false,
			Reason:      "no_previous_refresh",
		}
	}

	// Use consistent 5-minute rate limit for both manual and auto-refresh
	rateLimit := 5 * time.Minute
	timeSinceLastRefresh := time.Since(*lastManualRefresh)

	if timeSinceLastRefresh < rateLimit {
		remainingTime := rateLimit - timeSinceLastRefresh
		return RateLimitResult{
			ShouldBlock:   true,
			RemainingTime: remainingTime,
			Reason:        "rate_limit_active",
		}
	}

	return RateLimitResult{
		ShouldBlock: false,
		Reason:      "rate_limit_passed",
	}
}

// GetRateLimitDuration returns the rate limit duration for refresh operations
func GetRateLimitDuration() time.Duration {
	return 5 * time.Minute
}