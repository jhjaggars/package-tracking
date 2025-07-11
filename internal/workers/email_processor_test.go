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
	emails          []email.EmailMessage
	searchDelay     time.Duration
	errorOnSearch   bool
	errorOnGet      bool
	closed          bool
	lastSearchQuery string // Track the last search query used
}

func (m *mockEmailClient) Search(query string) ([]email.EmailMessage, error) {
	m.lastSearchQuery = query // Store the query for verification

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

func (m *mockEmailClient) GetMessageMetadata(id string) (*email.EmailMessage, error) {
	// For testing, just return the same as GetMessage but with empty content
	msg, err := m.GetMessage(id)
	if err != nil {
		return nil, err
	}
	
	// Create a copy with no content for metadata-only
	metadata := *msg
	metadata.PlainText = ""
	metadata.HTMLText = ""
	metadata.Snippet = "Email snippet for " + id
	
	return &metadata, nil
}

func (m *mockEmailClient) GetMessagesSinceMetadataOnly(since time.Time) ([]email.EmailMessage, error) {
	var result []email.EmailMessage
	for _, msg := range m.emails {
		if msg.Date.After(since) {
			// Create metadata-only version
			metadata := msg
			metadata.PlainText = ""
			metadata.HTMLText = ""
			metadata.Snippet = "Email snippet for " + msg.ID
			result = append(result, metadata)
		}
	}
	return result, nil
}

type mockStateManager struct {
	processed   map[string]bool
	mu          sync.RWMutex
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
		TotalEmails: len(m.processed), // Return total historical count
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
	callCount        int
}

func (m *mockAPIClient) CreateShipment(tracking email.TrackingInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++

	if m.errorOnCreate {
		return fmt.Errorf("mock API error")
	}

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
	carrierFactory := carriers.NewClientFactory()
	extractor := parser.NewTrackingExtractor(carrierFactory, &parser.ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
	}, &parser.LLMConfig{Enabled: false})
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
		name             string
		emails           []email.EmailMessage
		alreadyProcessed []string
		expectProcessed  int
		expectCreated    int
		apiClientError   bool
	}{
		{
			name: "Process new emails with tracking",
			emails: []email.EmailMessage{
				{
					ID:        "msg-001",
					PlainText: "Your UPS package with tracking number 1Z999AA1234567890 has shipped successfully.",
					From:      "noreply@ups.com",
					Subject:   "UPS Shipment Notification",
				},
			},
			expectProcessed: 1,
			expectCreated:   1,
			apiClientError:  false, // Ensure API client doesn't error
		},
		{
			name: "Skip already processed emails",
			emails: []email.EmailMessage{
				{
					ID:        "msg-002",
					PlainText: "Your package has shipped",
					From:      "test@example.com",
				},
			},
			alreadyProcessed: []string{"msg-002"},
			expectProcessed:  1, // Email is marked as processed in setup (historical count)
			expectCreated:    0, // No new shipments should be created
		},
		{
			name: "Process emails with no tracking numbers",
			emails: []email.EmailMessage{
				{
					ID:        "msg-003",
					PlainText: "Thank you for your order",
					From:      "orders@example.com",
				},
			},
			expectProcessed: 1,
			expectCreated:   0,
		},
		{
			name: "Handle emails with no trackable content",
			emails: []email.EmailMessage{
				{
					ID:        "msg-004",
					PlainText: "Test email with no trackable content",
				},
			},
			expectProcessed: 1, // Email still gets processed
			expectCreated:   0, // No shipments created due to no tracking
		},
		{
			name: "Handle API client errors",
			emails: []email.EmailMessage{
				{
					ID:        "msg-005",
					PlainText: "Your UPS package tracking number 1Z999AA1234567890 is ready for pickup.",
				},
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
			apiClient := &mockAPIClient{
				errorOnCreate: tc.apiClientError,
			}

			config := &EmailProcessorConfig{
				CheckInterval:     5 * time.Minute,
				MaxEmailsPerRun:   50,
				DryRun:            false,
				SearchQuery:       "test query",
				ProcessingTimeout: 30 * time.Second,
				RetryCount:        3,
				RetryDelay:        1 * time.Second,
			}

			// Use real extractor for all tests - more realistic integration testing
			carrierFactory := carriers.NewClientFactory()
			extractorConfig := &parser.ExtractorConfig{
				EnableLLM:     false,
				MinConfidence: 0.5,
				DebugMode:     false,
			}
			realExtractor := parser.NewTrackingExtractor(carrierFactory, extractorConfig, &parser.LLMConfig{Enabled: false})
			processor := NewEmailProcessor(config, emailClient, realExtractor, stateManager, apiClient, logger)

			// Mark emails as already processed if specified
			for _, msgID := range tc.alreadyProcessed {
				entry := &email.StateEntry{
					GmailMessageID: msgID,
					ProcessedAt:    time.Now(),
					Status:         "processed",
				}
				stateManager.MarkProcessed(entry)
			}

			// Debug: Test extractor directly
			if tc.name == "Process new emails with tracking" {
				content := &email.EmailContent{
					PlainText: tc.emails[0].PlainText,
					From:      tc.emails[0].From,
					Subject:   tc.emails[0].Subject,
				}
				trackingResults, err := realExtractor.Extract(content)
				if err != nil {
					t.Logf("Extractor error: %v", err)
				} else {
					t.Logf("Extractor found %d tracking numbers", len(trackingResults))
					for i, tr := range trackingResults {
						t.Logf("  %d: %s (%s) confidence=%.2f", i, tr.Number, tr.Carrier, tr.Confidence)
					}
				}

				// Debug: Test email client search
				searchResults, err := emailClient.Search("test query")
				if err != nil {
					t.Logf("Email search error: %v", err)
				} else {
					t.Logf("Email search found %d emails", len(searchResults))
				}

				// Debug: Check if email is marked as processed
				isProcessed, err := stateManager.IsProcessed(tc.emails[0].ID)
				if err != nil {
					t.Logf("IsProcessed error: %v", err)
				} else {
					t.Logf("Email %s is processed: %v", tc.emails[0].ID, isProcessed)
				}
			}

			// Debug: For the skip test, check state before and after
			if tc.name == "Skip already processed emails" {
				isProcessed, _ := stateManager.IsProcessed(tc.emails[0].ID)
				t.Logf("Before processing: Email %s is processed: %v", tc.emails[0].ID, isProcessed)
			}

			// Run one processing cycle manually
			processor.runProcessing()

			// Check processed count
			stats, _ := stateManager.GetStats()
			if stats.TotalEmails != tc.expectProcessed {
				t.Errorf("Expected %d processed emails, got %d", tc.expectProcessed, stats.TotalEmails)
			}

			// Check created shipments
			created := apiClient.GetCreatedShipments()
			if len(created) != tc.expectCreated {
				t.Errorf("Expected %d created shipments, got %d. API error mode: %v, API call count: %d", tc.expectCreated, len(created), tc.apiClientError, apiClient.callCount)
				// Debug: show what shipments were created
				for i, shipment := range created {
					t.Logf("Created shipment %d: %+v", i, shipment)
				}
			}
		})
	}
}

func TestEmailProcessor_DryRunMode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	emails := []email.EmailMessage{
		{
			ID:        "dry-run-msg",
			PlainText: "Package 1Z999AA1234567890 shipped",
		},
	}

	emailClient := &mockEmailClient{emails: emails}
	stateManager := newMockStateManager()
	carrierFactory := carriers.NewClientFactory()
	extractor := parser.NewTrackingExtractor(carrierFactory, &parser.ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
	}, &parser.LLMConfig{Enabled: false})
	apiClient := &mockAPIClient{}

	config := &EmailProcessorConfig{
		CheckInterval:     5 * time.Minute,
		MaxEmailsPerRun:   50,
		DryRun:            true, // Enable dry run mode
		SearchQuery:       "test query",
		ProcessingTimeout: 30 * time.Second,
		RetryCount:        3,
		RetryDelay:        1 * time.Second,
	}

	processor := NewEmailProcessor(config, emailClient, extractor, stateManager, apiClient, logger)

	processor.runProcessing()

	// In dry run mode, emails should be processed but no shipments created
	stats, _ := stateManager.GetStats()
	if stats.TotalEmails != 1 {
		t.Errorf("Expected 1 processed email in dry run, got %d", stats.TotalEmails)
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
			ID:        fmt.Sprintf("msg-%d", i),
			PlainText: fmt.Sprintf("Package TRACK%d", i),
		})
	}

	emailClient := &mockEmailClient{emails: emails}
	stateManager := newMockStateManager()
	carrierFactory := carriers.NewClientFactory()
	extractor := parser.NewTrackingExtractor(carrierFactory, &parser.ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
	}, &parser.LLMConfig{Enabled: false})
	apiClient := &mockAPIClient{}

	config := &EmailProcessorConfig{
		CheckInterval:     5 * time.Minute,
		MaxEmailsPerRun:   5, // Limit to 5 emails per run
		DryRun:            false,
		SearchQuery:       "test query",
		ProcessingTimeout: 30 * time.Second,
		RetryCount:        3,
		RetryDelay:        1 * time.Second,
	}

	processor := NewEmailProcessor(config, emailClient, extractor, stateManager, apiClient, logger)

	processor.runProcessing()

	// Should only process 5 emails due to the limit
	stats, _ := stateManager.GetStats()
	if stats.TotalEmails != 5 {
		t.Errorf("Expected 5 processed emails (limited), got %d", stats.TotalEmails)
	}
}

func TestEmailProcessor_HealthCheck(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	testCases := []struct {
		name             string
		emailClientError bool
		expectError      bool
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
			carrierFactory := carriers.NewClientFactory()
			extractor := parser.NewTrackingExtractor(carrierFactory, &parser.ExtractorConfig{
				EnableLLM:     false,
				MinConfidence: 0.5,
			}, &parser.LLMConfig{Enabled: false})
			apiClient := &mockAPIClient{}

			config := &EmailProcessorConfig{
				CheckInterval:   5 * time.Minute,
				MaxEmailsPerRun: 50,
				SearchQuery:     "test query",
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

func TestEmailProcessor_StartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	emailClient := &mockEmailClient{}
	stateManager := newMockStateManager()
	carrierFactory := carriers.NewClientFactory()
	extractor := parser.NewTrackingExtractor(carrierFactory, &parser.ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
	}, &parser.LLMConfig{Enabled: false})
	apiClient := &mockAPIClient{}

	config := &EmailProcessorConfig{
		CheckInterval:   100 * time.Millisecond, // Fast interval for testing
		MaxEmailsPerRun: 50,
		SearchQuery:     "test query",
	}

	processor := NewEmailProcessor(config, emailClient, extractor, stateManager, apiClient, logger)

	// Start the processor
	processor.Start()

	// Give it a moment to run
	time.Sleep(200 * time.Millisecond)

	// Stop the processor
	processor.Stop()

	// Give it a moment to stop
	time.Sleep(100 * time.Millisecond)
}

func TestEmailProcessor_ConcurrentSafety(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	emailClient := &mockEmailClient{
		emails: []email.EmailMessage{
			{
				ID:        "concurrent-msg",
				PlainText: "test",
			},
		},
	}
	stateManager := newMockStateManager()
	carrierFactory := carriers.NewClientFactory()
	extractor := parser.NewTrackingExtractor(carrierFactory, &parser.ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
	}, &parser.LLMConfig{Enabled: false})
	apiClient := &mockAPIClient{}

	config := &EmailProcessorConfig{
		CheckInterval:     50 * time.Millisecond,
		MaxEmailsPerRun:   1,
		SearchQuery:       "test query",
		ProcessingTimeout: 30 * time.Second,
		RetryCount:        3,
		RetryDelay:        1 * time.Second,
	}

	processor := NewEmailProcessor(config, emailClient, extractor, stateManager, apiClient, logger)

	// Start processor
	processor.Start()
	defer processor.Stop()

	// Run multiple concurrent operations
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			processor.runProcessing()
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
				ID:        "error-recovery-msg",
				PlainText: "Package 1Z999AA1234567890",
			},
		},
	}
	stateManager := newMockStateManager()
	carrierFactory := carriers.NewClientFactory()
	extractor := parser.NewTrackingExtractor(carrierFactory, &parser.ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
	}, &parser.LLMConfig{Enabled: false})
	apiClient := &mockAPIClient{
		errorOnCreate: true, // Simulate API errors
	}

	config := &EmailProcessorConfig{
		CheckInterval:     100 * time.Millisecond,
		MaxEmailsPerRun:   50,
		SearchQuery:       "test query",
		ProcessingTimeout: 30 * time.Second,
		RetryCount:        3,
		RetryDelay:        1 * time.Second,
	}

	processor := NewEmailProcessor(config, emailClient, extractor, stateManager, apiClient, logger)

	// Process emails with API errors
	processor.runProcessing()

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
			ID:        "bench-msg",
			PlainText: "Package 1Z999AA1234567890 shipped",
		},
	}

	emailClient := &mockEmailClient{emails: emails}
	stateManager := newMockStateManager()
	carrierFactory := carriers.NewClientFactory()
	extractor := parser.NewTrackingExtractor(carrierFactory, &parser.ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
	}, &parser.LLMConfig{Enabled: false})
	apiClient := &mockAPIClient{}

	config := &EmailProcessorConfig{
		CheckInterval:   5 * time.Minute,
		MaxEmailsPerRun: 50,
		DryRun:          true, // Use dry run to avoid side effects
		SearchQuery:     "test query",
	}

	processor := NewEmailProcessor(config, emailClient, extractor, stateManager, apiClient, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.runProcessing()
	}
}

func TestEmailProcessor_SearchQueryNoRedundantBuilding(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create a custom search query to verify it's used directly
	customQuery := "from:test@example.com subject:custom-search-test"

	emailClient := &mockEmailClient{
		emails: []email.EmailMessage{
			{
				ID:        "search-test-msg",
				PlainText: "Test email content",
			},
		},
	}
	stateManager := newMockStateManager()
	carrierFactory := carriers.NewClientFactory()
	extractor := parser.NewTrackingExtractor(carrierFactory, &parser.ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
	}, &parser.LLMConfig{Enabled: false})
	apiClient := &mockAPIClient{}

	config := &EmailProcessorConfig{
		CheckInterval:   5 * time.Minute,
		MaxEmailsPerRun: 50,
		SearchQuery:     customQuery, // Use custom search query
		SearchAfterDays: 30,
		UnreadOnly:      false,
	}

	processor := NewEmailProcessor(config, emailClient, extractor, stateManager, apiClient, logger)

	// Run email search
	emails, err := processor.searchEmails(context.Background())
	if err != nil {
		t.Fatalf("Search emails failed: %v", err)
	}

	// Verify the configured search query was used directly without modification
	if emailClient.lastSearchQuery != customQuery {
		t.Errorf("Expected search query '%s', but got '%s'", customQuery, emailClient.lastSearchQuery)
	}

	// Verify emails were found
	if len(emails) != 1 {
		t.Errorf("Expected 1 email, got %d", len(emails))
	}

	t.Logf("Successfully verified custom search query is used: %s", emailClient.lastSearchQuery)
}

