package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestCLIViperConfig_LoadFromDefaults(t *testing.T) {
	// Clear environment variables to test defaults
	clearCLIEnvVars()

	v := viper.New()
	config, err := LoadCLIConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test default values
	if config.ServerURL != "http://localhost:8080" {
		t.Errorf("Expected ServerURL to be 'http://localhost:8080', got '%s'", config.ServerURL)
	}
	if config.Format != "table" {
		t.Errorf("Expected Format to be 'table', got '%s'", config.Format)
	}
	if config.Quiet != false {
		t.Errorf("Expected Quiet to be false, got %v", config.Quiet)
	}
	if config.NoColor != false {
		t.Errorf("Expected NoColor to be false, got %v", config.NoColor)
	}
	if config.RequestTimeout != 180*time.Second {
		t.Errorf("Expected RequestTimeout to be 180s, got %v", config.RequestTimeout)
	}
}

func TestCLIViperConfig_LoadFromEnvironment(t *testing.T) {
	// Clear environment variables first
	clearCLIEnvVars()

	// Set test environment variables with new PKG_TRACKER prefix
	envVars := map[string]string{
		"PKG_TRACKER_CLI_SERVER_URL":     "http://example.com:9090",
		"PKG_TRACKER_CLI_FORMAT":         "json",
		"PKG_TRACKER_CLI_QUIET":          "true",
		"PKG_TRACKER_CLI_NO_COLOR":       "true",
		"PKG_TRACKER_CLI_TIMEOUT":        "300",
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
	config, err := LoadCLIConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test environment variable values
	if config.ServerURL != "http://example.com:9090" {
		t.Errorf("Expected ServerURL to be 'http://example.com:9090', got '%s'", config.ServerURL)
	}
	if config.Format != "json" {
		t.Errorf("Expected Format to be 'json', got '%s'", config.Format)
	}
	if config.Quiet != true {
		t.Errorf("Expected Quiet to be true, got %v", config.Quiet)
	}
	if config.NoColor != true {
		t.Errorf("Expected NoColor to be true, got %v", config.NoColor)
	}
	if config.RequestTimeout != 300*time.Second {
		t.Errorf("Expected RequestTimeout to be 300s, got %v", config.RequestTimeout)
	}
}

func TestCLIViperConfig_LoadFromYAMLFile(t *testing.T) {
	// Clear environment variables first
	clearCLIEnvVars()

	// Create temporary YAML config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "cli.yaml")
	configContent := `server_url: "http://yaml-test.com:8888"
format: "json"
quiet: true
no_color: false
request_timeout: "240s"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configFile)
	config, err := LoadCLIConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test YAML file values
	if config.ServerURL != "http://yaml-test.com:8888" {
		t.Errorf("Expected ServerURL to be 'http://yaml-test.com:8888', got '%s'", config.ServerURL)
	}
	if config.Format != "json" {
		t.Errorf("Expected Format to be 'json', got '%s'", config.Format)
	}
	if config.Quiet != true {
		t.Errorf("Expected Quiet to be true, got %v", config.Quiet)
	}
	if config.NoColor != false {
		t.Errorf("Expected NoColor to be false, got %v", config.NoColor)
	}
	if config.RequestTimeout != 240*time.Second {
		t.Errorf("Expected RequestTimeout to be 240s, got %v", config.RequestTimeout)
	}
}

func TestCLIViperConfig_BackwardCompatibility(t *testing.T) {
	// Clear environment variables first
	clearCLIEnvVars()

	// Set old environment variables to test backward compatibility
	oldEnvVars := map[string]string{
		"PACKAGE_TRACKER_SERVER":  "http://old-server.com:7070",
		"PACKAGE_TRACKER_FORMAT":  "json",
		"PACKAGE_TRACKER_QUIET":   "true",
		"PACKAGE_TRACKER_TIMEOUT": "120",
		"NO_COLOR":                "1",
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
	config, err := LoadCLIConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test that old environment variables still work
	if config.ServerURL != "http://old-server.com:7070" {
		t.Errorf("Expected ServerURL to be 'http://old-server.com:7070', got '%s'", config.ServerURL)
	}
	if config.Format != "json" {
		t.Errorf("Expected Format to be 'json', got '%s'", config.Format)
	}
	if config.Quiet != true {
		t.Errorf("Expected Quiet to be true, got %v", config.Quiet)
	}
	if config.NoColor != true {
		t.Errorf("Expected NoColor to be true, got %v", config.NoColor)
	}
	if config.RequestTimeout != 120*time.Second {
		t.Errorf("Expected RequestTimeout to be 120s, got %v", config.RequestTimeout)
	}
}

func TestCLIViperConfig_NewFormatOverridesOld(t *testing.T) {
	// Clear environment variables first
	clearCLIEnvVars()

	// Set both old and new environment variables
	envVars := map[string]string{
		// Old format
		"PACKAGE_TRACKER_SERVER": "http://old-server.com:7070",
		"PACKAGE_TRACKER_FORMAT": "json",
		// New format (should override old)
		"PKG_TRACKER_CLI_SERVER_URL": "http://new-server.com:9090",
		"PKG_TRACKER_CLI_FORMAT":     "table",
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
	config, err := LoadCLIConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test that new format overrides old
	if config.ServerURL != "http://new-server.com:9090" {
		t.Errorf("Expected ServerURL to be 'http://new-server.com:9090' (new format), got '%s'", config.ServerURL)
	}
	if config.Format != "table" {
		t.Errorf("Expected Format to be 'table' (new format), got '%s'", config.Format)
	}
}

func TestCLIViperConfig_ValidationErrors(t *testing.T) {
	// Clear environment variables first
	clearCLIEnvVars()

	tests := []struct {
		name       string
		envVars    map[string]string
		configFile string
		errorMsg   string
	}{
		{
			name: "empty server URL",
			configFile: `server_url: ""`,
			errorMsg: "invalid configuration: server URL cannot be empty",
		},
		{
			name: "invalid server URL",
			envVars: map[string]string{
				"PKG_TRACKER_CLI_SERVER_URL": "not-a-url",
			},
			errorMsg: "invalid configuration: invalid server URL format",
		},
		{
			name: "invalid format",
			envVars: map[string]string{
				"PKG_TRACKER_CLI_FORMAT": "invalid-format",
			},
			errorMsg: "invalid configuration: invalid format: invalid-format (must be one of: table, json)",
		},
		{
			name: "negative timeout",
			envVars: map[string]string{
				"PKG_TRACKER_CLI_TIMEOUT": "-1",
			},
			errorMsg: "failed to unmarshal config: request timeout must be positive, got -1 seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables
			clearCLIEnvVars()

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
				configFile := filepath.Join(tempDir, "cli.yaml")
				err := os.WriteFile(configFile, []byte(tt.configFile), 0644)
				if err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}
				v.SetConfigFile(configFile)
			}
			
			_, err := LoadCLIConfigWithViper(v)
			if err == nil {
				t.Errorf("Expected error, got nil")
			}
			if err != nil && err.Error() != tt.errorMsg {
				t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
			}
		})
	}
}

// Helper function to clear CLI environment variables
func clearCLIEnvVars() {
	// Clear new format variables
	newVars := []string{
		"PKG_TRACKER_CLI_SERVER_URL", "PKG_TRACKER_CLI_FORMAT", "PKG_TRACKER_CLI_QUIET",
		"PKG_TRACKER_CLI_NO_COLOR", "PKG_TRACKER_CLI_TIMEOUT",
	}

	// Clear old format variables
	oldVars := []string{
		"PACKAGE_TRACKER_SERVER", "PACKAGE_TRACKER_FORMAT", "PACKAGE_TRACKER_QUIET",
		"PACKAGE_TRACKER_NO_COLOR", "PACKAGE_TRACKER_TIMEOUT", "NO_COLOR",
	}

	allVars := append(newVars, oldVars...)
	for _, key := range allVars {
		os.Unsetenv(key)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
	       len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
	       (len(s) > len(substr) && findInString(s, substr))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}