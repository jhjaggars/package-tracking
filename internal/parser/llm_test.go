package parser

import (
	"testing"
	"time"

	"package-tracking/internal/email"
)

func TestLLMExtractor_EnhancedPrompt(t *testing.T) {
	config := &LLMConfig{
		Provider:    "local",
		Model:       "llama3.2",
		Endpoint:    "http://localhost:11434",
		MaxTokens:   1000,
		Temperature: 0.1,
		Timeout:     30 * time.Second,
		RetryCount:  2,
		Enabled:     true,
	}
	
	extractor := NewLocalLLMExtractor(config)
	
	testCases := []struct {
		name          string
		emailContent  *email.EmailContent
		expectedJSON  string
		shouldContain []string
	}{
		{
			name: "Enhanced prompt with merchant and description extraction",
			emailContent: &email.EmailContent{
				From:      "noreply@amazon.com",
				Subject:   "Your order has shipped - Order #123-4567890",
				PlainText: "Your order of Apple iPhone 15 Pro 256GB in Space Black has been shipped via UPS. Tracking number: 1Z999AA1234567890. Estimated delivery: January 5, 2025.",
				MessageID: "test-1",
				Date:      time.Now(),
			},
			expectedJSON: `{
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
			shouldContain: []string{
				"Extract shipping tracking numbers, product descriptions, and merchant information",
				"description",
				"merchant",
				"Apple iPhone 15 Pro 256GB Space Black",
				"Amazon",
			},
		},
		{
			name: "Multiple products with different merchants",
			emailContent: &email.EmailContent{
				From:      "orders@shopify.com",
				Subject:   "Your TechStore order has been shipped",
				PlainText: "Your order containing Dell XPS 13 Laptop and Logitech MX Master 3 Mouse has been shipped via FedEx. Tracking: 961234567890. From TechStore.",
				MessageID: "test-2",
				Date:      time.Now(),
			},
			expectedJSON: `{
				"tracking_numbers": [
					{
						"number": "961234567890",
						"carrier": "fedex",
						"confidence": 0.9,
						"description": "Dell XPS 13 Laptop, Logitech MX Master 3 Mouse",
						"merchant": "TechStore"
					}
				]
			}`,
			shouldContain: []string{
				"Dell XPS 13 Laptop",
				"Logitech MX Master 3 Mouse",
				"TechStore",
			},
		},
		{
			name: "Order confirmation with tracking details",
			emailContent: &email.EmailContent{
				From:      "support@bestbuy.com",
				Subject:   "Order Confirmation - Nike Air Max 270",
				PlainText: "Thank you for your order! Your Nike Air Max 270 sneakers in size 10 have been shipped via USPS. Tracking number: 9405511206213414325732. Order total: $120.00.",
				MessageID: "test-3",
				Date:      time.Now(),
			},
			expectedJSON: `{
				"tracking_numbers": [
					{
						"number": "9405511206213414325732",
						"carrier": "usps",
						"confidence": 0.92,
						"description": "Nike Air Max 270 sneakers size 10",
						"merchant": "Best Buy"
					}
				]
			}`,
			shouldContain: []string{
				"Nike Air Max 270",
				"Best Buy",
			},
		},
		{
			name: "Email with no tracking information",
			emailContent: &email.EmailContent{
				From:      "newsletter@example.com",
				Subject:   "Weekly Newsletter",
				PlainText: "Check out our latest deals and promotions this week!",
				MessageID: "test-4",
				Date:      time.Now(),
			},
			expectedJSON: `{
				"tracking_numbers": []
			}`,
			shouldContain: []string{
				"tracking_numbers",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the enhanced prompt building
			prompt := extractor.buildEnhancedPrompt(tc.emailContent)
			
			// Verify the prompt contains expected elements
			for _, expected := range tc.shouldContain {
				if !contains(prompt, expected) {
					t.Errorf("Expected prompt to contain '%s', but it didn't", expected)
				}
			}
			
			// Verify prompt structure for enhanced extraction
			if !contains(prompt, "Extract shipping tracking numbers, product descriptions, and merchant information") {
				t.Error("Prompt should contain enhanced extraction instructions")
			}
			
			if !contains(prompt, "description") {
				t.Error("Prompt should include description field in JSON schema")
			}
			
			if !contains(prompt, "merchant") {
				t.Error("Prompt should include merchant field in JSON schema")
			}
			
			// Verify few-shot examples are included
			if !contains(prompt, "Example 1:") {
				t.Error("Prompt should contain few-shot examples")
			}
		})
	}
}

func TestLLMExtractor_ParseEnhancedResponse(t *testing.T) {
	config := &LLMConfig{
		Provider:    "local",
		Model:       "llama3.2",
		Endpoint:    "http://localhost:11434",
		MaxTokens:   1000,
		Temperature: 0.1,
		Timeout:     30 * time.Second,
		RetryCount:  2,
		Enabled:     true,
	}
	
	extractor := NewLocalLLMExtractor(config)
	
	testCases := []struct {
		name            string
		response        string
		expectedCount   int
		expectedMerchant string
		expectedDesc    string
		shouldError     bool
	}{
		{
			name: "Enhanced response with merchant and description",
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
			expectedCount:   1,
			expectedMerchant: "Amazon",
			expectedDesc:    "Apple iPhone 15 Pro 256GB Space Black",
			shouldError:     false,
		},
		{
			name: "Multiple tracking numbers with different merchants",
			response: `{
				"tracking_numbers": [
					{
						"number": "1Z999AA1234567890",
						"carrier": "ups",
						"confidence": 0.95,
						"description": "Apple iPhone 15 Pro",
						"merchant": "Amazon"
					},
					{
						"number": "9405511206213414325732",
						"carrier": "usps",
						"confidence": 0.88,
						"description": "Samsung Galaxy Buds Pro",
						"merchant": "Best Buy"
					}
				]
			}`,
			expectedCount:   2,
			expectedMerchant: "Amazon",
			expectedDesc:    "Apple iPhone 15 Pro",
			shouldError:     false,
		},
		{
			name: "Response with markdown formatting",
			response: "```json\n{\n  \"tracking_numbers\": [\n    {\n      \"number\": \"1Z999AA1234567890\",\n      \"carrier\": \"ups\",\n      \"confidence\": 0.95,\n      \"description\": \"MacBook Pro 16-inch\",\n      \"merchant\": \"Apple Store\"\n    }\n  ]\n}\n```",
			expectedCount:   1,
			expectedMerchant: "Apple Store",
			expectedDesc:    "MacBook Pro 16-inch",
			shouldError:     false,
		},
		{
			name: "Empty response",
			response: `{
				"tracking_numbers": []
			}`,
			expectedCount: 0,
			shouldError:   false,
		},
		{
			name: "Invalid JSON response",
			response: `{
				"tracking_numbers": [
					{
						"number": "1Z999AA1234567890",
						"carrier": "ups",
						"confidence": 0.95,
						"description": "Apple iPhone 15 Pro",
						"merchant": "Amazon"
					}
				// Missing closing brace
			`,
			expectedCount: 0,
			shouldError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := extractor.parseEnhancedResponse(tc.response)
			
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
				// Check first result for merchant and description
				if results[0].Merchant != tc.expectedMerchant {
					t.Errorf("Expected merchant '%s', got '%s'", tc.expectedMerchant, results[0].Merchant)
				}
				
				if results[0].Description != tc.expectedDesc {
					t.Errorf("Expected description '%s', got '%s'", tc.expectedDesc, results[0].Description)
				}
			}
		})
	}
}

func TestLLMExtractor_ConfidenceBasedFallback(t *testing.T) {
	config := &LLMConfig{
		Provider:    "local",
		Model:       "llama3.2",
		Endpoint:    "http://localhost:11434",
		MaxTokens:   1000,
		Temperature: 0.1,
		Timeout:     30 * time.Second,
		RetryCount:  2,
		Enabled:     true,
	}
	
	extractor := NewLocalLLMExtractor(config)
	
	testCases := []struct {
		name              string
		llmResponse       string
		expectedFallback  bool
		confidenceThreshold float64
		description       string
	}{
		{
			name: "High confidence - should use LLM result",
			llmResponse: `{
				"tracking_numbers": [
					{
						"number": "1Z999AA1234567890",
						"carrier": "ups",
						"confidence": 0.95,
						"description": "Apple iPhone 15 Pro",
						"merchant": "Amazon"
					}
				]
			}`,
			expectedFallback:    false,
			confidenceThreshold: 0.7,
			description:        "High confidence should use LLM result",
		},
		{
			name: "Low confidence - should trigger fallback",
			llmResponse: `{
				"tracking_numbers": [
					{
						"number": "1Z999AA1234567890",
						"carrier": "ups",
						"confidence": 0.5,
						"description": "Unknown item",
						"merchant": "Unknown"
					}
				]
			}`,
			expectedFallback:    true,
			confidenceThreshold: 0.7,
			description:        "Low confidence should trigger fallback",
		},
		{
			name: "Mixed confidence - should use selective fallback",
			llmResponse: `{
				"tracking_numbers": [
					{
						"number": "1Z999AA1234567890",
						"carrier": "ups",
						"confidence": 0.95,
						"description": "Apple iPhone 15 Pro",
						"merchant": "Amazon"
					},
					{
						"number": "9405511206213414325732",
						"carrier": "usps",
						"confidence": 0.4,
						"description": "Unknown item",
						"merchant": "Unknown"
					}
				]
			}`,
			expectedFallback:    false,
			confidenceThreshold: 0.7,
			description:        "Mixed confidence should filter low confidence results",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := extractor.parseEnhancedResponse(tc.llmResponse)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			// Test confidence-based filtering
			filteredResults := extractor.filterByConfidence(results, tc.confidenceThreshold)
			
			if tc.expectedFallback {
				// When expecting fallback, we should have fewer results after filtering
				if len(filteredResults) >= len(results) {
					t.Error("Expected confidence-based filtering to reduce results")
				}
			} else {
				// When not expecting fallback, high confidence results should remain
				hasHighConfidence := false
				for _, result := range filteredResults {
					if result.Confidence >= tc.confidenceThreshold {
						hasHighConfidence = true
						break
					}
				}
				if !hasHighConfidence && len(results) > 0 {
					t.Error("Expected at least one high confidence result to remain")
				}
			}
		})
	}
}

func TestLLMExtractor_FewShotPromptExamples(t *testing.T) {
	config := &LLMConfig{
		Provider:    "local",
		Model:       "llama3.2",
		Endpoint:    "http://localhost:11434",
		MaxTokens:   1000,
		Temperature: 0.1,
		Timeout:     30 * time.Second,
		RetryCount:  2,
		Enabled:     true,
	}
	
	extractor := NewLocalLLMExtractor(config)
	
	// Test that few-shot examples are properly formatted
	emailContent := &email.EmailContent{
		From:      "noreply@amazon.com",
		Subject:   "Your order has shipped",
		PlainText: "Test content",
		MessageID: "test-1",
		Date:      time.Now(),
	}
	
	prompt := extractor.buildEnhancedPrompt(emailContent)
	
	// Check for few-shot examples
	expectedExamples := []string{
		"Example 1:",
		"Example 2:",
		"Example 3:",
		"From: noreply@amazon.com",
		"Subject: Your Amazon order has shipped",
		"Apple iPhone 15 Pro 256GB Space Black",
		"From: orders@shopify.com",
		"Subject: Your TechStore order is on its way",
		"Dell XPS 13 Laptop",
		"From: support@bestbuy.com",
		"Subject: Order Confirmation",
		"Nike Air Max 270",
	}
	
	for _, expected := range expectedExamples {
		if !contains(prompt, expected) {
			t.Errorf("Expected prompt to contain few-shot example: '%s'", expected)
		}
	}
	
	// Verify JSON schema includes new fields
	if !contains(prompt, "\"description\"") {
		t.Error("Prompt should include description field in JSON schema")
	}
	
	if !contains(prompt, "\"merchant\"") {
		t.Error("Prompt should include merchant field in JSON schema")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}