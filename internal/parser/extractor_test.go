package parser

import (
	"testing"
	"time"

	"package-tracking/internal/carriers"
	"package-tracking/internal/email"
)

func TestTrackingExtractor_Extract(t *testing.T) {
	// Initialize test dependencies
	carrierFactory := carriers.NewClientFactory()
	config := &ExtractorConfig{
		EnableLLM:           false, // Disable LLM for basic tests
		MinConfidence:       0.5,
		MaxCandidates:       10,
		UseHybridValidation: true,
		DebugMode:           false,
	}
	
	// LLM config for test (disabled)
	llmConfig := &LLMConfig{
		Enabled: false,
	}
	
	extractor := NewTrackingExtractor(carrierFactory, config, llmConfig)
	
	testCases := []struct {
		name             string
		emailContent     *email.EmailContent
		expectAtLeast    int
		expectedCarrier  string
		expectedNumber   string
		shouldFindNumber bool
	}{
		{
			name: "UPS tracking number in plain text",
			emailContent: &email.EmailContent{
				PlainText: "Your package with tracking number 1Z999AA1234567890 has been shipped.",
				From:      "noreply@ups.com",
				Subject:   "UPS Shipment Notification",
				MessageID: "test-1",
				Date:      time.Now(),
			},
			expectAtLeast:    1,
			expectedCarrier:  "ups",
			expectedNumber:   "1Z999AA1234567890",
			shouldFindNumber: true,
		},
		{
			name: "USPS Priority Mail tracking",
			emailContent: &email.EmailContent{
				PlainText: "Your USPS package 9400111699000367046792 is on its way.",
				From:      "inform@email.usps.com",
				Subject:   "USPS Tracking Update",
				MessageID: "test-2", 
				Date:      time.Now(),
			},
			expectAtLeast:    1,
			expectedCarrier:  "usps",
			expectedNumber:   "9400111699000367046792",
			shouldFindNumber: true,
		},
		{
			name: "FedEx tracking with label",
			emailContent: &email.EmailContent{
				PlainText: "Tracking Number: 123456789012\nYour FedEx package has shipped.",
				From:      "tracking@fedex.com",
				Subject:   "FedEx Shipment Notification",
				MessageID: "test-3",
				Date:      time.Now(),
			},
			expectAtLeast:    1,
			expectedCarrier:  "fedex", 
			expectedNumber:   "123456789012",
			shouldFindNumber: true,
		},
		{
			name: "No tracking numbers",
			emailContent: &email.EmailContent{
				PlainText: "Thank you for your order. We will send tracking information soon.",
				From:      "orders@example.com",
				Subject:   "Order Confirmation",
				MessageID: "test-4",
				Date:      time.Now(),
			},
			expectAtLeast:    0,
			shouldFindNumber: false,
		},
		{
			name: "Multiple tracking numbers",
			emailContent: &email.EmailContent{
				PlainText: "UPS: 1Z999AA1234567890\nUSPS: 9400111699000367046792\nBoth packages shipped.",
				From:      "shipping@amazon.com",
				Subject:   "Multiple Shipments",
				MessageID: "test-5",
				Date:      time.Now(),
			},
			expectAtLeast:    1, // Just check we find at least one
			expectedCarrier:  "ups", // Check for UPS specifically
			expectedNumber:   "1Z999AA1234567890",
			shouldFindNumber: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := extractor.Extract(tc.emailContent)
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if len(results) < tc.expectAtLeast {
				t.Errorf("Expected at least %d tracking numbers, got %d", tc.expectAtLeast, len(results))
			}
			
			if tc.shouldFindNumber {
				found := false
				for _, result := range results {
					if result.Number == tc.expectedNumber && result.Carrier == tc.expectedCarrier {
						found = true
						
						// Verify confidence is reasonable
						if result.Confidence < 0.5 {
							t.Errorf("Confidence too low: %f", result.Confidence)
						}
						
						// Verify source
						if result.Source != "regex" {
							t.Errorf("Expected source 'regex', got '%s'", result.Source)
						}
						
						break
					}
				}
				
				if !found {
					t.Errorf("Expected tracking number %s (%s) not found in results", tc.expectedNumber, tc.expectedCarrier)
				}
			} else if len(results) > 0 {
				t.Errorf("Expected no tracking numbers, but found %d", len(results))
			}
		})
	}
}

func TestPatternManager_ExtractForCarrier(t *testing.T) {
	pm := NewPatternManager()
	
	testCases := []struct {
		name        string
		carrier     string
		text        string
		expectCount int
		expectText  string
	}{
		{
			name:        "UPS standard format",
			carrier:     "ups",
			text:        "Your package 1Z999AA1234567890 has shipped",
			expectCount: 1,
			expectText:  "1Z999AA1234567890",
		},
		{
			name:        "USPS Priority Mail",
			carrier:     "usps", 
			text:        "USPS tracking: 9400111699000367046792",
			expectCount: 1,
			expectText:  "9400111699000367046792",
		},
		{
			name:        "FedEx 12-digit",
			carrier:     "fedex",
			text:        "FedEx tracking 123456789012 for your order",
			expectCount: 1,
			expectText:  "123456789012",
		},
		{
			name:        "DHL with label",
			carrier:     "dhl",
			text:        "DHL tracking number: 1234567890",
			expectCount: 1,
			expectText:  "1234567890",
		},
		{
			name:        "No matches",
			carrier:     "ups",
			text:        "Your order has been received",
			expectCount: 0,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			candidates := pm.ExtractForCarrier(tc.text, tc.carrier)
			
			if len(candidates) < tc.expectCount {
				t.Errorf("Expected at least %d candidates, got %d", tc.expectCount, len(candidates))
			}
			
			if tc.expectCount > 0 {
				found := false
				for _, candidate := range candidates {
					if candidate.Text == tc.expectText {
						found = true
						
						// Verify carrier matches
						if candidate.Carrier != tc.carrier {
							t.Errorf("Expected carrier %s, got %s", tc.carrier, candidate.Carrier)
						}
						
						// Verify confidence is reasonable
						if candidate.Confidence <= 0 || candidate.Confidence > 1.0 {
							t.Errorf("Invalid confidence: %f", candidate.Confidence)
						}
						
						break
					}
				}
				
				if !found {
					t.Errorf("Expected text '%s' not found in candidates", tc.expectText)
				}
			}
		})
	}
}

func TestEmailContent_Preprocessing(t *testing.T) {
	carrierFactory := carriers.NewClientFactory()
	config := &ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
		DebugMode:     true,
	}
	
	// LLM config for test (disabled)
	llmConfig := &LLMConfig{
		Enabled: false,
	}
	
	extractor := NewTrackingExtractor(carrierFactory, config, llmConfig)
	
	testCases := []struct {
		name     string
		content  *email.EmailContent
		expected string
	}{
		{
			name: "HTML to text conversion",
			content: &email.EmailContent{
				PlainText: "",
				HTMLText:  "<p>Your tracking number is <strong>1Z999AA1234567890</strong></p>",
				MessageID: "test-html",
			},
			expected: "1Z999AA1234567890",
		},
		{
			name: "Whitespace normalization",
			content: &email.EmailContent{
				PlainText: "Tracking:\n\n   1Z999AA1234567890   \n\nThank you",
				MessageID: "test-whitespace",
			},
			expected: "1Z999AA1234567890",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			processed := extractor.preprocessContent(tc.content)
			
			if processed.PlainText == "" {
				t.Error("Preprocessing resulted in empty plain text")
			}
			
			// Should contain the expected tracking number
			if tc.expected != "" {
				results, err := extractor.Extract(processed)
				if err != nil {
					t.Fatalf("Extraction failed: %v", err)
				}
				
				found := false
				for _, result := range results {
					if result.Number == tc.expected {
						found = true
						break
					}
				}
				
				if !found {
					t.Errorf("Expected tracking number %s not found after preprocessing", tc.expected)
				}
			}
		})
	}
}

func TestCarrierIdentification(t *testing.T) {
	carrierFactory := carriers.NewClientFactory()
	config := &ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
	}
	
	// LLM config for test (disabled)
	llmConfig := &LLMConfig{
		Enabled: false,
	}
	
	extractor := NewTrackingExtractor(carrierFactory, config, llmConfig)
	
	testCases := []struct {
		name           string
		content        *email.EmailContent
		expectedCarrier string
		minConfidence  float64
	}{
		{
			name: "UPS sender domain",
			content: &email.EmailContent{
				From:    "noreply@ups.com",
				Subject: "Package notification",
				PlainText: "Your package has shipped",
			},
			expectedCarrier: "ups",
			minConfidence:   0.8,
		},
		{
			name: "FedEx in subject",
			content: &email.EmailContent{
				From:    "shipping@example.com",
				Subject: "FedEx delivery update",
				PlainText: "Your package status",
			},
			expectedCarrier: "fedex",
			minConfidence:   0.6,
		},
		{
			name: "Generic shipping email",
			content: &email.EmailContent{
				From:    "orders@amazon.com",
				Subject: "Your package has shipped",
				PlainText: "Tracking information will be available soon",
			},
			expectedCarrier: "unknown",
			minConfidence:   0.4,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			processed := extractor.preprocessContent(tc.content)
			hints := extractor.identifyCarriers(processed)
			
			if len(hints) == 0 {
				t.Error("No carrier hints identified")
				return
			}
			
			// Find the expected carrier hint
			found := false
			for _, hint := range hints {
				if hint.Carrier == tc.expectedCarrier {
					found = true
					
					if hint.Confidence < tc.minConfidence {
						t.Errorf("Confidence too low: %f, expected >= %f", hint.Confidence, tc.minConfidence)
					}
					
					break
				}
			}
			
			if !found && tc.expectedCarrier != "unknown" {
				t.Errorf("Expected carrier %s not found in hints", tc.expectedCarrier)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkTrackingExtractor_Extract(b *testing.B) {
	carrierFactory := carriers.NewClientFactory()
	config := &ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
	}
	
	// LLM config for test (disabled)
	llmConfig := &LLMConfig{
		Enabled: false,
	}
	
	extractor := NewTrackingExtractor(carrierFactory, config, llmConfig)
	
	content := &email.EmailContent{
		PlainText: "Your UPS package 1Z999AA1234567890 and USPS package 9400111699000367046792 have shipped. FedEx tracking: 123456789012",
		From:      "shipping@amazon.com",
		Subject:   "Multiple shipments",
		MessageID: "bench-test",
		Date:      time.Now(),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := extractor.Extract(content)
		if err != nil {
			b.Fatalf("Extraction failed: %v", err)
		}
	}
}

func TestTrackingExtractor_LLMInitialization(t *testing.T) {
	carrierFactory := carriers.NewClientFactory()
	
	tests := []struct {
		name      string
		config    *ExtractorConfig
		expectErr bool
	}{
		{
			name: "LLM enabled should not cause nil pointer",
			config: &ExtractorConfig{
				EnableLLM:     true,
				MinConfidence: 0.5,
			},
			expectErr: false,
		},
		{
			name: "LLM disabled should work normally",
			config: &ExtractorConfig{
				EnableLLM:     false,
				MinConfidence: 0.5,
			},
			expectErr: false,
		},
		{
			name:      "Nil config should work with defaults",
			config:    nil,
			expectErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create extractor - should not panic
			extractor := NewTrackingExtractor(carrierFactory, tt.config, &LLMConfig{Enabled: false})
			
			// Verify LLM extractor was initialized
			if extractor.llmExtractor == nil {
				t.Error("Expected llmExtractor to be initialized, got nil")
			}
			
			// Test extraction with LLM-enabled content - should not panic
			content := &email.EmailContent{
				PlainText: "Your package 1Z999AA1234567890 has been shipped",
				Subject:   "Package shipped",
				From:      "test@ups.com",
			}
			
			results, err := extractor.Extract(content)
			if (err != nil) != tt.expectErr {
				t.Errorf("Extract() error = %v, expectErr %v", err, tt.expectErr)
			}
			
			// Should find at least the UPS tracking number
			if len(results) == 0 {
				t.Error("Expected to find tracking numbers, got none")
			}
			
			t.Logf("Configuration: EnableLLM=%v, found %d tracking numbers", 
				extractor.config.EnableLLM, len(results))
		})
	}
}

func TestTrackingExtractor_MergeResultsWithMerchant(t *testing.T) {
	// Initialize test dependencies
	carrierFactory := carriers.NewClientFactory()
	config := &ExtractorConfig{
		EnableLLM:           true,
		MinConfidence:       0.5,
		MaxCandidates:       10,
		UseHybridValidation: true,
		DebugMode:           false,
	}
	
	llmConfig := &LLMConfig{
		Enabled: true,
	}
	
	extractor := NewTrackingExtractor(carrierFactory, config, llmConfig)
	
	testCases := []struct {
		name        string
		regexResults []email.TrackingInfo
		llmResults   []email.TrackingInfo
		expected     []string // expected descriptions
	}{
		{
			name:         "LLM result with merchant and description",
			regexResults: []email.TrackingInfo{},
			llmResults: []email.TrackingInfo{
				{
					Number:      "1Z999AA1234567890",
					Carrier:     "ups",
					Description: "Apple iPhone 15 Pro 256GB Space Black",
					Merchant:    "Amazon",
					Confidence:  0.95,
					Source:      "llm",
				},
			},
			expected: []string{"Apple iPhone 15 Pro 256GB Space Black from Amazon"},
		},
		{
			name: "Merge regex with LLM enhancement",
			regexResults: []email.TrackingInfo{
				{
					Number:     "1Z999AA1234567890",
					Carrier:    "ups",
					Confidence: 0.8,
					Source:     "regex",
				},
			},
			llmResults: []email.TrackingInfo{
				{
					Number:      "1Z999AA1234567890",
					Carrier:     "ups",
					Description: "MacBook Pro 16-inch",
					Merchant:    "Apple Store",
					Confidence:  0.92,
					Source:      "llm",
				},
			},
			expected: []string{"MacBook Pro 16-inch from Apple Store"},
		},
		{
			name:         "LLM result with description only",
			regexResults: []email.TrackingInfo{},
			llmResults: []email.TrackingInfo{
				{
					Number:      "9405511206213414325732",
					Carrier:     "usps",
					Description: "Nike Air Max 270 sneakers",
					Merchant:    "",
					Confidence:  0.85,
					Source:      "llm",
				},
			},
			expected: []string{"Nike Air Max 270 sneakers"},
		},
		{
			name:         "LLM result with merchant only",
			regexResults: []email.TrackingInfo{},
			llmResults: []email.TrackingInfo{
				{
					Number:      "961234567890",
					Carrier:     "fedex",
					Description: "",
					Merchant:    "Best Buy",
					Confidence:  0.75,
					Source:      "llm",
				},
			},
			expected: []string{"Package from Best Buy"},
		},
		{
			name: "Multiple tracking numbers with different merchants",
			regexResults: []email.TrackingInfo{},
			llmResults: []email.TrackingInfo{
				{
					Number:      "1Z999AA1234567890",
					Carrier:     "ups",
					Description: "Dell XPS 13 Laptop",
					Merchant:    "Amazon",
					Confidence:  0.9,
					Source:      "llm",
				},
				{
					Number:      "9405511206213414325732",
					Carrier:     "usps",
					Description: "Wireless Mouse",
					Merchant:    "Best Buy",
					Confidence:  0.88,
					Source:      "llm",
				},
			},
			expected: []string{"Dell XPS 13 Laptop from Amazon", "Wireless Mouse from Best Buy"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results := extractor.mergeResults(tc.regexResults, tc.llmResults)
			
			if len(results) != len(tc.expected) {
				t.Errorf("Expected %d results, got %d", len(tc.expected), len(results))
				return
			}
			
			// Create a map of actual descriptions for comparison
			actualDescs := make(map[string]bool)
			for _, result := range results {
				actualDescs[result.Description] = true
			}
			
			// Check that all expected descriptions are present
			for _, expectedDesc := range tc.expected {
				if !actualDescs[expectedDesc] {
					t.Errorf("Expected description '%s' not found in results", expectedDesc)
				}
			}
		})
	}
}

func TestTrackingExtractor_CombineDescriptionAndMerchant(t *testing.T) {
	extractor := &TrackingExtractor{}
	
	testCases := []struct {
		name        string
		description string
		merchant    string
		expected    string
	}{
		{
			name:        "Both description and merchant",
			description: "Apple iPhone 15 Pro 256GB Space Black",
			merchant:    "Amazon",
			expected:    "Apple iPhone 15 Pro 256GB Space Black from Amazon",
		},
		{
			name:        "Description only",
			description: "Nike Air Max 270 sneakers",
			merchant:    "",
			expected:    "Nike Air Max 270 sneakers",
		},
		{
			name:        "Merchant only",
			description: "",
			merchant:    "Best Buy",
			expected:    "Package from Best Buy",
		},
		{
			name:        "Neither description nor merchant",
			description: "",
			merchant:    "",
			expected:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractor.combineDescriptionAndMerchant(tc.description, tc.merchant)
			
			if result != tc.expected {
				t.Errorf("Expected: '%s', got: '%s'", tc.expected, result)
			}
		})
	}
}

func BenchmarkPatternManager_ExtractForCarrier(b *testing.B) {
	pm := NewPatternManager()
	text := "Your UPS package 1Z999AA1234567890 has shipped via UPS Ground service"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.ExtractForCarrier(text, "ups")
	}
}