package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	ServerPort string
	ServerHost string

	// Database configuration
	DBPath string

	// Update intervals
	UpdateInterval time.Duration

	// Carrier API keys
	USPSAPIKey     string
	UPSAPIKey      string // Deprecated: Use UPSClientID and UPSClientSecret instead
	UPSClientID    string
	UPSClientSecret string
	FedExAPIKey    string
	FedExSecretKey string
	FedExAPIURL    string
	DHLAPIKey      string

	// Logging
	LogLevel string

	// Development/testing flags
	DisableRateLimit bool
	DisableCache     bool

	// Admin authentication
	DisableAdminAuth bool
	AdminAPIKey      string

	// Auto-update configuration
	AutoUpdateEnabled           bool
	AutoUpdateCutoffDays        int
	AutoUpdateBatchSize         int
	AutoUpdateMaxRetries        int
	AutoUpdateFailureThreshold  int
	
	// Per-carrier auto-update configuration
	UPSAutoUpdateEnabled        bool
	UPSAutoUpdateCutoffDays     int
	DHLAutoUpdateEnabled        bool
	DHLAutoUpdateCutoffDays     int

	// Cache configuration
	CacheTTL                    time.Duration

	// Timeout configuration
	AutoUpdateBatchTimeout      time.Duration
	AutoUpdateIndividualTimeout time.Duration
}

// Load loads configuration from environment variables with defaults
// If a .env file exists, it will be loaded first
func Load() (*Config, error) {
	// Try to load .env file if it exists
	if err := LoadEnvFile(".env"); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}
	config := &Config{
		// Server defaults
		ServerPort: getEnvOrDefault("SERVER_PORT", "8080"),
		ServerHost: getEnvOrDefault("SERVER_HOST", "localhost"),

		// Database defaults
		DBPath: getEnvOrDefault("DB_PATH", "./database.db"),

		// Update interval default
		UpdateInterval: getEnvDurationOrDefault("UPDATE_INTERVAL", "1h"),

		// API keys (optional)
		USPSAPIKey:      os.Getenv("USPS_API_KEY"),
		UPSAPIKey:       os.Getenv("UPS_API_KEY"),
		UPSClientID:     os.Getenv("UPS_CLIENT_ID"),
		UPSClientSecret: os.Getenv("UPS_CLIENT_SECRET"),
		FedExAPIKey:     os.Getenv("FEDEX_API_KEY"),
		FedExSecretKey:  os.Getenv("FEDEX_SECRET_KEY"),
		FedExAPIURL:     getEnvOrDefault("FEDEX_API_URL", "https://apis.fedex.com"),
		DHLAPIKey:       os.Getenv("DHL_API_KEY"),

		// Logging
		LogLevel: getEnvOrDefault("LOG_LEVEL", "info"),

		// Development/testing flags
		DisableRateLimit: getEnvBoolOrDefault("DISABLE_RATE_LIMIT", false),
		DisableCache:     getEnvBoolOrDefault("DISABLE_CACHE", false),

		// Admin authentication
		DisableAdminAuth: getEnvBoolOrDefault("DISABLE_ADMIN_AUTH", false),
		AdminAPIKey:      os.Getenv("ADMIN_API_KEY"),

		// Auto-update configuration
		AutoUpdateEnabled:          getEnvBoolOrDefault("AUTO_UPDATE_ENABLED", true),
		AutoUpdateCutoffDays:       getEnvIntOrDefault("AUTO_UPDATE_CUTOFF_DAYS", 30),
		AutoUpdateBatchSize:        getEnvIntOrDefault("AUTO_UPDATE_BATCH_SIZE", 10),
		AutoUpdateMaxRetries:       getEnvIntOrDefault("AUTO_UPDATE_MAX_RETRIES", 10),
		AutoUpdateFailureThreshold: getEnvIntOrDefault("AUTO_UPDATE_FAILURE_THRESHOLD", 10),
		
		// Per-carrier auto-update configuration
		UPSAutoUpdateEnabled:    getEnvBoolOrDefault("UPS_AUTO_UPDATE_ENABLED", true),
		UPSAutoUpdateCutoffDays: getEnvIntOrDefault("UPS_AUTO_UPDATE_CUTOFF_DAYS", 30),
		DHLAutoUpdateEnabled:    getEnvBoolOrDefault("DHL_AUTO_UPDATE_ENABLED", true),
		DHLAutoUpdateCutoffDays: getEnvIntOrDefault("DHL_AUTO_UPDATE_CUTOFF_DAYS", 0),

		// Cache configuration
		CacheTTL:                    getEnvDurationOrDefault("CACHE_TTL", "5m"),

		// Timeout configuration
		AutoUpdateBatchTimeout:      getEnvDurationOrDefault("AUTO_UPDATE_BATCH_TIMEOUT", "60s"),
		AutoUpdateIndividualTimeout: getEnvDurationOrDefault("AUTO_UPDATE_INDIVIDUAL_TIMEOUT", "30s"),
	}

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// validate checks if the configuration is valid
func (c *Config) validate() error {
	// Validate server port
	if c.ServerPort == "" {
		return fmt.Errorf("server port cannot be empty")
	}

	// Check if port is a valid number
	if _, err := strconv.Atoi(c.ServerPort); err != nil {
		return fmt.Errorf("invalid server port: %s", c.ServerPort)
	}

	// Validate database path
	if c.DBPath == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	// Validate update interval
	if c.UpdateInterval <= 0 {
		return fmt.Errorf("update interval must be positive")
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	isValidLogLevel := false
	for _, level := range validLogLevels {
		if c.LogLevel == level {
			isValidLogLevel = true
			break
		}
	}
	if !isValidLogLevel {
		return fmt.Errorf("invalid log level: %s (must be one of: debug, info, warn, error)", c.LogLevel)
	}

	// Validate auto-update configuration
	if c.AutoUpdateCutoffDays < 0 {
		return fmt.Errorf("auto update cutoff days must be non-negative")
	}
	if c.AutoUpdateBatchSize < 1 || c.AutoUpdateBatchSize > 10 {
		return fmt.Errorf("auto update batch size must be between 1 and 10")
	}
	if c.AutoUpdateMaxRetries < 0 {
		return fmt.Errorf("auto update max retries must be non-negative")
	}
	if c.AutoUpdateFailureThreshold < 0 {
		return fmt.Errorf("auto update failure threshold must be non-negative")
	}
	if c.UPSAutoUpdateCutoffDays < 0 {
		return fmt.Errorf("UPS auto update cutoff days must be non-negative")
	}
	if c.DHLAutoUpdateCutoffDays < 0 {
		return fmt.Errorf("DHL auto update cutoff days must be non-negative")
	}
	if c.CacheTTL <= 0 {
		return fmt.Errorf("cache TTL must be positive")
	}

	// Validate timeout configuration
	if c.AutoUpdateBatchTimeout <= 0 {
		return fmt.Errorf("auto update batch timeout must be positive")
	}
	if c.AutoUpdateIndividualTimeout <= 0 {
		return fmt.Errorf("auto update individual timeout must be positive")
	}

	// Validate admin authentication
	if !c.DisableAdminAuth && c.AdminAPIKey == "" {
		return fmt.Errorf("ADMIN_API_KEY is required when admin authentication is enabled (set DISABLE_ADMIN_AUTH=true to disable)")
	}

	return nil
}

// Address returns the full server address
func (c *Config) Address() string {
	return c.ServerHost + ":" + c.ServerPort
}

// GetFedExAPIKey returns the FedEx API key
func (c *Config) GetFedExAPIKey() string {
	return c.FedExAPIKey
}

// GetFedExSecretKey returns the FedEx secret key  
func (c *Config) GetFedExSecretKey() string {
	return c.FedExSecretKey
}

// GetFedExAPIURL returns the FedEx API URL
func (c *Config) GetFedExAPIURL() string {
	return c.FedExAPIURL
}

// GetDisableRateLimit returns the rate limit disable flag
func (c *Config) GetDisableRateLimit() bool {
	return c.DisableRateLimit
}

// GetDisableCache returns the cache disable flag
func (c *Config) GetDisableCache() bool {
	return c.DisableCache
}

// GetUPSClientID returns the UPS OAuth client ID
func (c *Config) GetUPSClientID() string {
	return c.UPSClientID
}

// GetUPSClientSecret returns the UPS OAuth client secret
func (c *Config) GetUPSClientSecret() string {
	return c.UPSClientSecret
}

// GetCacheTTL returns the cache TTL duration
func (c *Config) GetCacheTTL() time.Duration {
	return c.CacheTTL
}

// GetDisableAdminAuth returns the admin authentication disable flag
func (c *Config) GetDisableAdminAuth() bool {
	return c.DisableAdminAuth
}

// GetAdminAPIKey returns the admin API key (redacted for logging)
func (c *Config) GetAdminAPIKey() string {
	return c.AdminAPIKey
}

// GetAdminAPIKeyForLogging returns a redacted version of the admin API key for safe logging
func (c *Config) GetAdminAPIKeyForLogging() string {
	if c.AdminAPIKey == "" {
		return "(not set)"
	}
	if len(c.AdminAPIKey) <= 8 {
		return "***"
	}
	return c.AdminAPIKey[:4] + "***" + c.AdminAPIKey[len(c.AdminAPIKey)-4:]
}

