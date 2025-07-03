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

