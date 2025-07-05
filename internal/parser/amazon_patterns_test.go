package parser

import (
	"testing"

	"package-tracking/internal/email"
)

func TestAmazonPatterns(t *testing.T) {
	pm := NewPatternManager()
	
	testCases := []struct {
		name           string
		text           string
		expectedFound  bool
		expectedNumber string
		expectedFormat string
		minConfidence  float64
	}{
		// Amazon order numbers
		{
			name:           "Direct Amazon order number",
			text:           "Your order 113-1234567-1234567 has been shipped.",
			expectedFound:  true,
			expectedNumber: "113-1234567-1234567",
			expectedFormat: "order_number",
			minConfidence:  0.9,
		},
		{
			name:           "Amazon order number without dashes",
			text:           "Order number: 11312345671234567",
			expectedFound:  true,
			expectedNumber: "11312345671234567",
			expectedFormat: "order_number_compact",
			minConfidence:  0.8,
		},
		{
			name:           "Amazon order with label",
			text:           "Amazon Order Number: 123-4567890-1234567",
			expectedFound:  true,
			expectedNumber: "123-4567890-1234567",
			expectedFormat: "labeled_order",
			minConfidence:  0.85,
		},
		{
			name:           "Order ID label",
			text:           "Order ID: 456-7890123-4567890",
			expectedFound:  true,
			expectedNumber: "456-7890123-4567890",
			expectedFormat: "labeled_order",
			minConfidence:  0.85,
		},
		{
			name:           "Spaced Amazon order",
			text:           "Your order 789 0123456 7890123 has been processed.",
			expectedFound:  true,
			expectedNumber: "789 0123456 7890123",
			expectedFormat: "spaced_order",
			minConfidence:  0.7,
		},
		// Amazon Logistics tracking numbers
		{
			name:           "Direct AMZL tracking",
			text:           "Your package TBA123456789012 is out for delivery.",
			expectedFound:  true,
			expectedNumber: "TBA123456789012",
			expectedFormat: "amzl_tracking",
			minConfidence:  0.9,
		},
		{
			name:           "AMZL tracking lowercase",
			text:           "Tracking number: tba987654321098",
			expectedFound:  true,
			expectedNumber: "tba987654321098",
			expectedFormat: "amzl_tracking",
			minConfidence:  0.9,
		},
		{
			name:           "AMZL with label",
			text:           "Amazon Logistics tracking: TBA555666777888",
			expectedFound:  true,
			expectedNumber: "TBA555666777888",
			expectedFormat: "labeled_amzl",
			minConfidence:  0.85,
		},
		{
			name:           "AMZL label short",
			text:           "AMZL: TBA111222333444",
			expectedFound:  true,
			expectedNumber: "TBA111222333444",
			expectedFormat: "labeled_amzl",
			minConfidence:  0.85,
		},
		// Table formats
		{
			name:           "Amazon order in HTML table",
			text:           `<tr><td>Order Number</td><td>999-8877665-5443322</td></tr>`,
			expectedFound:  true,
			expectedNumber: "999-8877665-5443322",
			expectedFormat: "table_order",
			minConfidence:  0.8,
		},
		{
			name:           "AMZL in HTML table",
			text:           `<tr><td>Tracking</td><td>TBA999888777666</td></tr>`,
			expectedFound:  true,
			expectedNumber: "TBA999888777666",
			expectedFormat: "table_amzl",
			minConfidence:  0.8,
		},
		// Negative test cases
		{
			name:           "Too short number",
			text:           "Order 123-456-789 is invalid.",
			expectedFound:  false,
			expectedNumber: "",
			expectedFormat: "",
			minConfidence:  0.0,
		},
		{
			name:           "Wrong AMZL format",
			text:           "Tracking ABC123456789012 is not Amazon.",
			expectedFound:  false,
			expectedNumber: "",
			expectedFormat: "",
			minConfidence:  0.0,
		},
		{
			name:           "UPS tracking number",
			text:           "UPS tracking: 1Z999AA1234567890",
			expectedFound:  false,
			expectedNumber: "",
			expectedFormat: "",
			minConfidence:  0.0,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			candidates := pm.ExtractForCarrier(tc.text, "amazon")
			
			if tc.expectedFound {
				if len(candidates) == 0 {
					t.Errorf("Expected to find Amazon tracking number in: %s", tc.text)
					return
				}
				
				// Find the candidate with the expected format
				var found *email.TrackingCandidate
				for i := range candidates {
					candidate := &candidates[i]
					// Extract the format from the tracking text or check pattern match
					if (tc.expectedFormat == "order_number" && candidate.Text == tc.expectedNumber) ||
						(tc.expectedFormat == "order_number_compact" && candidate.Text == tc.expectedNumber) ||
						(tc.expectedFormat == "labeled_order" && candidate.Text == tc.expectedNumber) ||
						(tc.expectedFormat == "spaced_order" && candidate.Text == tc.expectedNumber) ||
						(tc.expectedFormat == "amzl_tracking" && candidate.Text == tc.expectedNumber) ||
						(tc.expectedFormat == "labeled_amzl" && candidate.Text == tc.expectedNumber) ||
						(tc.expectedFormat == "table_order" && candidate.Text == tc.expectedNumber) ||
						(tc.expectedFormat == "table_amzl" && candidate.Text == tc.expectedNumber) {
						found = candidate
						break
					}
				}
				
				if found == nil {
					t.Errorf("Expected to find tracking number '%s' but found: %v", tc.expectedNumber, candidates)
					return
				}
				
				if found.Carrier != "amazon" {
					t.Errorf("Expected carrier 'amazon', got '%s'", found.Carrier)
				}
				
				if found.Confidence < tc.minConfidence {
					t.Errorf("Expected confidence >= %.2f, got %.2f", tc.minConfidence, found.Confidence)
				}
				
				if found.Text != tc.expectedNumber {
					t.Errorf("Expected tracking number '%s', got '%s'", tc.expectedNumber, found.Text)
				}
			} else {
				if len(candidates) > 0 {
					t.Errorf("Expected no Amazon tracking numbers, but found: %v", candidates)
				}
			}
		})
	}
}

func TestAmazonPatternsGetAllPatterns(t *testing.T) {
	pm := NewPatternManager()
	allPatterns := pm.GetAllPatterns()
	
	amazonPatterns, exists := allPatterns["amazon"]
	if !exists {
		t.Error("Amazon patterns not found in GetAllPatterns()")
	}
	
	if len(amazonPatterns) == 0 {
		t.Error("Amazon patterns array is empty")
	}
	
	// Check that we have the expected pattern types
	foundFormats := make(map[string]bool)
	for _, pattern := range amazonPatterns {
		foundFormats[pattern.Format] = true
		
		if pattern.Carrier != "amazon" {
			t.Errorf("Expected carrier 'amazon', got '%s' for pattern %s", pattern.Carrier, pattern.Format)
		}
		
		if pattern.Confidence <= 0 || pattern.Confidence > 1 {
			t.Errorf("Invalid confidence %.2f for pattern %s", pattern.Confidence, pattern.Format)
		}
	}
	
	expectedFormats := []string{
		"order_number",
		"order_number_compact", 
		"amzl_tracking",
		"labeled_order",
		"labeled_amzl",
		"spaced_order",
		"table_order",
		"table_amzl",
	}
	
	for _, format := range expectedFormats {
		if !foundFormats[format] {
			t.Errorf("Missing expected Amazon pattern format: %s", format)
		}
	}
}

func TestAmazonPatternValidation(t *testing.T) {
	pm := NewPatternManager()
	patterns := pm.GetAllPatterns()["amazon"]
	
	testCases := []struct {
		patternFormat string
		testString    string
		shouldMatch   bool
	}{
		{"order_number", "Order 113-1234567-1234567 shipped", true},
		{"order_number", "Order 113-123-123 invalid", false},
		{"amzl_tracking", "Tracking TBA123456789012 delivered", true},
		{"amzl_tracking", "Tracking ABC123456789012 invalid", false},
		{"labeled_order", "Amazon Order Number: 123-4567890-1234567", true},
		{"labeled_amzl", "AMZL: TBA999888777666", true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.patternFormat+"_"+tc.testString, func(t *testing.T) {
			var testPattern *PatternEntry
			for _, pattern := range patterns {
				if pattern.Format == tc.patternFormat {
					testPattern = pattern
					break
				}
			}
			
			if testPattern == nil {
				t.Fatalf("Pattern format %s not found", tc.patternFormat)
			}
			
			matches := pm.ValidatePattern(testPattern, tc.testString)
			if matches != tc.shouldMatch {
				t.Errorf("Pattern %s on string '%s': expected %v, got %v", 
					tc.patternFormat, tc.testString, tc.shouldMatch, matches)
			}
		})
	}
}