package workers

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"package-tracking/internal/database"
	"package-tracking/internal/email"
)

// MockTwoPhaseEmailClient implements TwoPhaseEmailClient for testing
type MockTwoPhaseEmailClient struct {
	metadataMessages []email.EmailMessage
	fullMessages     map[string]*email.EmailMessage
	healthError      error
}

func NewMockTwoPhaseEmailClient() *MockTwoPhaseEmailClient {
	return &MockTwoPhaseEmailClient{
		metadataMessages: []email.EmailMessage{},
		fullMessages:     make(map[string]*email.EmailMessage),
	}
}

func (m *MockTwoPhaseEmailClient) GetMessage(id string) (*email.EmailMessage, error) {
	if msg, exists := m.fullMessages[id]; exists {
		return msg, nil
	}
	return nil, email.ErrNotFound
}

func (m *MockTwoPhaseEmailClient) GetMessageMetadata(id string) (*email.EmailMessage, error) {
	for _, msg := range m.metadataMessages {
		if msg.ID == id {
			return &msg, nil
		}
	}
	return nil, email.ErrNotFound
}

func (m *MockTwoPhaseEmailClient) GetMessagesSinceMetadataOnly(since time.Time) ([]email.EmailMessage, error) {
	var result []email.EmailMessage
	for _, msg := range m.metadataMessages {
		if msg.Date.After(since) {
			result = append(result, msg)
		}
	}
	return result, nil
}

func (m *MockTwoPhaseEmailClient) HealthCheck() error {
	return m.healthError
}

func (m *MockTwoPhaseEmailClient) Close() error {
	return nil
}

func (m *MockTwoPhaseEmailClient) AddMetadataMessage(msg email.EmailMessage) {
	m.metadataMessages = append(m.metadataMessages, msg)
}

func (m *MockTwoPhaseEmailClient) AddFullMessage(id string, msg *email.EmailMessage) {
	m.fullMessages[id] = msg
}

// TwoPhaseMockTrackingExtractor for testing two-phase processor
type TwoPhaseMockTrackingExtractor struct {
	extractResults map[string][]email.TrackingInfo
}

func NewTwoPhaseMockTrackingExtractor() *TwoPhaseMockTrackingExtractor {
	return &TwoPhaseMockTrackingExtractor{
		extractResults: make(map[string][]email.TrackingInfo),
	}
}

func (m *TwoPhaseMockTrackingExtractor) Extract(content *email.EmailContent) ([]email.TrackingInfo, error) {
	if results, exists := m.extractResults[content.MessageID]; exists {
		return results, nil
	}
	return []email.TrackingInfo{}, nil
}

func (m *TwoPhaseMockTrackingExtractor) SetExtractResult(messageID string, tracking []email.TrackingInfo) {
	m.extractResults[messageID] = tracking
}

// MockAPIClient for testing
type MockAPIClient struct {
	createShipmentError error
	createdShipments    []email.TrackingInfo
}

func NewMockAPIClient() *MockAPIClient {
	return &MockAPIClient{
		createdShipments: []email.TrackingInfo{},
	}
}

func (m *MockAPIClient) CreateShipment(tracking email.TrackingInfo) error {
	if m.createShipmentError != nil {
		return m.createShipmentError
	}
	m.createdShipments = append(m.createdShipments, tracking)
	return nil
}

func (m *MockAPIClient) GetCreatedShipments() []email.TrackingInfo {
	return m.createdShipments
}

func (m *MockAPIClient) HealthCheck() error {
	return nil
}

func TestTwoPhaseEmailProcessor_ProcessEmailsSince(t *testing.T) {
	// Create temporary database for testing  
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())
	
	db, err := database.Open(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	emailStore := database.NewEmailStore(db.DB)
	
	// Create mock dependencies
	emailClient := NewMockTwoPhaseEmailClient()
	extractor := NewTwoPhaseMockTrackingExtractor()
	apiClient := NewMockAPIClient()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Add test emails to mock client
	now := time.Now()
	
	// High relevance email with tracking
	highRelevanceEmail := email.EmailMessage{
		ID:       "msg1",
		ThreadID: "thread1",
		From:     "shipping@amazon.com",
		Subject:  "Your package has shipped",
		Snippet:  "UPS tracking number 1Z999AA1234567890",
		Date:     now.Add(-1 * time.Hour),
	}
	emailClient.AddMetadataMessage(highRelevanceEmail)
	
	// Add full content for the high relevance email
	fullEmail := &email.EmailMessage{
		ID:        "msg1",
		ThreadID:  "thread1",
		From:      "shipping@amazon.com",
		Subject:   "Your package has shipped",
		PlainText: "Your order has shipped via UPS. Tracking: 1Z999AA1234567890",
		HTMLText:  "<p>Your order has shipped via UPS. Tracking: 1Z999AA1234567890</p>",
		Date:      now.Add(-1 * time.Hour),
	}
	emailClient.AddFullMessage("msg1", fullEmail)
	
	// Set up extractor to find tracking number
	extractor.SetExtractResult("msg1", []email.TrackingInfo{
		{
			Number:  "1Z999AA1234567890",
			Carrier: "ups",
		},
	})

	// Low relevance email
	lowRelevanceEmail := email.EmailMessage{
		ID:      "msg2",
		From:    "newsletter@example.com",
		Subject: "Weekly deals",
		Snippet: "Check out our latest promotions",
		Date:    now.Add(-30 * time.Minute),
	}
	emailClient.AddMetadataMessage(lowRelevanceEmail)

	// Create processor
	config := &TwoPhaseEmailProcessorConfig{
		ScanDays:              7,
		MaxEmailsPerScan:      100,
		RelevanceThreshold:    0.3,
		MetadataOnlyBatchSize: 50,
		ContentBatchSize:      10,
		MaxContentExtractions: 20,
		BodyStorageEnabled:    true,
		DryRun:                false,
		RetryCount:            1,
		RetryDelay:            100 * time.Millisecond,
		RetentionDays:         30,
	}

	processor := NewTwoPhaseEmailProcessor(
		config,
		emailClient,
		extractor,
		emailStore,
		nil, // shipmentStore not needed for this test
		apiClient,
		logger,
		nil, // factory not needed for this test (would need mock)
		nil, // cacheManager not needed for this test
		nil, // rateLimiter not needed for this test
	)

	// Test processing
	since := now.Add(-2 * time.Hour)
	
	// This will fail on validation since we don't have a carrier factory mock
	// But we can test the metadata processing part
	err = processor.processPhase1MetadataOnly(since)
	if err != nil {
		t.Fatalf("Phase 1 processing failed: %v", err)
	}

	// Check metrics
	metrics := processor.GetMetrics()
	if metrics.MetadataEmailsScanned != 2 {
		t.Errorf("Expected 2 emails scanned, got %d", metrics.MetadataEmailsScanned)
	}

	if metrics.MetadataEmailsStored != 2 {
		t.Errorf("Expected 2 emails stored, got %d", metrics.MetadataEmailsStored)
	}

	// Check that emails were stored in database
	storedEmail, err := emailStore.GetByGmailMessageID("msg1")
	if err != nil {
		t.Fatalf("Failed to get stored email: %v", err)
	}

	if storedEmail.ProcessingPhase != "metadata_only" {
		t.Errorf("Expected processing phase 'metadata_only', got '%s'", storedEmail.ProcessingPhase)
	}

	if storedEmail.RelevanceScore < 0.3 {
		t.Errorf("Expected high relevance score for shipping email, got %f", storedEmail.RelevanceScore)
	}

	// Check low relevance email
	lowRelevanceStored, err := emailStore.GetByGmailMessageID("msg2")
	if err != nil {
		t.Fatalf("Failed to get low relevance email: %v", err)
	}

	if lowRelevanceStored.RelevanceScore > 0.3 {
		t.Errorf("Expected low relevance score for newsletter, got %f", lowRelevanceStored.RelevanceScore)
	}
}

func TestTwoPhaseEmailProcessor_RelevanceScoring(t *testing.T) {
	processor := NewTwoPhaseEmailProcessor(
		&TwoPhaseEmailProcessorConfig{},
		nil, nil, nil, nil, nil,
		slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		nil, nil, nil,
	)

	scorer := processor.GetRelevanceScorer()

	tests := []struct {
		name        string
		email       *email.EmailMessage
		expectHigh  bool
		expectLow   bool
	}{
		{
			name: "Amazon shipping email",
			email: &email.EmailMessage{
				From:    "auto-confirm@amazon.com",
				Subject: "Your package has shipped",
				Snippet: "UPS tracking 1Z999AA1234567890",
			},
			expectHigh: true,
		},
		{
			name: "Newsletter email",
			email: &email.EmailMessage{
				From:    "newsletter@store.com",
				Subject: "Weekly deals and offers",
				Snippet: "Check out our latest promotions",
			},
			expectLow: true,
		},
		{
			name: "Order confirmation",
			email: &email.EmailMessage{
				From:    "orders@shop.com",
				Subject: "Order confirmation #12345",
				Snippet: "Thank you for your order",
			},
			expectHigh: false,
			expectLow:  false, // Medium relevance
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scorer.CalculateRelevanceScore(tt.email)
			
			if tt.expectHigh && score < scorer.GetHighConfidenceThreshold() {
				t.Errorf("Expected high confidence score (>= %f), got %f", 
					scorer.GetHighConfidenceThreshold(), score)
			}
			
			if tt.expectLow && score >= scorer.GetRelevanceThreshold() {
				t.Errorf("Expected low relevance score (< %f), got %f", 
					scorer.GetRelevanceThreshold(), score)
			}
			
			t.Logf("%s: score = %f", tt.name, score)
		})
	}
}

func TestTwoPhaseEmailProcessor_Configuration(t *testing.T) {
	config := &TwoPhaseEmailProcessorConfig{
		ScanDays:              7,
		MaxEmailsPerScan:      100,
		RelevanceThreshold:    0.4,
		MetadataOnlyBatchSize: 50,
		ContentBatchSize:      10,
		MaxContentExtractions: 20,
		BodyStorageEnabled:    true,
		DryRun:                false,
		RetryCount:            3,
		RetryDelay:            500 * time.Millisecond,
		RetentionDays:         30,
	}

	processor := NewTwoPhaseEmailProcessor(
		config,
		nil, nil, nil, nil, nil,
		slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		nil, nil, nil,
	)

	// Test that configuration is properly stored
	if processor.config.RelevanceThreshold != 0.4 {
		t.Errorf("Expected relevance threshold 0.4, got %f", processor.config.RelevanceThreshold)
	}

	if processor.config.MaxEmailsPerScan != 100 {
		t.Errorf("Expected max emails per scan 100, got %d", processor.config.MaxEmailsPerScan)
	}

	// Test that relevance scorer uses the correct threshold
	scorer := processor.GetRelevanceScorer()
	if scorer.GetRelevanceThreshold() != 0.3 { // Default threshold from scorer
		t.Errorf("Expected default relevance threshold 0.3, got %f", scorer.GetRelevanceThreshold())
	}
}

func BenchmarkTwoPhaseEmailProcessor_RelevanceScoring(b *testing.B) {
	processor := NewTwoPhaseEmailProcessor(
		&TwoPhaseEmailProcessorConfig{},
		nil, nil, nil, nil, nil,
		slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		nil, nil, nil,
	)

	scorer := processor.GetRelevanceScorer()
	
	email := &email.EmailMessage{
		From:    "shipping@amazon.com",
		Subject: "Your package has shipped via UPS",
		Snippet: "Your order has been shipped. Tracking number: 1Z999AA1234567890",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scorer.CalculateRelevanceScore(email)
	}
}