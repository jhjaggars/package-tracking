package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"package-tracking/internal/database"
	"package-tracking/internal/email"
	"package-tracking/internal/parser"
)

// DescriptionEnhancer handles retroactive enhancement of shipment descriptions
type DescriptionEnhancer struct {
	shipmentStore *database.ShipmentStore
	emailStore    *database.EmailStore
	extractor     *parser.TrackingExtractor
	logger        *slog.Logger
}

// DescriptionEnhancementResult represents the result of enhancing a single shipment
type DescriptionEnhancementResult struct {
	ShipmentID      int    `json:"shipment_id"`
	TrackingNumber  string `json:"tracking_number"`
	OldDescription  string `json:"old_description"`
	NewDescription  string `json:"new_description"`
	EmailsFound     int    `json:"emails_found"`
	Success         bool   `json:"success"`
	Error           string `json:"error,omitempty"`
	ProcessedAt     time.Time `json:"processed_at"`
}

// DescriptionEnhancementSummary represents the overall results of an enhancement operation
type DescriptionEnhancementSummary struct {
	TotalShipments  int                            `json:"total_shipments"`
	SuccessCount    int                            `json:"success_count"`
	FailureCount    int                            `json:"failure_count"`
	Results         []DescriptionEnhancementResult `json:"results"`
	ProcessingTime  time.Duration                  `json:"processing_time"`
	StartedAt       time.Time                      `json:"started_at"`
	CompletedAt     time.Time                      `json:"completed_at"`
}

// NewDescriptionEnhancer creates a new description enhancer service
func NewDescriptionEnhancer(
	shipmentStore *database.ShipmentStore,
	emailStore *database.EmailStore,
	extractor *parser.TrackingExtractor,
	logger *slog.Logger,
) *DescriptionEnhancer {
	return &DescriptionEnhancer{
		shipmentStore: shipmentStore,
		emailStore:    emailStore,
		extractor:     extractor,
		logger:        logger,
	}
}

// EnhanceAllShipmentsWithPoorDescriptions enhances all shipments that have poor descriptions
func (de *DescriptionEnhancer) EnhanceAllShipmentsWithPoorDescriptions(limit int, dryRun bool) (*DescriptionEnhancementSummary, error) {
	startTime := time.Now()
	
	de.logger.Info("Starting enhancement of shipments with poor descriptions",
		"limit", limit,
		"dry_run", dryRun)

	// Get shipments with poor descriptions
	shipments, err := de.shipmentStore.GetShipmentsWithPoorDescriptions(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipments with poor descriptions: %w", err)
	}

	de.logger.Info("Found shipments with poor descriptions", "count", len(shipments))

	summary := &DescriptionEnhancementSummary{
		TotalShipments: len(shipments),
		Results:        make([]DescriptionEnhancementResult, 0, len(shipments)),
		StartedAt:      startTime,
	}

	// Process each shipment
	for _, shipment := range shipments {
		result := de.enhanceShipmentDescription(shipment, dryRun)
		summary.Results = append(summary.Results, result)
		
		if result.Success {
			summary.SuccessCount++
		} else {
			summary.FailureCount++
		}
	}

	summary.CompletedAt = time.Now()
	summary.ProcessingTime = summary.CompletedAt.Sub(startTime)

	de.logger.Info("Completed enhancement operation",
		"total", summary.TotalShipments,
		"success", summary.SuccessCount,
		"failures", summary.FailureCount,
		"duration", summary.ProcessingTime)

	return summary, nil
}

// EnhanceSpecificShipment enhances a specific shipment by ID
func (de *DescriptionEnhancer) EnhanceSpecificShipment(shipmentID int, dryRun bool) (*DescriptionEnhancementResult, error) {
	de.logger.Info("Enhancing specific shipment", "shipment_id", shipmentID, "dry_run", dryRun)

	// Get the shipment
	shipment, err := de.shipmentStore.GetByID(shipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment %d: %w", shipmentID, err)
	}

	result := de.enhanceShipmentDescription(*shipment, dryRun)
	return &result, nil
}

// enhanceShipmentDescription enhances the description of a single shipment
func (de *DescriptionEnhancer) enhanceShipmentDescription(shipment database.Shipment, dryRun bool) DescriptionEnhancementResult {
	result := DescriptionEnhancementResult{
		ShipmentID:     shipment.ID,
		TrackingNumber: shipment.TrackingNumber,
		OldDescription: shipment.Description,
		ProcessedAt:    time.Now(),
	}

	de.logger.Debug("Processing shipment",
		"shipment_id", shipment.ID,
		"tracking_number", shipment.TrackingNumber,
		"current_description", shipment.Description)

	// Find emails containing this tracking number
	emails, err := de.emailStore.GetEmailsForTrackingNumber(shipment.TrackingNumber)
	if err != nil {
		result.Error = fmt.Sprintf("failed to find emails: %v", err)
		de.logger.Warn("Failed to find emails for tracking number",
			"tracking_number", shipment.TrackingNumber,
			"error", err)
		return result
	}

	result.EmailsFound = len(emails)
	de.logger.Debug("Found emails for tracking number",
		"tracking_number", shipment.TrackingNumber,
		"email_count", len(emails))

	if len(emails) == 0 {
		result.Error = "no emails found for tracking number"
		return result
	}

	// Find the best email to extract description from
	bestEmail := de.selectBestEmailForExtraction(emails, shipment.TrackingNumber)
	if bestEmail == nil {
		result.Error = "no suitable email found for extraction"
		return result
	}

	// Extract enhanced description using LLM
	newDescription, err := de.extractEnhancedDescription(bestEmail, shipment.TrackingNumber, shipment.Carrier)
	if err != nil {
		result.Error = fmt.Sprintf("failed to extract description: %v", err)
		de.logger.Warn("Failed to extract description",
			"email_id", bestEmail.ID,
			"tracking_number", shipment.TrackingNumber,
			"error", err)
		return result
	}

	result.NewDescription = newDescription

	// Update the shipment description if not in dry run mode
	if !dryRun && newDescription != "" && newDescription != shipment.Description {
		err = de.shipmentStore.UpdateDescription(shipment.ID, newDescription)
		if err != nil {
			result.Error = fmt.Sprintf("failed to update description: %v", err)
			de.logger.Error("Failed to update shipment description",
				"shipment_id", shipment.ID,
				"new_description", newDescription,
				"error", err)
			return result
		}
		
		de.logger.Info("Updated shipment description",
			"shipment_id", shipment.ID,
			"tracking_number", shipment.TrackingNumber,
			"old_description", shipment.Description,
			"new_description", newDescription)
	}

	result.Success = true
	return result
}

// selectBestEmailForExtraction selects the most suitable email for description extraction
func (de *DescriptionEnhancer) selectBestEmailForExtraction(emails []database.EmailBodyEntry, trackingNumber string) *database.EmailBodyEntry {
	if len(emails) == 0 {
		return nil
	}

	// Prioritize emails with actual body content
	var emailsWithContent []database.EmailBodyEntry
	for _, email := range emails {
		if email.BodyText != "" || len(email.BodyCompressed) > 0 || email.Subject != "" {
			emailsWithContent = append(emailsWithContent, email)
		}
	}

	if len(emailsWithContent) == 0 {
		// Fall back to any email if no body content found
		emailsWithContent = emails
	}

	// Prioritize Amazon shipping notifications and confirmations
	for _, email := range emailsWithContent {
		sender := strings.ToLower(email.From)
		subject := strings.ToLower(email.Subject)
		
		if strings.Contains(sender, "amazon.com") {
			// Prioritize shipping notifications over order confirmations
			if strings.Contains(subject, "shipped") || strings.Contains(subject, "delivered") {
				return &email
			}
		}
	}

	// Look for any shipping-related email
	for _, email := range emailsWithContent {
		subject := strings.ToLower(email.Subject)
		if strings.Contains(subject, "shipped") || strings.Contains(subject, "delivered") || 
		   strings.Contains(subject, "tracking") || strings.Contains(subject, "shipment") {
			return &email
		}
	}

	// Return the most recent email with content
	return &emailsWithContent[0]
}

// extractEnhancedDescription extracts an enhanced description from an email using LLM
func (de *DescriptionEnhancer) extractEnhancedDescription(email *database.EmailBodyEntry, trackingNumber, carrier string) (string, error) {
	// Reconstruct email content for LLM processing
	emailContent, err := de.reconstructEmailContent(email)
	if err != nil {
		return "", fmt.Errorf("failed to reconstruct email content: %w", err)
	}

	de.logger.Debug("Reconstructed email content",
		"email_id", email.ID,
		"subject", emailContent.Subject,
		"from", emailContent.From,
		"body_length", len(emailContent.PlainText))

	// Use the existing LLM extractor to get enhanced description
	trackingInfo, err := de.extractor.Extract(emailContent)
	if err != nil {
		return "", fmt.Errorf("LLM extraction failed: %w", err)
	}

	// Find the tracking info for our specific tracking number
	for _, info := range trackingInfo {
		if info.Number == trackingNumber && info.Description != "" {
			de.logger.Debug("Found enhanced description via LLM",
				"tracking_number", trackingNumber,
				"description", info.Description,
				"merchant", info.Merchant,
				"confidence", info.Confidence)
			return info.Description, nil
		}
	}

	// If no LLM result, try subject line extraction as fallback
	if de.extractor != nil {
		subjectDescription := de.extractDescriptionFromSubject(emailContent.Subject, carrier)
		if subjectDescription != "" {
			de.logger.Debug("Found description via subject extraction",
				"tracking_number", trackingNumber,
				"description", subjectDescription)
			return subjectDescription, nil
		}
	}

	return "", fmt.Errorf("no enhanced description found for tracking number %s", trackingNumber)
}

// reconstructEmailContent reconstructs email content from stored database entry
func (de *DescriptionEnhancer) reconstructEmailContent(emailEntry *database.EmailBodyEntry) (*email.EmailContent, error) {
	content := &email.EmailContent{
		From:      emailEntry.From,
		Subject:   emailEntry.Subject,
		Date:      emailEntry.Date,
		MessageID: emailEntry.GmailMessageID,
		ThreadID:  emailEntry.GmailThreadID,
	}

	// Use body text if available
	if emailEntry.BodyText != "" {
		content.PlainText = emailEntry.BodyText
	} else if len(emailEntry.BodyCompressed) > 0 {
		// Decompress body if it's compressed
		decompressed, err := database.DecompressEmailBody(emailEntry.BodyCompressed)
		if err != nil {
			de.logger.Warn("Failed to decompress email body",
				"email_id", emailEntry.ID,
				"error", err)
		} else {
			content.PlainText = decompressed
		}
	}

	// Use HTML body if available
	if emailEntry.BodyHTML != "" {
		content.HTMLText = emailEntry.BodyHTML
	}

	// If no body content, at least we have subject line for extraction
	if content.PlainText == "" && content.HTMLText == "" {
		de.logger.Debug("No body content found, using subject only",
			"email_id", emailEntry.ID,
			"subject", content.Subject)
	}

	return content, nil
}

// extractDescriptionFromSubject is a simplified version for fallback (calls the parser method if available)
func (de *DescriptionEnhancer) extractDescriptionFromSubject(subject, carrier string) string {
	// This is a simplified fallback - in a real implementation we could
	// copy the subject extraction logic from the parser
	if subject == "" {
		return ""
	}

	// Amazon-specific patterns
	if strings.Contains(strings.ToLower(subject), "amazon") || carrier == "amazon" {
		// For simplicity, we'll use a basic string search instead of regex
		if strings.Contains(subject, `"`) {
			start := strings.Index(subject, `"`)
			if start != -1 {
				end := strings.Index(subject[start+1:], `"`)
				if end != -1 {
					description := subject[start+1 : start+1+end]
					if len(description) > 3 {
						// Clean up common suffixes
						description = strings.TrimSuffix(description, "...")
						description = strings.TrimSuffix(description, "…")
						description = strings.TrimSuffix(description, " and more")
						return strings.TrimSpace(description)
					}
				}
			}
		}
	}

	return ""
}

// AssociateEmailsWithShipments creates associations between existing emails and shipments
func (de *DescriptionEnhancer) AssociateEmailsWithShipments() error {
	de.logger.Info("Starting email-shipment association process")

	// Get all emails with tracking numbers
	emails, err := de.emailStore.GetEmailsWithTrackingNumbers()
	if err != nil {
		return fmt.Errorf("failed to get emails with tracking numbers: %w", err)
	}

	associationCount := 0
	for _, email := range emails {
		// Parse tracking numbers from the email
		var trackingNumbers []string
		if err := json.Unmarshal([]byte(email.TrackingNumbers), &trackingNumbers); err != nil {
			de.logger.Warn("Failed to parse tracking numbers",
				"email_id", email.ID,
				"tracking_numbers", email.TrackingNumbers,
				"error", err)
			continue
		}

		// For each tracking number, try to find corresponding shipment
		for _, trackingNumber := range trackingNumbers {
			shipment, err := de.shipmentStore.GetByTrackingNumber(trackingNumber)
			if err != nil {
				de.logger.Debug("No shipment found for tracking number",
					"tracking_number", trackingNumber,
					"email_id", email.ID)
				continue
			}

			// Create the association
			err = de.emailStore.LinkEmailToShipment(email.ID, shipment.ID, "automatic", trackingNumber, "description-enhancer")
			if err != nil {
				de.logger.Warn("Failed to link email to shipment",
					"email_id", email.ID,
					"shipment_id", shipment.ID,
					"tracking_number", trackingNumber,
					"error", err)
				continue
			}

			associationCount++
			de.logger.Debug("Associated email with shipment",
				"email_id", email.ID,
				"shipment_id", shipment.ID,
				"tracking_number", trackingNumber)
		}
	}

	de.logger.Info("Completed email-shipment association",
		"total_emails", len(emails),
		"associations_created", associationCount)

	return nil
}