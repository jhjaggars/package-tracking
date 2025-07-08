package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"package-tracking/internal/parser"
)

// SimplifiedConfig represents a simplified configuration for the email processing system
// This replaces the complex Viper-based multi-format configuration system with a
// simple, focused approach that only includes settings needed for the simplified workflow.
type SimplifiedConfig struct {
	// Core server settings
	ServerHost string
	ServerPort string
	DBPath     string
	LogLevel   string

	// Email processing settings
	EmailProcessor EmailProcessorConfig

	// LLM settings for description extraction only
	LLM parser.SimplifiedLLMConfig
}

// EmailProcessorConfig holds configuration for the simplified email processor
type EmailProcessorConfig struct {
	// Gmail settings
	GmailClientID     string
	GmailClientSecret string
	GmailRefreshToken string

	// Processing settings
	DaysToScan    int
	CheckInterval time.Duration
	DryRun        bool
	StateDBPath   string
	APIEndpoint   string

	// Search query for filtering emails
	SearchQuery string
}

// DefaultSimplifiedConfig returns a configuration with sensible defaults
func DefaultSimplifiedConfig() *SimplifiedConfig {
	return &SimplifiedConfig{
		ServerHost: "localhost",
		ServerPort: "8080",
		DBPath:     "./database.db",
		LogLevel:   "info",

		EmailProcessor: EmailProcessorConfig{
			DaysToScan:    30,
			CheckInterval: 5 * time.Minute,
			DryRun:        false,
			StateDBPath:   "./email-state.db",
			APIEndpoint:   "http://localhost:8080",
			SearchQuery:   buildSimplifiedSearchQuery(),
		},

		LLM: *parser.DefaultSimplifiedLLMConfig(),
	}
}

// LoadSimplifiedConfig loads configuration using simple environment variable lookup
// This replaces the complex Viper system with straightforward environment variable parsing
func LoadSimplifiedConfig() (*SimplifiedConfig, error) {
	config := DefaultSimplifiedConfig()

	// Load core server settings
	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.ServerHost = host
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		config.ServerPort = port
	}
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		config.DBPath = dbPath
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	// Load email processing settings
	if err := loadEmailProcessorConfig(&config.EmailProcessor); err != nil {
		return nil, fmt.Errorf("failed to load email processor config: %w", err)
	}

	// Load LLM settings
	if err := loadLLMConfig(&config.LLM); err != nil {
		return nil, fmt.Errorf("failed to load LLM config: %w", err)
	}

	return config, nil
}

// loadEmailProcessorConfig loads email processor configuration from environment variables
func loadEmailProcessorConfig(config *EmailProcessorConfig) error {
	// Gmail credentials
	if clientID := os.Getenv("GMAIL_CLIENT_ID"); clientID != "" {
		config.GmailClientID = clientID
	}
	if clientSecret := os.Getenv("GMAIL_CLIENT_SECRET"); clientSecret != "" {
		config.GmailClientSecret = clientSecret
	}
	if refreshToken := os.Getenv("GMAIL_REFRESH_TOKEN"); refreshToken != "" {
		config.GmailRefreshToken = refreshToken
	}

	// Processing settings
	if daysStr := os.Getenv("EMAIL_DAYS_TO_SCAN"); daysStr != "" {
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return fmt.Errorf("invalid EMAIL_DAYS_TO_SCAN value: %w", err)
		}
		config.DaysToScan = days
	}

	if intervalStr := os.Getenv("EMAIL_CHECK_INTERVAL"); intervalStr != "" {
		interval, err := time.ParseDuration(intervalStr)
		if err != nil {
			return fmt.Errorf("invalid EMAIL_CHECK_INTERVAL value: %w", err)
		}
		config.CheckInterval = interval
	}

	if dryRunStr := os.Getenv("EMAIL_DRY_RUN"); dryRunStr != "" {
		dryRun, err := strconv.ParseBool(dryRunStr)
		if err != nil {
			return fmt.Errorf("invalid EMAIL_DRY_RUN value: %w", err)
		}
		config.DryRun = dryRun
	}

	if stateDBPath := os.Getenv("EMAIL_STATE_DB_PATH"); stateDBPath != "" {
		config.StateDBPath = stateDBPath
	}

	if apiEndpoint := os.Getenv("EMAIL_API_URL"); apiEndpoint != "" {
		config.APIEndpoint = apiEndpoint
	}

	if searchQuery := os.Getenv("GMAIL_SEARCH_QUERY"); searchQuery != "" {
		config.SearchQuery = searchQuery
	}

	return nil
}

// loadLLMConfig loads LLM configuration from environment variables
func loadLLMConfig(config *parser.SimplifiedLLMConfig) error {
	// Provider and model
	if provider := os.Getenv("LLM_PROVIDER"); provider != "" {
		config.Provider = provider
	}
	if model := os.Getenv("LLM_MODEL"); model != "" {
		config.Model = model
	}
	if endpoint := os.Getenv("LLM_ENDPOINT"); endpoint != "" {
		config.Endpoint = endpoint
	}
	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}

	// Numeric settings
	if maxTokensStr := os.Getenv("LLM_MAX_TOKENS"); maxTokensStr != "" {
		maxTokens, err := strconv.Atoi(maxTokensStr)
		if err != nil {
			return fmt.Errorf("invalid LLM_MAX_TOKENS value: %w", err)
		}
		config.MaxTokens = maxTokens
	}

	if temperatureStr := os.Getenv("LLM_TEMPERATURE"); temperatureStr != "" {
		temperature, err := strconv.ParseFloat(temperatureStr, 64)
		if err != nil {
			return fmt.Errorf("invalid LLM_TEMPERATURE value: %w", err)
		}
		config.Temperature = temperature
	}

	if retryCountStr := os.Getenv("LLM_RETRY_COUNT"); retryCountStr != "" {
		retryCount, err := strconv.Atoi(retryCountStr)
		if err != nil {
			return fmt.Errorf("invalid LLM_RETRY_COUNT value: %w", err)
		}
		config.RetryCount = retryCount
	}

	if timeoutStr := os.Getenv("LLM_TIMEOUT"); timeoutStr != "" {
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return fmt.Errorf("invalid LLM_TIMEOUT value: %w", err)
		}
		config.Timeout = timeout
	}

	// Enabled flag
	if enabledStr := os.Getenv("LLM_ENABLED"); enabledStr != "" {
		enabled, err := strconv.ParseBool(enabledStr)
		if err != nil {
			return fmt.Errorf("invalid LLM_ENABLED value: %w", err)
		}
		config.Enabled = enabled
	}

	return nil
}

// buildSimplifiedSearchQuery creates the default Gmail search query for shipping emails
func buildSimplifiedSearchQuery() string {
	// Simple search query focused on shipping emails from major carriers
	carriers := []string{
		"from:ups.com OR from:usps.com OR from:fedex.com OR from:dhl.com OR from:amazon.com",
		"subject:shipped OR subject:tracking OR subject:delivery OR subject:package",
		"has:attachment OR body:tracking",
	}
	return "(" + strings.Join(carriers, " OR ") + ")"
}

// Validate validates the simplified configuration
func (c *SimplifiedConfig) Validate() error {
	// Validate core settings
	if c.ServerPort == "" {
		return fmt.Errorf("server port is required")
	}
	if c.DBPath == "" {
		return fmt.Errorf("database path is required")
	}

	// Validate LLM settings if enabled (check provider first)
	if c.LLM.Enabled {
		if c.LLM.Provider == "" || c.LLM.Provider == "disabled" {
			return fmt.Errorf("LLM provider is required when LLM is enabled")
		}
		if c.LLM.Provider == "ollama" && c.LLM.Endpoint == "" {
			return fmt.Errorf("LLM endpoint is required for Ollama provider")
		}
		if (c.LLM.Provider == "openai" || c.LLM.Provider == "anthropic") && c.LLM.APIKey == "" {
			return fmt.Errorf("LLM API key is required for cloud providers")
		}
		
		// Validate email processor settings if LLM is enabled
		if c.EmailProcessor.GmailClientID == "" {
			return fmt.Errorf("Gmail client ID is required when LLM is enabled")
		}
		if c.EmailProcessor.GmailClientSecret == "" {
			return fmt.Errorf("Gmail client secret is required when LLM is enabled")
		}
		if c.EmailProcessor.GmailRefreshToken == "" {
			return fmt.Errorf("Gmail refresh token is required when LLM is enabled")
		}
	}

	return nil
}

// ToEnvExample generates an example .env file content for this configuration
func (c *SimplifiedConfig) ToEnvExample() string {
	return `# Simplified Package Tracker Configuration
# This replaces the complex multi-format configuration system

# Core Server Settings
SERVER_HOST=localhost
SERVER_PORT=8080
DB_PATH=./database.db
LOG_LEVEL=info

# Email Processing Settings
GMAIL_CLIENT_ID=your-gmail-client-id
GMAIL_CLIENT_SECRET=your-gmail-client-secret
GMAIL_REFRESH_TOKEN=your-gmail-refresh-token
EMAIL_DAYS_TO_SCAN=30
EMAIL_CHECK_INTERVAL=5m
EMAIL_DRY_RUN=false
EMAIL_STATE_DB_PATH=./email-state.db
EMAIL_API_URL=http://localhost:8080
GMAIL_SEARCH_QUERY="(from:ups.com OR from:usps.com OR from:fedex.com OR from:dhl.com OR from:amazon.com OR subject:shipped OR subject:tracking OR subject:delivery OR subject:package)"

# LLM Settings (for description extraction only)
LLM_ENABLED=true
LLM_PROVIDER=ollama
LLM_MODEL=llama3.2
LLM_ENDPOINT=http://localhost:11434
LLM_API_KEY=your-api-key-for-cloud-providers
LLM_MAX_TOKENS=1000
LLM_TEMPERATURE=0.1
LLM_TIMEOUT=120s
LLM_RETRY_COUNT=2
`
}