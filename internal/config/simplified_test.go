package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"package-tracking/internal/parser"
)

func TestDefaultSimplifiedConfig(t *testing.T) {
	config := DefaultSimplifiedConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "localhost", config.ServerHost)
	assert.Equal(t, "8080", config.ServerPort)
	assert.Equal(t, "./database.db", config.DBPath)
	assert.Equal(t, "info", config.LogLevel)
	
	// Check email processor defaults
	assert.Equal(t, 30, config.EmailProcessor.DaysToScan)
	assert.Equal(t, 5*time.Minute, config.EmailProcessor.CheckInterval)
	assert.False(t, config.EmailProcessor.DryRun)
	assert.Equal(t, "./email-state.db", config.EmailProcessor.StateDBPath)
	assert.Equal(t, "http://localhost:8080", config.EmailProcessor.APIEndpoint)
	assert.Contains(t, config.EmailProcessor.SearchQuery, "from:ups.com")
	
	// Check LLM defaults
	assert.Equal(t, "disabled", config.LLM.Provider)
	assert.False(t, config.LLM.Enabled)
}

func TestLoadSimplifiedConfig_Defaults(t *testing.T) {
	// Clear environment to test defaults
	clearTestEnv()
	
	config, err := LoadSimplifiedConfig()
	
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "localhost", config.ServerHost)
	assert.Equal(t, "8080", config.ServerPort)
}

func TestLoadSimplifiedConfig_WithEnvironmentVariables(t *testing.T) {
	// Clear environment first
	clearTestEnv()
	
	// Set test environment variables
	os.Setenv("SERVER_HOST", "0.0.0.0")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("DB_PATH", "/tmp/test.db")
	os.Setenv("LOG_LEVEL", "debug")
	
	os.Setenv("GMAIL_CLIENT_ID", "test-client-id")
	os.Setenv("GMAIL_CLIENT_SECRET", "test-client-secret")
	os.Setenv("GMAIL_REFRESH_TOKEN", "test-refresh-token")
	os.Setenv("EMAIL_DAYS_TO_SCAN", "7")
	os.Setenv("EMAIL_CHECK_INTERVAL", "10m")
	os.Setenv("EMAIL_DRY_RUN", "true")
	
	os.Setenv("LLM_ENABLED", "true")
	os.Setenv("LLM_PROVIDER", "ollama")
	os.Setenv("LLM_MODEL", "llama3.2")
	os.Setenv("LLM_ENDPOINT", "http://localhost:11434")
	os.Setenv("LLM_MAX_TOKENS", "500")
	os.Setenv("LLM_TEMPERATURE", "0.2")
	os.Setenv("LLM_TIMEOUT", "60s")
	os.Setenv("LLM_RETRY_COUNT", "3")
	
	defer clearTestEnv()
	
	config, err := LoadSimplifiedConfig()
	
	assert.NoError(t, err)
	assert.NotNil(t, config)
	
	// Check server settings
	assert.Equal(t, "0.0.0.0", config.ServerHost)
	assert.Equal(t, "9090", config.ServerPort)
	assert.Equal(t, "/tmp/test.db", config.DBPath)
	assert.Equal(t, "debug", config.LogLevel)
	
	// Check email processor settings
	assert.Equal(t, "test-client-id", config.EmailProcessor.GmailClientID)
	assert.Equal(t, "test-client-secret", config.EmailProcessor.GmailClientSecret)
	assert.Equal(t, "test-refresh-token", config.EmailProcessor.GmailRefreshToken)
	assert.Equal(t, 7, config.EmailProcessor.DaysToScan)
	assert.Equal(t, 10*time.Minute, config.EmailProcessor.CheckInterval)
	assert.True(t, config.EmailProcessor.DryRun)
	
	// Check LLM settings
	assert.True(t, config.LLM.Enabled)
	assert.Equal(t, "ollama", config.LLM.Provider)
	assert.Equal(t, "llama3.2", config.LLM.Model)
	assert.Equal(t, "http://localhost:11434", config.LLM.Endpoint)
	assert.Equal(t, 500, config.LLM.MaxTokens)
	assert.Equal(t, 0.2, config.LLM.Temperature)
	assert.Equal(t, 60*time.Second, config.LLM.Timeout)
	assert.Equal(t, 3, config.LLM.RetryCount)
}

func TestLoadSimplifiedConfig_InvalidValues(t *testing.T) {
	clearTestEnv()
	
	tests := []struct {
		name   string
		envVar string
		value  string
		hasError bool
	}{
		{
			name:     "Invalid days to scan",
			envVar:   "EMAIL_DAYS_TO_SCAN",
			value:    "invalid",
			hasError: true,
		},
		{
			name:     "Invalid check interval",
			envVar:   "EMAIL_CHECK_INTERVAL",
			value:    "invalid",
			hasError: true,
		},
		{
			name:     "Invalid dry run flag",
			envVar:   "EMAIL_DRY_RUN",
			value:    "invalid",
			hasError: true,
		},
		{
			name:     "Invalid LLM max tokens",
			envVar:   "LLM_MAX_TOKENS",
			value:    "invalid",
			hasError: true,
		},
		{
			name:     "Invalid LLM temperature",
			envVar:   "LLM_TEMPERATURE",
			value:    "invalid",
			hasError: true,
		},
		{
			name:     "Invalid LLM timeout",
			envVar:   "LLM_TIMEOUT",
			value:    "invalid",
			hasError: true,
		},
		{
			name:     "Invalid LLM enabled flag",
			envVar:   "LLM_ENABLED",
			value:    "invalid",
			hasError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearTestEnv()
			os.Setenv(tt.envVar, tt.value)
			defer os.Unsetenv(tt.envVar)
			
			_, err := LoadSimplifiedConfig()
			
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSimplifiedConfig_Validate(t *testing.T) {
	tests := []struct {
		name     string
		config   *SimplifiedConfig
		hasError bool
		errorMsg string
	}{
		{
			name:     "Valid default config",
			config:   DefaultSimplifiedConfig(),
			hasError: false,
		},
		{
			name: "Missing server port",
			config: &SimplifiedConfig{
				ServerPort: "",
				DBPath:     "./test.db",
			},
			hasError: true,
			errorMsg: "server port is required",
		},
		{
			name: "Missing database path",
			config: &SimplifiedConfig{
				ServerPort: "8080",
				DBPath:     "",
			},
			hasError: true,
			errorMsg: "database path is required",
		},
		{
			name: "LLM enabled but missing Gmail credentials",
			config: &SimplifiedConfig{
				ServerPort: "8080",
				DBPath:     "./test.db",
				LLM: parser.SimplifiedLLMConfig{
					Enabled:  true,
					Provider: "ollama",
					Endpoint: "http://localhost:11434", // Valid endpoint so it gets to Gmail validation
				},
				EmailProcessor: EmailProcessorConfig{
					GmailClientID: "", // Missing
				},
			},
			hasError: true,
			errorMsg: "Gmail client ID is required when LLM is enabled",
		},
		{
			name: "LLM enabled but no provider",
			config: &SimplifiedConfig{
				ServerPort: "8080",
				DBPath:     "./test.db",
				LLM: parser.SimplifiedLLMConfig{
					Enabled:  true,
					Provider: "", // Missing
				},
			},
			hasError: true,
			errorMsg: "LLM provider is required when LLM is enabled",
		},
		{
			name: "Ollama provider but no endpoint",
			config: &SimplifiedConfig{
				ServerPort: "8080",
				DBPath:     "./test.db",
				LLM: parser.SimplifiedLLMConfig{
					Enabled:  true,
					Provider: "ollama",
					Endpoint: "", // Missing for Ollama
				},
				EmailProcessor: EmailProcessorConfig{
					GmailClientID:     "test",
					GmailClientSecret: "test",
					GmailRefreshToken: "test",
				},
			},
			hasError: true,
			errorMsg: "LLM endpoint is required for Ollama provider",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.hasError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildDefaultSearchQuery(t *testing.T) {
	query := buildSimplifiedSearchQuery()
	
	assert.NotEmpty(t, query)
	assert.Contains(t, query, "from:ups.com")
	assert.Contains(t, query, "from:usps.com")
	assert.Contains(t, query, "from:fedex.com")
	assert.Contains(t, query, "from:dhl.com")
	assert.Contains(t, query, "from:amazon.com")
	assert.Contains(t, query, "subject:shipped")
	assert.Contains(t, query, "subject:tracking")
}

func TestToEnvExample(t *testing.T) {
	config := DefaultSimplifiedConfig()
	example := config.ToEnvExample()
	
	assert.NotEmpty(t, example)
	assert.Contains(t, example, "SERVER_HOST=localhost")
	assert.Contains(t, example, "SERVER_PORT=8080")
	assert.Contains(t, example, "LLM_ENABLED=true")
	assert.Contains(t, example, "GMAIL_CLIENT_ID=")
	assert.Contains(t, example, "# Simplified Package Tracker Configuration")
}

// Helper function to clear test environment variables
func clearTestEnv() {
	envVars := []string{
		"SERVER_HOST", "SERVER_PORT", "DB_PATH", "LOG_LEVEL",
		"GMAIL_CLIENT_ID", "GMAIL_CLIENT_SECRET", "GMAIL_REFRESH_TOKEN",
		"EMAIL_DAYS_TO_SCAN", "EMAIL_CHECK_INTERVAL", "EMAIL_DRY_RUN",
		"EMAIL_STATE_DB_PATH", "EMAIL_API_URL", "GMAIL_SEARCH_QUERY",
		"LLM_ENABLED", "LLM_PROVIDER", "LLM_MODEL", "LLM_ENDPOINT",
		"LLM_API_KEY", "LLM_MAX_TOKENS", "LLM_TEMPERATURE", "LLM_TIMEOUT",
		"LLM_RETRY_COUNT",
	}
	
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}