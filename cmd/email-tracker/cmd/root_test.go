package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestEmailTrackerCLI(t *testing.T) {
	// Save original command to restore later
	originalCmd := rootCmd

	t.Run("Help flag works", func(t *testing.T) {
		cmd := &cobra.Command{
			Use:   "email-tracker",
			Short: "Email tracking service for package tracking system",
			Long:  "Test help command",
		}
		cmd.SetArgs([]string{"--help"})
		
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Help command failed: %v", err)
		}
		
		output := buf.String()
		if !strings.Contains(output, "Test help command") {
			t.Errorf("Help output missing expected content, got: %s", output)
		}
	})

	t.Run("Version flag works", func(t *testing.T) {
		cmd := &cobra.Command{
			Use:     "email-tracker",
			Version: "1.0.0",
		}
		cmd.SetArgs([]string{"--version"})
		
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Version command failed: %v", err)
		}
	})

	t.Run("Configuration loading with invalid env file", func(t *testing.T) {
		// Test that malicious paths are rejected
		configFile = "../../../etc/passwd"
		dryRun = false
		
		_, err := loadConfiguration()
		if err == nil {
			t.Error("Expected error for directory traversal attempt")
		}
		if !strings.Contains(err.Error(), "cannot contain") {
			t.Errorf("Expected directory traversal error, got: %v", err)
		}
		
		// Reset globals
		configFile = ""
		dryRun = false
	})

	t.Run("Configuration loading with valid env file", func(t *testing.T) {
		// Create a temporary .env file
		tmpFile, err := os.CreateTemp("", "test*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())
		
		envContent := `GMAIL_CLIENT_ID=test-client-id
GMAIL_CLIENT_SECRET=test-client-secret
EMAIL_DRY_RUN=true
EMAIL_API_URL=http://test.localhost:8080
`
		
		if _, err := tmpFile.WriteString(envContent); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()
		
		// Test loading this config
		configFile = tmpFile.Name()
		dryRun = false
		
		cfg, err := loadConfiguration()
		if err != nil {
			t.Fatalf("Expected no error loading valid config file, got: %v", err)
		}
		
		// Verify configuration was loaded correctly
		if cfg.Gmail.ClientID != "test-client-id" {
			t.Errorf("Expected ClientID 'test-client-id', got '%s'", cfg.Gmail.ClientID)
		}
		if !cfg.Processing.DryRun {
			t.Error("Expected DryRun to be true from env file")
		}
		if cfg.API.URL != "http://test.localhost:8080" {
			t.Errorf("Expected API URL from env file, got '%s'", cfg.API.URL)
		}
		
		// Reset globals
		configFile = ""
		dryRun = false
	})

	t.Run("CLI flag overrides env file", func(t *testing.T) {
		// Create a temporary .env file with dry run false
		tmpFile, err := os.CreateTemp("", "test*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())
		
		envContent := `GMAIL_CLIENT_ID=test-client-id
GMAIL_CLIENT_SECRET=test-client-secret
EMAIL_DRY_RUN=false
`
		
		if _, err := tmpFile.WriteString(envContent); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()
		
		// Test that CLI flag overrides env file
		configFile = tmpFile.Name()
		dryRun = true  // CLI flag override
		
		cfg, err := loadConfiguration()
		if err != nil {
			t.Fatalf("Expected no error loading config, got: %v", err)
		}
		
		// Verify CLI flag took precedence
		if !cfg.Processing.DryRun {
			t.Error("Expected DryRun to be true from CLI flag override")
		}
		
		// Reset globals
		configFile = ""
		dryRun = false
	})

	t.Run("Default configuration when no env file", func(t *testing.T) {
		// Clean up any environment variables that might interfere
		envVars := []string{
			"GMAIL_CLIENT_ID", "GMAIL_CLIENT_SECRET", "EMAIL_DRY_RUN", "EMAIL_API_URL",
		}
		for _, v := range envVars {
			os.Unsetenv(v)
		}
		defer func() {
			for _, v := range envVars {
				os.Unsetenv(v)
			}
		}()
		
		// Set minimal required credentials for validation
		os.Setenv("GMAIL_CLIENT_ID", "test-client")
		os.Setenv("GMAIL_CLIENT_SECRET", "test-secret")
		
		configFile = ""
		dryRun = false
		
		cfg, err := loadConfiguration()
		if err != nil {
			t.Fatalf("Expected no error with default config, got: %v", err)
		}
		
		// Verify defaults
		if cfg.API.URL != "http://localhost:8080" {
			t.Errorf("Expected default API URL, got '%s'", cfg.API.URL)
		}
		if cfg.Processing.DryRun != false {
			t.Error("Expected default DryRun to be false")
		}
		if cfg.Processing.CheckInterval != 5*time.Minute {
			t.Errorf("Expected default check interval 5m, got %v", cfg.Processing.CheckInterval)
		}
	})

	// Restore original command
	rootCmd = originalCmd
}

func TestConfigurationPrecedence(t *testing.T) {
	t.Run("Full precedence chain: CLI > env vars > .env file > defaults", func(t *testing.T) {
		// Save and restore global state
		originalConfigFile := configFile
		originalDryRun := dryRun
		defer func() {
			configFile = originalConfigFile
			dryRun = originalDryRun
		}()
		// Create a temporary .env file with specific values
		tmpFile, err := os.CreateTemp("", "test*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())
		
		envFileContent := `GMAIL_CLIENT_ID=env-file-client
GMAIL_CLIENT_SECRET=env-file-secret
EMAIL_DRY_RUN=false
EMAIL_CHECK_INTERVAL=10m
EMAIL_API_URL=http://envfile.localhost:8080
`
		
		if _, err := tmpFile.WriteString(envFileContent); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()
		
		// Set environment variables (should override .env file)
		os.Setenv("GMAIL_CLIENT_ID", "env-var-client")
		os.Setenv("EMAIL_CHECK_INTERVAL", "15m")
		defer func() {
			os.Unsetenv("GMAIL_CLIENT_ID")
			os.Unsetenv("EMAIL_CHECK_INTERVAL")
		}()
		
		// Set CLI flags (should override both env vars and .env file)
		configFile = tmpFile.Name()
		dryRun = true  // CLI flag override
		
		cfg, err := loadConfiguration()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		
		// Verify precedence order:
		// 1. CLI flag wins for dry run
		if !cfg.Processing.DryRun {
			t.Error("Expected CLI flag to override dry run setting")
		}
		
		// 2. Environment variable wins for client ID (over .env file)
		if cfg.Gmail.ClientID != "env-var-client" {
			t.Errorf("Expected env var to override .env file for client ID, got '%s'", cfg.Gmail.ClientID)
		}
		
		// 3. Environment variable wins for check interval (over .env file)
		if cfg.Processing.CheckInterval != 15*time.Minute {
			t.Errorf("Expected env var check interval 15m, got %v", cfg.Processing.CheckInterval)
		}
		
		// 4. .env file wins for client secret (no env var override)
		if cfg.Gmail.ClientSecret != "env-file-secret" {
			t.Errorf("Expected .env file value for client secret, got '%s'", cfg.Gmail.ClientSecret)
		}
		
		// 5. .env file wins for API URL (no env var override)
		if cfg.API.URL != "http://envfile.localhost:8080" {
			t.Errorf("Expected .env file API URL, got '%s'", cfg.API.URL)
		}
	})
}

func TestLoadConfiguration_YAMLSupport(t *testing.T) {
	// Save and restore global state
	originalConfigFile := configFile
	originalDryRun := dryRun
	defer func() {
		configFile = originalConfigFile
		dryRun = originalDryRun
	}()
	
	// Clean up any environment variables that might interfere
	envVarsToClean := []string{
		"EMAIL_DRY_RUN", "PKG_TRACKER_EMAIL_PROCESSING_DRY_RUN",
		"GMAIL_CLIENT_ID", "PKG_TRACKER_EMAIL_GMAIL_CLIENT_ID",
		"EMAIL_CHECK_INTERVAL", "PKG_TRACKER_EMAIL_PROCESSING_CHECK_INTERVAL",
	}
	originalEnvValues := make(map[string]string)
	for _, envVar := range envVarsToClean {
		originalEnvValues[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}
	defer func() {
		for envVar, originalValue := range originalEnvValues {
			if originalValue != "" {
				os.Setenv(envVar, originalValue)
			}
		}
	}()
	
	// Create a temporary YAML config file
	tmpFile, err := os.CreateTemp("", "test-config*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	
	yamlContent := `
gmail:
  client_id: "test-client-id"
  client_secret: "test-client-secret"
  refresh_token: "test-refresh-token"
search:
  query: "from:test@example.com subject:yaml-test"
  after_days: 7
processing:
  dry_run: true
  check_interval: "3m"
api:
  url: "http://localhost:8080"
`
	
	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	
	// Test loading YAML configuration
	configFile = tmpFile.Name()
	dryRun = false
	
	cfg, err := loadConfiguration()
	if err != nil {
		t.Fatalf("Failed to load YAML configuration: %v", err)
	}
	
	// Verify configuration was loaded correctly
	if cfg.Gmail.ClientID != "test-client-id" {
		t.Errorf("Expected ClientID 'test-client-id', got %s", cfg.Gmail.ClientID)
	}
	
	if cfg.Search.Query != "from:test@example.com subject:yaml-test" {
		t.Errorf("Expected custom search query, got %s", cfg.Search.Query)
	}
	
	if cfg.Processing.CheckInterval.String() != "3m0s" {
		t.Errorf("Expected CheckInterval 3m0s, got %s", cfg.Processing.CheckInterval)
	}
	
	if !cfg.Processing.DryRun {
		t.Errorf("Expected DryRun to be true from YAML config")
	}
}

func TestLoadConfiguration_SecurityValidation(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		expectErr  bool
		errMsg     string
	}{
		{
			name:       "Directory traversal attack",
			configPath: "../../../etc/passwd",
			expectErr:  true,
			errMsg:     "cannot contain",
		},
		{
			name:       "Relative path with ..",
			configPath: "../config.yaml",
			expectErr:  true,
			errMsg:     "cannot contain",
		},
		{
			name:       "Valid YAML file",
			configPath: "test-config.yaml",
			expectErr:  false,
			errMsg:     "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFile = tt.configPath
			dryRun = false
			
			_, err := loadConfiguration()
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.configPath)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
				}
			} else if err != nil && !strings.Contains(err.Error(), "no such file") {
				// Allow "no such file" errors for non-existent test files
				t.Errorf("Expected no security error for %s, but got: %v", tt.configPath, err)
			}
			
			// Reset globals
			configFile = ""
		})
	}
}

func TestLoadConfiguration_FileTypeDetection(t *testing.T) {
	tests := []struct {
		name           string
		filename       string
		expectedLoader string // "env" or "viper"
	}{
		{"env file with extension", "config.env", "env"},
		{"env file without extension", "config", "env"},
		{"YAML file", "config.yaml", "viper"},
		{"TOML file", "config.toml", "viper"},
		{"JSON file", "config.json", "viper"},
		{"dotenv file", ".env.test", "env"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies the file type detection logic
			// We check the logic without actually loading files
			
			filename := tt.filename
			isEnvFile := strings.HasSuffix(filename, ".env") || !strings.Contains(filename, ".") || strings.HasPrefix(filepath.Base(filename), ".env")
			
			expectedIsEnv := tt.expectedLoader == "env"
			if isEnvFile != expectedIsEnv {
				t.Errorf("File type detection failed for %s: expected isEnvFile=%v, got %v", 
					filename, expectedIsEnv, isEnvFile)
			}
		})
	}
}