package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{"SERVER_PORT", "SERVER_HOST", "DB_PATH", "UPDATE_INTERVAL", "LOG_LEVEL", "DISABLE_CACHE", "DISABLE_ADMIN_AUTH", "ADMIN_API_KEY"}
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
	}

	// Cleanup function
	cleanup := func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}
	defer cleanup()

	t.Run("DefaultValues", func(t *testing.T) {
		// Clear environment variables
		for _, key := range envVars {
			os.Unsetenv(key)
		}
		// Disable admin auth for default test
		os.Setenv("DISABLE_ADMIN_AUTH", "true")

		config, err := Load()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if config.ServerPort != "8080" {
			t.Errorf("Expected default port 8080, got %s", config.ServerPort)
		}

		if config.ServerHost != "localhost" {
			t.Errorf("Expected default host localhost, got %s", config.ServerHost)
		}

		if config.DBPath != "./database.db" {
			t.Errorf("Expected default DB path ./database.db, got %s", config.DBPath)
		}

		if config.UpdateInterval != time.Hour {
			t.Errorf("Expected default update interval 1h, got %v", config.UpdateInterval)
		}

		if config.LogLevel != "info" {
			t.Errorf("Expected default log level info, got %s", config.LogLevel)
		}

		if config.DisableCache != false {
			t.Errorf("Expected default disable cache false, got %v", config.DisableCache)
		}

		if config.DisableAdminAuth != true {
			t.Errorf("Expected disable admin auth true (overridden), got %v", config.DisableAdminAuth)
		}

		if config.AdminAPIKey != "" {
			t.Errorf("Expected default admin API key empty, got %s", config.AdminAPIKey)
		}
	})

	t.Run("EnvironmentOverrides", func(t *testing.T) {
		os.Setenv("SERVER_PORT", "9090")
		os.Setenv("SERVER_HOST", "0.0.0.0")
		os.Setenv("DB_PATH", "/tmp/test.db")
		os.Setenv("UPDATE_INTERVAL", "30m")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("DISABLE_ADMIN_AUTH", "true")

		config, err := Load()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if config.ServerPort != "9090" {
			t.Errorf("Expected port 9090, got %s", config.ServerPort)
		}

		if config.ServerHost != "0.0.0.0" {
			t.Errorf("Expected host 0.0.0.0, got %s", config.ServerHost)
		}

		if config.DBPath != "/tmp/test.db" {
			t.Errorf("Expected DB path /tmp/test.db, got %s", config.DBPath)
		}

		if config.UpdateInterval != 30*time.Minute {
			t.Errorf("Expected update interval 30m, got %v", config.UpdateInterval)
		}

		if config.LogLevel != "debug" {
			t.Errorf("Expected log level debug, got %s", config.LogLevel)
		}
	})

	t.Run("InvalidPort", func(t *testing.T) {
		os.Setenv("SERVER_PORT", "invalid")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for invalid port")
		}
	})

	t.Run("InvalidLogLevel", func(t *testing.T) {
		os.Setenv("LOG_LEVEL", "invalid")

		_, err := Load()
		if err == nil {
			t.Error("Expected error for invalid log level")
		}
	})

	t.Run("APIKeys", func(t *testing.T) {
		// Clear any invalid env vars from previous tests
		for _, key := range envVars {
			os.Unsetenv(key)
		}
		
		os.Setenv("USPS_API_KEY", "usps123")
		os.Setenv("UPS_API_KEY", "ups456")
		os.Setenv("DHL_API_KEY", "dhl789")
		os.Setenv("DISABLE_ADMIN_AUTH", "true")

		config, err := Load()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if config.USPSAPIKey != "usps123" {
			t.Errorf("Expected USPS API key usps123, got %s", config.USPSAPIKey)
		}

		if config.UPSAPIKey != "ups456" {
			t.Errorf("Expected UPS API key ups456, got %s", config.UPSAPIKey)
		}

		if config.DHLAPIKey != "dhl789" {
			t.Errorf("Expected DHL API key dhl789, got %s", config.DHLAPIKey)
		}
	})

	t.Run("DisableCache", func(t *testing.T) {
		// Clear any invalid env vars from previous tests
		for _, key := range envVars {
			os.Unsetenv(key)
		}
		
		os.Setenv("DISABLE_CACHE", "true")
		os.Setenv("DISABLE_ADMIN_AUTH", "true")

		config, err := Load()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if config.DisableCache != true {
			t.Errorf("Expected disable cache true, got %v", config.DisableCache)
		}

		if !config.GetDisableCache() {
			t.Errorf("Expected GetDisableCache() to return true")
		}
	})

	t.Run("AdminAuth", func(t *testing.T) {
		// Clear any invalid env vars from previous tests
		for _, key := range envVars {
			os.Unsetenv(key)
		}
		
		os.Setenv("ADMIN_API_KEY", "secret123")

		config, err := Load()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if config.AdminAPIKey != "secret123" {
			t.Errorf("Expected admin API key secret123, got %s", config.AdminAPIKey)
		}

		if config.GetAdminAPIKey() != "secret123" {
			t.Errorf("Expected GetAdminAPIKey() to return secret123, got %s", config.GetAdminAPIKey())
		}

		if config.GetDisableAdminAuth() {
			t.Errorf("Expected GetDisableAdminAuth() to return false")
		}
	})

	t.Run("DisableAdminAuth", func(t *testing.T) {
		// Clear any invalid env vars from previous tests
		for _, key := range envVars {
			os.Unsetenv(key)
		}
		
		os.Setenv("DISABLE_ADMIN_AUTH", "true")

		config, err := Load()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !config.GetDisableAdminAuth() {
			t.Errorf("Expected GetDisableAdminAuth() to return true")
		}

		// Should not require API key when auth is disabled
		if config.AdminAPIKey != "" {
			t.Errorf("Expected empty API key when auth disabled, got %s", config.AdminAPIKey)
		}
	})

	t.Run("DHLAutoUpdateConfig", func(t *testing.T) {
		// Clear any invalid env vars from previous tests
		for _, key := range envVars {
			os.Unsetenv(key)
		}
		
		os.Setenv("DHL_AUTO_UPDATE_ENABLED", "false")
		os.Setenv("DHL_AUTO_UPDATE_CUTOFF_DAYS", "45")
		os.Setenv("DISABLE_ADMIN_AUTH", "true")

		config, err := Load()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if config.DHLAutoUpdateEnabled != false {
			t.Errorf("Expected DHL auto-update enabled false, got %v", config.DHLAutoUpdateEnabled)
		}

		if config.DHLAutoUpdateCutoffDays != 45 {
			t.Errorf("Expected DHL auto-update cutoff days 45, got %d", config.DHLAutoUpdateCutoffDays)
		}
	})

	t.Run("DHLAutoUpdateDefaults", func(t *testing.T) {
		// Clear any invalid env vars from previous tests
		for _, key := range envVars {
			os.Unsetenv(key)
		}
		// Clear DHL-specific env vars from previous test
		os.Unsetenv("DHL_AUTO_UPDATE_ENABLED")
		os.Unsetenv("DHL_AUTO_UPDATE_CUTOFF_DAYS")
		
		os.Setenv("DISABLE_ADMIN_AUTH", "true")

		config, err := Load()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if config.DHLAutoUpdateEnabled != true {
			t.Errorf("Expected DHL auto-update enabled true (default), got %v", config.DHLAutoUpdateEnabled)
		}

		if config.DHLAutoUpdateCutoffDays != 0 {
			t.Errorf("Expected DHL auto-update cutoff days 0 (default), got %d", config.DHLAutoUpdateCutoffDays)
		}
	})
}

func TestAddress(t *testing.T) {
	config := &Config{
		ServerHost: "localhost",
		ServerPort: "8080",
	}

	expected := "localhost:8080"
	if config.Address() != expected {
		t.Errorf("Expected address %s, got %s", expected, config.Address())
	}
}

func TestValidate(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		config := &Config{
			ServerPort:                  "8080",  
			ServerHost:                  "localhost",
			DBPath:                      "./test.db",
			UpdateInterval:              time.Hour,
			LogLevel:                    "info",
			AutoUpdateBatchSize:         5, // Must be between 1 and 10
			AutoUpdateMaxRetries:        3,
			AutoUpdateFailureThreshold:  10,
			CacheTTL:                    5 * time.Minute,
			AutoUpdateBatchTimeout:      30 * time.Second,
			AutoUpdateIndividualTimeout: 10 * time.Second,
			DisableAdminAuth:            false,
			AdminAPIKey:                 "test-key-123",
		}

		if err := config.validate(); err != nil {
			t.Errorf("Expected valid config, got error: %v", err)
		}
	})

	t.Run("EmptyPort", func(t *testing.T) {
		config := &Config{
			ServerPort:     "",
			DBPath:         "./test.db",
			UpdateInterval: time.Hour,
			LogLevel:       "info",
		}

		if err := config.validate(); err == nil {
			t.Error("Expected error for empty port")
		}
	})

	t.Run("EmptyDBPath", func(t *testing.T) {
		config := &Config{
			ServerPort:     "8080",
			DBPath:         "",
			UpdateInterval: time.Hour,
			LogLevel:       "info",
		}

		if err := config.validate(); err == nil {
			t.Error("Expected error for empty DB path")
		}
	})

	t.Run("NegativeUpdateInterval", func(t *testing.T) {
		config := &Config{
			ServerPort:     "8080",
			DBPath:         "./test.db",
			UpdateInterval: -time.Hour,
			LogLevel:       "info",
		}

		if err := config.validate(); err == nil {
			t.Error("Expected error for negative update interval")
		}
	})

	t.Run("MissingAdminAPIKey", func(t *testing.T) {
		config := &Config{
			ServerPort:                  "8080",
			ServerHost:                  "localhost",
			DBPath:                      "./test.db",
			UpdateInterval:              time.Hour,
			LogLevel:                    "info",
			AutoUpdateBatchSize:         5,
			AutoUpdateMaxRetries:        3,
			AutoUpdateFailureThreshold:  10,
			CacheTTL:                    5 * time.Minute,
			AutoUpdateBatchTimeout:      30 * time.Second,
			AutoUpdateIndividualTimeout: 10 * time.Second,
			DisableAdminAuth:            false, // Auth enabled
			AdminAPIKey:                 "",    // But no key
		}

		if err := config.validate(); err == nil {
			t.Error("Expected error for missing admin API key when auth is enabled")
		}
	})

	t.Run("DisabledAdminAuthWithoutKey", func(t *testing.T) {
		config := &Config{
			ServerPort:                  "8080",
			ServerHost:                  "localhost",
			DBPath:                      "./test.db",
			UpdateInterval:              time.Hour,
			LogLevel:                    "info",
			AutoUpdateBatchSize:         5,
			AutoUpdateMaxRetries:        3,
			AutoUpdateFailureThreshold:  10,
			CacheTTL:                    5 * time.Minute,
			AutoUpdateBatchTimeout:      30 * time.Second,
			AutoUpdateIndividualTimeout: 10 * time.Second,
			DisableAdminAuth:            true, // Auth disabled
			AdminAPIKey:                 "",   // No key needed
		}

		if err := config.validate(); err != nil {
			t.Errorf("Expected no error when admin auth is disabled, got: %v", err)
		}
	})

	t.Run("NegativeDHLCutoffDays", func(t *testing.T) {
		config := &Config{
			ServerPort:                  "8080",
			ServerHost:                  "localhost",
			DBPath:                      "./test.db",
			UpdateInterval:              time.Hour,
			LogLevel:                    "info",
			AutoUpdateBatchSize:         5,
			AutoUpdateMaxRetries:        3,
			AutoUpdateFailureThreshold:  10,
			CacheTTL:                    5 * time.Minute,
			AutoUpdateBatchTimeout:      30 * time.Second,
			AutoUpdateIndividualTimeout: 10 * time.Second,
			DHLAutoUpdateCutoffDays:     -1, // Invalid
			DisableAdminAuth:            true,
		}

		if err := config.validate(); err == nil {
			t.Error("Expected error for negative DHL auto-update cutoff days")
		}
	})
}

func TestGetAdminAPIKeyForLogging(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "Empty key",
			apiKey:   "",
			expected: "(not set)",
		},
		{
			name:     "Short key",
			apiKey:   "abc",
			expected: "***",
		},
		{
			name:     "Normal key",
			apiKey:   "secret123456",
			expected: "secr***3456",
		},
		{
			name:     "Long key",
			apiKey:   "this-is-a-very-long-secret-key-123456789",
			expected: "this***6789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{AdminAPIKey: tt.apiKey}
			result := config.GetAdminAPIKeyForLogging()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}