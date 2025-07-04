package config

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/viper"
	
	"package-tracking/internal/cli"
)

// LoadCLIConfigWithViper loads CLI configuration using Viper
func LoadCLIConfigWithViper(v *viper.Viper) (*cli.Config, error) {
	// Set defaults
	setCLIDefaults(v)

	// Set up environment variable binding
	setupCLIEnvBinding(v)

	// Load configuration file if specified
	if err := loadCLIConfigFile(v); err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	// Unmarshal configuration
	config := &cli.Config{}
	if err := unmarshalCLIConfig(v, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateCLIConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// setCLIDefaults sets default values for CLI configuration
func setCLIDefaults(v *viper.Viper) {
	v.SetDefault("server_url", "http://localhost:8080")
	v.SetDefault("format", "table")
	v.SetDefault("quiet", false)
	v.SetDefault("no_color", false)
	v.SetDefault("request_timeout", "180s") // 3 minutes for SPA scraping
}

// setupCLIEnvBinding sets up environment variable binding for CLI configuration
func setupCLIEnvBinding(v *viper.Viper) {
	// Set environment variable prefix
	v.SetEnvPrefix("PKG_TRACKER")
	v.AutomaticEnv()

	// Bind new format environment variables
	envBindings := map[string]string{
		"server_url":      "CLI_SERVER_URL",
		"format":          "CLI_FORMAT",
		"quiet":           "CLI_QUIET",
		"no_color":        "CLI_NO_COLOR",
		"request_timeout": "CLI_TIMEOUT",
	}

	for configKey, envSuffix := range envBindings {
		v.BindEnv(configKey, "PKG_TRACKER_"+envSuffix)
	}

	// Bind old format environment variables for backward compatibility
	oldEnvBindings := map[string]string{
		"server_url":      "PACKAGE_TRACKER_SERVER",
		"format":          "PACKAGE_TRACKER_FORMAT", 
		"quiet":           "PACKAGE_TRACKER_QUIET",
		"no_color":        "PACKAGE_TRACKER_NO_COLOR",
		"request_timeout": "PACKAGE_TRACKER_TIMEOUT",
	}

	for configKey, envVar := range oldEnvBindings {
		v.BindEnv(configKey, envVar)
	}

	// Special handling for NO_COLOR environment variable
	v.BindEnv("no_color", "NO_COLOR")
}

// loadCLIConfigFile loads configuration file if it exists
func loadCLIConfigFile(v *viper.Viper) error {
	// Check if a specific config file was set
	if v.ConfigFileUsed() == "" {
		// Add configuration search paths
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("$HOME")

		// Set configuration file name (without extension)
		v.SetConfigName("cli")
	}

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		// Config file is optional, only return error if it's not a "not found" error
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	return nil
}

// unmarshalCLIConfig unmarshals Viper configuration into CLI Config struct
func unmarshalCLIConfig(v *viper.Viper, config *cli.Config) error {
	config.ServerURL = v.GetString("server_url")
	config.Format = v.GetString("format")
	config.Quiet = v.GetBool("quiet")
	config.NoColor = v.GetBool("no_color")

	// Parse timeout from string or int
	timeoutStr := v.GetString("request_timeout")
	if timeoutStr != "" {
		// Try parsing as duration first
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			config.RequestTimeout = duration
		} else {
			// Try parsing as seconds (int)
			if seconds, err := strconv.Atoi(timeoutStr); err == nil {
				if seconds <= 0 {
					return fmt.Errorf("request timeout must be positive, got %d seconds", seconds)
				}
				config.RequestTimeout = time.Duration(seconds) * time.Second
			} else {
				return fmt.Errorf("invalid request timeout: %s", timeoutStr)
			}
		}
	} else {
		// Use default
		config.RequestTimeout = 180 * time.Second
	}

	return nil
}

// validateCLIConfig validates CLI configuration
func validateCLIConfig(config *cli.Config) error {
	// Use the existing validation logic from the CLI config
	// But we need to implement it here since we can't access the private validate method
	
	if config.ServerURL == "" {
		return fmt.Errorf("server URL cannot be empty")
	}

	// Validate URL format - basic check for URL structure
	if config.ServerURL != "" {
		if len(config.ServerURL) < 7 || (!hasPrefix(config.ServerURL, "http://") && !hasPrefix(config.ServerURL, "https://")) {
			return fmt.Errorf("invalid server URL format")
		}
	}

	// Validate format
	validFormats := []string{"table", "json"}
	isValidFormat := false
	for _, format := range validFormats {
		if config.Format == format {
			isValidFormat = true
			break
		}
	}
	if !isValidFormat {
		return fmt.Errorf("invalid format: %s (must be one of: table, json)", config.Format)
	}

	// Validate timeout
	if config.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}

	return nil
}

// LoadCLIConfig loads CLI configuration using default Viper instance
func LoadCLIConfig() (*cli.Config, error) {
	v := viper.New()
	return LoadCLIConfigWithViper(v)
}

// LoadCLIConfigWithFile loads CLI configuration from a specific file
func LoadCLIConfigWithFile(configFile string) (*cli.Config, error) {
	v := viper.New()
	v.SetConfigFile(configFile)
	return LoadCLIConfigWithViper(v)
}

// hasPrefix checks if string s has the given prefix
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}