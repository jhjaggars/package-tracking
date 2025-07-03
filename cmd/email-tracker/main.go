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

package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"package-tracking/internal/api"
	"package-tracking/internal/carriers"
	"package-tracking/internal/config"
	"package-tracking/internal/email"
	"package-tracking/internal/parser"
	"package-tracking/internal/workers"
)

const (
	// Version information
	Version   = "1.0.0"
	BuildDate = "development"
	
	// Application name
	AppName = "email-tracker"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	
	logger.Info("Starting email tracker service",
		"version", Version,
		"build_date", BuildDate)
	
	// Load email configuration
	cfg, err := config.LoadEmailConfig()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	
	// Set configuration defaults
	cfg.SetDefaults()
	
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
		os.Exit(1)
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
	
	extractor := parser.NewTrackingExtractor(carrierFactory, extractorConfig)
	logger.Info("Tracking extractor initialized")
	
	// Initialize state manager
	stateManager, err := email.NewSQLiteStateManager(cfg.Processing.StateDBPath)
	if err != nil {
		logger.Error("Failed to initialize state manager", "error", err)
		os.Exit(1)
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
		os.Exit(1)
	}
	
	logger.Info("API client initialized successfully", "url", cfg.API.URL)
	
	// Initialize email processor
	processorConfig := &workers.EmailProcessorConfig{
		CheckInterval:     cfg.Processing.CheckInterval,
		SearchQuery:       cfg.GetSearchQuery(),
		SearchAfterDays:   cfg.Search.AfterDays,
		MaxEmailsPerRun:   cfg.Processing.MaxEmailsPerRun,
		UnreadOnly:        cfg.Search.UnreadOnly,
		DryRun:            cfg.Processing.DryRun,
		RetryCount:        cfg.API.RetryCount,
		RetryDelay:        cfg.API.RetryDelay,
		ProcessingTimeout: cfg.Processing.ProcessingTimeout,
	}
	
	processor := workers.NewEmailProcessor(
		processorConfig,
		emailClient,
		extractor,
		stateManager,
		apiClient,
		logger,
	)
	
	logger.Info("Email processor initialized")
	
	// Start the email processor
	processor.Start()
	defer processor.Stop()
	
	logger.Info("Email tracker service started successfully")
	
	// Handle graceful shutdown
	if err := handleSignals(processor, logger); err != nil {
		logger.Error("Service error", "error", err)
		os.Exit(1)
	}
	
	logger.Info("Email tracker service stopped gracefully")
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

// handleSignals handles graceful shutdown on system signals
func handleSignals(processor *workers.EmailProcessor, logger *slog.Logger) error {
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
		
		// Stop the email processor
		processor.Stop()
		
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

// printUsage prints usage information
func printUsage() {
	log.Printf(`
Email Tracker Service v%s

DESCRIPTION:
    Monitors Gmail accounts for shipping emails and automatically extracts
    tracking numbers to create shipments in the package tracking system.

CONFIGURATION:
    Configuration is done via environment variables:

    Gmail API Configuration:
        GMAIL_CLIENT_ID         - OAuth2 client ID
        GMAIL_CLIENT_SECRET     - OAuth2 client secret  
        GMAIL_REFRESH_TOKEN     - OAuth2 refresh token
        GMAIL_ACCESS_TOKEN      - OAuth2 access token
        GMAIL_TOKEN_FILE        - Token storage file (default: ./gmail-token.json)
        
    Gmail IMAP Fallback:
        GMAIL_USERNAME          - Gmail username/email
        GMAIL_APP_PASSWORD      - Gmail app-specific password
        
    Search Configuration:
        GMAIL_SEARCH_QUERY      - Custom Gmail search query
        GMAIL_SEARCH_AFTER_DAYS - Only process emails from last N days (default: 30)
        GMAIL_SEARCH_UNREAD_ONLY - Only process unread emails (default: false)
        GMAIL_SEARCH_MAX_RESULTS - Maximum emails per search (default: 100)
        
    Processing Configuration:
        EMAIL_CHECK_INTERVAL    - How often to check for new emails (default: 5m)
        EMAIL_MAX_PER_RUN       - Maximum emails to process per run (default: 50)
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
    %s
    
    # With custom search and API settings
    export GMAIL_SEARCH_QUERY="from:amazon.com subject:shipped"
    export EMAIL_API_URL="http://tracking.example.com:8080"
    export EMAIL_CHECK_INTERVAL="10m"
    %s
    
    # Dry run mode for testing
    export EMAIL_DRY_RUN="true"
    export EMAIL_DEBUG_MODE="true"
    %s

`, Version, AppName, AppName, AppName)
}