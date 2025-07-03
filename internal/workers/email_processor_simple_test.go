package workers

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"package-tracking/internal/carriers"
	"package-tracking/internal/email"
	"package-tracking/internal/parser"
)

// Simple mock implementations for basic testing

type simpleEmailClient struct {
	emails []email.EmailMessage
	searchCalled bool
}

func (s *simpleEmailClient) Search(query string) ([]email.EmailMessage, error) {
	s.searchCalled = true
	return s.emails, nil
}

func (s *simpleEmailClient) GetMessage(id string) (*email.EmailMessage, error) {
	for _, msg := range s.emails {
		if msg.ID == id {
			return &msg, nil
		}
	}
	return nil, nil
}

func (s *simpleEmailClient) HealthCheck() error {
	return nil
}

func (s *simpleEmailClient) Close() error {
	return nil
}

type simpleStateManager struct {
	processed map[string]bool
}

func newSimpleStateManager() *simpleStateManager {
	return &simpleStateManager{
		processed: make(map[string]bool),
	}
}

func (s *simpleStateManager) IsProcessed(messageID string) (bool, error) {
	return s.processed[messageID], nil
}

func (s *simpleStateManager) MarkProcessed(entry *email.StateEntry) error {
	s.processed[entry.GmailMessageID] = true
	return nil
}

func (s *simpleStateManager) Cleanup(olderThan time.Time) error {
	return nil
}

func (s *simpleStateManager) GetStats() (*email.EmailMetrics, error) {
	return &email.EmailMetrics{
		TotalEmails: len(s.processed),
	}, nil
}

type simpleAPIClient struct {
	created []email.TrackingInfo
}

func (s *simpleAPIClient) CreateShipment(tracking email.TrackingInfo) error {
	s.created = append(s.created, tracking)
	return nil
}

func (s *simpleAPIClient) HealthCheck() error {
	return nil
}

func TestEmailProcessor_Basic(t *testing.T) {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Setup email client with test emails
	emailClient := &simpleEmailClient{
		emails: []email.EmailMessage{
			{
				ID:        "test-msg-1",
				From:      "noreply@ups.com",
				Subject:   "UPS Shipment Notification",
				PlainText: "Your package with tracking number 1Z999AA1234567890 has shipped.",
				Date:      time.Now(),
			},
		},
	}

	// Setup state manager
	stateManager := newSimpleStateManager()

	// Setup real extractor
	carrierFactory := carriers.NewClientFactory()
	extractorConfig := &parser.ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
		DebugMode:     false,
	}
	extractor := parser.NewTrackingExtractor(carrierFactory, extractorConfig)

	// Setup API client
	apiClient := &simpleAPIClient{}

	// Setup processor config
	config := &EmailProcessorConfig{
		CheckInterval:     5 * time.Minute,
		MaxEmailsPerRun:   10,
		DryRun:            false,
		SearchQuery:       "",
		ProcessingTimeout: 30 * time.Second,
	}

	// Create processor
	processor := NewEmailProcessor(config, emailClient, extractor, stateManager, apiClient, logger)

	if processor == nil {
		t.Fatal("Expected processor to be created, got nil")
	}

	// Test that processor has correct config
	if processor.config.MaxEmailsPerRun != 10 {
		t.Errorf("Expected MaxEmailsPerRun=10, got %d", processor.config.MaxEmailsPerRun)
	}

	// Test health check
	if err := processor.healthCheck(); err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Test pause/resume
	processor.Pause()
	if !processor.IsPaused() {
		t.Error("Expected processor to be paused")
	}

	processor.Resume()
	if processor.IsPaused() {
		t.Error("Expected processor to be resumed")
	}

	// Test metrics
	metrics := processor.GetMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be available")
	}
}

func TestEmailProcessor_ProcessingSingle(t *testing.T) {
	// This test verifies that the processor can handle a single processing run
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	emailClient := &simpleEmailClient{
		emails: []email.EmailMessage{
			{
				ID:        "ups-msg",
				From:      "noreply@ups.com",
				Subject:   "Package shipped",
				PlainText: "Your UPS package 1Z999AA1234567890 is on the way",
				Date:      time.Now(),
			},
		},
	}

	stateManager := newSimpleStateManager()
	carrierFactory := carriers.NewClientFactory()
	extractor := parser.NewTrackingExtractor(carrierFactory, &parser.ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
	})
	apiClient := &simpleAPIClient{}

	config := &EmailProcessorConfig{
		CheckInterval:     1 * time.Hour,
		MaxEmailsPerRun:   5,
		DryRun:            false,
		ProcessingTimeout: 10 * time.Second,
		SearchQuery:       "from:noreply@ups.com", // Provide explicit query
		SearchAfterDays:   7,
		UnreadOnly:        false,
		RetryCount:        3,
		RetryDelay:        1 * time.Second,
	}

	processor := NewEmailProcessor(config, emailClient, extractor, stateManager, apiClient, logger)

	// Run a single processing cycle
	processor.runProcessing()

	// Debug: Check if search was called
	if !emailClient.searchCalled {
		t.Error("Email search was not called")
	}

	// Check that shipment was created
	if len(apiClient.created) != 1 {
		t.Errorf("Expected 1 shipment to be created, got %d", len(apiClient.created))
	}

	if len(apiClient.created) > 0 {
		tracking := apiClient.created[0]
		if tracking.Number != "1Z999AA1234567890" {
			t.Errorf("Expected tracking number 1Z999AA1234567890, got %s", tracking.Number)
		}
		if tracking.Carrier != "ups" {
			t.Errorf("Expected carrier ups, got %s", tracking.Carrier)
		}
	}

	// Check that email was marked as processed
	stats, err := stateManager.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalEmails != 1 {
		t.Errorf("Expected 1 processed email, got %d", stats.TotalEmails)
	}
}