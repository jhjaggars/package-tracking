package workers

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"package-tracking/internal/email"
	"package-tracking/internal/parser"
)

// EmailProcessor handles automatic background processing of emails for tracking numbers
type EmailProcessor struct {
	ctx           context.Context
	cancel        context.CancelFunc
	config        *EmailProcessorConfig
	emailClient   email.EmailClient
	extractor     *parser.TrackingExtractor
	stateManager  StateManager
	apiClient     APIClient
	paused        atomic.Bool
	logger        *slog.Logger
	metrics       *ProcessingMetrics
}

// EmailProcessorConfig configures the email processor behavior
type EmailProcessorConfig struct {
	CheckInterval    time.Duration
	SearchQuery      string
	SearchAfterDays  int
	MaxEmailsPerRun  int
	UnreadOnly       bool
	DryRun           bool
	RetryCount       int
	RetryDelay       time.Duration
	ProcessingTimeout time.Duration
}

// StateManager handles email processing state tracking
type StateManager interface {
	IsProcessed(messageID string) (bool, error)
	MarkProcessed(entry *email.StateEntry) error
	Cleanup(olderThan time.Time) error
	GetStats() (*email.EmailMetrics, error)
}

// APIClient handles shipment creation via REST API
type APIClient interface {
	CreateShipment(tracking email.TrackingInfo) error
	HealthCheck() error
}

// ProcessingMetrics tracks email processing statistics
type ProcessingMetrics struct {
	TotalRuns          atomic.Int64
	TotalEmails        atomic.Int64
	ProcessedEmails    atomic.Int64
	SkippedEmails      atomic.Int64
	ErrorEmails        atomic.Int64
	TrackingNumbers    atomic.Int64
	ShipmentsCreated   atomic.Int64
	LastRun            atomic.Value // time.Time
	LastError          atomic.Value // string
	AverageRunTime     atomic.Value // time.Duration
}

// NewEmailProcessor creates a new email processor service
func NewEmailProcessor(
	config *EmailProcessorConfig,
	emailClient email.EmailClient,
	extractor *parser.TrackingExtractor,
	stateManager StateManager,
	apiClient APIClient,
	logger *slog.Logger,
) *EmailProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &EmailProcessor{
		ctx:          ctx,
		cancel:       cancel,
		config:       config,
		emailClient:  emailClient,
		extractor:    extractor,
		stateManager: stateManager,
		apiClient:    apiClient,
		logger:       logger,
		metrics:      &ProcessingMetrics{},
	}
}

// Start begins the background email processing
func (p *EmailProcessor) Start() {
	p.logger.Info("Starting email processor service",
		"check_interval", p.config.CheckInterval,
		"search_query", p.config.SearchQuery,
		"dry_run", p.config.DryRun,
		"max_emails_per_run", p.config.MaxEmailsPerRun)
	
	// Verify connections before starting
	if err := p.healthCheck(); err != nil {
		p.logger.Error("Health check failed", "error", err)
		return
	}
	
	go p.processingLoop()
}

// Stop gracefully stops the email processing
func (p *EmailProcessor) Stop() {
	p.logger.Info("Stopping email processor service")
	p.cancel()
}

// Pause temporarily pauses email processing
func (p *EmailProcessor) Pause() {
	p.paused.Store(true)
	p.logger.Info("Email processor paused")
}

// Resume resumes email processing
func (p *EmailProcessor) Resume() {
	p.paused.Store(false)
	p.logger.Info("Email processor resumed")
}

// IsPaused returns true if the processor is currently paused
func (p *EmailProcessor) IsPaused() bool {
	return p.paused.Load()
}

// IsRunning returns true if the processor is currently running
func (p *EmailProcessor) IsRunning() bool {
	select {
	case <-p.ctx.Done():
		return false
	default:
		return true
	}
}

// GetMetrics returns current processing metrics
func (p *EmailProcessor) GetMetrics() *ProcessingMetrics {
	return p.metrics
}

// healthCheck verifies all connections are working
func (p *EmailProcessor) healthCheck() error {
	if err := p.emailClient.HealthCheck(); err != nil {
		return fmt.Errorf("email client health check failed: %w", err)
	}
	
	if err := p.apiClient.HealthCheck(); err != nil {
		return fmt.Errorf("API client health check failed: %w", err)
	}
	
	return nil
}

// processingLoop is the main background loop that processes emails
func (p *EmailProcessor) processingLoop() {
	ticker := time.NewTicker(p.config.CheckInterval)
	defer ticker.Stop()
	
	// Perform initial processing after a short delay
	initialDelay := time.NewTimer(10 * time.Second)
	defer initialDelay.Stop()
	
	for {
		select {
		case <-p.ctx.Done():
			p.logger.Info("Email processing loop stopped")
			return
			
		case <-initialDelay.C:
			p.runProcessing()
			
		case <-ticker.C:
			if !p.paused.Load() {
				p.runProcessing()
			}
		}
	}
}

// runProcessing performs a single email processing run
func (p *EmailProcessor) runProcessing() {
	startTime := time.Now()
	p.metrics.TotalRuns.Add(1)
	
	p.logger.Info("Starting email processing run")
	
	// Create timeout context for this run
	ctx, cancel := context.WithTimeout(p.ctx, p.config.ProcessingTimeout)
	defer cancel()
	
	// Search for emails
	emails, err := p.searchEmails(ctx)
	if err != nil {
		p.logger.Error("Failed to search emails", "error", err)
		p.metrics.LastError.Store(err.Error())
		return
	}
	
	p.logger.Info("Found emails to process", "count", len(emails))
	p.metrics.TotalEmails.Add(int64(len(emails)))
	
	// Process each email
	processed := 0
	skipped := 0
	errors := 0
	
	for _, emailMsg := range emails {
		select {
		case <-ctx.Done():
			p.logger.Warn("Processing cancelled due to timeout")
			return
		default:
		}
		
		result := p.processEmail(&emailMsg)
		
		switch result.Status {
		case "processed":
			processed++
		case "skipped":
			skipped++
		case "error":
			errors++
		}
		
		// Add small delay between emails to be respectful
		time.Sleep(100 * time.Millisecond)
	}
	
	// Update metrics
	p.metrics.ProcessedEmails.Add(int64(processed))
	p.metrics.SkippedEmails.Add(int64(skipped))
	p.metrics.ErrorEmails.Add(int64(errors))
	p.metrics.LastRun.Store(time.Now())
	
	duration := time.Since(startTime)
	p.metrics.AverageRunTime.Store(duration)
	
	p.logger.Info("Email processing run completed",
		"duration", duration,
		"processed", processed,
		"skipped", skipped,
		"errors", errors)
	
	// Cleanup old state entries
	if err := p.stateManager.Cleanup(time.Now().AddDate(0, 0, -30)); err != nil {
		p.logger.Warn("Failed to cleanup old state entries", "error", err)
	}
}

// searchEmails searches for new emails to process
func (p *EmailProcessor) searchEmails(ctx context.Context) ([]email.EmailMessage, error) {
	// Use configured search query or build default
	query := p.config.SearchQuery
	if query == "" {
		query = email.BuildSearchQuery(nil, p.config.SearchAfterDays, p.config.UnreadOnly, "")
	}
	
	emails, err := p.emailClient.Search(query)
	if err != nil {
		return nil, fmt.Errorf("email search failed: %w", err)
	}
	
	// Filter out already processed emails
	var newEmails []email.EmailMessage
	for _, emailMsg := range emails {
		processed, err := p.stateManager.IsProcessed(emailMsg.ID)
		if err != nil {
			p.logger.Warn("Failed to check if email was processed", "email_id", emailMsg.ID, "error", err)
			continue
		}
		
		if !processed {
			newEmails = append(newEmails, emailMsg)
		}
		
		// Respect max emails limit
		if len(newEmails) >= p.config.MaxEmailsPerRun {
			break
		}
	}
	
	return newEmails, nil
}

// processEmail processes a single email for tracking numbers
func (p *EmailProcessor) processEmail(emailMsg *email.EmailMessage) *ProcessingResult {
	logger := p.logger.With("email_id", emailMsg.ID, "from", emailMsg.From, "subject", emailMsg.Subject)
	
	result := &ProcessingResult{
		EmailID:     emailMsg.ID,
		StartTime:   time.Now(),
		Status:      "processing",
		Error:       "",
		Tracking:    nil,
	}
	
	// Convert to content format
	content := &email.EmailContent{
		PlainText: emailMsg.PlainText,
		HTMLText:  emailMsg.HTMLText,
		Subject:   emailMsg.Subject,
		From:      emailMsg.From,
		Headers:   emailMsg.Headers,
		MessageID: emailMsg.ID,
		ThreadID:  emailMsg.ThreadID,
		Date:      emailMsg.Date,
	}
	
	// Extract tracking numbers
	trackingInfo, err := p.extractor.Extract(content)
	if err != nil {
		logger.Error("Failed to extract tracking numbers", "error", err)
		result.Status = "error"
		result.Error = err.Error()
		p.markProcessed(emailMsg, result)
		return result
	}
	
	if len(trackingInfo) == 0 {
		logger.Debug("No tracking numbers found")
		result.Status = "skipped"
		result.Error = "no tracking numbers found"
		p.markProcessed(emailMsg, result)
		return result
	}
	
	logger.Info("Found tracking numbers", "count", len(trackingInfo))
	result.Tracking = trackingInfo
	
	// Create shipments if not in dry-run mode
	if !p.config.DryRun {
		created := 0
		for _, tracking := range trackingInfo {
			if err := p.createShipment(tracking); err != nil {
				logger.Warn("Failed to create shipment", 
					"tracking_number", tracking.Number,
					"carrier", tracking.Carrier,
					"error", err)
			} else {
				created++
				p.metrics.ShipmentsCreated.Add(1)
			}
		}
		
		logger.Info("Created shipments", "count", created, "total_tracking", len(trackingInfo))
	} else {
		logger.Info("Dry-run mode: would create shipments", "count", len(trackingInfo))
	}
	
	p.metrics.TrackingNumbers.Add(int64(len(trackingInfo)))
	result.Status = "processed"
	
	// Mark email as processed
	p.markProcessed(emailMsg, result)
	
	return result
}

// createShipment creates a shipment via the API
func (p *EmailProcessor) createShipment(tracking email.TrackingInfo) error {
	attempt := 0
	var lastErr error
	
	for attempt < p.config.RetryCount {
		err := p.apiClient.CreateShipment(tracking)
		if err == nil {
			return nil // Success
		}
		
		lastErr = err
		attempt++
		
		if attempt < p.config.RetryCount {
			p.logger.Warn("Shipment creation failed, retrying",
				"tracking_number", tracking.Number,
				"attempt", attempt,
				"error", err)
			time.Sleep(p.config.RetryDelay)
		}
	}
	
	return fmt.Errorf("failed to create shipment after %d attempts: %w", p.config.RetryCount, lastErr)
}

// markProcessed marks an email as processed in state storage
func (p *EmailProcessor) markProcessed(emailMsg *email.EmailMessage, result *ProcessingResult) {
	var trackingNumbers []string
	for _, tracking := range result.Tracking {
		trackingNumbers = append(trackingNumbers, tracking.Number)
	}
	
	entry := &email.StateEntry{
		GmailMessageID:  emailMsg.ID,
		GmailThreadID:   emailMsg.ThreadID,
		ProcessedAt:     time.Now(),
		TrackingNumbers: fmt.Sprintf("%v", trackingNumbers), // Simple JSON encoding
		Status:          result.Status,
		Sender:          emailMsg.From,
		Subject:         emailMsg.Subject,
		ErrorMessage:    result.Error,
	}
	
	if err := p.stateManager.MarkProcessed(entry); err != nil {
		p.logger.Error("Failed to mark email as processed", 
			"email_id", emailMsg.ID, 
			"error", err)
	}
}

// ProcessingResult represents the result of processing a single email
type ProcessingResult struct {
	EmailID   string                `json:"email_id"`
	StartTime time.Time            `json:"start_time"`
	Status    string               `json:"status"` // "processed", "skipped", "error"
	Error     string               `json:"error,omitempty"`
	Tracking  []email.TrackingInfo `json:"tracking,omitempty"`
}