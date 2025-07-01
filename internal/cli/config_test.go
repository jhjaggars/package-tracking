package cli

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.ServerURL != "http://localhost:8080" {
		t.Errorf("Expected default server URL to be 'http://localhost:8080', got '%s'", config.ServerURL)
	}
	
	if config.Format != "table" {
		t.Errorf("Expected default format to be 'table', got '%s'", config.Format)
	}
	
	if config.Quiet != false {
		t.Errorf("Expected default quiet to be false, got %v", config.Quiet)
	}
	
	if config.RequestTimeout != 180*time.Second {
		t.Errorf("Expected default timeout to be 180s, got %v", config.RequestTimeout)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("PACKAGE_TRACKER_SERVER", "http://test.example.com:9090")
	os.Setenv("PACKAGE_TRACKER_FORMAT", "json")
	os.Setenv("PACKAGE_TRACKER_QUIET", "true")
	os.Setenv("PACKAGE_TRACKER_TIMEOUT", "60")
	defer func() {
		os.Unsetenv("PACKAGE_TRACKER_SERVER")
		os.Unsetenv("PACKAGE_TRACKER_FORMAT")
		os.Unsetenv("PACKAGE_TRACKER_QUIET")
		os.Unsetenv("PACKAGE_TRACKER_TIMEOUT")
	}()
	
	config, err := LoadConfig("", "", false)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	if config.ServerURL != "http://test.example.com:9090" {
		t.Errorf("Expected server URL from env to be 'http://test.example.com:9090', got '%s'", config.ServerURL)
	}
	
	if config.Format != "json" {
		t.Errorf("Expected format from env to be 'json', got '%s'", config.Format)
	}
	
	if config.Quiet != true {
		t.Errorf("Expected quiet from env to be true, got %v", config.Quiet)
	}
	
	if config.RequestTimeout != 60*time.Second {
		t.Errorf("Expected timeout from env to be 60s, got %v", config.RequestTimeout)
	}
}

func TestLoadConfigFlagOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("PACKAGE_TRACKER_SERVER", "http://env.example.com")
	defer os.Unsetenv("PACKAGE_TRACKER_SERVER")
	
	// CLI flags should override environment variables
	config, err := LoadConfig("http://flag.example.com", "json", true)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	if config.ServerURL != "http://flag.example.com" {
		t.Errorf("Expected server URL from flag to override env, got '%s'", config.ServerURL)
	}
	
	if config.Format != "json" {
		t.Errorf("Expected format from flag to be 'json', got '%s'", config.Format)
	}
	
	if config.Quiet != true {
		t.Errorf("Expected quiet from flag to be true, got %v", config.Quiet)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		serverURL   string
		format      string
		shouldError bool
	}{
		{"valid config", "http://localhost:8080", "table", false},
		{"valid json format", "http://localhost:8080", "json", false},
		{"valid https config", "https://api.example.com", "table", false},
		{"just whitespace server URL", " ", "table", true},
		{"invalid format", "http://localhost:8080", "xml", true},
		{"invalid URL format", "://invalid", "table", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadConfig(tt.serverURL, tt.format, false)
			
			if tt.shouldError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.name)
			}
			
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.name, err)
			}
			
			if !tt.shouldError && config == nil {
				t.Errorf("Expected config for %s, but got nil", tt.name)
			}
		})
	}
}