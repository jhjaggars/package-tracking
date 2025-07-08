// Copyright 2024 Package Tracking System
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	"package-tracking/internal/api"
	"package-tracking/internal/carriers"
	"package-tracking/internal/config"
	"package-tracking/internal/database"
	"package-tracking/internal/email"
	"package-tracking/internal/parser"
	"package-tracking/internal/workers"
)

const (
	// Version information
	Version   = "1.0.0"
	BuildDate = "development"
)

var (
	configFile string
	dryRun     bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "email-tracker",
	Short: "Email tracking service for package tracking system",
	Long: `Email Tracker Service v1.0.0

DESCRIPTION:
    Monitors Gmail accounts for shipping emails and automatically extracts
    tracking numbers to create shipments in the package tracking system.

CONFIGURATION:
    Configuration is done via environment variables and .env files:

    Gmail API Configuration:
        GMAIL_CLIENT_ID         - OAuth2 client ID
        GMAIL_CLIENT_SECRET     - OAuth2 client secret  
        GMAIL_REFRESH_TOKEN     - OAuth2 refresh token
        GMAIL_ACCESS_TOKEN      - OAuth2 access token
        GMAIL_TOKEN_FILE        - Token storage file (default: ./gmail-token.json)
        
    Gmail IMAP Fallback:
        GMAIL_USERNAME          - Gmail username/email
        GMAIL_APP_PASSWORD      - Gmail app-specific password
        
    Time-Based Processing Configuration:
        EMAIL_SCAN_DAYS         - Number of days to scan back for emails (default: 7)
        EMAIL_BODY_STORAGE      - Store full email bodies for analysis (default: true)
        EMAIL_RETENTION_DAYS    - Days to retain email bodies before cleanup (default: 30)
        EMAIL_CHECK_INTERVAL    - How often to scan for new emails (default: 5m)
        EMAIL_MAX_PER_SCAN      - Maximum emails to process per scan (default: 100)
        EMAIL_DRY_RUN           - Only extract tracking numbers, don't create shipments (default: false)
        EMAIL_STATE_DB_PATH     - SQLite database for processing state (default: ./email-state.db)
        EMAIL_MIN_CONFIDENCE    - Minimum confidence for tracking number extraction (default: 0.5)
        EMAIL_DEBUG_MODE        - Enable debug logging (default: false)
        
    API Configuration:
        EMAIL_API_URL           - Package tracking API URL (default: http://localhost:8080)
        EMAIL_API_TIMEOUT       - API request timeout (default: 30s)
        EMAIL_API_RETRY_COUNT   - Number of API retries (default: 3)
        EMAIL_API_RETRY_DELAY   - Delay between retries (default: 1s)
        
    LLM Configuration (Optional):
        LLM_ENABLED             - Enable LLM-based parsing (default: false)
        LLM_PROVIDER            - LLM provider: openai, anthropic, local (default: disabled)
        LLM_MODEL               - Model name (e.g., gpt-4, claude-3)
        LLM_API_KEY             - API key for hosted LLMs
        LLM_ENDPOINT            - Endpoint for local LLMs
        LLM_MAX_TOKENS          - Maximum response tokens (default: 1000)
        LLM_TEMPERATURE         - Sampling temperature (default: 0.1)

EXAMPLES:
    # Basic usage with OAuth2
    export GMAIL_CLIENT_ID="your-client-id"
    export GMAIL_CLIENT_SECRET="your-client-secret"
    export GMAIL_REFRESH_TOKEN="your-refresh-token"
    email-tracker
    
    # With custom configuration file
    email-tracker --config=.env.production
    
    # Dry run mode for testing
    email-tracker --dry-run
    
    # Using .env file with dry run override
    echo "EMAIL_DRY_RUN=false" > .env.test
    email-tracker --config=.env.test --dry-run
    
    # Time-based scanning configuration
    echo "EMAIL_SCAN_DAYS=14" > .env.custom
    echo "EMAIL_BODY_STORAGE=true" >> .env.custom
    echo "EMAIL_RETENTION_DAYS=60" >> .env.custom
    email-tracker --config=.env.custom`,
	Version: "1.0.0",
	RunE:    runEmailTracker,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	fang.Execute(context.Background(), rootCmd)
}

func init() {
	cobra.OnInitialize(initConfig)

	// Add CLI flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is .env in current directory)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "only extract tracking numbers, don't create shipments")
}

// loadConfiguration loads configuration from files and environment variables
func loadConfiguration() (*config.EmailConfig, error) {
	var cfg *config.EmailConfig
	var err error
	
	// Load configuration with Viper (supports YAML, TOML, JSON, .env)
	if configFile != "" {
		// Check if it's a .env file or a structured config file
		if strings.HasSuffix(configFile, ".env") || !strings.Contains(configFile, ".") || strings.HasPrefix(filepath.Base(configFile), ".env") {
			// Use legacy .env loader for .env files (includes security validation)
			cfg, err = config.LoadEmailConfigWithEnvFile(configFile)
		} else {
			// Validate config file path for security (prevent directory traversal)
			if err := config.ValidateConfigFilePath(configFile); err != nil {
				return nil, fmt.Errorf("failed to load configuration: %w", err)
			}
			// Use Viper loader for YAML/TOML/JSON files
			cfg, err = config.LoadEmailConfigViperWithFile(configFile)
		}
	} else {
		// Try Viper first (supports auto-discovery), fall back to legacy
		cfg, err = config.LoadEmailConfigViper()
		if err != nil {
			// Fall back to legacy .env loader
			cfg, err = config.LoadEmailConfig()
		}
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Override with CLI flags
	if dryRun {
		originalValue := cfg.Processing.DryRun
		cfg.Processing.DryRun = true
		if originalValue != true {
			// Note: Using fmt.Printf since logger isn't available yet
			fmt.Printf("DEBUG: CLI flag --dry-run overriding config value: %v -> %v\n", originalValue, true)
		}
	}
	
	// Set configuration defaults
	cfg.SetDefaults()
	
	return cfg, nil
}

// initConfig is called by cobra during initialization
func initConfig() {
	// Configuration loading is now done in runEmailTracker
	// This function is kept for cobra initialization compatibility
}

// runEmailTracker is the main execution function for the email tracker
func runEmailTracker(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}
	// Initialize structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	
	logger.Info("Starting email tracker service",
		"version", Version,
		"build_date", BuildDate)
	
	logger.Info("Configuration loaded successfully",
		"dry_run", cfg.Processing.DryRun,
		"check_interval", cfg.Processing.CheckInterval,
		"llm_enabled", cfg.LLM.Enabled)
	
	// Log configuration (with sensitive fields redacted)
	if configJSON, err := cfg.ToJSON(); err == nil {
		logger.Debug("Configuration details", "config", configJSON)
	}
	
	// Initialize Gmail client
	emailClient, err := createEmailClient(cfg, logger)
	if err != nil {
		logger.Error("Failed to create email client", "error", err)
		return fmt.Errorf("failed to create email client: %w", err)
	}
	defer emailClient.Close()
	
	logger.Info("Email client initialized successfully")
	
	// Initialize carrier factory for tracking validation
	carrierFactory := carriers.NewClientFactory()
	
	// Initialize tracking extractor
	extractorConfig := &parser.ExtractorConfig{
		EnableLLM:           cfg.LLM.Enabled,
		MinConfidence:       cfg.Processing.MinConfidence,
		MaxCandidates:       cfg.Processing.MaxCandidates,
		UseHybridValidation: cfg.Processing.UseHybridValidation,
		DebugMode:           cfg.Processing.DebugMode,
	}
	
	// Convert to LLM config format
	llmConfig := &parser.LLMConfig{
		Provider:    cfg.LLM.Provider,
		Model:       cfg.LLM.Model,
		APIKey:      cfg.LLM.APIKey,
		Endpoint:    cfg.LLM.Endpoint,
		MaxTokens:   cfg.LLM.MaxTokens,
		Temperature: cfg.LLM.Temperature,
		Timeout:     cfg.LLM.Timeout,
		RetryCount:  cfg.LLM.RetryCount,
		Enabled:     cfg.LLM.Enabled,
	}
	
	extractor := parser.NewTrackingExtractor(carrierFactory, extractorConfig, llmConfig)
	logger.Info("Tracking extractor initialized")
	
	// Initialize state manager
	stateManager, err := email.NewSQLiteStateManager(cfg.Processing.StateDBPath)
	if err != nil {
		logger.Error("Failed to initialize state manager", "error", err)
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}
	defer stateManager.Close()
	
	logger.Info("State manager initialized", "db_path", cfg.Processing.StateDBPath)
	
	// Initialize API client
	apiConfig := &api.ClientConfig{
		BaseURL:       cfg.API.URL,
		Timeout:       cfg.API.Timeout,
		RetryCount:    cfg.API.RetryCount,
		RetryDelay:    cfg.API.RetryDelay,
		UserAgent:     cfg.API.UserAgent,
		BackoffFactor: cfg.API.BackoffFactor,
	}
	
	apiClient := api.NewClient(apiConfig)
	
	// Test API connection
	if err := apiClient.HealthCheck(); err != nil {
		logger.Error("API health check failed", "error", err, "url", cfg.API.URL)
		return fmt.Errorf("API health check failed: %w", err)
	}
	
	logger.Info("API client initialized successfully", "url", cfg.API.URL)
	
	// Initialize main database for email body storage (only if body storage is enabled)
	var emailStore *database.EmailStore
	var shipmentStore *database.ShipmentStore
	
	if cfg.TimeBased.BodyStorageEnabled {
		// Use a different database path for email body storage to avoid conflicts
		// We'll use the main database.db since that's where shipments are stored
		mainDBPath := "./database.db" // Use the main application database
		
		mainDB, err := database.Open(mainDBPath)
		if err != nil {
			logger.Error("Failed to initialize main database for email body storage", "error", err)
			return fmt.Errorf("failed to initialize main database: %w", err)
		}
		defer mainDB.Close()
		
		emailStore = mainDB.Emails
		shipmentStore = mainDB.Shipments
		
		logger.Info("Email body storage enabled", "db_path", mainDBPath)
	} else {
		logger.Info("Email body storage disabled")
	}
	
	// Initialize time-based email processor
	timeProcessorConfig := &workers.TimeBasedEmailProcessorConfig{
		ScanDays:           cfg.TimeBased.ScanDays,
		BodyStorageEnabled: cfg.TimeBased.BodyStorageEnabled,
		RetentionDays:      cfg.TimeBased.RetentionDays,
		MaxEmailsPerScan:   cfg.TimeBased.MaxEmailsPerScan,
		UnreadOnly:         cfg.TimeBased.UnreadOnly,
		CheckInterval:      cfg.Processing.CheckInterval,
		ProcessingTimeout:  cfg.Processing.ProcessingTimeout,
		RetryCount:         cfg.TimeBased.RetryCount,
		RetryDelay:         cfg.TimeBased.RetryDelay,
		DryRun:             cfg.Processing.DryRun,
	}
	
	// Cast email client to time-based interface
	timeBasedClient, ok := emailClient.(workers.TimeBasedEmailClient)
	if !ok {
		logger.Error("Email client does not support time-based operations")
		return fmt.Errorf("email client does not implement TimeBasedEmailClient interface")
	}
	
	// Wrap the API client with the adapter
	adaptedAPIClient := workers.NewAPIClientAdapter(apiClient)
	
	timeProcessor := workers.NewTimeBasedEmailProcessor(
		timeProcessorConfig,
		timeBasedClient,
		extractor,
		stateManager,  // Use stateManager for email state tracking
		emailStore,    // Use emailStore for body storage (may be nil if disabled)
		shipmentStore, // Use shipmentStore for linking emails to shipments
		adaptedAPIClient,
		logger,
	)
	
	logger.Info("Time-based email processor initialized")
	
	// Start the time-based email processor
	go startTimeBasedProcessor(timeProcessor, logger)
	defer func() {
		logger.Info("Stopping time-based email processor")
	}()
	
	logger.Info("Email tracker service started successfully")
	
	// Handle graceful shutdown
	if err := handleSignals(timeProcessor, logger); err != nil {
		logger.Error("Service error", "error", err)
		return fmt.Errorf("service error: %w", err)
	}
	
	logger.Info("Email tracker service stopped gracefully")
	return nil
}

// createEmailClient creates and configures the email client
func createEmailClient(cfg *config.EmailConfig, logger *slog.Logger) (email.EmailClient, error) {
	// Check which authentication method to use
	if cfg.IsOAuth2Configured() {
		logger.Info("Using Gmail API with OAuth2 authentication")
		
		gmailConfig := &email.GmailConfig{
			ClientID:       cfg.Gmail.ClientID,
			ClientSecret:   cfg.Gmail.ClientSecret,
			RefreshToken:   cfg.Gmail.RefreshToken,
			AccessToken:    cfg.Gmail.AccessToken,
			TokenFile:      cfg.Gmail.TokenFile,
			MaxResults:     cfg.Gmail.MaxResults,
			RequestTimeout: cfg.Gmail.RequestTimeout,
			RateLimitDelay: cfg.Gmail.RateLimitDelay,
		}
		
		return email.NewGmailClient(gmailConfig)
		
	} else if cfg.IsIMAPConfigured() {
		// TODO: Implement IMAP fallback client
		logger.Warn("IMAP fallback not yet implemented, using Gmail API")
		return nil, fmt.Errorf("IMAP client not implemented")
		
	} else {
		return nil, fmt.Errorf("no valid email authentication method configured")
	}
}

// startTimeBasedProcessor starts the time-based email processor with periodic scanning
func startTimeBasedProcessor(processor *workers.TimeBasedEmailProcessor, logger *slog.Logger) {
	// Perform initial scan after a short delay
	time.Sleep(10 * time.Second)
	
	// Get the last scan time (start from 7 days ago if no previous scan)
	since := time.Now().AddDate(0, 0, -7)
	
	logger.Info("Starting initial time-based email scan", "since", since)
	if err := processor.ProcessEmailsSince(since); err != nil {
		logger.Error("Initial email processing failed", "error", err)
	}
	
	// Start periodic scanning
	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Process emails since last 10 minutes to catch any new ones
			since := time.Now().Add(-10 * time.Minute)
			logger.Debug("Performing scheduled email scan", "since", since)
			if err := processor.ProcessEmailsSince(since); err != nil {
				logger.Error("Scheduled email processing failed", "error", err)
			}
		}
	}
}

// handleSignals handles graceful shutdown on system signals
func handleSignals(processor *workers.TimeBasedEmailProcessor, logger *slog.Logger) error {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Channel to receive OS signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	
	// Channel to receive shutdown completion
	shutdownChan := make(chan struct{})
	
	// Start signal handling goroutine
	go func() {
		sig := <-signalChan
		logger.Info("Received shutdown signal", "signal", sig)
		
		// Start graceful shutdown
		logger.Info("Starting graceful shutdown...")
		
		// Wait a bit for processor to finish current operations
		time.Sleep(2 * time.Second)
		
		// Signal shutdown completion
		close(shutdownChan)
	}()
	
	// Wait for either shutdown signal or context cancellation
	select {
	case <-shutdownChan:
		logger.Info("Graceful shutdown completed")
		return nil
		
	case <-ctx.Done():
		return ctx.Err()
	}
}