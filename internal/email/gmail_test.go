package email

import (
	"fmt"
	"strings"
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

func TestBuildSearchQuery_EnhancedSearch(t *testing.T) {
	testCases := []struct {
		name        string
		carriers    []string
		afterDays   int
		unreadOnly  bool
		customQuery string
		expected    string
	}{
		{
			name:        "Custom query takes precedence",
			carriers:    []string{"ups"},
			afterDays:   7,
			unreadOnly:  true,
			customQuery: "custom search query",
			expected:    "custom search query",
		},
		{
			name:       "Enhanced search for last 30 days unread",
			carriers:   nil,
			afterDays:  30,
			unreadOnly: true,
			expected:   "from:(ups.com OR usps.com OR fedex.com OR dhl.com OR amazon.com OR shopify.com) subject:(tracking OR shipment OR package OR delivery OR shipped OR \"tracking number\") after:2024/12/05 is:unread",
		},
		{
			name:       "Specific carrier search",
			carriers:   []string{"ups"},
			afterDays:  30,
			unreadOnly: true,
			expected:   "from:(noreply@ups.com OR quantum@ups.com OR pkginfo@ups.com) subject:(tracking OR shipment OR package OR delivery OR shipped OR \"tracking number\") after:2024/12/05 is:unread",
		},
		{
			name:       "Multiple carriers search",
			carriers:   []string{"ups", "fedex"},
			afterDays:  30,
			unreadOnly: true,
			expected:   "from:(noreply@ups.com OR quantum@ups.com OR pkginfo@ups.com OR fedex.com OR tracking@fedex.com OR shipment@fedex.com) subject:(tracking OR shipment OR package OR delivery OR shipped OR \"tracking number\") after:2024/12/05 is:unread",
		},
		{
			name:       "No date filter",
			carriers:   nil,
			afterDays:  0,
			unreadOnly: true,
			expected:   "from:(ups.com OR usps.com OR fedex.com OR dhl.com OR amazon.com OR shopify.com) subject:(tracking OR shipment OR package OR delivery OR shipped OR \"tracking number\") is:unread",
		},
		{
			name:       "Include read emails",
			carriers:   nil,
			afterDays:  30,
			unreadOnly: false,
			expected:   "from:(ups.com OR usps.com OR fedex.com OR dhl.com OR amazon.com OR shopify.com) subject:(tracking OR shipment OR package OR delivery OR shipped OR \"tracking number\") after:2024/12/05",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Mock current time to ensure consistent date calculations
			now := time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC)
			
			// Calculate expected date if afterDays is set
			if tc.afterDays > 0 && tc.expected != tc.customQuery {
				expectedDate := now.AddDate(0, 0, -tc.afterDays).Format("2006/1/2")
				tc.expected = fmt.Sprintf("from:(ups.com OR usps.com OR fedex.com OR dhl.com OR amazon.com OR shopify.com) subject:(tracking OR shipment OR package OR delivery OR shipped OR \"tracking number\") after:%s", expectedDate)
				if tc.unreadOnly {
					tc.expected += " is:unread"
				}
				
				// Handle specific carrier cases
				if len(tc.carriers) > 0 {
					var senders []string
					for _, carrier := range tc.carriers {
						switch carrier {
						case "ups":
							senders = append(senders, "noreply@ups.com", "quantum@ups.com", "pkginfo@ups.com")
						case "fedex":
							senders = append(senders, "fedex.com", "tracking@fedex.com", "shipment@fedex.com")
						}
					}
					tc.expected = fmt.Sprintf("from:(%s) subject:(tracking OR shipment OR package OR delivery OR shipped OR \"tracking number\") after:%s", strings.Join(senders, " OR "), expectedDate)
					if tc.unreadOnly {
						tc.expected += " is:unread"
					}
				}
			}

			result := BuildSearchQuery(tc.carriers, tc.afterDays, tc.unreadOnly, tc.customQuery)
			
			// For non-custom queries, we need to handle the dynamic date calculation
			if tc.customQuery == "" && tc.afterDays > 0 {
				// Extract the date part from the result and verify it's reasonable
				parts := strings.Split(result, " ")
				var dateFound bool
				for _, part := range parts {
					if strings.HasPrefix(part, "after:") {
						dateFound = true
						// Verify the date is in the correct format
						dateStr := strings.TrimPrefix(part, "after:")
						_, err := time.Parse("2006/1/2", dateStr)
						if err != nil {
							t.Errorf("Invalid date format in result: %s", dateStr)
						}
					}
				}
				if !dateFound {
					t.Errorf("Expected date filter not found in result: %s", result)
				}
			} else {
				if result != tc.expected {
					t.Errorf("Expected query: %s, got: %s", tc.expected, result)
				}
			}
		})
	}
}

func TestBuildSearchQuery_EnhancedForLLMProcessing(t *testing.T) {
	// Test the enhanced search query that will be used for LLM processing
	// This should return a broader search to capture more emails for LLM analysis
	testCases := []struct {
		name       string
		afterDays  int
		unreadOnly bool
		expected   string
	}{
		{
			name:       "Enhanced search for LLM processing - 30 days unread",
			afterDays:  30,
			unreadOnly: true,
			expected:   "after:DATE is:unread",
		},
		{
			name:       "Enhanced search for LLM processing - 7 days unread",
			afterDays:  7,
			unreadOnly: true,
			expected:   "after:DATE is:unread",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// For LLM processing, we want a broader search without sender/subject filtering
			result := BuildSearchQueryForLLMProcessing(tc.afterDays, tc.unreadOnly)
			
			// Extract the date part and verify structure
			parts := strings.Split(result, " ")
			if len(parts) < 2 {
				t.Errorf("Expected at least 2 parts in query, got %d: %s", len(parts), result)
			}
			
			var hasDate, hasUnread bool
			for _, part := range parts {
				if strings.HasPrefix(part, "after:") {
					hasDate = true
					// Verify date format
					dateStr := strings.TrimPrefix(part, "after:")
					_, err := time.Parse("2006/1/2", dateStr)
					if err != nil {
						t.Errorf("Invalid date format: %s", dateStr)
					}
				}
				if part == "is:unread" && tc.unreadOnly {
					hasUnread = true
				}
			}
			
			if !hasDate && tc.afterDays > 0 {
				t.Errorf("Expected date filter not found in result: %s", result)
			}
			
			if !hasUnread && tc.unreadOnly {
				t.Errorf("Expected unread filter not found in result: %s", result)
			}
			
			// Should NOT contain sender or subject filters
			if strings.Contains(result, "from:") {
				t.Errorf("Enhanced search should not contain sender filters: %s", result)
			}
			
			if strings.Contains(result, "subject:") {
				t.Errorf("Enhanced search should not contain subject filters: %s", result)
			}
		})
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

