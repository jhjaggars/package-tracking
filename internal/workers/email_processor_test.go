package workers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"package-tracking/internal/api"
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
		if msg.MessageID == id {
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

func (m *mockStateManager) MarkProcessed(messageID string, tracking []email.TrackingInfo) error {
	if m.errorOnMark {
		return fmt.Errorf("mock mark processed error")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processed[messageID] = true
	return nil
}

func (m *mockStateManager) GetProcessedEmails(since time.Time, limit int) ([]email.ProcessedEmail, error) {
	return nil, nil
}

func (m *mockStateManager) GetStats() (*email.ProcessingStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &email.ProcessingStats{
		TotalEmails: len(m.processed),
	}, nil
}

func (m *mockStateManager) CleanupOldEntries(maxAge time.Duration) (int, error) {
	return 0, nil
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
		config      *ProcessorConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid configuration",
			config: &ProcessorConfig{
				CheckInterval: 5 * time.Minute,
				MaxPerRun:     50,
				DryRun:        false,
				DebugMode:     false,
				SearchQuery:   "from:ups.com",
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
			config: &ProcessorConfig{
				CheckInterval: 0,
				MaxPerRun:     50,
			},
			expectError: true,
			errorMsg:    "check interval must be positive",
		},
		{
			name: "Invalid max per run",
			config: &ProcessorConfig{
				CheckInterval: 5 * time.Minute,
				MaxPerRun:     0,
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
			processor, err := NewEmailProcessor(
				emailClient,
				stateManager,
				extractor,
				apiClient,
				tc.config,
				logger,
			)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				} else if tc.errorMsg != "" && !containsString(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %v", tc.errorMsg, err)
				}
				if processor != nil {
					t.Errorf("Expected nil processor on error, but got: %v", processor)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
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
					MessageID: "msg-001",
					Content: &email.EmailContent{
						PlainText: "Your UPS package 1Z999AA1234567890 has shipped",
						From:      "noreply@ups.com",
						Subject:   "UPS Shipment Notification",
					},
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
					MessageID: "msg-002",
					Content: &email.EmailContent{
						PlainText: "Your package has shipped",
						From:      "test@example.com",
					},
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
					MessageID: "msg-003",
					Content: &email.EmailContent{
						PlainText: "Thank you for your order",
						From:      "orders@example.com",
					},
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
					MessageID: "msg-004",
					Content: &email.EmailContent{
						PlainText: "Test email",
					},
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
					MessageID: "msg-005",
					Content: &email.EmailContent{
						PlainText: "Package 1Z999AA1234567890",
					},
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
			extractor := &mockExtractor{
				trackingNumbers: tc.trackingNumbers,
				errorOnExtract:  tc.extractorError,
			}
			apiClient := &mockAPIClient{
				errorOnCreate: tc.apiClientError,
			}

			// Mark emails as already processed if specified
			for _, msgID := range tc.alreadyProcessed {
				stateManager.MarkProcessed(msgID, []email.TrackingInfo{})
			}

			config := &ProcessorConfig{
				CheckInterval: 5 * time.Minute,
				MaxPerRun:     50,
				DryRun:        false,
				SearchQuery:   "test query",
			}

			processor, err := NewEmailProcessor(emailClient, stateManager, extractor, apiClient, config, logger)
			if err != nil {
				t.Fatalf("Failed to create processor: %v", err)
			}

			// Run one processing cycle
			err = processor.processEmails()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
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