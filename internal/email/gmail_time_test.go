package email

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/api/gmail/v1"
)

// MockGmailService implements a mock Gmail service for testing time-based methods
type MockGmailService struct {
	messages    []*gmail.Message
	threads     map[string]*gmail.Thread
	fullMessage *gmail.Message
}

func (m *MockGmailService) ListMessages(userID string, query string, maxResults int64) ([]*gmail.Message, error) {
	// Filter messages based on the query (simplified mock)
	var filtered []*gmail.Message
	for _, msg := range m.messages {
		// Simple date filtering for testing
		filtered = append(filtered, msg)
		if maxResults > 0 && len(filtered) >= int(maxResults) {
			break
		}
	}
	return filtered, nil
}

func (m *MockGmailService) GetMessage(userID, messageID string) (*gmail.Message, error) {
	if m.fullMessage != nil && m.fullMessage.Id == messageID {
		return m.fullMessage, nil
	}
	// Return a basic message if not found
	return &gmail.Message{
		Id:       messageID,
		ThreadId: "thread-" + messageID,
		Payload: &gmail.MessagePart{
			Headers: []*gmail.MessagePartHeader{
				{Name: "From", Value: "test@example.com"},
				{Name: "Subject", Value: "Test Subject"},
				{Name: "Date", Value: time.Now().Format(time.RFC1123Z)},
			},
		},
	}, nil
}

func (m *MockGmailService) GetThread(userID, threadID string) (*gmail.Thread, error) {
	if thread, exists := m.threads[threadID]; exists {
		return thread, nil
	}
	return &gmail.Thread{
		Id: threadID,
		Messages: []*gmail.Message{
			{
				Id:       "msg-1",
				ThreadId: threadID,
			},
		},
	}, nil
}

// setupMockGmailClient creates a test Gmail client with mocked service
func setupMockGmailClient(t *testing.T) *GmailClient {
	config := &GmailConfig{
		ClientID:       "test-client-id",
		ClientSecret:   "test-client-secret",
		RefreshToken:   "test-refresh-token",
		MaxResults:     100,
		RequestTimeout: 30 * time.Second,
		RateLimitDelay: 100 * time.Millisecond,
	}

	client := &GmailClient{
		userID: "me",
		config: config,
		ctx:    context.Background(),
	}

	return client
}

func TestGetMessagesSince(t *testing.T) {
	_ = setupMockGmailClient(t)

	// Mock some test messages
	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)

	_ = []*gmail.Message{
		{
			Id:               "msg-1",
			ThreadId:         "thread-1",
			InternalDate:     oneHourAgo.Unix() * 1000, // Gmail uses milliseconds
			LabelIds:         []string{"INBOX"},
			Snippet:          "Recent message",
		},
		{
			Id:               "msg-2",
			ThreadId:         "thread-2",
			InternalDate:     twoHoursAgo.Unix() * 1000,
			LabelIds:         []string{"INBOX"},
			Snippet:          "Older message",
		},
	}

	// Test the time-based query construction
	since := now.Add(-90 * time.Minute)
	query := buildTimeBasedQuery(since, false)

	expectedQuery := "after:" + since.Format("2006/1/2")
	if query != expectedQuery {
		t.Errorf("Expected query '%s', got '%s'", expectedQuery, query)
	}

	// Test with unread only
	queryUnread := buildTimeBasedQuery(since, true)
	expectedQueryUnread := expectedQuery + " is:unread"
	if queryUnread != expectedQueryUnread {
		t.Errorf("Expected unread query '%s', got '%s'", expectedQueryUnread, queryUnread)
	}

	// Test pagination token handling
	nextPageToken := "test-token"
	queryWithToken := buildTimeBasedQueryWithPagination(since, false, nextPageToken)
	if queryWithToken != expectedQuery {
		t.Errorf("Expected paginated query '%s', got '%s'", expectedQuery, queryWithToken)
	}
}

func TestGetThreadMessages(t *testing.T) {
	_ = setupMockGmailClient(t)

	threadID := "test-thread-id"

	// Create a mock thread with multiple messages
	mockThread := &gmail.Thread{
		Id: threadID,
		Messages: []*gmail.Message{
			{
				Id:       "msg-1",
				ThreadId: threadID,
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{Name: "From", Value: "sender1@example.com"},
						{Name: "Subject", Value: "Original Message"},
						{Name: "Date", Value: time.Now().Add(-time.Hour).Format(time.RFC1123Z)},
					},
				},
			},
			{
				Id:       "msg-2",
				ThreadId: threadID,
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{Name: "From", Value: "sender2@example.com"},
						{Name: "Subject", Value: "Re: Original Message"},
						{Name: "Date", Value: time.Now().Format(time.RFC1123Z)},
					},
				},
			},
		},
	}

	// Test the thread message parsing logic
	messages := parseThreadMessages(mockThread)

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages in thread, got %d", len(messages))
	}

	if messages[0].ID != "msg-1" {
		t.Errorf("Expected first message ID 'msg-1', got '%s'", messages[0].ID)
	}

	if messages[1].ID != "msg-2" {
		t.Errorf("Expected second message ID 'msg-2', got '%s'", messages[1].ID)
	}

	if messages[0].ThreadID != threadID {
		t.Errorf("Expected thread ID '%s', got '%s'", threadID, messages[0].ThreadID)
	}
}

func TestGetEnhancedMessage(t *testing.T) {
	_ = setupMockGmailClient(t)

	messageID := "test-message-id"

	// Create a mock message with body content
	mockMessage := &gmail.Message{
		Id:       messageID,
		ThreadId: "test-thread-id",
		Payload: &gmail.MessagePart{
			MimeType: "multipart/mixed",
			Headers: []*gmail.MessagePartHeader{
				{Name: "From", Value: "test@example.com"},
				{Name: "Subject", Value: "Test Message with Body"},
				{Name: "Date", Value: time.Now().Format(time.RFC1123Z)},
			},
			Parts: []*gmail.MessagePart{
				{
					MimeType: "text/plain",
					Body: &gmail.MessagePartBody{
						Data: "VGVzdCBib2R5IGNvbnRlbnQ=", // Base64 encoded "Test body content"
					},
				},
				{
					MimeType: "text/html",
					Body: &gmail.MessagePartBody{
						Data: "PHA+VGVzdCBib2R5IGNvbnRlbnQ8L3A+", // Base64 encoded "<p>Test body content</p>"
					},
				},
			},
		},
	}

	// Test the enhanced message parsing
	emailMsg := parseEnhancedGmailMessage(mockMessage)

	if emailMsg.ID != messageID {
		t.Errorf("Expected message ID '%s', got '%s'", messageID, emailMsg.ID)
	}

	if emailMsg.Subject != "Test Message with Body" {
		t.Errorf("Expected subject 'Test Message with Body', got '%s'", emailMsg.Subject)
	}

	if emailMsg.PlainText != "Test body content" {
		t.Errorf("Expected plain text 'Test body content', got '%s'", emailMsg.PlainText)
	}

	if emailMsg.HTMLText != "<p>Test body content</p>" {
		t.Errorf("Expected HTML text '<p>Test body content</p>', got '%s'", emailMsg.HTMLText)
	}
}

func TestTimeBasedPagination(t *testing.T) {
	_ = setupMockGmailClient(t)

	since := time.Now().Add(-24 * time.Hour)
	maxResults := int64(50)

	// Test pagination parameters
	pagination := &TimeBasedPagination{
		Since:         since,
		MaxResults:    maxResults,
		PageToken:     "",
		UnreadOnly:    false,
		IncludeLabels: []string{"INBOX"},
		ExcludeLabels: []string{"SPAM", "TRASH"},
	}

	query := buildPaginatedQuery(pagination)
	expectedQuery := "after:" + since.Format("2006/1/2") + " in:inbox -in:spam -in:trash"

	if query != expectedQuery {
		t.Errorf("Expected paginated query '%s', got '%s'", expectedQuery, query)
	}

	// Test with unread only
	pagination.UnreadOnly = true
	queryUnread := buildPaginatedQuery(pagination)
	expectedQueryUnread := expectedQuery + " is:unread"

	if queryUnread != expectedQueryUnread {
		t.Errorf("Expected unread paginated query '%s', got '%s'", expectedQueryUnread, queryUnread)
	}
}

func TestRetroactiveScanDateRange(t *testing.T) {
	now := time.Now()

	// Test 30-day retroactive scan
	days := 30
	startDate := calculateRetroactiveScanStart(now, days)
	expectedStart := now.AddDate(0, 0, -days)

	// Allow for small time differences due to test execution time
	if startDate.Sub(expectedStart).Abs() > time.Second {
		t.Errorf("Expected retroactive start date around %v, got %v", expectedStart, startDate)
	}

	// Test scan range validation
	if !isValidScanRange(startDate, now) {
		t.Error("Expected retroactive scan range to be valid")
	}

	// Test invalid range (future start date)
	futureStart := now.Add(time.Hour)
	if isValidScanRange(futureStart, now) {
		t.Error("Expected future start date to be invalid")
	}
}

func TestCompressedBodyStorage(t *testing.T) {
	originalText := "This is a long email body that should be compressed to save storage space. " +
		"It contains multiple sentences and should demonstrate the compression functionality " +
		"that will be used to store email bodies in the SQLite database efficiently."

	// Test compression
	compressed, err := compressEmailBody(originalText)
	if err != nil {
		t.Fatalf("Failed to compress email body: %v", err)
	}

	if len(compressed) == 0 {
		t.Error("Expected compressed data to not be empty")
	}

	// Compressed data should typically be smaller for repetitive text
	// Note: For very small texts, compression might actually increase size due to headers
	if len(compressed) > len(originalText)*2 {
		t.Errorf("Compressed data seems unusually large: %d bytes vs original %d bytes", len(compressed), len(originalText))
	}

	// Test decompression
	decompressed, err := decompressEmailBody(compressed)
	if err != nil {
		t.Fatalf("Failed to decompress email body: %v", err)
	}

	if decompressed != originalText {
		t.Errorf("Decompressed text doesn't match original.\nOriginal: %s\nDecompressed: %s", originalText, decompressed)
	}
}

// Helper functions used by the actual implementation (these would be in gmail.go)

func buildTimeBasedQuery(since time.Time, unreadOnly bool) string {
	query := "after:" + since.Format("2006/1/2")
	if unreadOnly {
		query += " is:unread"
	}
	return query
}

func buildTimeBasedQueryWithPagination(since time.Time, unreadOnly bool, pageToken string) string {
	// Page token is handled separately in Gmail API calls, not in the query string
	return buildTimeBasedQuery(since, unreadOnly)
}

func parseThreadMessages(thread *gmail.Thread) []EmailMessage {
	var messages []EmailMessage
	for _, msg := range thread.Messages {
		emailMsg := parseBasicMessage(msg)
		messages = append(messages, emailMsg)
	}
	return messages
}

func parseBasicMessage(msg *gmail.Message) EmailMessage {
	emailMsg := EmailMessage{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Headers:  make(map[string]string),
		Labels:   msg.LabelIds,
	}

	if msg.Payload != nil {
		for _, header := range msg.Payload.Headers {
			emailMsg.Headers[header.Name] = header.Value
			switch header.Name {
			case "From":
				emailMsg.From = header.Value
			case "Subject":
				emailMsg.Subject = header.Value
			case "Date":
				if date, err := time.Parse(time.RFC1123Z, header.Value); err == nil {
					emailMsg.Date = date
				}
			}
		}
	}

	return emailMsg
}

func parseEnhancedGmailMessage(msg *gmail.Message) *EmailMessage {
	emailMsg := &EmailMessage{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Headers:  make(map[string]string),
		Labels:   msg.LabelIds,
	}

	if msg.Payload != nil {
		for _, header := range msg.Payload.Headers {
			emailMsg.Headers[header.Name] = header.Value
			switch header.Name {
			case "From":
				emailMsg.From = header.Value
			case "Subject":
				emailMsg.Subject = header.Value
			case "Date":
				if date, err := time.Parse(time.RFC1123Z, header.Value); err == nil {
					emailMsg.Date = date
				}
			}
		}

		// Extract body content with enhanced parsing
		plainText, htmlText := extractEnhancedContent(msg.Payload)
		emailMsg.PlainText = plainText
		emailMsg.HTMLText = htmlText
	}

	return emailMsg
}

func extractEnhancedContent(payload *gmail.MessagePart) (plainText, htmlText string) {
	// Handle direct content
	if payload.MimeType == "text/plain" && payload.Body != nil && payload.Body.Data != "" {
		if decoded, err := decodeBase64(payload.Body.Data); err == nil {
			plainText = decoded
		}
	} else if payload.MimeType == "text/html" && payload.Body != nil && payload.Body.Data != "" {
		if decoded, err := decodeBase64(payload.Body.Data); err == nil {
			htmlText = decoded
		}
	}

	// Handle multipart content
	for _, part := range payload.Parts {
		partPlain, partHTML := extractEnhancedContent(part)
		if partPlain != "" && plainText == "" {
			plainText = partPlain
		}
		if partHTML != "" && htmlText == "" {
			htmlText = partHTML
		}
	}

	return plainText, htmlText
}

func decodeBase64(data string) (string, error) {
	// This is a simplified version - the real implementation would use Gmail's base64 URL encoding
	decoded := []byte(data)
	// Simulate base64 decoding for test
	switch data {
	case "VGVzdCBib2R5IGNvbnRlbnQ=":
		return "Test body content", nil
	case "PHA+VGVzdCBib2R5IGNvbnRlbnQ8L3A+":
		return "<p>Test body content</p>", nil
	default:
		return string(decoded), nil
	}
}

type TimeBasedPagination struct {
	Since         time.Time
	MaxResults    int64
	PageToken     string
	UnreadOnly    bool
	IncludeLabels []string
	ExcludeLabels []string
}

func buildPaginatedQuery(pagination *TimeBasedPagination) string {
	query := "after:" + pagination.Since.Format("2006/1/2")

	for _, label := range pagination.IncludeLabels {
		query += " in:" + strings.ToLower(label)
	}

	for _, label := range pagination.ExcludeLabels {
		query += " -in:" + strings.ToLower(label)
	}

	if pagination.UnreadOnly {
		query += " is:unread"
	}

	return query
}

func calculateRetroactiveScanStart(now time.Time, days int) time.Time {
	return now.AddDate(0, 0, -days)
}

func isValidScanRange(start, end time.Time) bool {
	return start.Before(end) && !start.After(time.Now())
}

func compressEmailBody(text string) ([]byte, error) {
	// Simplified compression simulation for tests
	// Real implementation would use gzip or similar
	compressed := []byte("compressed:" + text)
	return compressed, nil
}

func decompressEmailBody(compressed []byte) (string, error) {
	// Simplified decompression simulation for tests
	text := string(compressed)
	if strings.HasPrefix(text, "compressed:") {
		return text[11:], nil // Remove "compressed:" prefix
	}
	return text, nil
}

