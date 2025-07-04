package email

import (
	"fmt"
	"testing"
	"time"
)

func TestGmailConfig_Validation(t *testing.T) {
	testCases := []struct {
		name        string
		config      *GmailConfig
		expectError bool
	}{
		{
			name: "Valid minimal config",
			config: &GmailConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
				RefreshToken: "test-refresh-token",
			},
			expectError: false,
		},
		{
			name: "Missing client ID",
			config: &GmailConfig{
				ClientSecret: "test-secret",
				RefreshToken: "test-refresh-token",
			},
			expectError: true,
		},
		{
			name: "Missing client secret",
			config: &GmailConfig{
				ClientID:     "test-client-id",
				RefreshToken: "test-refresh-token",
			},
			expectError: true,
		},
		{
			name: "Missing refresh token",
			config: &GmailConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
			},
			expectError: true,
		},
		{
			name:        "Nil config",
			config:      nil,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateGmailConfig(tc.config)

			if tc.expectError && err == nil {
				t.Errorf("Expected error, but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestEmailMessage_Structure(t *testing.T) {
	// Test that EmailMessage has correct structure
	msg := EmailMessage{
		ID:        "test-id",
		ThreadID:  "test-thread",
		From:      "test@example.com",
		Subject:   "Test Subject",
		Date:      time.Now(),
		Headers:   map[string]string{"X-Test": "value"},
		PlainText: "Plain text content",
		HTMLText:  "<p>HTML content</p>",
		Labels:    []string{"INBOX", "UNREAD"},
	}

	if msg.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", msg.ID)
	}

	if msg.From != "test@example.com" {
		t.Errorf("Expected From 'test@example.com', got '%s'", msg.From)
	}

	if len(msg.Headers) != 1 {
		t.Errorf("Expected 1 header, got %d", len(msg.Headers))
	}

	if len(msg.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(msg.Labels))
	}
}

func TestEmailContent_Structure(t *testing.T) {
	// Test that EmailContent has correct structure
	content := EmailContent{
		PlainText: "Plain text",
		HTMLText:  "<p>HTML</p>",
		Subject:   "Test Subject",
		From:      "test@example.com",
		Headers:   map[string]string{"Content-Type": "text/html"},
		MessageID: "msg-123",
		ThreadID:  "thread-456",
		Date:      time.Now(),
	}

	if content.MessageID != "msg-123" {
		t.Errorf("Expected MessageID 'msg-123', got '%s'", content.MessageID)
	}

	if content.ThreadID != "thread-456" {
		t.Errorf("Expected ThreadID 'thread-456', got '%s'", content.ThreadID)
	}
}

func TestTrackingInfo_Structure(t *testing.T) {
	// Test that TrackingInfo has correct structure
	tracking := TrackingInfo{
		Number:      "1Z999AA1234567890",
		Carrier:     "ups",
		Description: "Package from Amazon",
		Confidence:  0.9,
		Source:      "regex",
		Context:     "tracking number: 1Z999AA1234567890",
		ExtractedAt: time.Now(),
	}

	if tracking.Number != "1Z999AA1234567890" {
		t.Errorf("Expected Number '1Z999AA1234567890', got '%s'", tracking.Number)
	}

	if tracking.Carrier != "ups" {
		t.Errorf("Expected Carrier 'ups', got '%s'", tracking.Carrier)
	}

	if tracking.Confidence != 0.9 {
		t.Errorf("Expected Confidence 0.9, got %f", tracking.Confidence)
	}
}

func TestCarrierHint_Structure(t *testing.T) {
	// Test that CarrierHint has correct structure
	hint := CarrierHint{
		Carrier:    "ups",
		Confidence: 0.8,
		Source:     "sender",
		Reason:     "From ups.com domain",
	}

	if hint.Carrier != "ups" {
		t.Errorf("Expected Carrier 'ups', got '%s'", hint.Carrier)
	}

	if hint.Confidence != 0.8 {
		t.Errorf("Expected Confidence 0.8, got %f", hint.Confidence)
	}
}

func TestTrackingCandidate_Structure(t *testing.T) {
	// Test that TrackingCandidate has correct structure
	candidate := TrackingCandidate{
		Text:       "1Z999AA1234567890",
		Position:   25,
		Context:    "tracking number: 1Z999AA1234567890",
		Carrier:    "ups",
		Confidence: 0.9,
		Method:     "direct",
	}

	if candidate.Text != "1Z999AA1234567890" {
		t.Errorf("Expected Text '1Z999AA1234567890', got '%s'", candidate.Text)
	}

	if candidate.Position != 25 {
		t.Errorf("Expected Position 25, got %d", candidate.Position)
	}
}

// Helper function for config validation
func validateGmailConfig(config *GmailConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.ClientID == "" {
		return fmt.Errorf("client ID is required")
	}

	if config.ClientSecret == "" {
		return fmt.Errorf("client secret is required")
	}

	if config.RefreshToken == "" {
		return fmt.Errorf("refresh token is required")
	}

	return nil
}

func TestBuildSearchQuery_BroadenedSearch(t *testing.T) {
	testCases := []struct {
		name         string
		carriers     []string
		afterDays    int
		unreadOnly   bool
		customQuery  string
		expectFields []string
		expectNot    []string
	}{
		{
			name:         "broadened search - no carrier filtering",
			carriers:     []string{},
			afterDays:    30,
			unreadOnly:   true,
			customQuery:  "",
			expectFields: []string{"after:", "is:unread"},
			expectNot:    []string{"from:(ups.com", "from:(usps.com"},
		},
		{
			name:         "date-based unread emails only",
			carriers:     []string{},
			afterDays:    30,
			unreadOnly:   true,
			customQuery:  "",
			expectFields: []string{"after:", "is:unread"},
			expectNot:    []string{"subject:(tracking"},
		},
		{
			name:         "custom query overrides everything",
			carriers:     []string{"ups"},
			afterDays:    30,
			unreadOnly:   true,
			customQuery:  "after:2024/12/05 is:unread",
			expectFields: []string{"after:2024/12/05", "is:unread"},
			expectNot:    []string{"from:(ups.com", "subject:(tracking"},
		},
		{
			name:         "legacy mode with carriers",
			carriers:     []string{"ups", "usps"},
			afterDays:    7,
			unreadOnly:   false,
			customQuery:  "",
			expectFields: []string{"from:(", "subject:(tracking", "after:"},
			expectNot:    []string{"is:unread"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := BuildSearchQuery(tc.carriers, tc.afterDays, tc.unreadOnly, tc.customQuery)
			
			// Check expected fields are present
			for _, field := range tc.expectFields {
				if !containsSubstring(query, field) {
					t.Errorf("Expected query to contain '%s', but got: %s", field, query)
				}
			}
			
			// Check fields that should NOT be present
			for _, field := range tc.expectNot {
				if containsSubstring(query, field) {
					t.Errorf("Expected query NOT to contain '%s', but got: %s", field, query)
				}
			}
		})
	}
}

func TestBuildSearchQuery_BroadenedSearchDates(t *testing.T) {
	// Test specific date calculations
	testCases := []struct {
		name      string
		afterDays int
		expected  string
	}{
		{
			name:      "30 days ago",
			afterDays: 30,
			expected:  time.Now().AddDate(0, 0, -30).Format("2006/1/2"),
		},
		{
			name:      "7 days ago",
			afterDays: 7,
			expected:  time.Now().AddDate(0, 0, -7).Format("2006/1/2"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := BuildSearchQuery([]string{}, tc.afterDays, true, "")
			expectedDate := fmt.Sprintf("after:%s", tc.expected)
			
			if !containsSubstring(query, expectedDate) {
				t.Errorf("Expected query to contain '%s', but got: %s", expectedDate, query)
			}
		})
	}
}

func TestBuildSearchQuery_EnhancedMode(t *testing.T) {
	// Test the new enhanced mode that should be used for issue #45
	query := BuildSearchQuery([]string{}, 30, true, "")
	
	// Should contain date filter
	expectedDate := time.Now().AddDate(0, 0, -30).Format("2006/1/2")
	if !containsSubstring(query, fmt.Sprintf("after:%s", expectedDate)) {
		t.Errorf("Expected query to contain after:%s", expectedDate)
	}
	
	// Should contain unread filter
	if !containsSubstring(query, "is:unread") {
		t.Errorf("Expected query to contain 'is:unread'")
	}
	
	// Should NOT contain restrictive carrier filters
	if containsSubstring(query, "from:(ups.com") {
		t.Errorf("Expected query NOT to contain carrier-specific from: filters")
	}
	
	// Should NOT contain restrictive subject filters
	if containsSubstring(query, "subject:(tracking") {
		t.Errorf("Expected query NOT to contain restrictive subject filters")
	}
}

// Helper function for substring checking
func containsSubstring(text, substr string) bool {
	if len(substr) > len(text) {
		return false
	}
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

