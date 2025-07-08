package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"package-tracking/internal/carriers"
	"package-tracking/internal/database"
	"package-tracking/internal/email"
)

// TwoPhaseEmailProcessor implements a two-phase email processing approach:
// Phase 1: Fetch metadata only and score for relevance
// Phase 2: Fetch full content only for relevant emails and process tracking numbers
type TwoPhaseEmailProcessor struct {
	config           *TwoPhaseEmailProcessorConfig
	emailClient      TwoPhaseEmailClient
	extractor        TrackingExtractor
	emailStore       *database.EmailStore
	shipmentStore    *database.ShipmentStore
	apiClient        APIClient
	logger           *slog.Logger
	metrics          *TwoPhaseProcessingMetrics
	factory          CarrierFactory
	cacheManager     CacheManager
	rateLimiter      RateLimiter
	relevanceScorer  *RelevanceScorer
}

// TwoPhaseEmailClient extends the basic email client with metadata-only methods
type TwoPhaseEmailClient interface {
	// Basic email methods
	GetMessage(id string) (*email.EmailMessage, error)
	HealthCheck() error
	Close() error
	
	// Two-phase specific methods
	GetMessageMetadata(id string) (*email.EmailMessage, error)
	GetMessagesSinceMetadataOnly(since time.Time) ([]email.EmailMessage, error)
}

// TwoPhaseEmailProcessorConfig holds configuration for two-phase processing
type TwoPhaseEmailProcessorConfig struct {
	// Phase 1 configuration
	ScanDays              int     `json:"scan_days"`
	MaxEmailsPerScan      int     `json:"max_emails_per_scan"`
	RelevanceThreshold    float64 `json:"relevance_threshold"`
	MetadataOnlyBatchSize int     `json:"metadata_only_batch_size"`
	
	// Phase 2 configuration
	ContentBatchSize       int `json:"content_batch_size"`
	MaxContentExtractions  int `json:"max_content_extractions"`
	BodyStorageEnabled     bool `json:"body_storage_enabled"`
	
	// General configuration
	DryRun       bool          `json:"dry_run"`
	RetryCount   int           `json:"retry_count"`
	RetryDelay   time.Duration `json:"retry_delay"`
	RetentionDays int          `json:"retention_days"`
}

// TwoPhaseProcessingMetrics tracks metrics for two-phase processing
type TwoPhaseProcessingMetrics struct {
	// Phase 1 metrics
	MetadataEmailsScanned   int64     `json:"metadata_emails_scanned"`
	MetadataEmailsStored    int64     `json:"metadata_emails_stored"`
	MetadataEmailsFiltered  int64     `json:"metadata_emails_filtered"`
	LastMetadataScanTime    time.Time `json:"last_metadata_scan_time"`
	
	// Phase 2 metrics
	ContentEmailsProcessed  int64     `json:"content_emails_processed"`
	ContentEmailsWithTracking int64   `json:"content_emails_with_tracking"`
	ShipmentsCreated        int64     `json:"shipments_created"`
	LastContentScanTime     time.Time `json:"last_content_scan_time"`
	
	// Overall metrics
	TotalScanDuration       time.Duration `json:"total_scan_duration"`
	ProcessingErrors        int64         `json:"processing_errors"`
}

// NewTwoPhaseEmailProcessor creates a new two-phase email processor
func NewTwoPhaseEmailProcessor(
	config *TwoPhaseEmailProcessorConfig,
	emailClient TwoPhaseEmailClient,
	extractor TrackingExtractor,
	emailStore *database.EmailStore,
	shipmentStore *database.ShipmentStore,
	apiClient APIClient,
	logger *slog.Logger,
	factory CarrierFactory,
	cacheManager CacheManager,
	rateLimiter RateLimiter,
) *TwoPhaseEmailProcessor {
	
	return &TwoPhaseEmailProcessor{
		config:          config,
		emailClient:     emailClient,
		extractor:       extractor,
		emailStore:      emailStore,
		shipmentStore:   shipmentStore,
		apiClient:       apiClient,
		logger:          logger,
		metrics:         &TwoPhaseProcessingMetrics{},
		factory:         factory,
		cacheManager:    cacheManager,
		rateLimiter:     rateLimiter,
		relevanceScorer: NewRelevanceScorer(),
	}
}

// ProcessEmailsSince performs two-phase processing of emails since the specified time
func (p *TwoPhaseEmailProcessor) ProcessEmailsSince(since time.Time) error {
	startTime := time.Now()
	p.logger.Info("Starting two-phase email processing",
		"since", since,
		"relevance_threshold", p.config.RelevanceThreshold,
		"max_emails", p.config.MaxEmailsPerScan)
	
	// Phase 1: Process metadata only
	if err := p.processPhase1MetadataOnly(since); err != nil {
		return fmt.Errorf("phase 1 (metadata) failed: %w", err)
	}
	
	// Phase 2: Process content for relevant emails
	if err := p.processPhase2ContentExtraction(); err != nil {
		return fmt.Errorf("phase 2 (content) failed: %w", err)
	}
	
	// Update overall metrics
	p.metrics.TotalScanDuration = time.Since(startTime)
	
	p.logger.Info("Two-phase email processing completed",
		"duration", p.metrics.TotalScanDuration,
		"metadata_scanned", p.metrics.MetadataEmailsScanned,
		"content_processed", p.metrics.ContentEmailsProcessed,
		"shipments_created", p.metrics.ShipmentsCreated)
	
	return nil
}

// processPhase1MetadataOnly fetches and scores emails using metadata only
func (p *TwoPhaseEmailProcessor) processPhase1MetadataOnly(since time.Time) error {
	p.logger.Info("Phase 1: Starting metadata-only processing")
	p.metrics.LastMetadataScanTime = time.Now()
	
	// Get emails with metadata only
	messages, err := p.emailClient.GetMessagesSinceMetadataOnly(since)
	if err != nil {
		return fmt.Errorf("failed to get metadata-only messages: %w", err)
	}
	
	p.logger.Info("Retrieved messages for metadata processing", "count", len(messages))
	p.metrics.MetadataEmailsScanned = int64(len(messages))
	
	processed := 0
	filtered := 0
	
	for i, msg := range messages {
		// Respect max emails limit
		if p.config.MaxEmailsPerScan > 0 && i >= p.config.MaxEmailsPerScan {
			p.logger.Info("Reached max emails per scan limit", "limit", p.config.MaxEmailsPerScan)
			break
		}
		
		// Check if already processed
		existing, err := p.emailStore.GetByGmailMessageID(msg.ID)
		if err == nil && existing != nil {
			// Email already exists, skip
			continue
		}
		
		// Calculate relevance score
		relevanceScore := p.relevanceScorer.CalculateRelevanceScore(&msg)
		
		p.logger.Debug("Calculated relevance score",
			"email_id", msg.ID,
			"from", msg.From,
			"subject", msg.Subject,
			"score", relevanceScore)
		
		// Create metadata entry
		emailEntry := &database.EmailBodyEntry{
			GmailMessageID:       msg.ID,
			GmailThreadID:        msg.ThreadID,
			From:                 msg.From,
			Subject:              msg.Subject,
			Date:                 msg.Date,
			Snippet:              msg.Snippet,
			InternalTimestamp:    msg.InternalDate,
			ScanMethod:           "two-phase",
			ProcessedAt:          time.Now(),
			Status:               "metadata_extracted",
			ProcessingPhase:      "metadata_only",
			RelevanceScore:       relevanceScore,
			HasContent:           false,
		}
		
		// Store metadata entry
		if err := p.emailStore.CreateMetadataEntry(emailEntry); err != nil {
			p.logger.Error("Failed to store metadata entry",
				"email_id", msg.ID,
				"error", err)
			p.metrics.ProcessingErrors++
			continue
		}
		
		processed++
		
		// Track filtering
		if relevanceScore < p.config.RelevanceThreshold {
			filtered++
		}
		
		// Small delay between processing
		time.Sleep(50 * time.Millisecond)
	}
	
	p.metrics.MetadataEmailsStored = int64(processed)
	p.metrics.MetadataEmailsFiltered = int64(filtered)
	
	p.logger.Info("Phase 1 completed",
		"processed", processed,
		"filtered_out", filtered,
		"threshold", p.config.RelevanceThreshold)
	
	return nil
}

// processPhase2ContentExtraction processes emails that passed relevance filtering
func (p *TwoPhaseEmailProcessor) processPhase2ContentExtraction() error {
	p.logger.Info("Phase 2: Starting content extraction for relevant emails")
	p.metrics.LastContentScanTime = time.Now()
	
	// Get emails that need content extraction (above relevance threshold)
	candidateEmails, err := p.emailStore.GetEmailsByRelevanceScore(
		p.config.RelevanceThreshold,
		p.config.MaxContentExtractions,
	)
	if err != nil {
		return fmt.Errorf("failed to get candidate emails: %w", err)
	}
	
	p.logger.Info("Found candidate emails for content extraction",
		"count", len(candidateEmails),
		"threshold", p.config.RelevanceThreshold)
	
	processed := 0
	withTracking := 0
	
	for _, emailEntry := range candidateEmails {
		// Skip if already has content
		if emailEntry.HasContent {
			continue
		}
		
		p.logger.Debug("Processing email for content extraction",
			"email_id", emailEntry.GmailMessageID,
			"relevance_score", emailEntry.RelevanceScore)
		
		// Get full email content
		fullMessage, err := p.emailClient.GetMessage(emailEntry.GmailMessageID)
		if err != nil {
			p.logger.Error("Failed to get full message content",
				"email_id", emailEntry.GmailMessageID,
				"error", err)
			p.metrics.ProcessingErrors++
			continue
		}
		
		// Update email store with content
		var compressed []byte
		if len(fullMessage.PlainText) > 1000 {
			compressed, err = database.CompressEmailBody(fullMessage.PlainText)
			if err != nil {
				p.logger.Warn("Failed to compress email body", "error", err)
			}
		}
		
		if err := p.emailStore.UpdateWithContent(
			emailEntry.GmailMessageID,
			fullMessage.PlainText,
			fullMessage.HTMLText,
			compressed,
		); err != nil {
			p.logger.Error("Failed to update email with content",
				"email_id", emailEntry.GmailMessageID,
				"error", err)
			p.metrics.ProcessingErrors++
			continue
		}
		
		// Extract tracking numbers
		content := &email.EmailContent{
			PlainText: fullMessage.PlainText,
			HTMLText:  fullMessage.HTMLText,
			Subject:   fullMessage.Subject,
			From:      fullMessage.From,
			Headers:   fullMessage.Headers,
			MessageID: fullMessage.ID,
			ThreadID:  fullMessage.ThreadID,
			Date:      fullMessage.Date,
		}
		
		trackingInfo, err := p.extractor.Extract(content)
		if err != nil {
			p.logger.Error("Failed to extract tracking numbers",
				"email_id", emailEntry.GmailMessageID,
				"error", err)
			p.metrics.ProcessingErrors++
			continue
		}
		
		// Process tracking numbers if found
		if len(trackingInfo) > 0 {
			p.logger.Info("Found tracking numbers",
				"email_id", emailEntry.GmailMessageID,
				"count", len(trackingInfo))
			
			// Create shipments for valid tracking numbers
			successfulTracking := []email.TrackingInfo{}
			for _, tracking := range trackingInfo {
				if err := p.createShipment(tracking); err != nil {
					p.logger.Error("Failed to create shipment",
						"tracking_number", tracking.Number,
						"error", err)
				} else {
					successfulTracking = append(successfulTracking, tracking)
				}
			}
			
			if len(successfulTracking) > 0 {
				withTracking++
				p.metrics.ShipmentsCreated += int64(len(successfulTracking))
				
				// Update tracking numbers in email record
				trackingJSON, _ := json.Marshal(successfulTracking)
				emailEntry.TrackingNumbers = string(trackingJSON)
				emailEntry.Status = "processed_with_tracking"
			}
		}
		
		processed++
		
		// Rate limiting between content extractions
		time.Sleep(200 * time.Millisecond)
	}
	
	p.metrics.ContentEmailsProcessed = int64(processed)
	p.metrics.ContentEmailsWithTracking = int64(withTracking)
	
	p.logger.Info("Phase 2 completed",
		"processed", processed,
		"with_tracking", withTracking,
		"shipments_created", p.metrics.ShipmentsCreated)
	
	return nil
}

// createShipment creates a shipment via the API client (reused from original processor)
func (p *TwoPhaseEmailProcessor) createShipment(tracking email.TrackingInfo) error {
	if p.config.DryRun {
		p.logger.Info("Dry run: would create shipment",
			"tracking_number", tracking.Number,
			"carrier", tracking.Carrier)
		return nil
	}
	
	// Validate tracking number before creating shipment
	ctx := context.Background()
	validationResult, err := p.validateTracking(ctx, tracking.Number, tracking.Carrier)
	if err != nil || !validationResult.IsValid {
		p.logger.WarnContext(ctx, "Tracking validation failed",
			"tracking_number", tracking.Number,
			"carrier", tracking.Carrier,
			"error", err)
		return fmt.Errorf("tracking validation failed: %w", err)
	}
	
	if p.apiClient == nil {
		return fmt.Errorf("no API client configured")
	}
	
	attempt := 0
	var lastErr error
	
	for attempt < p.config.RetryCount {
		err := p.apiClient.CreateShipment(tracking)
		if err == nil {
			return nil
		}
		
		lastErr = err
		attempt++
		
		if attempt < p.config.RetryCount {
			time.Sleep(p.config.RetryDelay)
		}
	}
	
	return fmt.Errorf("failed to create shipment after %d attempts: %w", p.config.RetryCount, lastErr)
}

// validateTracking validates a tracking number (reused from original processor)
func (p *TwoPhaseEmailProcessor) validateTracking(ctx context.Context, trackingNumber, carrier string) (*ValidationResult, error) {
	// Create cache key
	cacheKey := fmt.Sprintf("validation_%s_%s", carrier, trackingNumber)
	
	// Check cache first
	if p.cacheManager != nil && p.cacheManager.IsEnabled() {
		if cached, err := p.cacheManager.Get(cacheKey); err == nil && cached != nil {
			p.logger.InfoContext(ctx, "Using cached validation result",
				"tracking_number", trackingNumber,
				"carrier", carrier,
				"cache_key", cacheKey)
			
			return &ValidationResult{
				IsValid:        len(cached.Events) > 0,
				TrackingEvents: cached.Events,
				Error:          nil,
			}, nil
		}
	}
	
	// Check rate limiting
	if p.rateLimiter != nil {
		rateLimitResult := p.rateLimiter.CheckValidationRateLimit(trackingNumber)
		if rateLimitResult.ShouldBlock {
			return &ValidationResult{
				IsValid: false,
				Error:   fmt.Errorf("rate limited: %s", rateLimitResult.Reason),
			}, fmt.Errorf("validation rate limited")
		}
	}
	
	// Create carrier client for validation
	client, _, err := p.factory.CreateClient(carrier)
	if err != nil {
		p.logger.WarnContext(ctx, "Failed to create carrier client for validation",
			"tracking_number", trackingNumber,
			"carrier", carrier,
			"error", err)
		return &ValidationResult{
			IsValid: false,
			Error:   err,
		}, err
	}
	
	// Perform validation
	req := &carriers.TrackingRequest{
		TrackingNumbers: []string{trackingNumber},
		Carrier:         carrier,
	}
	
	resp, err := client.Track(ctx, req)
	if err != nil {
		p.logger.WarnContext(ctx, "Tracking validation failed",
			"tracking_number", trackingNumber,
			"carrier", carrier,
			"error", err)
		return &ValidationResult{
			IsValid: false,
			Error:   err,
		}, err
	}
	
	// Process response
	if len(resp.Results) == 0 {
		return &ValidationResult{
			IsValid: false,
			Error:   fmt.Errorf("no tracking results returned"),
		}, fmt.Errorf("no tracking results returned")
	}
	
	// Convert carrier events to database events
	trackingInfo := resp.Results[0]
	events := make([]database.TrackingEvent, 0, len(trackingInfo.Events))
	
	for _, event := range trackingInfo.Events {
		dbEvent := database.TrackingEvent{
			ShipmentID:  -1, // Validation context
			Timestamp:   event.Timestamp,
			Location:    event.Location,
			Status:      string(event.Status),
			Description: event.Description,
		}
		if event.Details != "" {
			if dbEvent.Description != "" {
				dbEvent.Description += " - " + event.Details
			} else {
				dbEvent.Description = event.Details
			}
		}
		events = append(events, dbEvent)
	}
	
	// Cache the successful validation result
	if p.cacheManager != nil && p.cacheManager.IsEnabled() {
		validationResponse := &database.RefreshResponse{
			ShipmentID:  -1,
			UpdatedAt:   time.Now(),
			EventsAdded: len(events),
			TotalEvents: len(events),
			Events:      events,
		}
		
		if err := p.cacheManager.Set(cacheKey, validationResponse); err != nil {
			p.logger.WarnContext(ctx, "Failed to cache validation response", "error", err)
		}
	}
	
	return &ValidationResult{
		IsValid:        true,
		TrackingEvents: events,
		Error:          nil,
	}, nil
}

// GetMetrics returns the current processing metrics
func (p *TwoPhaseEmailProcessor) GetMetrics() *TwoPhaseProcessingMetrics {
	return p.metrics
}

// GetRelevanceScorer returns the relevance scorer for testing/analysis
func (p *TwoPhaseEmailProcessor) GetRelevanceScorer() *RelevanceScorer {
	return p.relevanceScorer
}