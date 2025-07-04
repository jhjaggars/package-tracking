package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// LoadServerConfigWithViper loads server configuration using Viper
func LoadServerConfigWithViper(v *viper.Viper) (*Config, error) {
	// Set defaults
	setServerDefaults(v)

	// Set up environment variable binding
	setupServerEnvBinding(v)

	// Load configuration file if specified
	if err := loadConfigFile(v); err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	// Unmarshal configuration
	config := &Config{}
	if err := unmarshalServerConfig(v, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// setServerDefaults sets default values for server configuration
func setServerDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.host", "localhost")

	// Database defaults
	v.SetDefault("database.path", "./database.db")

	// Logging defaults
	v.SetDefault("logging.level", "info")

	// Update defaults
	v.SetDefault("update.interval", "1h")
	v.SetDefault("update.auto_enabled", true)
	v.SetDefault("update.cutoff_days", 30)
	v.SetDefault("update.batch_size", 10)
	v.SetDefault("update.max_retries", 10)
	v.SetDefault("update.failure_threshold", 10)
	v.SetDefault("update.batch_timeout", "60s")
	v.SetDefault("update.individual_timeout", "30s")

	// Per-carrier auto-update defaults
	v.SetDefault("carriers.ups.auto_update_enabled", true)
	v.SetDefault("carriers.ups.auto_update_cutoff_days", 30)
	v.SetDefault("carriers.dhl.auto_update_enabled", true)
	v.SetDefault("carriers.dhl.auto_update_cutoff_days", 0)

	// Cache defaults
	v.SetDefault("cache.ttl", "5m")
	v.SetDefault("cache.disabled", false)

	// Development/testing defaults
	v.SetDefault("rate_limit.disabled", false)

	// Admin defaults
	v.SetDefault("admin.auth_disabled", false)
	v.SetDefault("admin.api_key", "")

	// FedEx defaults
	v.SetDefault("carriers.fedex.api_url", "https://apis.fedex.com")
}

// setupServerEnvBinding sets up environment variable binding for server configuration
func setupServerEnvBinding(v *viper.Viper) {
	// Set environment variable prefix
	v.SetEnvPrefix("PKG_TRACKER")
	v.AutomaticEnv()

	// Bind new format environment variables
	envBindings := map[string]string{
		"server.port":                          "SERVER_PORT",
		"server.host":                          "SERVER_HOST",
		"database.path":                        "DATABASE_PATH",
		"logging.level":                        "LOGGING_LEVEL",
		"update.interval":                      "UPDATE_INTERVAL",
		"update.auto_enabled":                  "UPDATE_AUTO_ENABLED",
		"update.cutoff_days":                   "UPDATE_CUTOFF_DAYS",
		"update.batch_size":                    "UPDATE_BATCH_SIZE",
		"update.max_retries":                   "UPDATE_MAX_RETRIES",
		"update.failure_threshold":             "UPDATE_FAILURE_THRESHOLD",
		"update.batch_timeout":                 "UPDATE_BATCH_TIMEOUT",
		"update.individual_timeout":            "UPDATE_INDIVIDUAL_TIMEOUT",
		"carriers.usps.api_key":                "CARRIERS_USPS_API_KEY",
		"carriers.ups.api_key":                 "CARRIERS_UPS_API_KEY",
		"carriers.ups.client_id":               "CARRIERS_UPS_CLIENT_ID",
		"carriers.ups.client_secret":           "CARRIERS_UPS_CLIENT_SECRET",
		"carriers.ups.auto_update_enabled":     "CARRIERS_UPS_AUTO_UPDATE_ENABLED",
		"carriers.ups.auto_update_cutoff_days": "CARRIERS_UPS_AUTO_UPDATE_CUTOFF_DAYS",
		"carriers.fedex.api_key":               "CARRIERS_FEDEX_API_KEY",
		"carriers.fedex.secret_key":            "CARRIERS_FEDEX_SECRET_KEY",
		"carriers.fedex.api_url":               "CARRIERS_FEDEX_API_URL",
		"carriers.dhl.api_key":                 "CARRIERS_DHL_API_KEY",
		"carriers.dhl.auto_update_enabled":     "CARRIERS_DHL_AUTO_UPDATE_ENABLED",
		"carriers.dhl.auto_update_cutoff_days": "CARRIERS_DHL_AUTO_UPDATE_CUTOFF_DAYS",
		"cache.ttl":                            "CACHE_TTL",
		"cache.disabled":                       "CACHE_DISABLED",
		"rate_limit.disabled":                  "RATE_LIMIT_DISABLED",
		"admin.api_key":                        "ADMIN_API_KEY",
		"admin.auth_disabled":                  "ADMIN_AUTH_DISABLED",
	}

	for configKey, envSuffix := range envBindings {
		v.BindEnv(configKey, "PKG_TRACKER_"+envSuffix)
	}

	// Bind old format environment variables for backward compatibility
	oldEnvBindings := map[string]string{
		"server.port":                          "SERVER_PORT",
		"server.host":                          "SERVER_HOST",
		"database.path":                        "DB_PATH",
		"logging.level":                        "LOG_LEVEL",
		"update.interval":                      "UPDATE_INTERVAL",
		"update.auto_enabled":                  "AUTO_UPDATE_ENABLED",
		"update.cutoff_days":                   "AUTO_UPDATE_CUTOFF_DAYS",
		"update.batch_size":                    "AUTO_UPDATE_BATCH_SIZE",
		"update.max_retries":                   "AUTO_UPDATE_MAX_RETRIES",
		"update.failure_threshold":             "AUTO_UPDATE_FAILURE_THRESHOLD",
		"update.batch_timeout":                 "AUTO_UPDATE_BATCH_TIMEOUT",
		"update.individual_timeout":            "AUTO_UPDATE_INDIVIDUAL_TIMEOUT",
		"carriers.usps.api_key":                "USPS_API_KEY",
		"carriers.ups.api_key":                 "UPS_API_KEY",
		"carriers.ups.client_id":               "UPS_CLIENT_ID",
		"carriers.ups.client_secret":           "UPS_CLIENT_SECRET",
		"carriers.ups.auto_update_enabled":     "UPS_AUTO_UPDATE_ENABLED",
		"carriers.ups.auto_update_cutoff_days": "UPS_AUTO_UPDATE_CUTOFF_DAYS",
		"carriers.fedex.api_key":               "FEDEX_API_KEY",
		"carriers.fedex.secret_key":            "FEDEX_SECRET_KEY",
		"carriers.fedex.api_url":               "FEDEX_API_URL",
		"carriers.dhl.api_key":                 "DHL_API_KEY",
		"carriers.dhl.auto_update_enabled":     "DHL_AUTO_UPDATE_ENABLED",
		"carriers.dhl.auto_update_cutoff_days": "DHL_AUTO_UPDATE_CUTOFF_DAYS",
		"cache.ttl":                            "CACHE_TTL",
		"cache.disabled":                       "DISABLE_CACHE",
		"rate_limit.disabled":                  "DISABLE_RATE_LIMIT",
		"admin.api_key":                        "ADMIN_API_KEY",
		"admin.auth_disabled":                  "DISABLE_ADMIN_AUTH",
	}

	for configKey, envVar := range oldEnvBindings {
		v.BindEnv(configKey, envVar)
	}

	// Prioritize new format over old format by binding them separately
	// New format values will override old format values due to AutomaticEnv()
}

// loadConfigFile loads configuration file if it exists
func loadConfigFile(v *viper.Viper) error {
	// Check if a specific config file was set
	if v.ConfigFileUsed() == "" {
		// Add configuration search paths
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("$HOME/.package-tracker")

		// Set configuration file name (without extension)
		v.SetConfigName("config")
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

// unmarshalServerConfig unmarshals Viper configuration into Config struct
func unmarshalServerConfig(v *viper.Viper, config *Config) error {
	// Map Viper keys to struct fields
	config.ServerPort = v.GetString("server.port")
	config.ServerHost = v.GetString("server.host")
	config.DBPath = v.GetString("database.path")
	config.LogLevel = v.GetString("logging.level")

	// Parse duration fields
	var err error
	config.UpdateInterval, err = time.ParseDuration(v.GetString("update.interval"))
	if err != nil {
		return fmt.Errorf("invalid update interval: %w", err)
	}

	config.CacheTTL, err = time.ParseDuration(v.GetString("cache.ttl"))
	if err != nil {
		return fmt.Errorf("invalid cache TTL: %w", err)
	}

	config.AutoUpdateBatchTimeout, err = time.ParseDuration(v.GetString("update.batch_timeout"))
	if err != nil {
		return fmt.Errorf("invalid batch timeout: %w", err)
	}

	config.AutoUpdateIndividualTimeout, err = time.ParseDuration(v.GetString("update.individual_timeout"))
	if err != nil {
		return fmt.Errorf("invalid individual timeout: %w", err)
	}

	// Carrier API keys
	config.USPSAPIKey = v.GetString("carriers.usps.api_key")
	config.UPSAPIKey = v.GetString("carriers.ups.api_key")
	config.UPSClientID = v.GetString("carriers.ups.client_id")
	config.UPSClientSecret = v.GetString("carriers.ups.client_secret")
	config.FedExAPIKey = v.GetString("carriers.fedex.api_key")
	config.FedExSecretKey = v.GetString("carriers.fedex.secret_key")
	config.FedExAPIURL = v.GetString("carriers.fedex.api_url")
	config.DHLAPIKey = v.GetString("carriers.dhl.api_key")

	// Boolean flags
	config.AutoUpdateEnabled = v.GetBool("update.auto_enabled")
	config.UPSAutoUpdateEnabled = v.GetBool("carriers.ups.auto_update_enabled")
	config.DHLAutoUpdateEnabled = v.GetBool("carriers.dhl.auto_update_enabled")
	config.DisableRateLimit = v.GetBool("rate_limit.disabled")
	config.DisableCache = v.GetBool("cache.disabled")
	config.DisableAdminAuth = v.GetBool("admin.auth_disabled")

	// Integer values
	config.AutoUpdateCutoffDays = v.GetInt("update.cutoff_days")
	config.AutoUpdateBatchSize = v.GetInt("update.batch_size")
	config.AutoUpdateMaxRetries = v.GetInt("update.max_retries")
	config.AutoUpdateFailureThreshold = v.GetInt("update.failure_threshold")
	config.UPSAutoUpdateCutoffDays = v.GetInt("carriers.ups.auto_update_cutoff_days")
	config.DHLAutoUpdateCutoffDays = v.GetInt("carriers.dhl.auto_update_cutoff_days")

	// Admin API key
	config.AdminAPIKey = v.GetString("admin.api_key")

	return nil
}

// LoadServerConfig loads server configuration using default Viper instance
func LoadServerConfig() (*Config, error) {
	v := viper.New()
	return LoadServerConfigWithViper(v)
}

// LoadServerConfigWithFile loads server configuration from a specific file
func LoadServerConfigWithFile(configFile string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configFile)
	return LoadServerConfigWithViper(v)
}

// LoadServerConfigWithEnvFile loads server configuration with .env file support
func LoadServerConfigWithEnvFile(envFile string) (*Config, error) {
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
	return LoadServerConfigWithViper(v)
}

// Ensure backward compatibility by providing a new Load function that works with Viper
func LoadWithViper() (*Config, error) {
	return LoadServerConfigWithEnvFile("")
}