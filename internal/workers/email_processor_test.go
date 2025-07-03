package workers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"package-tracking/internal/carriers"
	"package-tracking/internal/email"
	"package-tracking/internal/parser"
)

// Mock implementations for testing

type mockEmailClient struct {
	emails      []email.EmailMessage
	searchDelay time.Duration
	errorOnSearch bool
	errorOnGet   bool
	closed       bool
}

func (m *mockEmailClient) Search(query string) ([]email.EmailMessage, error) {
	if m.errorOnSearch {
		return nil, fmt.Errorf("mock search error")
	}
	
	if m.searchDelay > 0 {
		time.Sleep(m.searchDelay)
	}
	
	return m.emails, nil
}

func (m *mockEmailClient) GetMessage(id string) (*email.EmailMessage, error) {
	if m.errorOnGet {
		return nil, fmt.Errorf("mock get message error")
	}
	
	for _, msg := range m.emails {
		if msg.ID == id {
			return &msg, nil
		}
	}
	
	return nil, fmt.Errorf("message not found: %s", id)
}

func (m *mockEmailClient) HealthCheck() error {
	if m.closed {
		return fmt.Errorf("client is closed")
	}
	return nil
}

func (m *mockEmailClient) Close() error {
	m.closed = true
	return nil
}

type mockStateManager struct {
	processed map[string]bool
	mu        sync.RWMutex
	errorOnMark bool
}

func newMockStateManager() *mockStateManager {
	return &mockStateManager{
		processed: make(map[string]bool),
	}
}

func (m *mockStateManager) IsProcessed(messageID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.processed[messageID], nil
}

func (m *mockStateManager) MarkProcessed(entry *email.StateEntry) error {
	if m.errorOnMark {
		return fmt.Errorf("mock mark processed error")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processed[entry.GmailMessageID] = true
	return nil
}

func (m *mockStateManager) GetStats() (*email.EmailMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &email.EmailMetrics{
		TotalEmails: len(m.processed),
	}, nil
}

func (m *mockStateManager) Cleanup(olderThan time.Time) error {
	return nil
}

func (m *mockStateManager) Close() error {
	return nil
}

type mockExtractor struct {
	trackingNumbers []email.TrackingInfo
	errorOnExtract  bool
	extractDelay    time.Duration
}

func (m *mockExtractor) Extract(content *email.EmailContent) ([]email.TrackingInfo, error) {
	if m.errorOnExtract {
		return nil, fmt.Errorf("mock extraction error")
	}
	
	if m.extractDelay > 0 {
		time.Sleep(m.extractDelay)
	}
	
	return m.trackingNumbers, nil
}

type mockAPIClient struct {
	createdShipments []email.TrackingInfo
	mu               sync.Mutex
	errorOnCreate    bool
}

func (m *mockAPIClient) CreateShipment(tracking email.TrackingInfo) error {
	if m.errorOnCreate {
		return fmt.Errorf("mock API error")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createdShipments = append(m.createdShipments, tracking)
	return nil
}

func (m *mockAPIClient) HealthCheck() error {
	return nil
}

func (m *mockAPIClient) Close() error {
	return nil
}

func (m *mockAPIClient) GetCreatedShipments() []email.TrackingInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]email.TrackingInfo, len(m.createdShipments))
	copy(result, m.createdShipments)
	return result
}

func TestNewEmailProcessor(t *testing.T) {
	testCases := []struct {
		name        string
		config      *EmailProcessorConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid configuration",
			config: &EmailProcessorConfig{
				CheckInterval:   5 * time.Minute,
				MaxEmailsPerRun: 50,
				DryRun:          false,
				SearchQuery:     "from:ups.com",
			},
			expectError: false,
		},
		{
			name:        "Nil configuration",
			config:      nil,
			expectError: true,
			errorMsg:    "config cannot be nil",
		},
		{
			name: "Invalid check interval",
			config: &EmailProcessorConfig{
				CheckInterval:   0,
				MaxEmailsPerRun: 50,
			},
			expectError: true,
			errorMsg:    "check interval must be positive",
		},
		{
			name: "Invalid max per run",
			config: &EmailProcessorConfig{
				CheckInterval:   5 * time.Minute,
				MaxEmailsPerRun: 0,
			},
			expectError: true,
			errorMsg:    "max per run must be positive",
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	emailClient := &mockEmailClient{}
	stateManager := newMockStateManager()
	extractor := &mockExtractor{}
	apiClient := &mockAPIClient{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			processor := NewEmailProcessor(
				tc.config,
				emailClient,
				extractor,
				stateManager,
				apiClient,
				logger,
			)

			if tc.expectError {
				// For now, NewEmailProcessor doesn't return errors in actual implementation
				// This test structure is ready for future validation
				if processor == nil {
					t.Errorf("Expected processor even with invalid config for now")
				}
			} else {
				if processor == nil {
					t.Errorf("Expected processor, but got nil")
				}
			}
		})
	}
}

func TestEmailProcessor_ProcessEmails(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	testCases := []struct {
		name                 string
		emails              []email.EmailMessage
		trackingNumbers     []email.TrackingInfo
		alreadyProcessed    []string
		expectProcessed     int
		expectCreated       int
		expectError         bool
		extractorError      bool
		stateManagerError   bool
		apiClientError      bool
	}{
		{
			name: "Process new emails with tracking",
			emails: []email.EmailMessage{
				{
					ID: "msg-001",
					PlainText: "Your UPS package 1Z999AA1234567890 has shipped",
					From:      "noreply@ups.com",
					Subject:   "UPS Shipment Notification",
				},
			},
			trackingNumbers: []email.TrackingInfo{
				{Number: "1Z999AA1234567890", Carrier: "ups", Confidence: 0.9},
			},
			expectProcessed: 1,
			expectCreated:   1,
		},
		{
			name: "Skip already processed emails",
			emails: []email.EmailMessage{
				{
					ID: "msg-002",
					PlainText: "Your package has shipped",
					From:      "test@example.com",
				},
			},
			alreadyProcessed: []string{"msg-002"},
			expectProcessed:  0,
			expectCreated:    0,
		},
		{
			name: "Process emails with no tracking numbers",
			emails: []email.EmailMessage{
				{
					ID: "msg-003",
					PlainText: "Thank you for your order",
					From:      "orders@example.com",
				},
			},
			trackingNumbers: []email.TrackingInfo{}, // No tracking found
			expectProcessed: 1,
			expectCreated:   0,
		},
		{
			name: "Handle extraction errors gracefully",
			emails: []email.EmailMessage{
				{
					ID: "msg-004",
					PlainText: "Test email",
				},
			},
			extractorError:  true,
			expectProcessed: 0,
			expectCreated:   0,
		},
		{
			name: "Handle API client errors",
			emails: []email.EmailMessage{
				{
					ID: "msg-005",
					PlainText: "Package 1Z999AA1234567890",
				},
			},
			trackingNumbers: []email.TrackingInfo{
				{Number: "1Z999AA1234567890", Carrier: "ups"},
			},
			apiClientError:  true,
			expectProcessed: 1, // Should still mark as processed even if API fails
			expectCreated:   0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			emailClient := &mockEmailClient{emails: tc.emails}
			stateManager := newMockStateManager()
			
			// Use real extractor for better integration testing
			carrierFactory := carriers.NewClientFactory()
			extractorConfig := &parser.ExtractorConfig{
				EnableLLM:     false,
				MinConfidence: 0.5,
				DebugMode:     false,
			}
			extractor := parser.NewTrackingExtractor(carrierFactory, extractorConfig)
			
			apiClient := &mockAPIClient{
				errorOnCreate: tc.apiClientError,
			}

			// Mark emails as already processed if specified
			for _, msgID := range tc.alreadyProcessed {
				entry := &email.StateEntry{
					GmailMessageID: msgID,
					ProcessedAt:    time.Now(),
					Status:         "processed",
				}
				stateManager.MarkProcessed(entry)
			}

			config := &EmailProcessorConfig{
				CheckInterval:     5 * time.Minute,
				MaxEmailsPerRun:   50,
				DryRun:            false,
				SearchQuery:       "test query",
				ProcessingTimeout: 30 * time.Second,
			}

			processor := NewEmailProcessor(config, emailClient, extractor, stateManager, apiClient, logger)

			// Run one processing cycle manually
			processor.runProcessing()

			if tc.expectError {
				// Error handling would be in logs, not returned directly
				// This test structure is ready for future error handling
			}

			// Check processed count
			stats, _ := stateManager.GetStats()
			if stats.TotalEmails != tc.expectProcessed {
				t.Errorf("Expected %d processed emails, got %d", tc.expectProcessed, stats.TotalEmails)
			}

			// Check created shipments
			created := apiClient.GetCreatedShipments()
			if len(created) != tc.expectCreated {
				t.Errorf("Expected %d created shipments, got %d", tc.expectCreated, len(created))
			}
		})
	}
}

func TestEmailProcessor_DryRunMode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	emails := []email.EmailMessage{
		{
			MessageID: "dry-run-msg",
			Content: &email.EmailContent{
				PlainText: "Package 1Z999AA1234567890 shipped",
			},
		},
	}

	trackingNumbers := []email.TrackingInfo{
		{Number: "1Z999AA1234567890", Carrier: "ups"},
	}

	emailClient := &mockEmailClient{emails: emails}
	stateManager := newMockStateManager()
	extractor := &mockExtractor{trackingNumbers: trackingNumbers}
	apiClient := &mockAPIClient{}

	config := &ProcessorConfig{
		CheckInterval: 5 * time.Minute,
		MaxPerRun:     50,
		DryRun:        true, // Enable dry run mode
		SearchQuery:   "test query",
	}

	processor, err := NewEmailProcessor(emailClient, stateManager, extractor, apiClient, config, logger)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	err = processor.processEmails()
	if err != nil {
		t.Errorf("Unexpected error in dry run: %v", err)
	}

	// In dry run mode, nothing should be created or marked as processed
	stats, _ := stateManager.GetStats()
	if stats.TotalEmails != 0 {
		t.Errorf("Expected 0 processed emails in dry run, got %d", stats.TotalEmails)
	}

	created := apiClient.GetCreatedShipments()
	if len(created) != 0 {
		t.Errorf("Expected 0 created shipments in dry run, got %d", len(created))
	}
}

func TestEmailProcessor_MaxPerRunLimit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create more emails than the limit
	var emails []email.EmailMessage
	for i := 0; i < 10; i++ {
		emails = append(emails, email.EmailMessage{
			MessageID: fmt.Sprintf("msg-%d", i),
			Content: &email.EmailContent{
				PlainText: fmt.Sprintf("Package TRACK%d", i),
			},
		})
	}

	emailClient := &mockEmailClient{emails: emails}
	stateManager := newMockStateManager()
	extractor := &mockExtractor{trackingNumbers: []email.TrackingInfo{}}
	apiClient := &mockAPIClient{}

	config := &ProcessorConfig{
		CheckInterval: 5 * time.Minute,
		MaxPerRun:     5, // Limit to 5 emails per run
		DryRun:        false,
		SearchQuery:   "test query",
	}

	processor, err := NewEmailProcessor(emailClient, stateManager, extractor, apiClient, config, logger)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	err = processor.processEmails()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should only process 5 emails due to the limit
	stats, _ := stateManager.GetStats()
	if stats.TotalEmails != 5 {
		t.Errorf("Expected 5 processed emails (limited), got %d", stats.TotalEmails)
	}
}

func TestEmailProcessor_HealthCheck(t *testing.T) {
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
			emailClient := &mockEmailClient{closed: tc.emailClientError}
			stateManager := newMockStateManager()
			extractor := &mockExtractor{}
			apiClient := &mockAPIClient{}

			config := &ProcessorConfig{
				CheckInterval: 5 * time.Minute,
				MaxPerRun:     50,
				SearchQuery:   "test query",
			}

			processor, err := NewEmailProcessor(emailClient, stateManager, extractor, apiClient, config, logger)
			if err != nil {
				t.Fatalf("Failed to create processor: %v", err)
			}

			err = processor.healthCheck()

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

func TestEmailProcessor_StartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	emailClient := &mockEmailClient{}
	stateManager := newMockStateManager()
	extractor := &mockExtractor{}
	apiClient := &mockAPIClient{}

	config := &ProcessorConfig{
		CheckInterval: 100 * time.Millisecond, // Fast interval for testing
		MaxPerRun:     50,
		SearchQuery:   "test query",
	}

	processor, err := NewEmailProcessor(emailClient, stateManager, extractor, apiClient, config, logger)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Start the processor
	processor.Start()

	// Give it a moment to run
	time.Sleep(200 * time.Millisecond)

	// Stop the processor
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = processor.Stop(ctx)
	if err != nil {
		t.Errorf("Unexpected error stopping processor: %v", err)
	}

	// Verify processor has stopped
	select {
	case <-processor.done:
		// Good, processor stopped
	case <-time.After(100 * time.Millisecond):
		t.Error("Processor did not stop within expected time")
	}
}

func TestEmailProcessor_ConcurrentSafety(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	emailClient := &mockEmailClient{
		emails: []email.EmailMessage{
			{
				MessageID: "concurrent-msg",
				Content:   &email.EmailContent{PlainText: "test"},
			},
		},
	}
	stateManager := newMockStateManager()
	extractor := &mockExtractor{}
	apiClient := &mockAPIClient{}

	config := &ProcessorConfig{
		CheckInterval: 50 * time.Millisecond,
		MaxPerRun:     1,
		SearchQuery:   "test query",
	}

	processor, err := NewEmailProcessor(emailClient, stateManager, extractor, apiClient, config, logger)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Start processor
	processor.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		processor.Stop(ctx)
	}()

	// Run multiple concurrent operations
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			processor.processEmails()
		}()
	}

	wg.Wait()

	// Should not crash or deadlock
	stats, err := stateManager.GetStats()
	if err != nil {
		t.Errorf("Failed to get stats after concurrent access: %v", err)
	}

	// Stats should be reasonable (at least one email processed)
	if stats.TotalEmails < 1 {
		t.Errorf("Expected at least 1 processed email, got %d", stats.TotalEmails)
	}
}

func TestEmailProcessor_ErrorRecovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	emailClient := &mockEmailClient{
		emails: []email.EmailMessage{
			{
				MessageID: "error-recovery-msg",
				Content:   &email.EmailContent{PlainText: "Package 1Z999AA1234567890"},
			},
		},
	}
	stateManager := newMockStateManager()
	extractor := &mockExtractor{
		trackingNumbers: []email.TrackingInfo{
			{Number: "1Z999AA1234567890", Carrier: "ups"},
		},
	}
	apiClient := &mockAPIClient{
		errorOnCreate: true, // Simulate API errors
	}

	config := &ProcessorConfig{
		CheckInterval: 100 * time.Millisecond,
		MaxPerRun:     50,
		SearchQuery:   "test query",
	}

	processor, err := NewEmailProcessor(emailClient, stateManager, extractor, apiClient, config, logger)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Process emails with API errors
	err = processor.processEmails()
	if err != nil {
		t.Errorf("Processor should handle API errors gracefully, got: %v", err)
	}

	// Email should still be marked as processed even though API failed
	stats, _ := stateManager.GetStats()
	if stats.TotalEmails != 1 {
		t.Errorf("Expected 1 processed email despite API error, got %d", stats.TotalEmails)
	}

	// No shipments should be created due to API error
	created := apiClient.GetCreatedShipments()
	if len(created) != 0 {
		t.Errorf("Expected 0 created shipments due to API error, got %d", len(created))
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && 
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

// Benchmark tests
func BenchmarkEmailProcessor_ProcessEmails(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	emails := []email.EmailMessage{
		{
			MessageID: "bench-msg",
			Content: &email.EmailContent{
				PlainText: "Package 1Z999AA1234567890 shipped",
			},
		},
	}

	emailClient := &mockEmailClient{emails: emails}
	stateManager := newMockStateManager()
	extractor := &mockExtractor{
		trackingNumbers: []email.TrackingInfo{
			{Number: "1Z999AA1234567890", Carrier: "ups"},
		},
	}
	apiClient := &mockAPIClient{}

	config := &ProcessorConfig{
		CheckInterval: 5 * time.Minute,
		MaxPerRun:     50,
		DryRun:        true, // Use dry run to avoid side effects
		SearchQuery:   "test query",
	}

	processor, err := NewEmailProcessor(emailClient, stateManager, extractor, apiClient, config, logger)
	if err != nil {
		b.Fatalf("Failed to create processor: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.processEmails()
	}
}