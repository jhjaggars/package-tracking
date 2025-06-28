package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds CLI configuration
type Config struct {
	ServerURL      string        `json:"server_url"`
	Format         string        `json:"format"`
	Quiet          bool          `json:"quiet"`
	RequestTimeout time.Duration `json:"request_timeout"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		ServerURL:      "http://localhost:8080",
		Format:         "table",
		Quiet:          false,
		RequestTimeout: 30 * time.Second,
	}
}

// LoadConfig loads configuration from file, environment variables, and CLI flags
func LoadConfig(serverFlag, formatFlag string, quietFlag bool) (*Config, error) {
	config := DefaultConfig()

	// Try to load from config file
	if err := config.loadFromFile(); err != nil {
		// Config file is optional, continue with defaults
	}

	// Override with environment variables
	config.loadFromEnv()

	// Override with CLI flags (highest priority)
	if serverFlag != "" {
		config.ServerURL = serverFlag
	}
	if formatFlag != "" {
		config.Format = formatFlag
	}
	if quietFlag {
		config.Quiet = quietFlag
	}

	return config, config.validate()
}

// loadFromFile loads configuration from ~/.package-tracker.json
func (c *Config) loadFromFile() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".package-tracker.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err // File doesn't exist or can't be read
	}

	return json.Unmarshal(data, c)
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() {
	if serverURL := os.Getenv("PACKAGE_TRACKER_SERVER"); serverURL != "" {
		c.ServerURL = serverURL
	}
	if format := os.Getenv("PACKAGE_TRACKER_FORMAT"); format != "" {
		c.Format = format
	}
	if os.Getenv("PACKAGE_TRACKER_QUIET") == "true" {
		c.Quiet = true
	}
	if timeoutStr := os.Getenv("PACKAGE_TRACKER_TIMEOUT"); timeoutStr != "" {
		if timeoutSec, err := strconv.Atoi(timeoutStr); err == nil && timeoutSec > 0 {
			c.RequestTimeout = time.Duration(timeoutSec) * time.Second
		}
	}
}

// validate checks if the configuration is valid
func (c *Config) validate() error {
	if strings.TrimSpace(c.ServerURL) == "" {
		return fmt.Errorf("server URL cannot be empty")
	}

	// Validate URL format and warn about non-HTTPS URLs
	if parsedURL, err := url.Parse(c.ServerURL); err != nil {
		return fmt.Errorf("invalid server URL format: %v", err)
	} else if parsedURL.Scheme == "http" && parsedURL.Host != "localhost" && !strings.HasPrefix(parsedURL.Host, "127.0.0.1") {
		// Only warn for non-localhost HTTP URLs (production concern)
		fmt.Fprintf(os.Stderr, "Warning: Using HTTP instead of HTTPS for server URL. This is not recommended for production use.\n")
	}

	validFormats := []string{"table", "json"}
	isValidFormat := false
	for _, format := range validFormats {
		if c.Format == format {
			isValidFormat = true
			break
		}
	}
	if !isValidFormat {
		return fmt.Errorf("invalid format: %s (must be one of: table, json)", c.Format)
	}

	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}

	return nil
}

// SaveConfig saves the current configuration to ~/.package-tracker.json
func (c *Config) SaveConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".package-tracker.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}