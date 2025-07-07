package workers

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"package-tracking/internal/database"
	"package-tracking/internal/email"
)

// MockTimeBasedEmailClient implements a mock email client for time-based scanning tests
type MockTimeBasedEmailClient struct {
	messages      []email.EmailMessage
	threadMessages map[string][]email.EmailMessage
	shouldError   bool
	callLog       []string
}

func (m *MockTimeBasedEmailClient) GetMessagesSince(since time.Time) ([]email.EmailMessage, error) {
	m.callLog = append(m.callLog, "GetMessagesSince")
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	
	// Filter messages based on time
	var filtered []email.EmailMessage
	for _, msg := range m.messages {
		if msg.Date.After(since) || msg.Date.Equal(since) {
			filtered = append(filtered, msg)
		}
	}
	return filtered, nil
}

func (m *MockTimeBasedEmailClient) GetEnhancedMessage(id string) (*email.EmailMessage, error) {
	m.callLog = append(m.callLog, "GetEnhancedMessage")
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	
	for _, msg := range m.messages {
		if msg.ID == id {
			return &msg, nil
		}
	}
	return nil, fmt.Errorf("message not found")
}

func (m *MockTimeBasedEmailClient) GetThreadMessages(threadID string) ([]email.EmailMessage, error) {
	m.callLog = append(m.callLog, "GetThreadMessages")
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	
	if messages, exists := m.threadMessages[threadID]; exists {
		return messages, nil
	}
	return []email.EmailMessage{}, nil
}

func (m *MockTimeBasedEmailClient) PerformRetroactiveScan(days int) ([]email.EmailMessage, error) {
	m.callLog = append(m.callLog, "PerformRetroactiveScan")
	since := time.Now().AddDate(0, 0, -days)
	return m.GetMessagesSince(since)
}

// Legacy methods for backward compatibility
func (m *MockTimeBasedEmailClient) Search(query string) ([]email.EmailMessage, error) {
	m.callLog = append(m.callLog, "Search")
	return m.messages, nil
}

func (m *MockTimeBasedEmailClient) GetMessage(id string) (*email.EmailMessage, error) {
	return m.GetEnhancedMessage(id)
}

func (m *MockTimeBasedEmailClient) HealthCheck() error {
	if m.shouldError {
		return fmt.Errorf("mock health check error")
	}
	return nil
}

func (m *MockTimeBasedEmailClient) Close() error {
	return nil
}

// MockTimeBasedStateManager implements state management for time-based processing
type MockTimeBasedStateManager struct {
	processedEmails map[string]*email.StateEntry
	shouldError     bool
	callLog         []string
}

func (m *MockTimeBasedStateManager) IsProcessed(messageID string) (bool, error) {
	m.callLog = append(m.callLog, "IsProcessed")
	if m.shouldError {
		return false, fmt.Errorf("mock error")
	}
	_, exists := m.processedEmails[messageID]
	return exists, nil
}

func (m *MockTimeBasedStateManager) MarkProcessed(entry *email.StateEntry) error {
	m.callLog = append(m.callLog, "MarkProcessed")
	if m.shouldError {
		return fmt.Errorf("mock error")
	}
	m.processedEmails[entry.GmailMessageID] = entry
	return nil
}

func (m *MockTimeBasedStateManager) StoreEmailBody(messageID string, bodyText, bodyHTML string, compressed []byte) error {
	m.callLog = append(m.callLog, "StoreEmailBody")
	if m.shouldError {
		return fmt.Errorf("mock error")
	}
	// In a real implementation, this would store the email body
	return nil
}

func (m *MockTimeBasedStateManager) LinkEmailToShipment(messageID string, shipmentID int, trackingNumber string) error {
	m.callLog = append(m.callLog, "LinkEmailToShipment")
	if m.shouldError {
		return fmt.Errorf("mock error")
	}
	// In a real implementation, this would create the link
	return nil
}

func (m *MockTimeBasedStateManager) CreateOrUpdateThread(threadID, subject string, participants []string, messageCount int, firstDate, lastDate time.Time) error {
	m.callLog = append(m.callLog, "CreateOrUpdateThread")
	if m.shouldError {
		return fmt.Errorf("mock error")
	}
	// In a real implementation, this would store thread data
	return nil
}

func (m *MockTimeBasedStateManager) Cleanup(olderThan time.Time) error {
	m.callLog = append(m.callLog, "Cleanup")
	return nil
}

func (m *MockTimeBasedStateManager) GetStats() (*email.EmailMetrics, error) {
	return &email.EmailMetrics{
		TotalEmails:     len(m.processedEmails),
		ProcessedEmails: len(m.processedEmails),
	}, nil
}

// MockTrackingExtractor for testing
type MockTrackingExtractor struct{}

func (m *MockTrackingExtractor) Extract(content *email.EmailContent) ([]email.TrackingInfo, error) {
	// Simple mock: if content contains "TEST123456789", return it as tracking info
	if strings.Contains(content.PlainText, "TEST123456789") {
		return []email.TrackingInfo{
			{
				Number:   "TEST123456789",
				Carrier:  "ups",
				Source:   "mock",
				Context:  "test",
			},
		}, nil
	}
	return []email.TrackingInfo{}, nil
}

func setupTimeBasedProcessor(t *testing.T) (*TimeBasedEmailProcessor, *MockTimeBasedEmailClient, *database.DB) {
	client := &MockTimeBasedEmailClient{
		messages:      []email.EmailMessage{},
		threadMessages: make(map[string][]email.EmailMessage),
		shouldError:   false,
		callLog:       []string{},
	}

	// Create a mock tracking extractor
	extractor := &MockTrackingExtractor{}

	config := &TimeBasedEmailProcessorConfig{
		ScanDays:          30,
		BodyStorageEnabled: true,
		RetentionDays:     90,
		MaxEmailsPerScan:  100,
		UnreadOnly:        false,
		CheckInterval:     5 * time.Minute,
		ProcessingTimeout: 30 * time.Minute,
		RetryCount:        3,
		RetryDelay:        time.Second,
	}

	// Create real database for testing
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create a simple logger for tests
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	processor := NewTimeBasedEmailProcessor(
		config,
		client,
		extractor,
		db.Emails,
		db.Shipments,
		nil, // No API client for these tests
		logger,
	)

	return processor, client, db
}

func TestTimeBasedEmailProcessor_ProcessEmailsSince(t *testing.T) {
	processor, client, db := setupTimeBasedProcessor(t)
	defer db.Close()

	// Set up test emails
	now := time.Now()
	testEmails := []email.EmailMessage{
		{
			ID:        "msg-1",
			ThreadID:  "thread-1",
			From:      "test@example.com",
			Subject:   "Package shipped",
			Date:      now.Add(-time.Hour),
			PlainText: "Your package TEST123456789 has been shipped",
			HTMLText:  "<p>Your package <strong>TEST123456789</strong> has been shipped</p>",
		},
		{
			ID:        "msg-2",
			ThreadID:  "thread-2",
			From:      "fedex@example.com",
			Subject:   "FedEx delivery notification",
			Date:      now.Add(-2 * time.Hour),
			PlainText: "FedEx tracking number: TEST123456789",
			HTMLText:  "<p>FedEx tracking number: <strong>TEST123456789</strong></p>",
		},
	}

	client.messages = testEmails

	// Test processing emails since 3 hours ago
	since := now.Add(-3 * time.Hour)
	err := processor.ProcessEmailsSince(since)
	if err != nil {
		t.Fatalf("ProcessEmailsSince failed: %v", err)
	}

	// Verify that GetMessagesSince was called
	if !contains(client.callLog, "GetMessagesSince") {
		t.Error("Expected GetMessagesSince to be called")
	}

	// Verify emails were processed by checking the database
	emails, err := db.Emails.GetByGmailMessageID("msg-1")
	if err != nil {
		t.Fatalf("Failed to get email from database: %v", err)
	}
	if emails == nil {
		t.Error("Expected msg-1 to be processed and stored in database")
	}

	emails2, err := db.Emails.GetByGmailMessageID("msg-2")
	if err != nil {
		t.Fatalf("Failed to get email from database: %v", err)
	}
	if emails2 == nil {
		t.Error("Expected msg-2 to be processed and stored in database")
	}
}

func TestTimeBasedEmailProcessor_PerformRetroactiveScan(t *testing.T) {
	processor, client, db := setupTimeBasedProcessor(t)
	defer db.Close()

	// Set up test emails spanning different time periods
	now := time.Now()
	testEmails := []email.EmailMessage{
		{
			ID:        "msg-recent",
			ThreadID:  "thread-1",
			Date:      now.Add(-5 * time.Hour),
			PlainText: "Recent email with tracking 1Z999AA1234567890",
		},
		{
			ID:        "msg-old",
			ThreadID:  "thread-2",
			Date:      now.Add(-25 * 24 * time.Hour), // 25 days ago
			PlainText: "Old email with tracking 1234567890123456",
		},
		{
			ID:        "msg-very-old",
			ThreadID:  "thread-3",
			Date:      now.Add(-35 * 24 * time.Hour), // 35 days ago
			PlainText: "Very old email with tracking 9876543210987654",
		},
	}

	client.messages = testEmails

	// Test retroactive scan for 30 days
	err := processor.PerformRetroactiveScan()
	if err != nil {
		t.Fatalf("PerformRetroactiveScan failed: %v", err)
	}

	// Verify that PerformRetroactiveScan was called
	if !contains(client.callLog, "PerformRetroactiveScan") {
		t.Error("Expected PerformRetroactiveScan to be called")
	}

	// Verify only emails within the 30-day window were processed
	// Should process msg-recent and msg-old, but not msg-very-old
	recentEmail, err := db.Emails.GetByGmailMessageID("msg-recent")
	if err != nil || recentEmail == nil {
		t.Error("Expected msg-recent to be processed")
	}

	oldEmail, err := db.Emails.GetByGmailMessageID("msg-old")
	if err != nil || oldEmail == nil {
		t.Error("Expected msg-old to be processed")
	}

	// Very old email should not be processed
	veryOldEmail, _ := db.Emails.GetByGmailMessageID("msg-very-old")
	if veryOldEmail != nil {
		t.Error("Expected msg-very-old NOT to be processed")
	}
}

func TestTimeBasedEmailProcessor_ProcessEmailWithBodyStorage(t *testing.T) {
	processor, client, db := setupTimeBasedProcessor(t)
	defer db.Close()

	// Set up test email with body content
	testEmail := email.EmailMessage{
		ID:        "msg-with-body",
		ThreadID:  "thread-1",
		From:      "test@example.com",
		Subject:   "Email with body content",
		Date:      time.Now(),
		PlainText: "This is the plain text body with tracking number 1Z999AA1234567890",
		HTMLText:  "<p>This is the <strong>HTML body</strong> with tracking number <em>1Z999AA1234567890</em></p>",
	}

	client.messages = []email.EmailMessage{testEmail}

	// Process the email
	since := time.Now().Add(-time.Hour)
	err := processor.ProcessEmailsSince(since)
	if err != nil {
		t.Fatalf("ProcessEmailsSince failed: %v", err)
	}

	// Verify email was processed and body was stored
	processedEmail, err := db.Emails.GetByGmailMessageID("msg-with-body")
	if err != nil {
		t.Fatalf("Failed to get processed email: %v", err)
	}
	if processedEmail == nil {
		t.Error("Expected email with body to be processed")
	} else {
		// Verify body content was stored
		if processedEmail.BodyText == "" && len(processedEmail.BodyCompressed) == 0 {
			t.Error("Expected email body to be stored")
		}
	}
}

func TestTimeBasedEmailProcessor_ThreadProcessing(t *testing.T) {
	processor, client, db := setupTimeBasedProcessor(t)
	defer db.Close()

	// Set up thread with multiple messages
	threadID := "thread-conversation"
	threadMessages := []email.EmailMessage{
		{
			ID:        "msg-1",
			ThreadID:  threadID,
			From:      "sender@example.com",
			Subject:   "Package order",
			Date:      time.Now().Add(-2 * time.Hour),
			PlainText: "Your order has been placed",
		},
		{
			ID:        "msg-2",
			ThreadID:  threadID,
			From:      "carrier@example.com",
			Subject:   "Re: Package order",
			Date:      time.Now().Add(-time.Hour),
			PlainText: "Your package 1Z999AA1234567890 is on the way",
		},
	}

	client.messages = threadMessages
	client.threadMessages[threadID] = threadMessages

	// Process emails
	since := time.Now().Add(-3 * time.Hour)
	err := processor.ProcessEmailsSince(since)
	if err != nil {
		t.Fatalf("ProcessEmailsSince failed: %v", err)
	}

	// Verify thread was created
	thread, err := db.Emails.GetThreadByGmailThreadID(threadID)
	if err != nil {
		t.Fatalf("Failed to get thread: %v", err)
	}
	if thread == nil {
		t.Error("Expected thread to be created")
	}

	// Verify both messages in thread were processed
	msg1, err := db.Emails.GetByGmailMessageID("msg-1")
	if err != nil || msg1 == nil {
		t.Error("Expected msg-1 to be processed")
	}

	msg2, err := db.Emails.GetByGmailMessageID("msg-2")
	if err != nil || msg2 == nil {
		t.Error("Expected msg-2 to be processed")
	}
}

func TestTimeBasedEmailProcessor_ErrorHandling(t *testing.T) {
	processor, client, db := setupTimeBasedProcessor(t)
	defer db.Close()

	// Test client error
	client.shouldError = true
	since := time.Now().Add(-time.Hour)
	err := processor.ProcessEmailsSince(since)
	if err == nil {
		t.Error("Expected error when client fails")
	}

	// Reset client and test with valid data (no state manager to fail in real implementation)
	client.shouldError = false

	client.messages = []email.EmailMessage{
		{
			ID:       "test-msg",
			ThreadID: "test-thread",
			Date:     time.Now(),
			PlainText: "Test message",
		},
	}

	// This should now succeed since client error is reset
	err = processor.ProcessEmailsSince(since)
	if err != nil {
		t.Errorf("Expected success after client error reset, got: %v", err)
	}
}

func TestTimeBasedEmailProcessor_DuplicateDetection(t *testing.T) {
	processor, client, db := setupTimeBasedProcessor(t)
	defer db.Close()

	testEmail := email.EmailMessage{
		ID:       "duplicate-msg",
		ThreadID: "duplicate-thread",
		Date:     time.Now(),
		PlainText: "Test email",
	}

	client.messages = []email.EmailMessage{testEmail}

	// Store email as already processed in database
	alreadyProcessedEmail := &database.EmailBodyEntry{
		GmailMessageID:    "duplicate-msg",
		GmailThreadID:     "duplicate-thread", 
		From:              "test@example.com",
		Subject:           "Test email",
		Date:              time.Now(),
		BodyText:          "Test email",
		InternalTimestamp: time.Now(),
		ScanMethod:        "time-based",
		ProcessedAt:       time.Now().Add(-time.Hour),
		Status:            "processed",
	}
	err := db.Emails.CreateOrUpdate(alreadyProcessedEmail)
	if err != nil {
		t.Fatalf("Failed to create already processed email: %v", err)
	}

	// Process emails - should skip the duplicate
	since := time.Now().Add(-time.Hour)
	err = processor.ProcessEmailsSince(since)
	if err != nil {
		t.Fatalf("ProcessEmailsSince failed: %v", err)
	}

	// Verify duplicate was detected by checking the email wasn't updated
	// Get the email from database and verify it wasn't updated recently
	processedEmail, err := db.Emails.GetByGmailMessageID("duplicate-msg")
	if err != nil {
		t.Fatalf("Failed to get processed email: %v", err)
	}
	if processedEmail == nil {
		t.Error("Expected email to exist in database")
	} else {
		// Email should still have the old processed time (not updated)
		if processedEmail.ProcessedAt.After(time.Now().Add(-30 * time.Minute)) {
			t.Error("Email appears to have been reprocessed (ProcessedAt was updated)")
		}
	}
}

func TestTimeBasedEmailProcessor_ConfigurationHandling(t *testing.T) {
	// Test with body storage disabled
	processor, client, db := setupTimeBasedProcessor(t)
	defer db.Close()
	processor.config.BodyStorageEnabled = false

	testEmail := email.EmailMessage{
		ID:        "config-test-msg",
		ThreadID:  "config-test-thread",
		Date:      time.Now(),
		PlainText: "Test email for config",
		HTMLText:  "<p>Test email for config</p>",
	}

	client.messages = []email.EmailMessage{testEmail}

	// Process emails
	since := time.Now().Add(-time.Hour)
	err := processor.ProcessEmailsSince(since)
	if err != nil {
		t.Fatalf("ProcessEmailsSince failed: %v", err)
	}

	// Verify email was processed but body was not stored when body storage is disabled
	processedEmail, err := db.Emails.GetByGmailMessageID("config-test-msg")
	if err != nil {
		t.Fatalf("Failed to get processed email: %v", err)
	}
	if processedEmail == nil {
		t.Error("Expected email to be processed even with body storage disabled")
	} else {
		// Verify body was NOT stored when disabled
		if processedEmail.BodyText != "" || len(processedEmail.BodyCompressed) > 0 {
			t.Error("Expected email body NOT to be stored when body storage is disabled")
		}
	}
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func countOccurrences(slice []string, item string) int {
	count := 0
	for _, s := range slice {
		if s == item {
			count++
		}
	}
	return count
}


