package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	UPSAPIKey      string
	FedExAPIKey    string
	FedExSecretKey string
	FedExAPIURL    string
	DHLAPIKey      string

	// Logging
	LogLevel string

	// Development/testing flags
	DisableRateLimit bool
	DisableCache     bool

	// Auto-update configuration
	AutoUpdateEnabled    bool
	AutoUpdateCutoffDays int
	AutoUpdateBatchSize  int
	AutoUpdateMaxRetries int

	// Timeout configuration
	AutoUpdateBatchTimeout      time.Duration
	AutoUpdateIndividualTimeout time.Duration
	AutoUpdateRateLimit         time.Duration
}

// Load loads configuration from environment variables with defaults
// If a .env file exists, it will be loaded first
func Load() (*Config, error) {
	// Try to load .env file if it exists
	loadEnvFile(".env")
	config := &Config{
		// Server defaults
		ServerPort: getEnvOrDefault("SERVER_PORT", "8080"),
		ServerHost: getEnvOrDefault("SERVER_HOST", "localhost"),

		// Database defaults
		DBPath: getEnvOrDefault("DB_PATH", "./database.db"),

		// Update interval default
		UpdateInterval: getEnvDurationOrDefault("UPDATE_INTERVAL", "1h"),

		// API keys (optional)
		USPSAPIKey:     os.Getenv("USPS_API_KEY"),
		UPSAPIKey:      os.Getenv("UPS_API_KEY"),
		FedExAPIKey:    os.Getenv("FEDEX_API_KEY"),
		FedExSecretKey: os.Getenv("FEDEX_SECRET_KEY"),
		FedExAPIURL:    getEnvOrDefault("FEDEX_API_URL", "https://apis.fedex.com"),
		DHLAPIKey:      os.Getenv("DHL_API_KEY"),

		// Logging
		LogLevel: getEnvOrDefault("LOG_LEVEL", "info"),

		// Development/testing flags
		DisableRateLimit: getEnvBoolOrDefault("DISABLE_RATE_LIMIT", false),
		DisableCache:     getEnvBoolOrDefault("DISABLE_CACHE", false),

		// Auto-update configuration
		AutoUpdateEnabled:    getEnvBoolOrDefault("AUTO_UPDATE_ENABLED", true),
		AutoUpdateCutoffDays: getEnvIntOrDefault("AUTO_UPDATE_CUTOFF_DAYS", 30),
		AutoUpdateBatchSize:  getEnvIntOrDefault("AUTO_UPDATE_BATCH_SIZE", 10),
		AutoUpdateMaxRetries: getEnvIntOrDefault("AUTO_UPDATE_MAX_RETRIES", 10),

		// Timeout configuration
		AutoUpdateBatchTimeout:      getEnvDurationOrDefault("AUTO_UPDATE_BATCH_TIMEOUT", "60s"),
		AutoUpdateIndividualTimeout: getEnvDurationOrDefault("AUTO_UPDATE_INDIVIDUAL_TIMEOUT", "30s"),
		AutoUpdateRateLimit:         getEnvDurationOrDefault("AUTO_UPDATE_RATE_LIMIT", "5m"),
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

	// Validate timeout configuration
	if c.AutoUpdateBatchTimeout <= 0 {
		return fmt.Errorf("auto update batch timeout must be positive")
	}
	if c.AutoUpdateIndividualTimeout <= 0 {
		return fmt.Errorf("auto update individual timeout must be positive")
	}
	if c.AutoUpdateRateLimit <= 0 {
		return fmt.Errorf("auto update rate limit must be positive")
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

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvDurationOrDefault returns environment variable as duration or default
func getEnvDurationOrDefault(key string, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	
	// Parse default value
	duration, err := time.ParseDuration(defaultValue)
	if err != nil {
		return time.Hour // Fallback to 1 hour
	}
	return duration
}

// getEnvBoolOrDefault returns environment variable as boolean or default
func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getEnvIntOrDefault returns environment variable as integer or default
func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// loadEnvFile loads environment variables from a .env file if it exists
func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		// .env file doesn't exist or can't be opened, which is fine
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Split on first '=' character
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		   (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}
		
		// Only set if not already set in environment
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}