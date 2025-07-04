package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// LoadEmailConfigWithViper loads email configuration using Viper
func LoadEmailConfigWithViper(v *viper.Viper) (*EmailConfig, error) {
	// Set defaults
	setEmailDefaults(v)

	// Set up environment variable binding
	setupEmailEnvBinding(v)

	// Load configuration file if specified
	if err := loadEmailConfigFile(v); err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	// Unmarshal configuration
	config := &EmailConfig{}
	if err := unmarshalEmailConfig(v, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults for model names based on provider
	config.SetDefaults()

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// setEmailDefaults sets default values for email configuration
func setEmailDefaults(v *viper.Viper) {
	// Gmail defaults
	v.SetDefault("gmail.token_file", "./gmail-token.json")
	v.SetDefault("gmail.max_results", 100)
	v.SetDefault("gmail.request_timeout", "30s")
	v.SetDefault("gmail.rate_limit_delay", "100ms")

	// Search defaults
	v.SetDefault("search.query", "")
	v.SetDefault("search.after_days", 30)
	v.SetDefault("search.unread_only", false)
	v.SetDefault("search.max_results", 100)

	// Processing defaults
	v.SetDefault("processing.check_interval", "5m")
	v.SetDefault("processing.max_emails_per_run", 50)
	v.SetDefault("processing.dry_run", false)
	v.SetDefault("processing.state_db_path", "./email-state.db")
	v.SetDefault("processing.processing_timeout", "10m")
	v.SetDefault("processing.min_confidence", 0.5)
	v.SetDefault("processing.max_candidates", 10)
	v.SetDefault("processing.use_hybrid_validation", true)
	v.SetDefault("processing.debug_mode", false)

	// API defaults
	v.SetDefault("api.url", "http://localhost:8080")
	v.SetDefault("api.timeout", "30s")
	v.SetDefault("api.retry_count", 3)
	v.SetDefault("api.retry_delay", "1s")
	v.SetDefault("api.user_agent", "email-tracker/1.0")
	v.SetDefault("api.backoff_factor", 2.0)

	// LLM defaults
	v.SetDefault("llm.provider", LLMProviderDisabled)
	v.SetDefault("llm.model", "")
	v.SetDefault("llm.api_key", "")
	v.SetDefault("llm.endpoint", "")
	v.SetDefault("llm.max_tokens", 1000)
	v.SetDefault("llm.temperature", 0.1)
	v.SetDefault("llm.timeout", "120s")
	v.SetDefault("llm.retry_count", 2)
	v.SetDefault("llm.enabled", false)
}

// setupEmailEnvBinding sets up environment variable binding for email configuration
func setupEmailEnvBinding(v *viper.Viper) {
	// Set environment variable prefix
	v.SetEnvPrefix("PKG_TRACKER")
	v.AutomaticEnv()

	// Bind new format environment variables
	envBindings := map[string]string{
		// Gmail
		"gmail.client_id":       "EMAIL_GMAIL_CLIENT_ID",
		"gmail.client_secret":   "EMAIL_GMAIL_CLIENT_SECRET",
		"gmail.refresh_token":   "EMAIL_GMAIL_REFRESH_TOKEN",
		"gmail.access_token":    "EMAIL_GMAIL_ACCESS_TOKEN",
		"gmail.token_file":      "EMAIL_GMAIL_TOKEN_FILE",
		"gmail.username":        "EMAIL_GMAIL_USERNAME",
		"gmail.app_password":    "EMAIL_GMAIL_APP_PASSWORD",
		"gmail.max_results":     "EMAIL_GMAIL_MAX_RESULTS",
		"gmail.request_timeout": "EMAIL_GMAIL_REQUEST_TIMEOUT",
		"gmail.rate_limit_delay": "EMAIL_GMAIL_RATE_LIMIT_DELAY",
		
		// Search
		"search.query":           "EMAIL_SEARCH_QUERY",
		"search.after_days":      "EMAIL_SEARCH_AFTER_DAYS",
		"search.unread_only":     "EMAIL_SEARCH_UNREAD_ONLY",
		"search.max_results":     "EMAIL_SEARCH_MAX_RESULTS",
		"search.include_labels":  "EMAIL_SEARCH_INCLUDE_LABELS",
		"search.exclude_labels":  "EMAIL_SEARCH_EXCLUDE_LABELS",
		"search.custom_carriers": "EMAIL_SEARCH_CUSTOM_CARRIERS",
		
		// Processing
		"processing.check_interval":       "EMAIL_PROCESSING_CHECK_INTERVAL",
		"processing.max_emails_per_run":   "EMAIL_PROCESSING_MAX_EMAILS_PER_RUN",
		"processing.dry_run":              "EMAIL_PROCESSING_DRY_RUN",
		"processing.state_db_path":        "EMAIL_PROCESSING_STATE_DB_PATH",
		"processing.processing_timeout":   "EMAIL_PROCESSING_PROCESSING_TIMEOUT",
		"processing.min_confidence":       "EMAIL_PROCESSING_MIN_CONFIDENCE",
		"processing.max_candidates":       "EMAIL_PROCESSING_MAX_CANDIDATES",
		"processing.use_hybrid_validation": "EMAIL_PROCESSING_USE_HYBRID_VALIDATION",
		"processing.debug_mode":           "EMAIL_PROCESSING_DEBUG_MODE",
		
		// API
		"api.url":            "EMAIL_API_URL",
		"api.timeout":        "EMAIL_API_TIMEOUT",
		"api.retry_count":    "EMAIL_API_RETRY_COUNT",
		"api.retry_delay":    "EMAIL_API_RETRY_DELAY",
		"api.user_agent":     "EMAIL_API_USER_AGENT",
		"api.backoff_factor": "EMAIL_API_BACKOFF_FACTOR",
		
		// LLM
		"llm.provider":    "EMAIL_LLM_PROVIDER",
		"llm.model":       "EMAIL_LLM_MODEL",
		"llm.api_key":     "EMAIL_LLM_API_KEY",
		"llm.endpoint":    "EMAIL_LLM_ENDPOINT",
		"llm.max_tokens":  "EMAIL_LLM_MAX_TOKENS",
		"llm.temperature": "EMAIL_LLM_TEMPERATURE",
		"llm.timeout":     "EMAIL_LLM_TIMEOUT",
		"llm.retry_count": "EMAIL_LLM_RETRY_COUNT",
		"llm.enabled":     "EMAIL_LLM_ENABLED",
	}

	for configKey, envSuffix := range envBindings {
		v.BindEnv(configKey, "PKG_TRACKER_"+envSuffix)
	}

	// Bind old format environment variables for backward compatibility
	oldEnvBindings := map[string]string{
		// Gmail
		"gmail.client_id":       "GMAIL_CLIENT_ID",
		"gmail.client_secret":   "GMAIL_CLIENT_SECRET",
		"gmail.refresh_token":   "GMAIL_REFRESH_TOKEN",
		"gmail.access_token":    "GMAIL_ACCESS_TOKEN",
		"gmail.token_file":      "GMAIL_TOKEN_FILE",
		"gmail.username":        "GMAIL_USERNAME",
		"gmail.app_password":    "GMAIL_APP_PASSWORD",
		"gmail.max_results":     "GMAIL_MAX_RESULTS",
		"gmail.request_timeout": "GMAIL_REQUEST_TIMEOUT",
		"gmail.rate_limit_delay": "GMAIL_RATE_LIMIT_DELAY",
		
		// Search
		"search.query":           "GMAIL_SEARCH_QUERY",
		"search.after_days":      "GMAIL_SEARCH_AFTER_DAYS",
		"search.unread_only":     "GMAIL_SEARCH_UNREAD_ONLY",
		"search.max_results":     "GMAIL_SEARCH_MAX_RESULTS",
		"search.include_labels":  "GMAIL_INCLUDE_LABELS",
		"search.exclude_labels":  "GMAIL_EXCLUDE_LABELS",
		"search.custom_carriers": "GMAIL_CUSTOM_CARRIERS",
		
		// Processing
		"processing.check_interval":       "EMAIL_CHECK_INTERVAL",
		"processing.max_emails_per_run":   "EMAIL_MAX_PER_RUN",
		"processing.dry_run":              "EMAIL_DRY_RUN",
		"processing.state_db_path":        "EMAIL_STATE_DB_PATH",
		"processing.processing_timeout":   "EMAIL_PROCESSING_TIMEOUT",
		"processing.min_confidence":       "EMAIL_MIN_CONFIDENCE",
		"processing.max_candidates":       "EMAIL_MAX_CANDIDATES",
		"processing.use_hybrid_validation": "EMAIL_USE_HYBRID_VALIDATION",
		"processing.debug_mode":           "EMAIL_DEBUG_MODE",
		
		// API
		"api.url":            "EMAIL_API_URL",
		"api.timeout":        "EMAIL_API_TIMEOUT",
		"api.retry_count":    "EMAIL_API_RETRY_COUNT",
		"api.retry_delay":    "EMAIL_API_RETRY_DELAY",
		"api.user_agent":     "EMAIL_API_USER_AGENT",
		"api.backoff_factor": "EMAIL_API_BACKOFF_FACTOR",
		
		// LLM
		"llm.provider":    "LLM_PROVIDER",
		"llm.model":       "LLM_MODEL",
		"llm.api_key":     "LLM_API_KEY",
		"llm.endpoint":    "LLM_ENDPOINT",
		"llm.max_tokens":  "LLM_MAX_TOKENS",
		"llm.temperature": "LLM_TEMPERATURE",
		"llm.timeout":     "LLM_TIMEOUT",
		"llm.retry_count": "LLM_RETRY_COUNT",
		"llm.enabled":     "LLM_ENABLED",
	}

	for configKey, envVar := range oldEnvBindings {
		v.BindEnv(configKey, envVar)
	}
}

// loadEmailConfigFile loads configuration file if it exists
func loadEmailConfigFile(v *viper.Viper) error {
	// Check if a specific config file was set
	if v.ConfigFileUsed() == "" {
		// Add configuration search paths
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("$HOME/.email-tracker")

		// Set configuration file name (without extension)
		v.SetConfigName("email-tracker")
	}

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		// Config file is optional, only return error if it's not a "not found" error
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return err
		}
	}

	return nil
}

// unmarshalEmailConfig unmarshals Viper configuration into EmailConfig struct
func unmarshalEmailConfig(v *viper.Viper, config *EmailConfig) error {
	// Gmail configuration
	config.Gmail.ClientID = v.GetString("gmail.client_id")
	config.Gmail.ClientSecret = v.GetString("gmail.client_secret")
	config.Gmail.RefreshToken = v.GetString("gmail.refresh_token")
	config.Gmail.AccessToken = v.GetString("gmail.access_token")
	config.Gmail.TokenFile = v.GetString("gmail.token_file")
	config.Gmail.Username = v.GetString("gmail.username")
	config.Gmail.AppPassword = v.GetString("gmail.app_password")
	config.Gmail.MaxResults = v.GetInt64("gmail.max_results")

	// Parse Gmail durations
	var err error
	config.Gmail.RequestTimeout, err = time.ParseDuration(v.GetString("gmail.request_timeout"))
	if err != nil {
		return fmt.Errorf("invalid gmail request timeout: %w", err)
	}

	config.Gmail.RateLimitDelay, err = time.ParseDuration(v.GetString("gmail.rate_limit_delay"))
	if err != nil {
		return fmt.Errorf("invalid gmail rate limit delay: %w", err)
	}

	// Search configuration
	config.Search.Query = v.GetString("search.query")
	config.Search.AfterDays = v.GetInt("search.after_days")
	config.Search.UnreadOnly = v.GetBool("search.unread_only")
	config.Search.MaxResults = v.GetInt("search.max_results")
	config.Search.IncludeLabels = parseStringSlice(v.GetString("search.include_labels"))
	config.Search.ExcludeLabels = parseStringSlice(v.GetString("search.exclude_labels"))
	config.Search.CustomCarriers = parseStringSlice(v.GetString("search.custom_carriers"))

	// Processing configuration
	config.Processing.CheckInterval, err = time.ParseDuration(v.GetString("processing.check_interval"))
	if err != nil {
		return fmt.Errorf("invalid processing check interval: %w", err)
	}

	config.Processing.MaxEmailsPerRun = v.GetInt("processing.max_emails_per_run")
	config.Processing.DryRun = v.GetBool("processing.dry_run")
	config.Processing.StateDBPath = v.GetString("processing.state_db_path")

	config.Processing.ProcessingTimeout, err = time.ParseDuration(v.GetString("processing.processing_timeout"))
	if err != nil {
		return fmt.Errorf("invalid processing timeout: %w", err)
	}

	config.Processing.MinConfidence = v.GetFloat64("processing.min_confidence")
	config.Processing.MaxCandidates = v.GetInt("processing.max_candidates")
	config.Processing.UseHybridValidation = v.GetBool("processing.use_hybrid_validation")
	config.Processing.DebugMode = v.GetBool("processing.debug_mode")

	// API configuration
	config.API.URL = v.GetString("api.url")
	config.API.Timeout, err = time.ParseDuration(v.GetString("api.timeout"))
	if err != nil {
		return fmt.Errorf("invalid API timeout: %w", err)
	}

	config.API.RetryCount = v.GetInt("api.retry_count")
	config.API.RetryDelay, err = time.ParseDuration(v.GetString("api.retry_delay"))
	if err != nil {
		return fmt.Errorf("invalid API retry delay: %w", err)
	}

	config.API.UserAgent = v.GetString("api.user_agent")
	config.API.BackoffFactor = v.GetFloat64("api.backoff_factor")

	// LLM configuration
	config.LLM.Provider = v.GetString("llm.provider")
	config.LLM.Model = v.GetString("llm.model")
	config.LLM.APIKey = v.GetString("llm.api_key")
	config.LLM.Endpoint = v.GetString("llm.endpoint")
	config.LLM.MaxTokens = v.GetInt("llm.max_tokens")
	config.LLM.Temperature = v.GetFloat64("llm.temperature")

	config.LLM.Timeout, err = time.ParseDuration(v.GetString("llm.timeout"))
	if err != nil {
		return fmt.Errorf("invalid LLM timeout: %w", err)
	}

	config.LLM.RetryCount = v.GetInt("llm.retry_count")
	config.LLM.Enabled = v.GetBool("llm.enabled")

	return nil
}

// parseStringSlice parses comma-separated string into slice
func parseStringSlice(s string) []string {
	if s == "" {
		return []string{}
	}
	
	parts := []string{}
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// LoadEmailConfigViper loads email configuration using default Viper instance
func LoadEmailConfigViper() (*EmailConfig, error) {
	v := viper.New()
	return LoadEmailConfigWithViper(v)
}

// LoadEmailConfigViperWithFile loads email configuration from a specific file
func LoadEmailConfigViperWithFile(configFile string) (*EmailConfig, error) {
	v := viper.New()
	v.SetConfigFile(configFile)
	return LoadEmailConfigWithViper(v)
}

// LoadEmailConfigViperWithEnvFile loads email configuration with .env file support
func LoadEmailConfigViperWithEnvFile(envFile string) (*EmailConfig, error) {
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

	// Load configuration with Viper
	v := viper.New()
	return LoadEmailConfigWithViper(v)
}