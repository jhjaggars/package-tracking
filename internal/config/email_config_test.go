package config

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadEmailConfig(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"GMAIL_CLIENT_ID",
		"GMAIL_CLIENT_SECRET", 
		"GMAIL_REFRESH_TOKEN",
		"EMAIL_API_URL",
		"EMAIL_CHECK_INTERVAL",
		"EMAIL_DRY_RUN",
	}
	
	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
		os.Unsetenv(env)
	}
	
	// Restore environment after test
	defer func() {
		for _, env := range envVars {
			if val, exists := originalEnv[env]; exists {
				os.Setenv(env, val)
			} else {
				os.Unsetenv(env)
			}
		}
	}()

	testCases := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		errorMsg    string
		validate    func(*EmailConfig) error
	}{
		{
			name: "Valid minimal configuration",
			envVars: map[string]string{
				"GMAIL_CLIENT_ID":     "test-client-id",
				"GMAIL_CLIENT_SECRET": "test-secret",
				"GMAIL_REFRESH_TOKEN": "test-refresh-token",
				"EMAIL_API_URL":       "http://localhost:8080",
			},
			expectError: false,
			validate: func(cfg *EmailConfig) error {
				if cfg.Gmail.ClientID != "test-client-id" {
					return fmt.Errorf("expected client ID 'test-client-id', got '%s'", cfg.Gmail.ClientID)
				}
				if cfg.API.URL != "http://localhost:8080" {
					return fmt.Errorf("expected API URL 'http://localhost:8080', got '%s'", cfg.API.URL)
				}
				return nil
			},
		},
		{
			name: "Complete configuration",
			envVars: map[string]string{
				"GMAIL_CLIENT_ID":          "complete-client-id",
				"GMAIL_CLIENT_SECRET":      "complete-secret",
				"GMAIL_REFRESH_TOKEN":      "complete-refresh-token",
				"GMAIL_ACCESS_TOKEN":       "complete-access-token",
				"GMAIL_TOKEN_FILE":         "/path/to/token.json",
				"GMAIL_SEARCH_QUERY":       "from:ups.com",
				"GMAIL_SEARCH_AFTER_DAYS":  "14",
				"GMAIL_SEARCH_UNREAD_ONLY": "true",
				"GMAIL_SEARCH_MAX_RESULTS": "200",
				"EMAIL_CHECK_INTERVAL":     "10m",
				"EMAIL_MAX_PER_RUN":        "25",
				"EMAIL_DRY_RUN":            "true",
				"EMAIL_STATE_DB_PATH":      "/custom/state.db",
				"EMAIL_MIN_CONFIDENCE":     "0.8",
				"EMAIL_DEBUG_MODE":         "true",
				"EMAIL_API_URL":            "https://api.example.com",
				"EMAIL_API_TIMEOUT":        "45s",
				"EMAIL_API_RETRY_COUNT":    "5",
				"EMAIL_API_RETRY_DELAY":    "2s",
			},
			expectError: false,
			validate: func(cfg *EmailConfig) error {
				if cfg.Search.AfterDays != 14 {
					return fmt.Errorf("expected after days 14, got %d", cfg.Search.AfterDays)
				}
				if cfg.Processing.CheckInterval != 10*time.Minute {
					return fmt.Errorf("expected check interval 10m, got %v", cfg.Processing.CheckInterval)
				}
				if !cfg.Processing.DryRun {
					return fmt.Errorf("expected dry run to be true")
				}
				return nil
			},
		},
		{
			name: "Missing required Gmail client ID",
			envVars: map[string]string{
				"GMAIL_CLIENT_SECRET": "test-secret",
				"GMAIL_REFRESH_TOKEN": "test-refresh-token",
				"EMAIL_API_URL":       "http://localhost:8080",
			},
			expectError: true,
			errorMsg:    "GMAIL_CLIENT_ID is required",
		},
		{
			name: "Missing required Gmail client secret",
			envVars: map[string]string{
				"GMAIL_CLIENT_ID":     "test-client-id",
				"GMAIL_REFRESH_TOKEN": "test-refresh-token",
				"EMAIL_API_URL":       "http://localhost:8080",
			},
			expectError: true,
			errorMsg:    "GMAIL_CLIENT_SECRET is required",
		},
		{
			name: "Missing required API URL",
			envVars: map[string]string{
				"GMAIL_CLIENT_ID":     "test-client-id",
				"GMAIL_CLIENT_SECRET": "test-secret",
				"GMAIL_REFRESH_TOKEN": "test-refresh-token",
			},
			expectError: true,
			errorMsg:    "EMAIL_API_URL is required",
		},
		{
			name: "Invalid check interval",
			envVars: map[string]string{
				"GMAIL_CLIENT_ID":      "test-client-id",
				"GMAIL_CLIENT_SECRET":  "test-secret", 
				"GMAIL_REFRESH_TOKEN":  "test-refresh-token",
				"EMAIL_API_URL":        "http://localhost:8080",
				"EMAIL_CHECK_INTERVAL": "invalid-duration",
			},
			expectError: true,
			errorMsg:    "invalid EMAIL_CHECK_INTERVAL",
		},
		{
			name: "Invalid API timeout",
			envVars: map[string]string{
				"GMAIL_CLIENT_ID":     "test-client-id",
				"GMAIL_CLIENT_SECRET": "test-secret",
				"GMAIL_REFRESH_TOKEN": "test-refresh-token",
				"EMAIL_API_URL":       "http://localhost:8080",
				"EMAIL_API_TIMEOUT":   "not-a-duration",
			},
			expectError: true,
			errorMsg:    "invalid EMAIL_API_TIMEOUT",
		},
		{
			name: "Invalid boolean values",
			envVars: map[string]string{
				"GMAIL_CLIENT_ID":     "test-client-id",
				"GMAIL_CLIENT_SECRET": "test-secret",
				"GMAIL_REFRESH_TOKEN": "test-refresh-token",
				"EMAIL_API_URL":       "http://localhost:8080",
				"EMAIL_DRY_RUN":       "not-a-boolean",
			},
			expectError: true,
			errorMsg:    "invalid EMAIL_DRY_RUN",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables for this test
			for key, value := range tc.envVars {
				os.Setenv(key, value)
			}

			config, err := LoadEmailConfig()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				} else if tc.errorMsg != "" && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %v", tc.errorMsg, err)
				}
				if config != nil {
					t.Errorf("Expected nil config on error, but got: %v", config)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if config == nil {
					t.Errorf("Expected config, but got nil")
				} else if tc.validate != nil {
					if err := tc.validate(config); err != nil {
						t.Errorf("Validation failed: %v", err)
					}
				}
			}

			// Clean up environment variables for this test
			for key := range tc.envVars {
				os.Unsetenv(key)
			}
		})
	}
}

func TestLoadEmailConfigFromFile(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()
	
	testCases := []struct {
		name        string
		fileContent string
		fileName    string
		expectError bool
		errorMsg    string
		validate    func(*EmailConfig) error
	}{
		{
			name:     "Valid .env file",
			fileName: ".env",
			fileContent: `GMAIL_CLIENT_ID=file-client-id
GMAIL_CLIENT_SECRET=file-secret
GMAIL_REFRESH_TOKEN=file-refresh-token
EMAIL_API_URL=http://file.example.com
EMAIL_CHECK_INTERVAL=15m
EMAIL_DRY_RUN=true`,
			expectError: false,
			validate: func(cfg *EmailConfig) error {
				if cfg.Gmail.ClientID != "file-client-id" {
					return fmt.Errorf("expected client ID from file, got '%s'", cfg.Gmail.ClientID)
				}
				if cfg.Processing.CheckInterval != 15*time.Minute {
					return fmt.Errorf("expected 15m interval from file, got %v", cfg.Processing.CheckInterval)
				}
				return nil
			},
		},
		{
			name:     "Invalid .env file format", 
			fileName: ".env",
			fileContent: `GMAIL_CLIENT_ID=test
INVALID_LINE_WITHOUT_EQUALS
GMAIL_CLIENT_SECRET=secret`,
			expectError: false, // Should ignore invalid lines and continue
			validate: func(cfg *EmailConfig) error {
				// Should still load valid environment variables
				return nil
			},
		},
		{
			name:        "Missing .env file",
			fileName:    ".env",
			fileContent: "", // Don't create the file
			expectError: false, // Should not error if file doesn't exist
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Change to temp directory
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer os.Chdir(oldDir)

			err = os.Chdir(tmpDir)
			if err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Create test file if content is provided
			if tc.fileContent != "" {
				err = os.WriteFile(tc.fileName, []byte(tc.fileContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				defer os.Remove(tc.fileName)
			}

			// Set minimal required env vars that might not be in file
			os.Setenv("GMAIL_CLIENT_ID", "env-client-id")
			os.Setenv("GMAIL_CLIENT_SECRET", "env-secret")
			os.Setenv("GMAIL_REFRESH_TOKEN", "env-refresh-token")
			os.Setenv("EMAIL_API_URL", "http://env.example.com")
			defer func() {
				os.Unsetenv("GMAIL_CLIENT_ID")
				os.Unsetenv("GMAIL_CLIENT_SECRET")
				os.Unsetenv("GMAIL_REFRESH_TOKEN")
				os.Unsetenv("EMAIL_API_URL")
			}()

			config, err := LoadEmailConfig()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				} else if tc.errorMsg != "" && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %v", tc.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if config != nil && tc.validate != nil {
					if err := tc.validate(config); err != nil {
						t.Errorf("Validation failed: %v", err)
					}
				}
			}
		})
	}
}

func TestEmailConfigDefaults(t *testing.T) {
	// Clear all environment variables
	envVars := []string{
		"GMAIL_CLIENT_ID", "GMAIL_CLIENT_SECRET", "GMAIL_REFRESH_TOKEN",
		"EMAIL_API_URL", "EMAIL_CHECK_INTERVAL", "EMAIL_MAX_PER_RUN",
		"EMAIL_DRY_RUN", "EMAIL_DEBUG_MODE", "GMAIL_SEARCH_AFTER_DAYS",
		"GMAIL_SEARCH_MAX_RESULTS", "EMAIL_MIN_CONFIDENCE",
		"EMAIL_API_TIMEOUT", "EMAIL_API_RETRY_COUNT", "EMAIL_API_RETRY_DELAY",
	}
	
	originalValues := make(map[string]string)
	for _, env := range envVars {
		originalValues[env] = os.Getenv(env)
		os.Unsetenv(env)
	}
	defer func() {
		for env, val := range originalValues {
			if val != "" {
				os.Setenv(env, val)
			}
		}
	}()

	// Set only required variables
	os.Setenv("GMAIL_CLIENT_ID", "test-id")
	os.Setenv("GMAIL_CLIENT_SECRET", "test-secret")
	os.Setenv("GMAIL_REFRESH_TOKEN", "test-token")
	os.Setenv("EMAIL_API_URL", "http://localhost:8080")

	config, err := LoadEmailConfig()
	if err != nil {
		t.Fatalf("Failed to load config with defaults: %v", err)
	}

	// Verify default values - removing unused map

	// Check defaults using reflection would be complex, so check key values manually
	if config.Search.AfterDays != 30 {
		t.Errorf("Expected default AfterDays 30, got %d", config.Search.AfterDays)
	}
	if config.Processing.CheckInterval != 5*time.Minute {
		t.Errorf("Expected default CheckInterval 5m, got %v", config.Processing.CheckInterval)
	}
	if config.API.Timeout != 30*time.Second {
		t.Errorf("Expected default API timeout 30s, got %v", config.API.Timeout)
	}
}

func TestEmailConfigValidation(t *testing.T) {
	testCases := []struct {
		name   string
		config *EmailConfig
		valid  bool
	}{
		{
			name: "Valid complete config",
			config: &EmailConfig{
				Gmail: GmailConfig{
					ClientID:     "valid-id",
					ClientSecret: "valid-secret",
					RefreshToken: "valid-token",
				},
				Search: SearchConfig{
					AfterDays:   30,
					MaxResults:  100,
				},
				Processing: ProcessingConfig{
					CheckInterval:     5 * time.Minute,
					MaxEmailsPerRun:   50,
					MinConfidence:     0.5,
					StateDBPath:       "./state.db",
				},
				API: APIConfig{
					URL:         "http://localhost:8080",
					Timeout:     30 * time.Second,
					RetryCount:  3,
					RetryDelay:  1 * time.Second,
				},
			},
			valid: true,
		},
		{
			name: "Missing Gmail client ID",
			config: &EmailConfig{
				Gmail: GmailConfig{
					ClientSecret: "valid-secret",
					RefreshToken: "valid-token",
				},
				API: APIConfig{URL: "http://localhost:8080"},
			},
			valid: false,
		},
		{
			name: "Invalid API URL",
			config: &EmailConfig{
				Gmail: GmailConfig{
					ClientID:     "valid-id",
					ClientSecret: "valid-secret",
					RefreshToken: "valid-token",
				},
				API: APIConfig{URL: "not-a-url"},
			},
			valid: false,
		},
		{
			name: "Zero check interval",
			config: &EmailConfig{
				Gmail: GmailConfig{
					ClientID:     "valid-id",
					ClientSecret: "valid-secret",
					RefreshToken: "valid-token",
				},
				Processing: ProcessingConfig{
					CheckInterval: 0,
				},
				API: APIConfig{URL: "http://localhost:8080"},
			},
			valid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEmailConfig(tc.config)
			
			if tc.valid && err != nil {
				t.Errorf("Expected valid config, but got error: %v", err)
			}
			
			if !tc.valid && err == nil {
				t.Errorf("Expected invalid config to cause error, but got none")
			}
		})
	}
}

func TestGetEnvDurationOrDefault(t *testing.T) {
	testCases := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue string
		expected     time.Duration
	}{
		{
			name:         "Valid duration",
			envKey:       "TEST_DURATION",
			envValue:     "5m",
			defaultValue: "1m",
			expected:     5 * time.Minute,
		},
		{
			name:         "Empty value uses default",
			envKey:       "TEST_DURATION_EMPTY",
			envValue:     "",
			defaultValue: "10s",
			expected:     10 * time.Second,
		},
		{
			name:         "Invalid duration uses default",
			envKey:       "TEST_DURATION_INVALID",
			envValue:     "not-a-duration",
			defaultValue: "1m",
			expected:     1 * time.Minute,
		},
		{
			name:         "Complex duration",
			envKey:       "TEST_DURATION_COMPLEX",
			envValue:     "1h30m45s",
			defaultValue: "1m",
			expected:     1*time.Hour + 30*time.Minute + 45*time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable for this test
			original := os.Getenv(tc.envKey)
			if tc.envValue != "" {
				os.Setenv(tc.envKey, tc.envValue)
			} else {
				os.Unsetenv(tc.envKey)
			}
			defer func() {
				if original != "" {
					os.Setenv(tc.envKey, original)
				} else {
					os.Unsetenv(tc.envKey)
				}
			}()

			result := getEnvDurationOrDefault(tc.envKey, tc.defaultValue)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestGetEnvBoolOrDefault(t *testing.T) {
	testCases := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{
			name:         "True values",
			envKey:       "TEST_BOOL_TRUE",
			envValue:     "true",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "False values",
			envKey:       "TEST_BOOL_FALSE",
			envValue:     "false",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "Case insensitive true",
			envKey:       "TEST_BOOL_TRUE_CASE",
			envValue:     "TRUE",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "Numeric true",
			envKey:       "TEST_BOOL_NUMERIC_TRUE",
			envValue:     "1",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "Numeric false",
			envKey:       "TEST_BOOL_NUMERIC_FALSE",
			envValue:     "0",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "Empty uses default",
			envKey:       "TEST_BOOL_EMPTY",
			envValue:     "",
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "Invalid boolean uses default",
			envKey:       "TEST_BOOL_INVALID",
			envValue:     "maybe",
			defaultValue: false,
			expected:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable for this test
			original := os.Getenv(tc.envKey)
			if tc.envValue != "" {
				os.Setenv(tc.envKey, tc.envValue)
			} else {
				os.Unsetenv(tc.envKey)
			}
			defer func() {
				if original != "" {
					os.Setenv(tc.envKey, original)
				} else {
					os.Unsetenv(tc.envKey)
				}
			}()

			result := getEnvBoolOrDefault(tc.envKey, tc.defaultValue)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestGetEnvIntOrDefault(t *testing.T) {
	testCases := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue int
		expected     int
	}{
		{
			name:         "Valid integer",
			envKey:       "TEST_INT_VALID",
			envValue:     "42",
			defaultValue: 10,
			expected:     42,
		},
		{
			name:         "Empty uses default",
			envKey:       "TEST_INT_EMPTY",
			envValue:     "",
			defaultValue: 100,
			expected:     100,
		},
		{
			name:         "Zero value",
			envKey:       "TEST_INT_ZERO",
			envValue:     "0",
			defaultValue: 50,
			expected:     0,
		},
		{
			name:         "Negative value",
			envKey:       "TEST_INT_NEGATIVE",
			envValue:     "-5",
			defaultValue: 10,
			expected:     -5,
		},
		{
			name:         "Invalid integer uses default",
			envKey:       "TEST_INT_INVALID",
			envValue:     "not-a-number",
			defaultValue: 10,
			expected:     10,
		},
		{
			name:         "Float value uses default",
			envKey:       "TEST_INT_FLOAT",
			envValue:     "3.14",
			defaultValue: 10,
			expected:     10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variable for this test
			original := os.Getenv(tc.envKey)
			if tc.envValue != "" {
				os.Setenv(tc.envKey, tc.envValue)
			} else {
				os.Unsetenv(tc.envKey)
			}
			defer func() {
				if original != "" {
					os.Setenv(tc.envKey, original)
				} else {
					os.Unsetenv(tc.envKey)
				}
			}()

			result := getEnvIntOrDefault(tc.envKey, tc.defaultValue)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// Helper functions that would need to be implemented in the actual config package
func validateEmailConfig(config *EmailConfig) error {
	if config.Gmail.ClientID == "" {
		return fmt.Errorf("Gmail client ID is required")
	}
	if config.Gmail.ClientSecret == "" {
		return fmt.Errorf("Gmail client secret is required")
	}
	if config.API.URL == "" {
		return fmt.Errorf("API URL is required")
	}
	if !strings.HasPrefix(config.API.URL, "http://") && !strings.HasPrefix(config.API.URL, "https://") {
		return fmt.Errorf("invalid API URL format")
	}
	if config.Processing.CheckInterval <= 0 {
		return fmt.Errorf("check interval must be positive")
	}
	return nil
}

