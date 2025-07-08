package workers

import (
	"context"
	"fmt"
	"log"
	"time"

	"package-tracking/internal/parser"
)

// EmailMessage represents an email message for processing
type EmailMessage struct {
	ID      string
	From    string
	Subject string
	Body    string
	Date    time.Time
}

// ShipmentRequest represents a request to create a shipment
type ShipmentRequest struct {
	TrackingNumber string
	Carrier        string
	Description    string
}

// EmailClient interface for email operations
type EmailClient interface {
	SearchEmails(ctx context.Context, query string, since time.Time) ([]EmailMessage, error)
	GetMessage(ctx context.Context, messageID string) (*EmailMessage, error)
	HealthCheck(ctx context.Context) error
	Close() error
}

// ShipmentCreator interface for creating shipments
type ShipmentCreator interface {
	CreateShipment(ctx context.Context, req ShipmentRequest) error
}

// EmailStateManager interface for tracking processed emails
type EmailStateManager interface {
	IsProcessed(messageID string) (bool, error)
	MarkProcessed(messageID string) error
}

// SimplifiedEmailProcessor implements the simplified email processing algorithm
type SimplifiedEmailProcessor struct {
	emailClient          EmailClient
	trackingExtractor    parser.SimplifiedTrackingExtractorInterface
	descriptionExtractor parser.SimplifiedDescriptionExtractorInterface
	shipmentCreator      ShipmentCreator
	stateManager         EmailStateManager
	daysToScan           int
	dryRun               bool
}

// NewSimplifiedEmailProcessor creates a new simplified email processor
func NewSimplifiedEmailProcessor(
	emailClient EmailClient,
	trackingExtractor parser.SimplifiedTrackingExtractorInterface,
	descriptionExtractor parser.SimplifiedDescriptionExtractorInterface,
	shipmentCreator ShipmentCreator,
	stateManager EmailStateManager,
	daysToScan int,
	dryRun bool,
) *SimplifiedEmailProcessor {
	return &SimplifiedEmailProcessor{
		emailClient:          emailClient,
		trackingExtractor:    trackingExtractor,
		descriptionExtractor: descriptionExtractor,
		shipmentCreator:      shipmentCreator,
		stateManager:         stateManager,
		daysToScan:           daysToScan,
		dryRun:               dryRun,
	}
}

// ProcessEmails implements the simplified email processing algorithm:
// 1. Read last N days of unprocessed emails
// 2. Extract tracking numbers via pattern matching
// 3. Validate tracking numbers
// 4. Extract descriptions using LLM
// 5. Create shipments (unless in dry run mode)
func (p *SimplifiedEmailProcessor) ProcessEmails(ctx context.Context) error {
	// Calculate the date range for email search
	since := time.Now().AddDate(0, 0, -p.daysToScan)
	
	// Search for emails in the specified time range
	query := p.buildSearchQuery()
	emails, err := p.emailClient.SearchEmails(ctx, query, since)
	if err != nil {
		return fmt.Errorf("failed to search emails: %w", err)
	}

	log.Printf("Found %d emails to process", len(emails))

	// Process each email
	for _, email := range emails {
		if err := p.processEmail(ctx, email); err != nil {
			log.Printf("Error processing email %s: %v", email.ID, err)
			continue // Continue with next email instead of failing entirely
		}
	}

	return nil
}

// processEmail processes a single email message
func (p *SimplifiedEmailProcessor) processEmail(ctx context.Context, email EmailMessage) error {
	// Check if this email has already been processed
	processed, err := p.stateManager.IsProcessed(email.ID)
	if err != nil {
		return fmt.Errorf("failed to check if email is processed: %w", err)
	}
	
	if processed {
		log.Printf("Email %s already processed, skipping", email.ID)
		return nil
	}

	// Extract tracking numbers from email content
	emailContent := email.Subject + " " + email.Body
	trackingResults, err := p.trackingExtractor.ExtractTrackingNumbers(emailContent)
	if err != nil {
		return fmt.Errorf("failed to extract tracking numbers: %w", err)
	}

	// Process each valid tracking number found
	shipmentsCreated := 0
	for _, result := range trackingResults {
		if !result.Valid {
			continue
		}

		// Extract description using LLM
		description, err := p.descriptionExtractor.ExtractDescription(ctx, emailContent, result.Number)
		if err != nil {
			log.Printf("Failed to extract description for tracking %s: %v", result.Number, err)
			description = "" // Continue with empty description
		}

		// Create shipment request
		shipmentReq := ShipmentRequest{
			TrackingNumber: result.Number,
			Carrier:        result.Carrier,
			Description:    description,
		}

		// Create shipment (unless in dry run mode)
		if !p.dryRun {
			if err := p.shipmentCreator.CreateShipment(ctx, shipmentReq); err != nil {
				log.Printf("Failed to create shipment for tracking %s: %v", result.Number, err)
				continue // Continue with next tracking number
			}
		} else {
			log.Printf("DRY RUN: Would create shipment for tracking %s (%s) with description: %s", 
				result.Number, result.Carrier, description)
		}

		shipmentsCreated++
	}

	// Mark email as processed
	if err := p.stateManager.MarkProcessed(email.ID); err != nil {
		return fmt.Errorf("failed to mark email as processed: %w", err)
	}

	if shipmentsCreated > 0 {
		log.Printf("Successfully processed email %s and created %d shipments", email.ID, shipmentsCreated)
	} else {
		log.Printf("Processed email %s but found no valid tracking numbers", email.ID)
	}

	return nil
}

// buildSearchQuery builds the Gmail search query for finding shipping emails
func (p *SimplifiedEmailProcessor) buildSearchQuery() string {
	// Search for emails from major shipping carriers and common shipping keywords
	query := `(from:ups.com OR from:usps.com OR from:fedex.com OR from:dhl.com OR from:amazon.com OR ` +
		`from:shipment-tracking OR subject:shipped OR subject:tracking OR subject:delivery OR ` +
		`subject:"on its way" OR subject:"package" OR body:tracking OR body:"tracking number")`
	
	return query
}