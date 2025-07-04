package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// LLM Provider constants
const (
	LLMProviderOpenAI    = "openai"
	LLMProviderAnthropic = "anthropic"
	LLMProviderLocal     = "local"
	LLMProviderDisabled  = "disabled"
)

// EmailConfig holds all email processing configuration
type EmailConfig struct {
	// Gmail API Configuration
	Gmail GmailConfig `json:"gmail"`
	
	// Search Configuration
	Search SearchConfig `json:"search"`
	
	// Processing Configuration
	Processing ProcessingConfig `json:"processing"`
	
	// API Configuration
	API APIConfig `json:"api"`
	
	// LLM Configuration
	LLM LLMConfig `json:"llm"`
}

// GmailConfig holds Gmail-specific configuration
type GmailConfig struct {
	// OAuth2 Settings
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	TokenFile    string `json:"token_file"`
	
	// IMAP Fallback Settings
	Username    string `json:"username"`
	AppPassword string `json:"app_password"`
	
	// Request Settings
	MaxResults      int64         `json:"max_results"`
	RequestTimeout  time.Duration `json:"request_timeout"`
	RateLimitDelay  time.Duration `json:"rate_limit_delay"`
}

// SearchConfig holds email search configuration
type SearchConfig struct {
	Query          string   `json:"query"`
	AfterDays      int      `json:"after_days"`
	UnreadOnly     bool     `json:"unread_only"`
	MaxResults     int      `json:"max_results"`
	IncludeLabels  []string `json:"include_labels"`
	ExcludeLabels  []string `json:"exclude_labels"`
	CustomCarriers []string `json:"custom_carriers"`
}

// ProcessingConfig holds email processing configuration
type ProcessingConfig struct {
	CheckInterval     time.Duration `json:"check_interval"`
	MaxEmailsPerRun   int           `json:"max_emails_per_run"`
	DryRun            bool          `json:"dry_run"`
	StateDBPath       string        `json:"state_db_path"`
	ProcessingTimeout time.Duration `json:"processing_timeout"`
	
	// Parsing Configuration
	MinConfidence       float64 `json:"min_confidence"`
	MaxCandidates       int     `json:"max_candidates"`
	UseHybridValidation bool    `json:"use_hybrid_validation"`
	DebugMode           bool    `json:"debug_mode"`
}

// APIConfig holds API client configuration
type APIConfig struct {
	URL           string        `json:"url"`
	Timeout       time.Duration `json:"timeout"`
	RetryCount    int           `json:"retry_count"`
	RetryDelay    time.Duration `json:"retry_delay"`
	UserAgent     string        `json:"user_agent"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// LLMConfig holds LLM integration configuration
type LLMConfig struct {
	Provider    string        `json:"provider"`     // "openai", "anthropic", "local", "disabled"
	Model       string        `json:"model"`        // "gpt-4", "claude-3", "llama2", etc.
	APIKey      string        `json:"api_key"`      // API key for hosted services
	Endpoint    string        `json:"endpoint"`     // For local LLMs
	MaxTokens   int           `json:"max_tokens"`   // Response length limit
	Temperature float64       `json:"temperature"`  // Creativity vs consistency
	Timeout     time.Duration `json:"timeout"`      // Request timeout
	RetryCount  int           `json:"retry_count"`  // Number of retries
	Enabled     bool          `json:"enabled"`      // Enable/disable LLM parsing
}

// LoadEmailConfig loads email configuration from environment variables
func LoadEmailConfig() (*EmailConfig, error) {
	return LoadEmailConfigWithEnvFile("")
}

// LoadEmailConfigWithEnvFile loads email configuration from environment variables
// and optionally loads a .env file first
func LoadEmailConfigWithEnvFile(envFile string) (*EmailConfig, error) {
	// Load .env file if specified
	if envFile != "" {
		if err := LoadEnvFile(envFile); err != nil {
			return nil, fmt.Errorf("failed to load env file %s: %w", envFile, err)
		}
	} else {
		// Try to load default .env file
		if err := LoadEnvFile(".env"); err != nil {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
	}
	config := &EmailConfig{
		Gmail: GmailConfig{
			ClientID:       getEnvOrDefault("GMAIL_CLIENT_ID", ""),
			ClientSecret:   getEnvOrDefault("GMAIL_CLIENT_SECRET", ""),
			RefreshToken:   getEnvOrDefault("GMAIL_REFRESH_TOKEN", ""),
			AccessToken:    getEnvOrDefault("GMAIL_ACCESS_TOKEN", ""),
			TokenFile:      getEnvOrDefault("GMAIL_TOKEN_FILE", "./gmail-token.json"),
			Username:       getEnvOrDefault("GMAIL_USERNAME", ""),
			AppPassword:    getEnvOrDefault("GMAIL_APP_PASSWORD", ""),
			MaxResults:     getEnvInt64OrDefault("GMAIL_MAX_RESULTS", 100),
			RequestTimeout: getEnvDurationOrDefault("GMAIL_REQUEST_TIMEOUT", "30s"),
			RateLimitDelay: getEnvDurationOrDefault("GMAIL_RATE_LIMIT_DELAY", "100ms"),
		},
		
		Search: SearchConfig{
			Query:         getEnvOrDefault("GMAIL_SEARCH_QUERY", ""),
			AfterDays:     getEnvIntOrDefault("GMAIL_SEARCH_AFTER_DAYS", 30),
			UnreadOnly:    getEnvBoolOrDefault("GMAIL_SEARCH_UNREAD_ONLY", false),
			MaxResults:    getEnvIntOrDefault("GMAIL_SEARCH_MAX_RESULTS", 100),
			IncludeLabels: getEnvSliceOrDefault("GMAIL_INCLUDE_LABELS", []string{}),
			ExcludeLabels: getEnvSliceOrDefault("GMAIL_EXCLUDE_LABELS", []string{}),
			CustomCarriers: getEnvSliceOrDefault("GMAIL_CUSTOM_CARRIERS", []string{}),
		},
		
		Processing: ProcessingConfig{
			CheckInterval:       getEnvDurationOrDefault("EMAIL_CHECK_INTERVAL", "5m"),
			MaxEmailsPerRun:     getEnvIntOrDefault("EMAIL_MAX_PER_RUN", 50),
			DryRun:              getEnvBoolOrDefault("EMAIL_DRY_RUN", false),
			StateDBPath:         getEnvOrDefault("EMAIL_STATE_DB_PATH", "./email-state.db"),
			ProcessingTimeout:   getEnvDurationOrDefault("EMAIL_PROCESSING_TIMEOUT", "10m"),
			MinConfidence:       getEnvFloatOrDefault("EMAIL_MIN_CONFIDENCE", 0.5),
			MaxCandidates:       getEnvIntOrDefault("EMAIL_MAX_CANDIDATES", 10),
			UseHybridValidation: getEnvBoolOrDefault("EMAIL_USE_HYBRID_VALIDATION", true),
			DebugMode:           getEnvBoolOrDefault("EMAIL_DEBUG_MODE", false),
		},
		
		API: APIConfig{
			URL:           getEnvOrDefault("EMAIL_API_URL", "http://localhost:8080"),
			Timeout:       getEnvDurationOrDefault("EMAIL_API_TIMEOUT", "30s"),
			RetryCount:    getEnvIntOrDefault("EMAIL_API_RETRY_COUNT", 3),
			RetryDelay:    getEnvDurationOrDefault("EMAIL_API_RETRY_DELAY", "1s"),
			UserAgent:     getEnvOrDefault("EMAIL_API_USER_AGENT", "email-tracker/1.0"),
			BackoffFactor: getEnvFloatOrDefault("EMAIL_API_BACKOFF_FACTOR", 2.0),
		},
		
		LLM: LLMConfig{
			Provider:    getEnvOrDefault("LLM_PROVIDER", LLMProviderDisabled),
			Model:       getEnvOrDefault("LLM_MODEL", ""),
			APIKey:      getEnvOrDefault("LLM_API_KEY", ""),
			Endpoint:    getEnvOrDefault("LLM_ENDPOINT", ""),
			MaxTokens:   getEnvIntOrDefault("LLM_MAX_TOKENS", 1000),
			Temperature: getEnvFloatOrDefault("LLM_TEMPERATURE", 0.1),
			Timeout:     getEnvDurationOrDefault("LLM_TIMEOUT", "120s"),
			RetryCount:  getEnvIntOrDefault("LLM_RETRY_COUNT", 2),
			Enabled:     getEnvBoolOrDefault("LLM_ENABLED", false),
		},
	}
	
	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid email configuration: %w", err)
	}
	
	return config, nil
}

// validate checks if the configuration is valid
func (c *EmailConfig) validate() error {
	// Validate Gmail configuration
	if c.Gmail.ClientID == "" && c.Gmail.Username == "" {
		return fmt.Errorf("either Gmail OAuth2 (client_id) or IMAP (username) credentials must be provided")
	}
	
	if c.Gmail.ClientID != "" && c.Gmail.ClientSecret == "" {
		return fmt.Errorf("gmail_client_secret is required when using OAuth2")
	}
	
	if c.Gmail.Username != "" && c.Gmail.AppPassword == "" {
		return fmt.Errorf("gmail_app_password is required when using IMAP")
	}
	
	// Validate search configuration
	if c.Search.AfterDays < 0 {
		return fmt.Errorf("search after_days must be non-negative")
	}
	
	if c.Search.MaxResults < 1 || c.Search.MaxResults > 1000 {
		return fmt.Errorf("search max_results must be between 1 and 1000")
	}
	
	// Validate processing configuration
	if c.Processing.CheckInterval < time.Minute {
		return fmt.Errorf("check_interval must be at least 1 minute")
	}
	
	if c.Processing.MaxEmailsPerRun < 1 || c.Processing.MaxEmailsPerRun > 1000 {
		return fmt.Errorf("max_emails_per_run must be between 1 and 1000")
	}
	
	if c.Processing.StateDBPath == "" {
		return fmt.Errorf("state_db_path cannot be empty")
	}
	
	if c.Processing.MinConfidence < 0 || c.Processing.MinConfidence > 1.0 {
		return fmt.Errorf("min_confidence must be between 0.0 and 1.0")
	}
	
	// Validate API configuration
	if c.API.URL == "" {
		return fmt.Errorf("API URL cannot be empty")
	}
	
	if c.API.RetryCount < 0 || c.API.RetryCount > 10 {
		return fmt.Errorf("API retry_count must be between 0 and 10")
	}
	
	// Validate LLM configuration if enabled
	if c.LLM.Enabled {
		if c.LLM.Provider == "" || c.LLM.Provider == "disabled" {
			return fmt.Errorf("LLM provider must be specified when LLM is enabled")
		}
		
		validProviders := []string{LLMProviderOpenAI, LLMProviderAnthropic, LLMProviderLocal}
		isValid := false
		for _, provider := range validProviders {
			if c.LLM.Provider == provider {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid LLM provider: %s (must be one of: %v)", c.LLM.Provider, validProviders)
		}
		
		if c.LLM.Provider != LLMProviderLocal && c.LLM.APIKey == "" {
			return fmt.Errorf("LLM API key is required for provider: %s", c.LLM.Provider)
		}
		
		if c.LLM.Provider == LLMProviderLocal && c.LLM.Endpoint == "" {
			return fmt.Errorf("LLM endpoint is required for local provider")
		}
		
		if c.LLM.Temperature < 0 || c.LLM.Temperature > 1.0 {
			return fmt.Errorf("LLM temperature must be between 0.0 and 1.0")
		}
	}
	
	return nil
}

// SetDefaults sets default model names based on provider
func (c *EmailConfig) SetDefaults() {
	// Set default models if not specified
	if c.LLM.Enabled && c.LLM.Model == "" {
		switch c.LLM.Provider {
		case LLMProviderOpenAI:
			c.LLM.Model = "gpt-4"
		case LLMProviderAnthropic:
			c.LLM.Model = "claude-3-sonnet-20240229"
		case LLMProviderLocal:
			c.LLM.Model = "llama2"
		}
	}
	
	// Set default search query if not specified
	if c.Search.Query == "" {
		c.Search.Query = `from:(ups.com OR usps.com OR fedex.com OR dhl.com OR amazon.com OR shopify.com) subject:(tracking OR shipment OR package OR delivery)`
	}
}

// GetSearchQuery returns the configured search query or builds a default one
func (c *EmailConfig) GetSearchQuery() string {
	if c.Search.Query != "" {
		return c.Search.Query
	}
	
	// Build default query
	carriers := []string{"ups", "usps", "fedex", "dhl"}
	if len(c.Search.CustomCarriers) > 0 {
		carriers = c.Search.CustomCarriers
	}
	
	// Use helper function to build query
	return buildDefaultSearchQuery(carriers, c.Search.AfterDays, c.Search.UnreadOnly)
}

// buildDefaultSearchQuery constructs a Gmail search query
func buildDefaultSearchQuery(carriers []string, afterDays int, unreadOnly bool) string {
	query := `from:(ups.com OR usps.com OR fedex.com OR dhl.com OR amazon.com OR shopify.com) subject:(tracking OR shipment OR package OR delivery)`
	
	if afterDays > 0 {
		// Add date filter
		// Gmail date format: YYYY/MM/DD
		afterDate := time.Now().AddDate(0, 0, -afterDays).Format("2006/1/2")
		query += fmt.Sprintf(" after:%s", afterDate)
	}
	
	if unreadOnly {
		query += " is:unread"
	}
	
	return query
}

// IsOAuth2Configured returns true if OAuth2 is configured
func (c *EmailConfig) IsOAuth2Configured() bool {
	return c.Gmail.ClientID != "" && c.Gmail.ClientSecret != ""
}

// IsIMAPConfigured returns true if IMAP fallback is configured
func (c *EmailConfig) IsIMAPConfigured() bool {
	return c.Gmail.Username != "" && c.Gmail.AppPassword != ""
}

// IsLLMEnabled returns true if LLM integration is enabled and configured
func (c *EmailConfig) IsLLMEnabled() bool {
	return c.LLM.Enabled && c.LLM.Provider != LLMProviderDisabled
}

// Helper functions for environment variable parsing
// Note: getEnvInt64OrDefault and getEnvFloatOrDefault are now available in helpers.go

// getEnvSliceOrDefault returns environment variable as string slice or default
func getEnvSliceOrDefault(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		parts := []string{}
		for _, part := range splitAndTrim(value, ",") {
			if part != "" {
				parts = append(parts, part)
			}
		}
		if len(parts) > 0 {
			return parts
		}
	}
	return defaultValue
}

func splitAndTrim(s, sep string) []string {
	parts := []string{}
	for _, part := range strings.Split(s, sep) {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// ToJSON serializes the configuration to JSON (for debugging)
func (c *EmailConfig) ToJSON() (string, error) {
	// Create a copy with sensitive fields redacted
	safe := *c
	safe.Gmail.ClientSecret = redact(safe.Gmail.ClientSecret)
	safe.Gmail.RefreshToken = redact(safe.Gmail.RefreshToken)
	safe.Gmail.AccessToken = redact(safe.Gmail.AccessToken)
	safe.Gmail.AppPassword = redact(safe.Gmail.AppPassword)
	safe.LLM.APIKey = redact(safe.LLM.APIKey)
	
	data, err := json.MarshalIndent(safe, "", "  ")
	if err != nil {
		return "", err
	}
	
	return string(data), nil
}

func redact(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "***"
	}
	return value[:4] + "***" + value[len(value)-4:]
}