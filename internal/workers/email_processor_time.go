package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"package-tracking/internal/carriers"
	"package-tracking/internal/database"
	"package-tracking/internal/email"
)

// TrackingExtractor interface for extracting tracking information from emails
type TrackingExtractor interface {
	Extract(content *email.EmailContent) ([]email.TrackingInfo, error)
}

// TimeBasedEmailProcessor handles time-based email scanning with body storage
type TimeBasedEmailProcessor struct {
	config        *TimeBasedEmailProcessorConfig
	emailClient   TimeBasedEmailClient
	extractor     TrackingExtractor
	stateManager  StateManager
	emailStore    *database.EmailStore  // Optional: for storing email bodies with valid tracking
	shipmentStore *database.ShipmentStore
	apiClient     APIClient
	logger        *slog.Logger
	metrics       *TimeBasedProcessingMetrics
	factory       CarrierFactory // For validation
	cacheManager  CacheManager   // For validation caching
	rateLimiter   RateLimiter    // For validation rate limiting
}

// CacheManager interface for caching validation results
type CacheManager interface {
	Get(key interface{}) (*database.RefreshResponse, error)
	Set(key interface{}, response *database.RefreshResponse) error
	IsEnabled() bool
}

// RateLimiter interface for rate limiting validation requests
type RateLimiter interface {
	CheckValidationRateLimit(trackingNumber string) RateLimitResult
}

// RateLimitResult contains rate limiting information
type RateLimitResult struct {
	ShouldBlock   bool
	RemainingTime time.Duration
	Reason        string
	Allowed       bool   // For backward compatibility
	Message       string // For backward compatibility
}

// CarrierFactory interface for creating carrier clients
type CarrierFactory interface {
	CreateClient(carrier string) (carriers.Client, carriers.ClientType, error)
	SetCarrierConfig(carrier string, config *carriers.CarrierConfig)
}

// ValidationResult represents the result of tracking number validation
type ValidationResult struct {
	IsValid        bool                     `json:"is_valid"`
	TrackingEvents []database.TrackingEvent `json:"tracking_events"`
	Error          error                    `json:"error,omitempty"`
}

// TimeBasedEmailProcessorConfig configures the time-based email processor
type TimeBasedEmailProcessorConfig struct {
	ScanDays           int           `json:"scan_days"`
	BodyStorageEnabled bool          `json:"body_storage_enabled"`
	RetentionDays      int           `json:"retention_days"`
	MaxEmailsPerScan   int           `json:"max_emails_per_scan"`
	UnreadOnly         bool          `json:"unread_only"`
	CheckInterval      time.Duration `json:"check_interval"`
	ProcessingTimeout  time.Duration `json:"processing_timeout"`
	ValidationTimeout  time.Duration `json:"validation_timeout"` // Configurable timeout for validation
	RetryCount         int           `json:"retry_count"`
	RetryDelay         time.Duration `json:"retry_delay"`
	DryRun             bool          `json:"dry_run"`
}

// TimeBasedEmailClient defines the interface for time-based email scanning
type TimeBasedEmailClient interface {
	GetMessagesSince(since time.Time) ([]email.EmailMessage, error)
	GetEnhancedMessage(id string) (*email.EmailMessage, error)
	GetThreadMessages(threadID string) ([]email.EmailMessage, error)
	PerformRetroactiveScan(days int) ([]email.EmailMessage, error)
	HealthCheck() error
	Close() error
}

// TimeBasedProcessingMetrics tracks time-based processing statistics
type TimeBasedProcessingMetrics struct {
	mu                      sync.RWMutex
	TotalScans              int64     `json:"total_scans"`
	TotalEmailsScanned      int64     `json:"total_emails_scanned"`
	EmailsWithBodiesStored  int64     `json:"emails_with_bodies_stored"`
	ThreadsCreated          int64     `json:"threads_created"`
	AutomaticLinksCreated   int64     `json:"automatic_links_created"`
	ShipmentsCreated        int64     `json:"shipments_created"`
	LastScanTime            time.Time `json:"last_scan_time"`
	LastRetroactiveScanTime time.Time `json:"last_retroactive_scan_time"`
	AverageScanDuration     time.Duration `json:"average_scan_duration"`
}

// NewTimeBasedEmailProcessor creates a new time-based email processor
func NewTimeBasedEmailProcessor(
	config *TimeBasedEmailProcessorConfig,
	emailClient TimeBasedEmailClient,
	extractor TrackingExtractor,
	stateManager StateManager,
	emailStore *database.EmailStore,
	shipmentStore *database.ShipmentStore,
	apiClient APIClient,
	logger *slog.Logger,
) *TimeBasedEmailProcessor {
	return &TimeBasedEmailProcessor{
		config:        config,
		emailClient:   emailClient,
		extractor:     extractor,
		stateManager:  stateManager,
		emailStore:    emailStore,
		shipmentStore: shipmentStore,
		apiClient:     apiClient,
		logger:        logger,
		metrics:       &TimeBasedProcessingMetrics{},
		factory:       nil, // Will be set separately if validation is needed
		cacheManager:  nil, // Will be set separately if caching is needed
		rateLimiter:   nil, // Will be set separately if rate limiting is needed
	}
}

// validateTracking validates a tracking number by performing a carrier lookup
// This method integrates with the existing refresh system for caching and rate limiting
func (p *TimeBasedEmailProcessor) validateTracking(ctx context.Context, trackingNumber, carrier string) (*ValidationResult, error) {
	// Check if factory is available for validation
	if p.factory == nil {
		return &ValidationResult{
			IsValid: false,
			Error:   fmt.Errorf("carrier factory not configured for validation"),
		}, fmt.Errorf("carrier factory not configured")
	}

	// FR2: Cache Integration - Check cache first if enabled
	// Include carrier in cache key to prevent collisions between carriers with similar tracking number formats
	cacheKey := fmt.Sprintf("validation:%s:%s", carrier, trackingNumber)
	if p.cacheManager != nil && p.cacheManager.IsEnabled() {
		if cachedResponse, err := p.cacheManager.Get(cacheKey); err == nil && cachedResponse != nil {
			p.logger.InfoContext(ctx, "Serving cached validation response",
				"tracking_number", trackingNumber,
				"carrier", carrier,
				"cache_key", cacheKey)
			
			return &ValidationResult{
				IsValid:        true,
				TrackingEvents: cachedResponse.Events,
				Error:          nil,
			}, nil
		}
	}

	// FR3: Rate Limiting Integration - Check rate limits
	if p.rateLimiter != nil {
		rateLimitResult := p.rateLimiter.CheckValidationRateLimit(trackingNumber)
		if rateLimitResult.ShouldBlock {
			return &ValidationResult{
				IsValid: false,
				Error:   fmt.Errorf("rate limit exceeded: %s", rateLimitResult.Reason),
			}, fmt.Errorf("rate limit exceeded for tracking %s: %s", trackingNumber, rateLimitResult.Reason)
		}
	}

	// Create carrier client
	client, _, err := p.factory.CreateClient(carrier)
	if err != nil {
		return &ValidationResult{
			IsValid: false,
			Error:   err,
		}, fmt.Errorf("failed to create carrier client: %w", err)
	}

	// Create tracking request
	req := &carriers.TrackingRequest{
		TrackingNumbers: []string{trackingNumber},
		Carrier:         carrier,
	}

	// Perform the tracking call with configurable timeout
	validationTimeout := 120 * time.Second // Default timeout
	if p.config.ValidationTimeout > 0 {
		validationTimeout = p.config.ValidationTimeout
	}
	trackingCtx, cancel := context.WithTimeout(ctx, validationTimeout)
	defer cancel()

	resp, err := client.Track(trackingCtx, req)
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

	// Convert carrier events to database events for compatibility
	trackingInfo := resp.Results[0]
	// Pre-allocate slice for better memory efficiency
	events := make([]database.TrackingEvent, 0, len(trackingInfo.Events))
	
	for _, event := range trackingInfo.Events {
		dbEvent := database.TrackingEvent{
			ShipmentID:  -1, // Use -1 to indicate validation context (not associated with shipment yet)
			Timestamp:   event.Timestamp,
			Location:    event.Location,
			Status:      string(event.Status),
			Description: event.Description,
			// Note: database.TrackingEvent doesn't have Details field, combining with Description
		}
		// If there are details, append them to the description
		if event.Details != "" {
			if dbEvent.Description != "" {
				dbEvent.Description += " - " + event.Details
			} else {
				dbEvent.Description = event.Details
			}
		}
		events = append(events, dbEvent)
	}

	// FR2: Cache the successful validation result
	if p.cacheManager != nil && p.cacheManager.IsEnabled() {
		validationResponse := &database.RefreshResponse{
			ShipmentID:  -1, // Use -1 to indicate validation context
			UpdatedAt:   time.Now(),
			EventsAdded: len(events),
			TotalEvents: len(events),
			Events:      events,
		}
		
		if err := p.cacheManager.Set(cacheKey, validationResponse); err != nil {
			p.logger.WarnContext(ctx, "Failed to cache validation response",
				"tracking_number", trackingNumber,
				"carrier", carrier,
				"cache_key", cacheKey,
				"error", err)
			// Continue anyway - caching failure shouldn't break validation
		} else {
			p.logger.InfoContext(ctx, "Cached validation response",
				"tracking_number", trackingNumber,
				"carrier", carrier,
				"cache_key", cacheKey,
				"events_cached", len(events))
		}
	}

	return &ValidationResult{
		IsValid:        true,
		TrackingEvents: events,
		Error:          nil,
	}, nil
}

// truncateForLogging truncates a string for safe logging
func truncateForLogging(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}

// ProcessEmailsSince processes all emails since the specified time using time-based scanning
func (p *TimeBasedEmailProcessor) ProcessEmailsSince(since time.Time) error {
	startTime := time.Now()
	p.metrics.incrementTotalScans()

	p.logger.Info("Starting time-based email processing",
		"since", since,
		"scan_days", p.config.ScanDays,
		"body_storage_enabled", p.config.BodyStorageEnabled,
		"max_emails", p.config.MaxEmailsPerScan)

	// Get all messages since the specified time
	messages, err := p.emailClient.GetMessagesSince(since)
	if err != nil {
		return fmt.Errorf("failed to get messages since %v: %w", since, err)
	}

	p.logger.Info("Retrieved messages for time-based processing",
		"count", len(messages),
		"since", since)

	p.metrics.addEmailsScanned(int64(len(messages)))

	// Process each message
	processed := 0
	skipped := 0
	errors := 0

	for i, msg := range messages {
		// Respect max emails limit
		if p.config.MaxEmailsPerScan > 0 && i >= p.config.MaxEmailsPerScan {
			p.logger.Info("Reached max emails per scan limit", "limit", p.config.MaxEmailsPerScan)
			break
		}

		// Check if already processed
		alreadyProcessed, err := p.stateManager.IsProcessed(msg.ID)
		if err != nil {
			p.logger.Warn("Failed to check if email is processed", "email_id", msg.ID, "error", err)
			errors++
			continue
		}

		if alreadyProcessed {
			skipped++
			continue
		}

		// Process the individual email
		if err := p.processIndividualEmail(&msg); err != nil {
			p.logger.Error("Failed to process individual email",
				"email_id", msg.ID,
				"from", msg.From,
				"subject", msg.Subject,
				"error", err)
			errors++
			continue
		}

		processed++

		// Small delay between processing to be respectful to APIs
		time.Sleep(100 * time.Millisecond)
	}

	// Update metrics
	duration := time.Since(startTime)
	p.metrics.updateScanMetrics(duration)

	p.logger.Info("Time-based email processing completed",
		"duration", duration,
		"processed", processed,
		"skipped", skipped,
		"errors", errors,
		"total_messages", len(messages))

	// Cleanup old email state if retention is configured
	if p.config.RetentionDays > 0 {
		cleanupTime := time.Now().AddDate(0, 0, -p.config.RetentionDays)
		if err := p.stateManager.Cleanup(cleanupTime); err != nil {
			p.logger.Warn("Failed to cleanup old email state", "error", err)
		}
	}

	return nil
}

// PerformRetroactiveScan performs a full retroactive scan for the configured number of days
func (p *TimeBasedEmailProcessor) PerformRetroactiveScan() error {
	p.logger.Info("Starting retroactive scan", "days", p.config.ScanDays)

	messages, err := p.emailClient.PerformRetroactiveScan(p.config.ScanDays)
	if err != nil {
		return fmt.Errorf("retroactive scan failed: %w", err)
	}

	p.logger.Info("Retroactive scan retrieved messages", "count", len(messages))

	p.metrics.updateRetroactiveScanTime()
	p.metrics.addEmailsScanned(int64(len(messages)))

	// Process all retrieved messages
	for _, msg := range messages {
		// Check if already processed
		alreadyProcessed, err := p.stateManager.IsProcessed(msg.ID)
		if err != nil {
			p.logger.Warn("Failed to check if email is processed during retroactive scan",
				"email_id", msg.ID, "error", err)
			continue
		}

		if alreadyProcessed {
			continue
		}

		// Process the email
		if err := p.processIndividualEmail(&msg); err != nil {
			p.logger.Error("Failed to process email during retroactive scan",
				"email_id", msg.ID, "error", err)
			continue
		}

		// Small delay between processing
		time.Sleep(50 * time.Millisecond)
	}

	p.logger.Info("Retroactive scan completed", "total_messages", len(messages))
	return nil
}

// processIndividualEmail processes a single email with time-based workflow
func (p *TimeBasedEmailProcessor) processIndividualEmail(msg *email.EmailMessage) error {
	logger := p.logger.With("email_id", msg.ID, "from", msg.From, "subject", msg.Subject)

	// Convert to state entry format for storage
	stateEntry := &email.StateEntry{
		GmailMessageID: msg.ID,
		GmailThreadID:  msg.ThreadID,
		Sender:         msg.From,
		Subject:        msg.Subject,
		ProcessedAt:    time.Now(),
		Status:         "processing",
	}

	// Extract tracking numbers
	content := &email.EmailContent{
		PlainText: msg.PlainText,
		HTMLText:  msg.HTMLText,
		Subject:   msg.Subject,
		From:      msg.From,
		Headers:   msg.Headers,
		MessageID: msg.ID,
		ThreadID:  msg.ThreadID,
		Date:      msg.Date,
	}

	trackingInfo, err := p.extractor.Extract(content)
	if err != nil {
		logger.Error("Failed to extract tracking numbers", "error", err)
		stateEntry.Status = "error"
		stateEntry.ErrorMessage = err.Error()
	} else {
		// Store tracking numbers found
		if len(trackingInfo) > 0 {
			trackingJSON, _ := json.Marshal(trackingInfo)
			stateEntry.TrackingNumbers = string(trackingJSON)
			stateEntry.Status = "processed"

			logger.Info("Found tracking numbers", "count", len(trackingInfo))

			// Create shipments via API and store email body if successful
			successfulTrackingNumbers := []email.TrackingInfo{}
			for _, tracking := range trackingInfo {
				if err := p.createShipment(tracking); err != nil {
					logger.Error("Failed to create shipment", "tracking_number", tracking.Number, "error", err)
				} else {
					successfulTrackingNumbers = append(successfulTrackingNumbers, tracking)
				}
			}
			
			// Store email body only if we successfully created shipments and email store is available
			if len(successfulTrackingNumbers) > 0 && p.emailStore != nil && p.config.BodyStorageEnabled {
				if err := p.storeEmailBodyWithTracking(msg, successfulTrackingNumbers); err != nil {
					logger.Warn("Failed to store email body", "error", err)
					// Don't fail the entire process for email body storage issues
				}
			}
		} else {
			stateEntry.Status = "processed"
			logger.Debug("No tracking numbers found")
		}
	}

	// Store the email state
	if err := p.stateManager.MarkProcessed(stateEntry); err != nil {
		return fmt.Errorf("failed to store email: %w", err)
	}

	return nil
}


// createShipment creates a shipment via the API client
func (p *TimeBasedEmailProcessor) createShipment(tracking email.TrackingInfo) error {
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

	p.logger.InfoContext(ctx, "Tracking number validated successfully",
		"tracking_number", tracking.Number,
		"carrier", tracking.Carrier,
		"events_found", len(validationResult.TrackingEvents))

	if p.apiClient == nil {
		return fmt.Errorf("no API client configured")
	}

	attempt := 0
	var lastErr error

	for attempt < p.config.RetryCount {
		err := p.apiClient.CreateShipment(tracking)
		if err == nil {
			p.metrics.incrementShipmentsCreated()
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

// storeEmailBodyWithTracking stores the email body for emails with valid tracking numbers
func (p *TimeBasedEmailProcessor) storeEmailBodyWithTracking(msg *email.EmailMessage, trackingNumbers []email.TrackingInfo) error {
	if p.emailStore == nil {
		return fmt.Errorf("email store not available")
	}

	// Convert to database format for storage
	emailEntry := &database.EmailBodyEntry{
		GmailMessageID:    msg.ID,
		GmailThreadID:     msg.ThreadID,
		From:              msg.From,
		Subject:           msg.Subject,
		Date:              msg.Date,
		BodyText:          msg.PlainText,
		BodyHTML:          msg.HTMLText,
		InternalTimestamp: time.Now(),
		ScanMethod:        "time-based",
		ProcessedAt:       time.Now(),
		Status:            "processed",
	}

	// Store tracking numbers found
	trackingJSON, err := json.Marshal(trackingNumbers)
	if err != nil {
		return fmt.Errorf("failed to marshal tracking numbers: %w", err)
	}
	emailEntry.TrackingNumbers = string(trackingJSON)

	// Compress body if it's large to save space
	if len(msg.PlainText) > 1000 { // Compress if larger than 1KB
		compressed, err := database.CompressEmailBody(msg.PlainText)
		if err != nil {
			p.logger.Warn("Failed to compress email body", "error", err)
		} else {
			emailEntry.BodyCompressed = compressed
			// Clear uncompressed text to save space
			emailEntry.BodyText = ""
		}
	}

	// Store the email body in the main database
	if err := p.emailStore.CreateOrUpdate(emailEntry); err != nil {
		return fmt.Errorf("failed to store email body: %w", err)
	}

	p.logger.Info("Stored email body for shipment context",
		"email_id", msg.ID,
		"tracking_count", len(trackingNumbers),
		"compressed", len(emailEntry.BodyCompressed) > 0)

	// Link email to shipments for easy retrieval
	// Note: Linking is temporarily disabled until GetByTrackingNumber is implemented
	for _, tracking := range trackingNumbers {
		p.logger.Debug("Would link email to shipment",
			"email_id", emailEntry.ID,
			"tracking_number", tracking.Number)
		// TODO: Implement proper linking when GetByTrackingNumber is available
	}

	return nil
}

// linkEmailToShipment links an email to a shipment by tracking number
func (p *TimeBasedEmailProcessor) linkEmailToShipment(emailID int, trackingNumber string) error {
	if p.shipmentStore == nil {
		return fmt.Errorf("shipment store not available")
	}

	// Find the shipment by tracking number using direct SQL query
	// Since GetByTrackingNumber doesn't exist, we'll query directly
	shipmentID, err := p.findShipmentIDByTrackingNumber(trackingNumber)
	if err != nil {
		return fmt.Errorf("failed to find shipment with tracking number %s: %w", trackingNumber, err)
	}

	// Create the email-shipment link
	if err := p.emailStore.LinkEmailToShipment(emailID, shipmentID, "automatic", trackingNumber, "email-tracker"); err != nil {
		return fmt.Errorf("failed to create email-shipment link: %w", err)
	}

	p.logger.Debug("Linked email to shipment",
		"email_id", emailID,
		"shipment_id", shipmentID,
		"tracking_number", trackingNumber)

	return nil
}

// findShipmentIDByTrackingNumber finds a shipment ID by tracking number
func (p *TimeBasedEmailProcessor) findShipmentIDByTrackingNumber(trackingNumber string) (int, error) {
	// We need direct database access for this query
	// For now, let's return an error and handle linking later
	// This is a temporary solution until we can implement proper database access
	return 0, fmt.Errorf("shipment linking not yet implemented - tracking number: %s", trackingNumber)
}

// incrementTotalScans safely increments the total scans counter
func (m *TimeBasedProcessingMetrics) incrementTotalScans() {
	m.mu.Lock()
	m.TotalScans++
	m.mu.Unlock()
}

// addEmailsScanned safely adds to the total emails scanned counter
func (m *TimeBasedProcessingMetrics) addEmailsScanned(count int64) {
	m.mu.Lock()
	m.TotalEmailsScanned += count
	m.mu.Unlock()
}

// incrementEmailsWithBodiesStored safely increments the emails with bodies stored counter
func (m *TimeBasedProcessingMetrics) incrementEmailsWithBodiesStored() {
	m.mu.Lock()
	m.EmailsWithBodiesStored++
	m.mu.Unlock()
}

// incrementThreadsCreated safely increments the threads created counter
func (m *TimeBasedProcessingMetrics) incrementThreadsCreated() {
	m.mu.Lock()
	m.ThreadsCreated++
	m.mu.Unlock()
}

// incrementAutomaticLinksCreated safely increments the automatic links created counter
func (m *TimeBasedProcessingMetrics) incrementAutomaticLinksCreated() {
	m.mu.Lock()
	m.AutomaticLinksCreated++
	m.mu.Unlock()
}

// incrementShipmentsCreated safely increments the shipments created counter
func (m *TimeBasedProcessingMetrics) incrementShipmentsCreated() {
	m.mu.Lock()
	m.ShipmentsCreated++
	m.mu.Unlock()
}

// updateScanMetrics safely updates scan-related metrics
func (m *TimeBasedProcessingMetrics) updateScanMetrics(duration time.Duration) {
	m.mu.Lock()
	m.LastScanTime = time.Now()
	m.AverageScanDuration = duration
	m.mu.Unlock()
}

// updateRetroactiveScanTime safely updates retroactive scan time
func (m *TimeBasedProcessingMetrics) updateRetroactiveScanTime() {
	m.mu.Lock()
	m.LastRetroactiveScanTime = time.Now()
	m.mu.Unlock()
}

// GetMetrics returns current processing metrics (thread-safe copy)
func (p *TimeBasedEmailProcessor) GetMetrics() *TimeBasedProcessingMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	
	// Return a copy to prevent external modification
	return &TimeBasedProcessingMetrics{
		TotalScans:              p.metrics.TotalScans,
		TotalEmailsScanned:      p.metrics.TotalEmailsScanned,
		EmailsWithBodiesStored:  p.metrics.EmailsWithBodiesStored,
		ThreadsCreated:          p.metrics.ThreadsCreated,
		AutomaticLinksCreated:   p.metrics.AutomaticLinksCreated,
		ShipmentsCreated:        p.metrics.ShipmentsCreated,
		LastScanTime:            p.metrics.LastScanTime,
		LastRetroactiveScanTime: p.metrics.LastRetroactiveScanTime,
		AverageScanDuration:     p.metrics.AverageScanDuration,
	}
}

// IsHealthy checks if the processor is healthy
func (p *TimeBasedEmailProcessor) IsHealthy() error {
	if p.emailClient == nil {
		return fmt.Errorf("email client not configured")
	}

	return p.emailClient.HealthCheck()
}