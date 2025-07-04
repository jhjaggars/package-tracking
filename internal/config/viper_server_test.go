package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestServerViperConfig_LoadFromDefaults(t *testing.T) {
	// Clear environment variables to test defaults
	clearEnvVars()

	// Set admin auth to disabled for the defaults test
	os.Setenv("PKG_TRACKER_ADMIN_AUTH_DISABLED", "true")
	defer os.Unsetenv("PKG_TRACKER_ADMIN_AUTH_DISABLED")

	v := viper.New()
	config, err := LoadServerConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test default values
	if config.ServerPort != "8080" {
		t.Errorf("Expected ServerPort to be '8080', got '%s'", config.ServerPort)
	}
	if config.ServerHost != "localhost" {
		t.Errorf("Expected ServerHost to be 'localhost', got '%s'", config.ServerHost)
	}
	if config.DBPath != "./database.db" {
		t.Errorf("Expected DBPath to be './database.db', got '%s'", config.DBPath)
	}
	if config.LogLevel != "info" {
		t.Errorf("Expected LogLevel to be 'info', got '%s'", config.LogLevel)
	}
	if config.UpdateInterval != time.Hour {
		t.Errorf("Expected UpdateInterval to be 1h, got %v", config.UpdateInterval)
	}
}

func TestServerViperConfig_LoadFromEnvironment(t *testing.T) {
	// Clear environment variables first
	clearEnvVars()

	// Set test environment variables with new PKG_TRACKER prefix
	envVars := map[string]string{
		"PKG_TRACKER_SERVER_PORT":             "9090",
		"PKG_TRACKER_SERVER_HOST":             "0.0.0.0",
		"PKG_TRACKER_DATABASE_PATH":           "./test.db",
		"PKG_TRACKER_LOGGING_LEVEL":           "debug",
		"PKG_TRACKER_UPDATE_INTERVAL":         "30m",
		"PKG_TRACKER_CARRIERS_USPS_API_KEY":   "test-usps-key",
		"PKG_TRACKER_CARRIERS_UPS_CLIENT_ID":  "test-ups-client",
		"PKG_TRACKER_CARRIERS_UPS_CLIENT_SECRET": "test-ups-secret",
		"PKG_TRACKER_ADMIN_API_KEY":           "test-admin-key",
		"PKG_TRACKER_ADMIN_AUTH_DISABLED":     "true",
		"PKG_TRACKER_CACHE_TTL":               "10m",
		"PKG_TRACKER_CACHE_DISABLED":          "true",
	}

	for key, value := range envVars {
		os.Setenv(key, value)
	}
	defer func() {
		for key := range envVars {
			os.Unsetenv(key)
		}
	}()

	v := viper.New()
	config, err := LoadServerConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test environment variable values
	if config.ServerPort != "9090" {
		t.Errorf("Expected ServerPort to be '9090', got '%s'", config.ServerPort)
	}
	if config.ServerHost != "0.0.0.0" {
		t.Errorf("Expected ServerHost to be '0.0.0.0', got '%s'", config.ServerHost)
	}
	if config.DBPath != "./test.db" {
		t.Errorf("Expected DBPath to be './test.db', got '%s'", config.DBPath)
	}
	if config.LogLevel != "debug" {
		t.Errorf("Expected LogLevel to be 'debug', got '%s'", config.LogLevel)
	}
	if config.UpdateInterval != 30*time.Minute {
		t.Errorf("Expected UpdateInterval to be 30m, got %v", config.UpdateInterval)
	}
	if config.USPSAPIKey != "test-usps-key" {
		t.Errorf("Expected USPSAPIKey to be 'test-usps-key', got '%s'", config.USPSAPIKey)
	}
	if config.UPSClientID != "test-ups-client" {
		t.Errorf("Expected UPSClientID to be 'test-ups-client', got '%s'", config.UPSClientID)
	}
	if config.DisableAdminAuth != true {
		t.Errorf("Expected DisableAdminAuth to be true, got %v", config.DisableAdminAuth)
	}
	if config.DisableCache != true {
		t.Errorf("Expected DisableCache to be true, got %v", config.DisableCache)
	}
}

func TestServerViperConfig_LoadFromYAMLFile(t *testing.T) {
	// Clear environment variables first
	clearEnvVars()

	// Create temporary YAML config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	configContent := `server:
  host: "test-host"
  port: 8888

database:
  path: "./yaml-test.db"

logging:
  level: "warn"

carriers:
  usps:
    api_key: "yaml-usps-key"
  ups:
    client_id: "yaml-ups-client"
    client_secret: "yaml-ups-secret"

admin:
  api_key: "yaml-admin-key"
  auth_disabled: true

cache:
  ttl: "15m"
  disabled: false

update:
  interval: "45m"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configFile)
	config, err := LoadServerConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test YAML file values
	if config.ServerHost != "test-host" {
		t.Errorf("Expected ServerHost to be 'test-host', got '%s'", config.ServerHost)
	}
	if config.ServerPort != "8888" {
		t.Errorf("Expected ServerPort to be '8888', got '%s'", config.ServerPort)
	}
	if config.DBPath != "./yaml-test.db" {
		t.Errorf("Expected DBPath to be './yaml-test.db', got '%s'", config.DBPath)
	}
	if config.LogLevel != "warn" {
		t.Errorf("Expected LogLevel to be 'warn', got '%s'", config.LogLevel)
	}
	if config.UpdateInterval != 45*time.Minute {
		t.Errorf("Expected UpdateInterval to be 45m, got %v", config.UpdateInterval)
	}
	if config.USPSAPIKey != "yaml-usps-key" {
		t.Errorf("Expected USPSAPIKey to be 'yaml-usps-key', got '%s'", config.USPSAPIKey)
	}
	if config.UPSClientID != "yaml-ups-client" {
		t.Errorf("Expected UPSClientID to be 'yaml-ups-client', got '%s'", config.UPSClientID)
	}
	if config.AdminAPIKey != "yaml-admin-key" {
		t.Errorf("Expected AdminAPIKey to be 'yaml-admin-key', got '%s'", config.AdminAPIKey)
	}
	if config.DisableAdminAuth != true {
		t.Errorf("Expected DisableAdminAuth to be true, got %v", config.DisableAdminAuth)
	}
	if config.CacheTTL != 15*time.Minute {
		t.Errorf("Expected CacheTTL to be 15m, got %v", config.CacheTTL)
	}
}

func TestServerViperConfig_BackwardCompatibility(t *testing.T) {
	// Clear environment variables first
	clearEnvVars()

	// Set old environment variables to test backward compatibility
	oldEnvVars := map[string]string{
		"SERVER_PORT":     "7070",
		"SERVER_HOST":     "old-host",
		"DB_PATH":         "./old.db",
		"USPS_API_KEY":    "old-usps-key",
		"UPS_CLIENT_ID":   "old-ups-client",
		"ADMIN_API_KEY":   "old-admin-key",
		"LOG_LEVEL":       "error",
		"UPDATE_INTERVAL": "2h",
	}

	for key, value := range oldEnvVars {
		os.Setenv(key, value)
	}
	defer func() {
		for key := range oldEnvVars {
			os.Unsetenv(key)
		}
	}()

	v := viper.New()
	config, err := LoadServerConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test that old environment variables still work
	if config.ServerPort != "7070" {
		t.Errorf("Expected ServerPort to be '7070', got '%s'", config.ServerPort)
	}
	if config.ServerHost != "old-host" {
		t.Errorf("Expected ServerHost to be 'old-host', got '%s'", config.ServerHost)
	}
	if config.DBPath != "./old.db" {
		t.Errorf("Expected DBPath to be './old.db', got '%s'", config.DBPath)
	}
	if config.USPSAPIKey != "old-usps-key" {
		t.Errorf("Expected USPSAPIKey to be 'old-usps-key', got '%s'", config.USPSAPIKey)
	}
	if config.UPSClientID != "old-ups-client" {
		t.Errorf("Expected UPSClientID to be 'old-ups-client', got '%s'", config.UPSClientID)
	}
	if config.AdminAPIKey != "old-admin-key" {
		t.Errorf("Expected AdminAPIKey to be 'old-admin-key', got '%s'", config.AdminAPIKey)
	}
	if config.LogLevel != "error" {
		t.Errorf("Expected LogLevel to be 'error', got '%s'", config.LogLevel)
	}
	if config.UpdateInterval != 2*time.Hour {
		t.Errorf("Expected UpdateInterval to be 2h, got %v", config.UpdateInterval)
	}
}

func TestServerViperConfig_NewFormatOverridesOld(t *testing.T) {
	// Clear environment variables first
	clearEnvVars()

	// Set both old and new environment variables
	envVars := map[string]string{
		// Old format
		"SERVER_PORT":     "7070",
		"USPS_API_KEY":    "old-usps-key",
		// New format (should override old)
		"PKG_TRACKER_SERVER_PORT":           "9090",
		"PKG_TRACKER_CARRIERS_USPS_API_KEY": "new-usps-key",
		"PKG_TRACKER_ADMIN_AUTH_DISABLED":   "true",
	}

	for key, value := range envVars {
		os.Setenv(key, value)
	}
	defer func() {
		for key := range envVars {
			os.Unsetenv(key)
		}
	}()

	v := viper.New()
	config, err := LoadServerConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test that new format overrides old
	if config.ServerPort != "9090" {
		t.Errorf("Expected ServerPort to be '9090' (new format), got '%s'", config.ServerPort)
	}
	if config.USPSAPIKey != "new-usps-key" {
		t.Errorf("Expected USPSAPIKey to be 'new-usps-key' (new format), got '%s'", config.USPSAPIKey)
	}
}

func TestServerViperConfig_ValidationErrors(t *testing.T) {
	// Clear environment variables first
	clearEnvVars()

	tests := []struct {
		name       string
		envVars    map[string]string
		configFile string
		errorMsg   string
	}{
		{
			name: "empty server port",
			envVars: map[string]string{
				"PKG_TRACKER_SERVER_PORT":         "",
				"PKG_TRACKER_ADMIN_AUTH_DISABLED": "true",
			},
			configFile: `
server:
  port: ""
admin:
  auth_disabled: true
`,
			errorMsg: "invalid configuration: server port cannot be empty",
		},
		{
			name: "invalid server port",
			envVars: map[string]string{
				"PKG_TRACKER_SERVER_PORT":         "not-a-number",
				"PKG_TRACKER_ADMIN_AUTH_DISABLED": "true",
			},
			errorMsg: "invalid configuration: invalid server port: not-a-number",
		},
		{
			name: "empty database path",
			envVars: map[string]string{
				"PKG_TRACKER_ADMIN_AUTH_DISABLED": "true",
			},
			configFile: `
database:
  path: ""
admin:
  auth_disabled: true
`,
			errorMsg: "invalid configuration: database path cannot be empty",
		},
		{
			name: "invalid log level",
			envVars: map[string]string{
				"PKG_TRACKER_LOGGING_LEVEL":       "invalid",
				"PKG_TRACKER_ADMIN_AUTH_DISABLED": "true",
			},
			errorMsg: "invalid configuration: invalid log level: invalid (must be one of: debug, info, warn, error)",
		},
		{
			name: "admin key required when auth enabled",
			envVars: map[string]string{
				"PKG_TRACKER_ADMIN_AUTH_DISABLED": "false",
				"PKG_TRACKER_ADMIN_API_KEY":       "",
			},
			errorMsg: "invalid configuration: ADMIN_API_KEY is required when admin authentication is enabled (set DISABLE_ADMIN_AUTH=true to disable)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables
			clearEnvVars()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			v := viper.New()
			
			// Create config file if specified
			if tt.configFile != "" {
				tempDir := t.TempDir()
				configFile := filepath.Join(tempDir, "config.yaml")
				err := os.WriteFile(configFile, []byte(tt.configFile), 0644)
				if err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}
				v.SetConfigFile(configFile)
			}
			
			_, err := LoadServerConfigWithViper(v)
			if err == nil {
				t.Errorf("Expected error, got nil")
			}
			if err != nil && err.Error() != tt.errorMsg {
				t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
			}
		})
	}
}

// Helper function to clear environment variables
func clearEnvVars() {
	// Clear new format variables
	newVars := []string{
		"PKG_TRACKER_SERVER_PORT", "PKG_TRACKER_SERVER_HOST", "PKG_TRACKER_DATABASE_PATH",
		"PKG_TRACKER_LOGGING_LEVEL", "PKG_TRACKER_UPDATE_INTERVAL",
		"PKG_TRACKER_CARRIERS_USPS_API_KEY", "PKG_TRACKER_CARRIERS_UPS_CLIENT_ID",
		"PKG_TRACKER_CARRIERS_UPS_CLIENT_SECRET", "PKG_TRACKER_ADMIN_API_KEY",
		"PKG_TRACKER_ADMIN_AUTH_DISABLED", "PKG_TRACKER_CACHE_TTL", "PKG_TRACKER_CACHE_DISABLED",
	}

	// Clear old format variables
	oldVars := []string{
		"SERVER_PORT", "SERVER_HOST", "DB_PATH", "LOG_LEVEL", "UPDATE_INTERVAL",
		"USPS_API_KEY", "UPS_CLIENT_ID", "UPS_CLIENT_SECRET", "ADMIN_API_KEY",
		"DISABLE_ADMIN_AUTH", "CACHE_TTL", "DISABLE_CACHE",
	}

	allVars := append(newVars, oldVars...)
	for _, key := range allVars {
		os.Unsetenv(key)
	}
}