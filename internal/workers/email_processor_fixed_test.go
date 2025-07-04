package workers

import (
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"package-tracking/internal/carriers"
	"package-tracking/internal/email"
	"package-tracking/internal/parser"
)

// Fixed tests that match the actual implementation

func TestEmailProcessor_HealthCheckFixed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	testCases := []struct {
		name              string
		emailClientError  bool
		expectError       bool
	}{
		{
			name:             "Healthy dependencies",
			emailClientError: false,
			expectError:      false,
		},
		{
			name:             "Email client unhealthy",
			emailClientError: true,
			expectError:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			emailClient := &simpleEmailClient{closed: tc.emailClientError}
			stateManager := newSimpleStateManager()
			
			carrierFactory := carriers.NewClientFactory()
			extractor := parser.NewTrackingExtractor(carrierFactory, &parser.ExtractorConfig{
				EnableLLM:     false,
				MinConfidence: 0.5,
			}, &parser.LLMConfig{Enabled: false})
			apiClient := &simpleAPIClient{}

			config := &EmailProcessorConfig{
				CheckInterval:     5 * time.Minute,
				MaxEmailsPerRun:   50,
				SearchQuery:       "test query",
				ProcessingTimeout: 30 * time.Second,
				RetryCount:        3,
				RetryDelay:        1 * time.Second,
			}

			processor := NewEmailProcessor(config, emailClient, extractor, stateManager, apiClient, logger)

			err := processor.healthCheck()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected health check error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected health check error: %v", err)
				}
			}
		})
	}
}

func TestEmailProcessor_WorkflowFixed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	emailClient := &simpleEmailClient{
		emails: []email.EmailMessage{
			{
				ID:        "test-msg",
				From:      "noreply@ups.com",
				Subject:   "UPS Shipment",
				PlainText: "Package 1Z999AA1234567890 shipped",
				Date:      time.Now(),
			},
		},
	}
	stateManager := newSimpleStateManager()
	
	carrierFactory := carriers.NewClientFactory()
	extractor := parser.NewTrackingExtractor(carrierFactory, &parser.ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
	}, &parser.LLMConfig{Enabled: false})
	apiClient := &simpleAPIClient{}

	config := &EmailProcessorConfig{
		CheckInterval:     1 * time.Hour,
		MaxEmailsPerRun:   10,
		DryRun:            false,
		SearchQuery:       "test query",
		ProcessingTimeout: 30 * time.Second,
		RetryCount:        3,
		RetryDelay:        1 * time.Second,
	}

	processor := NewEmailProcessor(config, emailClient, extractor, stateManager, apiClient, logger)

	// Test basic functionality
	if processor == nil {
		t.Fatal("Processor should not be nil")
	}

	// Test pause/resume
	processor.Pause()
	if !processor.IsPaused() {
		t.Error("Processor should be paused")
	}

	processor.Resume()
	if processor.IsPaused() {
		t.Error("Processor should not be paused")
	}

	// Test metrics
	metrics := processor.GetMetrics()
	if metrics == nil {
		t.Error("Metrics should not be nil")
	}

	// Test processing
	processor.runProcessing()

	// Check results
	if len(apiClient.created) != 1 {
		t.Errorf("Expected 1 shipment created, got %d", len(apiClient.created))
	}

	stats, err := stateManager.GetStats()
	if err != nil {
		t.Errorf("Failed to get stats: %v", err)
	}

	if stats.TotalEmails != 1 {
		t.Errorf("Expected 1 processed email, got %d", stats.TotalEmails)
	}
}

// Simple mock implementations that match interfaces

type simpleEmailClient struct {
	emails []email.EmailMessage
	closed bool
}

func (s *simpleEmailClient) Search(query string) ([]email.EmailMessage, error) {
	if s.closed {
		return nil, fmt.Errorf("client is closed")
	}
	return s.emails, nil
}

func (s *simpleEmailClient) GetMessage(id string) (*email.EmailMessage, error) {
	for _, msg := range s.emails {
		if msg.ID == id {
			return &msg, nil
		}
	}
	return nil, fmt.Errorf("message not found")
}

func (s *simpleEmailClient) HealthCheck() error {
	if s.closed {
		return fmt.Errorf("client is closed")
	}
	return nil
}

func (s *simpleEmailClient) Close() error {
	s.closed = true
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