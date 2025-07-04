package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestEmailViperConfig_LoadFromDefaults(t *testing.T) {
	// Clear environment variables to test defaults
	clearEmailEnvVars()

	// Set minimal required config to disable auth requirements
	os.Setenv("PKG_TRACKER_EMAIL_GMAIL_USERNAME", "test@gmail.com")
	os.Setenv("PKG_TRACKER_EMAIL_GMAIL_APP_PASSWORD", "test-password")
	defer func() {
		os.Unsetenv("PKG_TRACKER_EMAIL_GMAIL_USERNAME")
		os.Unsetenv("PKG_TRACKER_EMAIL_GMAIL_APP_PASSWORD")
	}()

	v := viper.New()
	config, err := LoadEmailConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test default values
	if config.Gmail.Username != "test@gmail.com" {
		t.Errorf("Expected Gmail.Username to be 'test@gmail.com', got '%s'", config.Gmail.Username)
	}
	if config.Gmail.MaxResults != 100 {
		t.Errorf("Expected Gmail.MaxResults to be 100, got %d", config.Gmail.MaxResults)
	}
	if config.Processing.CheckInterval != 5*time.Minute {
		t.Errorf("Expected Processing.CheckInterval to be 5m, got %v", config.Processing.CheckInterval)
	}
	if config.Processing.DryRun != false {
		t.Errorf("Expected Processing.DryRun to be false, got %v", config.Processing.DryRun)
	}
	if config.API.URL != "http://localhost:8080" {
		t.Errorf("Expected API.URL to be 'http://localhost:8080', got '%s'", config.API.URL)
	}
	if config.LLM.Provider != LLMProviderDisabled {
		t.Errorf("Expected LLM.Provider to be '%s', got '%s'", LLMProviderDisabled, config.LLM.Provider)
	}
}

func TestEmailViperConfig_LoadFromEnvironment(t *testing.T) {
	// Clear environment variables first
	clearEmailEnvVars()

	// Set test environment variables with new PKG_TRACKER prefix
	envVars := map[string]string{
		"PKG_TRACKER_EMAIL_GMAIL_CLIENT_ID":       "test-client-id",
		"PKG_TRACKER_EMAIL_GMAIL_CLIENT_SECRET":   "test-client-secret",
		"PKG_TRACKER_EMAIL_GMAIL_REFRESH_TOKEN":   "test-refresh-token",
		"PKG_TRACKER_EMAIL_GMAIL_MAX_RESULTS":     "50",
		"PKG_TRACKER_EMAIL_PROCESSING_CHECK_INTERVAL": "10m",
		"PKG_TRACKER_EMAIL_PROCESSING_DRY_RUN":    "true",
		"PKG_TRACKER_EMAIL_PROCESSING_STATE_DB_PATH": "./test-email.db",
		"PKG_TRACKER_EMAIL_API_URL":               "http://test-api:9090",
		"PKG_TRACKER_EMAIL_LLM_PROVIDER":          "openai",
		"PKG_TRACKER_EMAIL_LLM_API_KEY":           "test-llm-key",
		"PKG_TRACKER_EMAIL_LLM_ENABLED":           "true",
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
	config, err := LoadEmailConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test environment variable values
	if config.Gmail.ClientID != "test-client-id" {
		t.Errorf("Expected Gmail.ClientID to be 'test-client-id', got '%s'", config.Gmail.ClientID)
	}
	if config.Gmail.ClientSecret != "test-client-secret" {
		t.Errorf("Expected Gmail.ClientSecret to be 'test-client-secret', got '%s'", config.Gmail.ClientSecret)
	}
	if config.Gmail.MaxResults != 50 {
		t.Errorf("Expected Gmail.MaxResults to be 50, got %d", config.Gmail.MaxResults)
	}
	if config.Processing.CheckInterval != 10*time.Minute {
		t.Errorf("Expected Processing.CheckInterval to be 10m, got %v", config.Processing.CheckInterval)
	}
	if config.Processing.DryRun != true {
		t.Errorf("Expected Processing.DryRun to be true, got %v", config.Processing.DryRun)
	}
	if config.Processing.StateDBPath != "./test-email.db" {
		t.Errorf("Expected Processing.StateDBPath to be './test-email.db', got '%s'", config.Processing.StateDBPath)
	}
	if config.API.URL != "http://test-api:9090" {
		t.Errorf("Expected API.URL to be 'http://test-api:9090', got '%s'", config.API.URL)
	}
	if config.LLM.Provider != "openai" {
		t.Errorf("Expected LLM.Provider to be 'openai', got '%s'", config.LLM.Provider)
	}
	if config.LLM.APIKey != "test-llm-key" {
		t.Errorf("Expected LLM.APIKey to be 'test-llm-key', got '%s'", config.LLM.APIKey)
	}
	if config.LLM.Enabled != true {
		t.Errorf("Expected LLM.Enabled to be true, got %v", config.LLM.Enabled)
	}
}

func TestEmailViperConfig_LoadFromYAMLFile(t *testing.T) {
	// Clear environment variables first
	clearEmailEnvVars()

	// Create temporary YAML config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "email.yaml")
	configContent := `gmail:
  client_id: "yaml-client-id"
  client_secret: "yaml-client-secret"
  refresh_token: "yaml-refresh-token"
  max_results: 75

search:
  query: "custom yaml search query"
  after_days: 14
  max_results: 75

processing:
  check_interval: "15m"
  dry_run: false
  state_db_path: "./yaml-email.db"
  min_confidence: 0.8

api:
  url: "http://yaml-api:8888"
  timeout: "45s"

llm:
  provider: "anthropic"
  api_key: "yaml-llm-key"
  enabled: false
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configFile)
	config, err := LoadEmailConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test YAML file values
	if config.Gmail.ClientID != "yaml-client-id" {
		t.Errorf("Expected Gmail.ClientID to be 'yaml-client-id', got '%s'", config.Gmail.ClientID)
	}
	if config.Gmail.MaxResults != 75 {
		t.Errorf("Expected Gmail.MaxResults to be 75, got %d", config.Gmail.MaxResults)
	}
	if config.Search.Query != "custom yaml search query" {
		t.Errorf("Expected Search.Query to be 'custom yaml search query', got '%s'", config.Search.Query)
	}
	if config.Search.AfterDays != 14 {
		t.Errorf("Expected Search.AfterDays to be 14, got %d", config.Search.AfterDays)
	}
	if config.Processing.CheckInterval != 15*time.Minute {
		t.Errorf("Expected Processing.CheckInterval to be 15m, got %v", config.Processing.CheckInterval)
	}
	if config.Processing.MinConfidence != 0.8 {
		t.Errorf("Expected Processing.MinConfidence to be 0.8, got %f", config.Processing.MinConfidence)
	}
	if config.API.URL != "http://yaml-api:8888" {
		t.Errorf("Expected API.URL to be 'http://yaml-api:8888', got '%s'", config.API.URL)
	}
	if config.API.Timeout != 45*time.Second {
		t.Errorf("Expected API.Timeout to be 45s, got %v", config.API.Timeout)
	}
	if config.LLM.Provider != "anthropic" {
		t.Errorf("Expected LLM.Provider to be 'anthropic', got '%s'", config.LLM.Provider)
	}
	if config.LLM.Enabled != false {
		t.Errorf("Expected LLM.Enabled to be false, got %v", config.LLM.Enabled)
	}
}

func TestEmailViperConfig_BackwardCompatibility(t *testing.T) {
	// Clear environment variables first
	clearEmailEnvVars()

	// Set old environment variables to test backward compatibility
	oldEnvVars := map[string]string{
		"GMAIL_CLIENT_ID":       "old-client-id",
		"GMAIL_CLIENT_SECRET":   "old-client-secret",
		"GMAIL_REFRESH_TOKEN":   "old-refresh-token",
		"EMAIL_CHECK_INTERVAL":  "20m",
		"EMAIL_DRY_RUN":         "true",
		"EMAIL_API_URL":         "http://old-api:7070",
		"LLM_PROVIDER":          "openai",
		"LLM_API_KEY":           "old-llm-key",
		"LLM_ENABLED":           "true",
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
	config, err := LoadEmailConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test that old environment variables still work
	if config.Gmail.ClientID != "old-client-id" {
		t.Errorf("Expected Gmail.ClientID to be 'old-client-id', got '%s'", config.Gmail.ClientID)
	}
	if config.Gmail.ClientSecret != "old-client-secret" {
		t.Errorf("Expected Gmail.ClientSecret to be 'old-client-secret', got '%s'", config.Gmail.ClientSecret)
	}
	if config.Processing.CheckInterval != 20*time.Minute {
		t.Errorf("Expected Processing.CheckInterval to be 20m, got %v", config.Processing.CheckInterval)
	}
	if config.Processing.DryRun != true {
		t.Errorf("Expected Processing.DryRun to be true, got %v", config.Processing.DryRun)
	}
	if config.API.URL != "http://old-api:7070" {
		t.Errorf("Expected API.URL to be 'http://old-api:7070', got '%s'", config.API.URL)
	}
	if config.LLM.Provider != "openai" {
		t.Errorf("Expected LLM.Provider to be 'openai', got '%s'", config.LLM.Provider)
	}
	if config.LLM.Enabled != true {
		t.Errorf("Expected LLM.Enabled to be true, got %v", config.LLM.Enabled)
	}
}

func TestEmailViperConfig_NewFormatOverridesOld(t *testing.T) {
	// Clear environment variables first
	clearEmailEnvVars()

	// Set both old and new environment variables
	envVars := map[string]string{
		// Old format
		"GMAIL_CLIENT_ID":      "old-client-id",
		"GMAIL_CLIENT_SECRET":  "old-client-secret",
		"EMAIL_API_URL":        "http://old-api:7070",
		"LLM_PROVIDER":         "openai",
		// New format (should override old)
		"PKG_TRACKER_EMAIL_GMAIL_CLIENT_ID":     "new-client-id",
		"PKG_TRACKER_EMAIL_GMAIL_CLIENT_SECRET": "new-client-secret",
		"PKG_TRACKER_EMAIL_API_URL":             "http://new-api:9090",
		"PKG_TRACKER_EMAIL_LLM_PROVIDER":        "anthropic",
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
	config, err := LoadEmailConfigWithViper(v)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Test that new format overrides old
	if config.Gmail.ClientID != "new-client-id" {
		t.Errorf("Expected Gmail.ClientID to be 'new-client-id' (new format), got '%s'", config.Gmail.ClientID)
	}
	if config.Gmail.ClientSecret != "new-client-secret" {
		t.Errorf("Expected Gmail.ClientSecret to be 'new-client-secret' (new format), got '%s'", config.Gmail.ClientSecret)
	}
	if config.API.URL != "http://new-api:9090" {
		t.Errorf("Expected API.URL to be 'http://new-api:9090' (new format), got '%s'", config.API.URL)
	}
	if config.LLM.Provider != "anthropic" {
		t.Errorf("Expected LLM.Provider to be 'anthropic' (new format), got '%s'", config.LLM.Provider)
	}
}

func TestEmailViperConfig_ValidationErrors(t *testing.T) {
	// Clear environment variables first
	clearEmailEnvVars()

	tests := []struct {
		name       string
		envVars    map[string]string
		configFile string
		errorMsg   string
	}{
		{
			name: "no authentication configured",
			envVars: map[string]string{
				// No Gmail auth configured
			},
			errorMsg: "invalid configuration: either Gmail OAuth2 (client_id) or IMAP (username) credentials must be provided",
		},
		{
			name: "OAuth2 missing client secret",
			envVars: map[string]string{
				"PKG_TRACKER_EMAIL_GMAIL_CLIENT_ID": "test-client-id",
				// Missing client secret
			},
			errorMsg: "invalid configuration: gmail_client_secret is required when using OAuth2",
		},
		{
			name: "IMAP missing app password",
			envVars: map[string]string{
				"PKG_TRACKER_EMAIL_GMAIL_USERNAME": "test@gmail.com",
				// Missing app password
			},
			errorMsg: "invalid configuration: gmail_app_password is required when using IMAP",
		},
		{
			name: "invalid check interval",
			envVars: map[string]string{
				"PKG_TRACKER_EMAIL_GMAIL_USERNAME":     "test@gmail.com",
				"PKG_TRACKER_EMAIL_GMAIL_APP_PASSWORD": "test-password",
				"PKG_TRACKER_EMAIL_PROCESSING_CHECK_INTERVAL": "30s",
			},
			errorMsg: "invalid configuration: check_interval must be at least 1 minute",
		},
		{
			name: "empty API URL",
			configFile: `gmail:
  username: "test@gmail.com"
  app_password: "test-password"
api:
  url: ""`,
			errorMsg: "invalid configuration: API URL cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables
			clearEmailEnvVars()

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
				configFile := filepath.Join(tempDir, "email.yaml")
				err := os.WriteFile(configFile, []byte(tt.configFile), 0644)
				if err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}
				v.SetConfigFile(configFile)
			}
			
			_, err := LoadEmailConfigWithViper(v)
			if err == nil {
				t.Errorf("Expected error, got nil")
			}
			if err != nil && err.Error() != tt.errorMsg {
				t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
			}
		})
	}
}

// Helper function to clear email environment variables
func clearEmailEnvVars() {
	// Clear new format variables
	newVars := []string{
		"PKG_TRACKER_EMAIL_GMAIL_CLIENT_ID", "PKG_TRACKER_EMAIL_GMAIL_CLIENT_SECRET",
		"PKG_TRACKER_EMAIL_GMAIL_REFRESH_TOKEN", "PKG_TRACKER_EMAIL_GMAIL_USERNAME",
		"PKG_TRACKER_EMAIL_GMAIL_APP_PASSWORD", "PKG_TRACKER_EMAIL_GMAIL_MAX_RESULTS",
		"PKG_TRACKER_EMAIL_PROCESSING_CHECK_INTERVAL", "PKG_TRACKER_EMAIL_PROCESSING_DRY_RUN",
		"PKG_TRACKER_EMAIL_PROCESSING_STATE_DB_PATH", "PKG_TRACKER_EMAIL_API_URL",
		"PKG_TRACKER_EMAIL_LLM_PROVIDER", "PKG_TRACKER_EMAIL_LLM_API_KEY",
		"PKG_TRACKER_EMAIL_LLM_ENABLED",
	}

	// Clear old format variables
	oldVars := []string{
		"GMAIL_CLIENT_ID", "GMAIL_CLIENT_SECRET", "GMAIL_REFRESH_TOKEN",
		"GMAIL_USERNAME", "GMAIL_APP_PASSWORD", "GMAIL_MAX_RESULTS",
		"EMAIL_CHECK_INTERVAL", "EMAIL_DRY_RUN", "EMAIL_STATE_DB_PATH",
		"EMAIL_API_URL", "LLM_PROVIDER", "LLM_API_KEY", "LLM_ENABLED",
	}

	allVars := append(newVars, oldVars...)
	for _, key := range allVars {
		os.Unsetenv(key)
	}
}