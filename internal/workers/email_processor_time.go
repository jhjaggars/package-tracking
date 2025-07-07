package workers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"package-tracking/internal/database"
	"package-tracking/internal/email"
)

// TrackingExtractor interface for extracting tracking information from emails
type TrackingExtractor interface {
	Extract(content *email.EmailContent) ([]email.TrackingInfo, error)
}

// TimeBasedEmailProcessor handles time-based email scanning with body storage
type TimeBasedEmailProcessor struct {
	config       *TimeBasedEmailProcessorConfig
	emailClient  TimeBasedEmailClient
	extractor    TrackingExtractor
	emailStore   *database.EmailStore
	shipmentStore *database.ShipmentStore
	apiClient    APIClient
	logger       *slog.Logger
	metrics      *TimeBasedProcessingMetrics
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
	emailStore *database.EmailStore,
	shipmentStore *database.ShipmentStore,
	apiClient APIClient,
	logger *slog.Logger,
) *TimeBasedEmailProcessor {
	return &TimeBasedEmailProcessor{
		config:        config,
		emailClient:   emailClient,
		extractor:     extractor,
		emailStore:    emailStore,
		shipmentStore: shipmentStore,
		apiClient:     apiClient,
		logger:        logger,
		metrics:       &TimeBasedProcessingMetrics{},
	}
}

// ProcessEmailsSince processes all emails since the specified time using time-based scanning
func (p *TimeBasedEmailProcessor) ProcessEmailsSince(since time.Time) error {
	startTime := time.Now()
	p.metrics.TotalScans++

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

	p.metrics.TotalEmailsScanned += int64(len(messages))

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
		alreadyProcessed, err := p.emailStore.IsProcessed(msg.ID)
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
	p.metrics.LastScanTime = time.Now()
	p.metrics.AverageScanDuration = duration

	p.logger.Info("Time-based email processing completed",
		"duration", duration,
		"processed", processed,
		"skipped", skipped,
		"errors", errors,
		"total_messages", len(messages))

	// Cleanup old email bodies if retention is configured
	if p.config.RetentionDays > 0 {
		cleanupTime := time.Now().AddDate(0, 0, -p.config.RetentionDays)
		if err := p.emailStore.CleanupOldEmails(cleanupTime); err != nil {
			p.logger.Warn("Failed to cleanup old email bodies", "error", err)
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

	p.metrics.LastRetroactiveScanTime = time.Now()
	p.metrics.TotalEmailsScanned += int64(len(messages))

	// Process all retrieved messages
	for _, msg := range messages {
		// Check if already processed
		alreadyProcessed, err := p.emailStore.IsProcessed(msg.ID)
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

	// Convert to database format for storage
	emailEntry := &database.EmailBodyEntry{
		GmailMessageID:    msg.ID,
		GmailThreadID:     msg.ThreadID,
		From:              msg.From,
		Subject:           msg.Subject,
		Date:              msg.Date,
		InternalTimestamp: time.Now(),
		ScanMethod:        "time-based",
		ProcessedAt:       time.Now(),
		Status:            "processing",
	}

	// Store email body if enabled
	if p.config.BodyStorageEnabled {
		emailEntry.BodyText = msg.PlainText
		emailEntry.BodyHTML = msg.HTMLText

		// Compress body if it's large
		if len(msg.PlainText) > 1000 { // Compress if larger than 1KB
			compressed, err := email.CompressEmailBody(msg.PlainText)
			if err != nil {
				logger.Warn("Failed to compress email body", "error", err)
			} else {
				emailEntry.BodyCompressed = compressed
				// Clear uncompressed text to save space
				emailEntry.BodyText = ""
			}
		}

		p.metrics.EmailsWithBodiesStored++
		logger.Debug("Stored email body", "compressed", len(emailEntry.BodyCompressed) > 0)
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
		emailEntry.Status = "error"
		emailEntry.ErrorMessage = err.Error()
	} else {
		// Store tracking numbers found
		if len(trackingInfo) > 0 {
			trackingJSON, _ := json.Marshal(trackingInfo)
			emailEntry.TrackingNumbers = string(trackingJSON)
			emailEntry.Status = "processed"

			logger.Info("Found tracking numbers", "count", len(trackingInfo))

			// Create shipments and link them to the email
			if err := p.createShipmentsAndLinks(trackingInfo, emailEntry); err != nil {
				logger.Error("Failed to create shipments and links", "error", err)
			}
		} else {
			emailEntry.Status = "processed"
			logger.Debug("No tracking numbers found")
		}
	}

	// Store the email in the database
	if err := p.emailStore.CreateOrUpdate(emailEntry); err != nil {
		return fmt.Errorf("failed to store email: %w", err)
	}

	// Create or update thread information
	if err := p.createOrUpdateThread(msg); err != nil {
		logger.Warn("Failed to create/update thread", "error", err)
	}

	return nil
}

// createShipmentsAndLinks creates shipments for found tracking numbers and links them to the email
func (p *TimeBasedEmailProcessor) createShipmentsAndLinks(trackingInfo []email.TrackingInfo, emailEntry *database.EmailBodyEntry) error {
	for _, tracking := range trackingInfo {
		if p.config.DryRun {
			p.logger.Info("Dry run: would create shipment",
				"tracking_number", tracking.Number,
				"carrier", tracking.Carrier)
			continue
		}

		// Create shipment via API
		if err := p.createShipment(tracking); err != nil {
			p.logger.Warn("Failed to create shipment",
				"tracking_number", tracking.Number,
				"carrier", tracking.Carrier,
				"error", err)
			continue
		}

		p.metrics.ShipmentsCreated++

		// Find the created shipment to get its ID
		// This is a simplified approach - in a real implementation, we'd get the ID from the API response
		// For now, we'll create the link based on tracking number matching
		if err := p.linkEmailToShipmentByTracking(emailEntry.ID, tracking.Number); err != nil {
			p.logger.Warn("Failed to link email to shipment",
				"email_id", emailEntry.ID,
				"tracking_number", tracking.Number,
				"error", err)
		} else {
			p.metrics.AutomaticLinksCreated++
		}
	}

	return nil
}

// createShipment creates a shipment via the API client
func (p *TimeBasedEmailProcessor) createShipment(tracking email.TrackingInfo) error {
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

// linkEmailToShipmentByTracking links an email to a shipment based on tracking number
func (p *TimeBasedEmailProcessor) linkEmailToShipmentByTracking(emailID int, trackingNumber string) error {
	// This is a placeholder implementation
	// In a real implementation, we would query the shipments table to find the shipment ID
	// and then create the link using emailStore.LinkEmailToShipment
	
	// For now, we'll log that we would create the link
	p.logger.Debug("Would create email-shipment link",
		"email_id", emailID,
		"tracking_number", trackingNumber)

	return nil
}

// createOrUpdateThread creates or updates thread information
func (p *TimeBasedEmailProcessor) createOrUpdateThread(msg *email.EmailMessage) error {
	// Extract participants from the From field (simplified)
	participants := []string{msg.From}
	participantsJSON, _ := json.Marshal(participants)

	thread := &database.EmailThread{
		GmailThreadID:    msg.ThreadID,
		Subject:          msg.Subject,
		Participants:     string(participantsJSON),
		MessageCount:     1, // This would be calculated properly in a real implementation
		FirstMessageDate: msg.Date,
		LastMessageDate:  msg.Date,
	}

	if err := p.emailStore.CreateOrUpdateThread(thread); err != nil {
		return fmt.Errorf("failed to create/update thread: %w", err)
	}

	p.metrics.ThreadsCreated++
	return nil
}

// GetMetrics returns current processing metrics
func (p *TimeBasedEmailProcessor) GetMetrics() *TimeBasedProcessingMetrics {
	return p.metrics
}

// IsHealthy checks if the processor is healthy
func (p *TimeBasedEmailProcessor) IsHealthy() error {
	if p.emailClient == nil {
		return fmt.Errorf("email client not configured")
	}

	return p.emailClient.HealthCheck()
}