package ratelimit

import (
	"testing"
	"time"
)

// TestConfig implements the Config interface for testing
type TestConfig struct {
	DisableRateLimit bool
}

func (c *TestConfig) GetDisableRateLimit() bool {
	return c.DisableRateLimit
}

func TestCheckRefreshRateLimit_Disabled(t *testing.T) {
	cfg := &TestConfig{DisableRateLimit: true}
	
	// Even with recent refresh, should not block when disabled
	recentRefresh := time.Now().Add(-1 * time.Minute)
	result := CheckRefreshRateLimit(cfg, &recentRefresh, false)
	
	if result.ShouldBlock {
		t.Error("Rate limiting should be disabled")
	}
	if result.Reason != "rate_limiting_disabled" {
		t.Errorf("Expected reason 'rate_limiting_disabled', got '%s'", result.Reason)
	}
}

func TestCheckRefreshRateLimit_Enabled(t *testing.T) {
	cfg := &TestConfig{DisableRateLimit: false}
	now := time.Now()
	
	t.Run("RecentRefresh", func(t *testing.T) {
		// Within 5-minute rate limit
		recentRefresh := now.Add(-2 * time.Minute)
		result := CheckRefreshRateLimit(cfg, &recentRefresh, false)
		
		if !result.ShouldBlock {
			t.Error("Recent refresh should be blocked")
		}
		if result.Reason != "rate_limit_active" {
			t.Errorf("Expected reason 'rate_limit_active', got '%s'", result.Reason)
		}
		if result.RemainingTime <= 0 {
			t.Error("Should have remaining time")
		}
	})
	
	t.Run("OldRefresh", func(t *testing.T) {
		// Outside 5-minute rate limit
		oldRefresh := now.Add(-6 * time.Minute)
		result := CheckRefreshRateLimit(cfg, &oldRefresh, false)
		
		if result.ShouldBlock {
			t.Error("Old refresh should not be blocked")
		}
		if result.Reason != "rate_limit_passed" {
			t.Errorf("Expected reason 'rate_limit_passed', got '%s'", result.Reason)
		}
	})
	
	t.Run("NoRefresh", func(t *testing.T) {
		// No previous refresh
		result := CheckRefreshRateLimit(cfg, nil, false)
		
		if result.ShouldBlock {
			t.Error("No refresh should not be blocked")
		}
		if result.Reason != "no_previous_refresh" {
			t.Errorf("Expected reason 'no_previous_refresh', got '%s'", result.Reason)
		}
	})
	
	t.Run("ForcedRefresh", func(t *testing.T) {
		// Forced refresh should bypass rate limiting
		recentRefresh := now.Add(-1 * time.Minute)
		result := CheckRefreshRateLimit(cfg, &recentRefresh, true)
		
		if result.ShouldBlock {
			t.Error("Forced refresh should not be blocked")
		}
		if result.Reason != "forced_refresh" {
			t.Errorf("Expected reason 'forced_refresh', got '%s'", result.Reason)
		}
	})
}

func TestGetRateLimitDuration(t *testing.T) {
	duration := GetRateLimitDuration()
	expected := 5 * time.Minute
	
	if duration != expected {
		t.Errorf("Expected rate limit duration %v, got %v", expected, duration)
	}
}

func TestRateLimitRemainingTime(t *testing.T) {
	cfg := &TestConfig{DisableRateLimit: false}
	
	// Test that remaining time calculation is correct
	now := time.Now()
	refreshTime := now.Add(-3 * time.Minute) // 3 minutes ago
	
	result := CheckRefreshRateLimit(cfg, &refreshTime, false)
	
	if !result.ShouldBlock {
		t.Error("Should be blocked within rate limit")
	}
	
	expectedRemaining := 2 * time.Minute // 5 - 3 = 2 minutes remaining
	tolerance := 5 * time.Second        // Allow some tolerance for test execution time
	
	if result.RemainingTime < expectedRemaining-tolerance || result.RemainingTime > expectedRemaining+tolerance {
		t.Errorf("Expected remaining time around %v, got %v", expectedRemaining, result.RemainingTime)
	}
}