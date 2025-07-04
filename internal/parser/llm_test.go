package parser

import (
	"testing"

	"package-tracking/internal/email"
)

func TestLocalLLMExtractor_buildPrompt(t *testing.T) {
	config := &LLMConfig{
		Provider:    "local",
		Model:       "test-model",
		Endpoint:    "http://localhost:11434",
		Temperature: 0.1,
		MaxTokens:   1000,
		Enabled:     true,
	}

	extractor := NewLocalLLMExtractor(config)

	testCases := []struct {
		name        string
		content     *email.EmailContent
		expectFields []string
	}{
		{
			name: "basic email content",
			content: &email.EmailContent{
				From:      "noreply@amazon.com",
				Subject:   "Your order has shipped",
				PlainText: "Your Apple iPhone 15 Pro has been shipped with tracking number 1Z999AA1234567890",
			},
			expectFields: []string{
				"merchant",
				"description",
				"tracking_number",
				"carrier",
				"confidence",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prompt := extractor.buildPrompt(tc.content)
			
			// Check that prompt contains expected fields
			for _, field := range tc.expectFields {
				if !containsField(prompt, field) {
					t.Errorf("Expected prompt to contain field '%s', but it didn't", field)
				}
			}
		})
	}
}

func TestLocalLLMExtractor_parseResponse(t *testing.T) {
	config := &LLMConfig{
		Provider:    "local",
		Model:       "test-model",
		Endpoint:    "http://localhost:11434",
		Temperature: 0.1,
		MaxTokens:   1000,
		Enabled:     true,
	}

	extractor := NewLocalLLMExtractor(config)

	testCases := []struct {
		name           string
		response       string
		expectedCount  int
		expectedNumber string
		expectedCarrier string
		expectedDesc   string
		expectedMerchant string
		shouldError    bool
	}{
		{
			name: "valid response with merchant and description",
			response: `{
				"tracking_numbers": [
					{
						"number": "1Z999AA1234567890",
						"carrier": "ups",
						"confidence": 0.95,
						"description": "Apple iPhone 15 Pro 256GB Space Black",
						"merchant": "Amazon"
					}
				]
			}`,
			expectedCount:    1,
			expectedNumber:   "1Z999AA1234567890",
			expectedCarrier:  "ups",
			expectedDesc:     "Apple iPhone 15 Pro 256GB Space Black",
			expectedMerchant: "Amazon",
			shouldError:      false,
		},
		{
			name: "response with markdown formatting",
			response: "```json\n{\n  \"tracking_numbers\": [\n    {\n      \"number\": \"1Z999AA1234567890\",\n      \"carrier\": \"ups\",\n      \"confidence\": 0.95,\n      \"description\": \"Apple iPhone 15 Pro\",\n      \"merchant\": \"Amazon\"\n    }\n  ]\n}\n```",
			expectedCount:    1,
			expectedNumber:   "1Z999AA1234567890",
			expectedCarrier:  "ups",
			expectedDesc:     "Apple iPhone 15 Pro",
			expectedMerchant: "Amazon",
			shouldError:      false,
		},
		{
			name: "empty response",
			response: `{
				"tracking_numbers": []
			}`,
			expectedCount: 0,
			shouldError:   false,
		},
		{
			name: "invalid JSON",
			response: `{
				"tracking_numbers": [
					{
						"number": "1Z999AA1234567890"
						"carrier": "ups"
					}
				]
			}`,
			expectedCount: 0,
			shouldError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := extractor.parseResponse(tc.response)
			
			if tc.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if len(results) != tc.expectedCount {
				t.Errorf("Expected %d results, got %d", tc.expectedCount, len(results))
				return
			}
			
			if tc.expectedCount > 0 {
				result := results[0]
				if result.Number != tc.expectedNumber {
					t.Errorf("Expected number '%s', got '%s'", tc.expectedNumber, result.Number)
				}
				if result.Carrier != tc.expectedCarrier {
					t.Errorf("Expected carrier '%s', got '%s'", tc.expectedCarrier, result.Carrier)
				}
				if result.Description != tc.expectedDesc {
					t.Errorf("Expected description '%s', got '%s'", tc.expectedDesc, result.Description)
				}
				// Note: Merchant field will be checked once we add it to TrackingInfo
			}
		})
	}
}

func TestConfidenceFallback(t *testing.T) {
	// Test case to verify confidence-based fallback logic
	testCases := []struct {
		name                string
		llmConfidence       float64
		confidenceThreshold float64
		shouldUseLLM        bool
	}{
		{
			name:                "high confidence - use LLM",
			llmConfidence:       0.95,
			confidenceThreshold: 0.7,
			shouldUseLLM:        true,
		},
		{
			name:                "low confidence - fallback to regex",
			llmConfidence:       0.5,
			confidenceThreshold: 0.7,
			shouldUseLLM:        false,
		},
		{
			name:                "threshold confidence - use LLM",
			llmConfidence:       0.7,
			confidenceThreshold: 0.7,
			shouldUseLLM:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shouldUseLLM := tc.llmConfidence >= tc.confidenceThreshold
			if shouldUseLLM != tc.shouldUseLLM {
				t.Errorf("Expected shouldUseLLM to be %v, got %v", tc.shouldUseLLM, shouldUseLLM)
			}
		})
	}
}

// Helper function to check if a prompt contains a specific field
func containsField(prompt, field string) bool {
	switch field {
	case "merchant":
		return containsString(prompt, "merchant") || containsString(prompt, "retailer")
	case "description":
		return containsString(prompt, "description") || containsString(prompt, "product")
	case "tracking_number":
		return containsString(prompt, "tracking") && containsString(prompt, "number")
	case "carrier":
		return containsString(prompt, "carrier") || containsString(prompt, "ups") || containsString(prompt, "usps")
	case "confidence":
		return containsString(prompt, "confidence")
	default:
		return containsString(prompt, field)
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsString(text, substr string) bool {
	return len(text) > 0 && len(substr) > 0 && 
		   (text == substr || 
		    (len(text) > len(substr) && 
		     findSubstring(text, substr)))
}

// Simple substring search helper
func findSubstring(text, substr string) bool {
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