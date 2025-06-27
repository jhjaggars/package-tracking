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
	USPSAPIKey  string
	UPSAPIKey   string
	FedExAPIKey string
	DHLAPIKey   string

	// Logging
	LogLevel string
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	config := &Config{
		// Server defaults
		ServerPort: getEnvOrDefault("SERVER_PORT", "8080"),
		ServerHost: getEnvOrDefault("SERVER_HOST", "localhost"),

		// Database defaults
		DBPath: getEnvOrDefault("DB_PATH", "./database.db"),

		// Update interval default
		UpdateInterval: getEnvDurationOrDefault("UPDATE_INTERVAL", "1h"),

		// API keys (optional)
		USPSAPIKey:  os.Getenv("USPS_API_KEY"),
		UPSAPIKey:   os.Getenv("UPS_API_KEY"),
		FedExAPIKey: os.Getenv("FEDEX_API_KEY"),
		DHLAPIKey:   os.Getenv("DHL_API_KEY"),

		// Logging
		LogLevel: getEnvOrDefault("LOG_LEVEL", "info"),
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

	return nil
}

// Address returns the full server address
func (c *Config) Address() string {
	return c.ServerHost + ":" + c.ServerPort
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