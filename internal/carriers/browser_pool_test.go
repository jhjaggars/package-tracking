package carriers

import (
	"context"
	"testing"
	"time"
)

func TestValidateChromeAvailable(t *testing.T) {
	// Test Chrome availability
	err := ValidateChromeAvailable()
	if err != nil {
		// This might fail in CI environments without Chrome
		t.Logf("Chrome validation failed (expected in some environments): %v", err)
		t.Skip("Chrome not available in test environment")
	} else {
		t.Log("Chrome validation succeeded")
	}
}

func TestSimpleBrowserPool_Stats(t *testing.T) {
	config := DefaultBrowserPoolConfig()
	options := DefaultHeadlessOptions()
	pool := NewBrowserPool(config, options)
	defer pool.Close()

	// Test initial stats
	stats := pool.Stats()
	if stats.Total != 0 {
		t.Errorf("Initial Total = %d; expected 0", stats.Total)
	}
	if stats.Active != 0 {
		t.Errorf("Initial Active = %d; expected 0", stats.Active)
	}
	if stats.Idle != 0 {
		t.Errorf("Initial Idle = %d; expected 0", stats.Idle)
	}
}

func TestSimpleBrowserPool_ExecuteWithBrowser_ContextTimeout(t *testing.T) {
	// Skip if Chrome is not available
	if err := ValidateChromeAvailable(); err != nil {
		t.Skip("Chrome not available in test environment")
	}

	config := DefaultBrowserPoolConfig()
	options := DefaultHeadlessOptions()
	options.Timeout = 5 * time.Second
	pool := NewBrowserPool(config, options)
	defer pool.Close()

	// Test with parent context that has shorter timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	err := pool.ExecuteWithBrowser(ctx, func(browserCtx context.Context) error {
		// This should timeout after ~1 second due to parent context
		select {
		case <-time.After(3 * time.Second):
			return nil // Should not reach here
		case <-browserCtx.Done():
			return browserCtx.Err()
		}
	})

	elapsed := time.Since(start)
	if err == nil {
		t.Error("Expected timeout error")
	}
	if elapsed > 2*time.Second {
		t.Errorf("Timeout took too long: %v", elapsed)
	}
}

func TestDefaultBrowserPoolConfig(t *testing.T) {
	config := DefaultBrowserPoolConfig()
	
	if config.MaxBrowsers <= 0 {
		t.Error("MaxBrowsers should be positive")
	}
	if config.IdleTimeout <= 0 {
		t.Error("IdleTimeout should be positive")
	}
	if config.MaxIdleBrowsers < 0 {
		t.Error("MaxIdleBrowsers should be non-negative")
	}
}

func TestDefaultHeadlessOptions(t *testing.T) {
	options := DefaultHeadlessOptions()
	
	if !options.Headless {
		t.Error("Default should be headless")
	}
	if options.Timeout <= 0 {
		t.Error("Timeout should be positive")
	}
	if options.UserAgent == "" {
		t.Error("UserAgent should not be empty")
	}
	if options.ViewportWidth <= 0 || options.ViewportHeight <= 0 {
		t.Error("Viewport dimensions should be positive")
	}
}