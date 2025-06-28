package carriers

import (
	"context"
	"testing"
	"time"
)

func TestFedExHeadlessClient_ValidateTrackingNumber(t *testing.T) {
	client := NewFedExHeadlessClient()
	defer client.Close()

	tests := []struct {
		name           string
		trackingNumber string
		expected       bool
	}{
		{
			name:           "valid 12 digit number",
			trackingNumber: "123456789012",
			expected:       true,
		},
		{
			name:           "valid 14 digit number",
			trackingNumber: "12345678901234",
			expected:       true,
		},
		{
			name:           "invalid short number",
			trackingNumber: "123456789",
			expected:       false,
		},
		{
			name:           "invalid with letters",
			trackingNumber: "12345678901A",
			expected:       false,
		},
		{
			name:           "empty string",
			trackingNumber: "",
			expected:       false,
		},
		{
			name:           "valid with spaces (should be cleaned)",
			trackingNumber: "1234 5678 9012",
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.ValidateTrackingNumber(tt.trackingNumber)
			if result != tt.expected {
				t.Errorf("ValidateTrackingNumber(%q) = %v; expected %v", tt.trackingNumber, result, tt.expected)
			}
		})
	}
}

func TestFedExHeadlessClient_ClientInterface(t *testing.T) {
	client := NewFedExHeadlessClient()
	defer client.Close()

	// Test that it implements the Client interface
	var _ Client = client

	// Test basic methods
	if client.GetCarrierName() != "fedex" {
		t.Errorf("GetCarrierName() = %q; expected %q", client.GetCarrierName(), "fedex")
	}

	rateLimit := client.GetRateLimit()
	if rateLimit == nil {
		t.Error("GetRateLimit() returned nil")
	}
}

func TestFedExHeadlessClient_HeadlessInterface(t *testing.T) {
	client := NewFedExHeadlessClient()
	defer client.Close()

	// Test that it implements the HeadlessBrowserClient interface
	var _ HeadlessBrowserClient = client

	// Test headless-specific methods with minimal functionality
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test WaitForContent with a short timeout (should fail quickly)
	err := client.WaitForContent(ctx, "body", 1*time.Second)
	// We expect this to fail since we're not navigating to a real page
	if err == nil {
		t.Log("WaitForContent unexpectedly succeeded (this might be okay in some environments)")
	}
}

func TestNewFedExHeadlessClient(t *testing.T) {
	client := NewFedExHeadlessClient()
	defer client.Close()

	if client == nil {
		t.Fatal("NewFedExHeadlessClient() returned nil")
	}

	if client.baseURL != "https://www.fedex.com" {
		t.Errorf("baseURL = %q; expected %q", client.baseURL, "https://www.fedex.com")
	}

	if client.GetCarrierName() != "fedex" {
		t.Errorf("GetCarrierName() = %q; expected %q", client.GetCarrierName(), "fedex")
	}
}

// TestFedExHeadlessClient_BrowserPoolStats tests browser pool functionality
func TestFedExHeadlessClient_BrowserPoolStats(t *testing.T) {
	client := NewFedExHeadlessClient()
	defer client.Close()

	stats := client.browserPool.Stats()
	if stats.Total < 0 {
		t.Errorf("Stats.Total = %d; expected >= 0", stats.Total)
	}
	if stats.Active < 0 {
		t.Errorf("Stats.Active = %d; expected >= 0", stats.Active)
	}
	if stats.Idle < 0 {
		t.Errorf("Stats.Idle = %d; expected >= 0", stats.Idle)
	}
}

// TestHeadlessOptions tests the options functionality
func TestHeadlessOptions(t *testing.T) {
	opts := DefaultHeadlessOptions()
	
	if !opts.Headless {
		t.Error("DefaultHeadlessOptions.Headless should be true")
	}
	
	if opts.Timeout <= 0 {
		t.Error("DefaultHeadlessOptions.Timeout should be positive")
	}
	
	if opts.UserAgent == "" {
		t.Error("DefaultHeadlessOptions.UserAgent should not be empty")
	}
	
	if opts.ViewportWidth <= 0 || opts.ViewportHeight <= 0 {
		t.Error("DefaultHeadlessOptions viewport dimensions should be positive")
	}
}

// TestBrowserPoolConfig tests browser pool configuration
func TestBrowserPoolConfig(t *testing.T) {
	config := DefaultBrowserPoolConfig()
	
	if config.MaxBrowsers <= 0 {
		t.Error("DefaultBrowserPoolConfig.MaxBrowsers should be positive")
	}
	
	if config.IdleTimeout <= 0 {
		t.Error("DefaultBrowserPoolConfig.IdleTimeout should be positive")
	}
	
	if config.MaxIdleBrowsers < 0 {
		t.Error("DefaultBrowserPoolConfig.MaxIdleBrowsers should be non-negative")
	}
}