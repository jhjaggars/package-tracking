package parser

import (
	"testing"

	"package-tracking/internal/carriers"
	"package-tracking/internal/email"
)

func TestExtractDescriptionFromSubject(t *testing.T) {
	extractor := &TrackingExtractor{}

	tests := []struct {
		name     string
		subject  string
		carrier  string
		expected string
	}{
		{
			name:     "Amazon shipped with quotes",
			subject:  `Shipped: "Kuject 320PCS Heat Shrink..." and 1 more item`,
			carrier:  "amazon",
			expected: "Kuject 320PCS Heat Shrink",
		},
		{
			name:     "Amazon shipped with single quotes",
			subject:  `Shipped: 'WOLFBOX MF50 Electric Air...'`,
			carrier:  "amazon",
			expected: "WOLFBOX MF50 Electric Air",
		},
		{
			name:     "Amazon ordered",
			subject:  `Ordered: "BAMBOO COOL Men's UPF 50+..."`,
			carrier:  "amazon",
			expected: "BAMBOO COOL Men's UPF 50+",
		},
		{
			name:     "Amazon delivered generic",
			subject:  "Delivered: 1 item | Order # 114-0213341-4089071",
			carrier:  "amazon",
			expected: "",
		},
		{
			name:     "Generic shipping pattern",
			subject:  "Your iPhone Case has shipped",
			carrier:  "ups",
			expected: "iPhone Case",
		},
		{
			name:     "No description found",
			subject:  "Package tracking notification",
			carrier:  "ups",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.extractDescriptionFromSubject(tt.subject, tt.carrier)
			if result != tt.expected {
				t.Errorf("extractDescriptionFromSubject() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCombineDescriptionAndMerchant(t *testing.T) {
	extractor := &TrackingExtractor{}

	tests := []struct {
		name        string
		description string
		merchant    string
		expected    string
	}{
		{
			name:        "Both description and merchant",
			description: "iPhone Case",
			merchant:    "Amazon",
			expected:    "iPhone Case from Amazon",
		},
		{
			name:        "Only description",
			description: "iPhone Case",
			merchant:    "",
			expected:    "iPhone Case",
		},
		{
			name:        "Only merchant",
			description: "",
			merchant:    "Amazon",
			expected:    "Package from Amazon",
		},
		{
			name:        "Empty merchant should return empty",
			description: "",
			merchant:    "",
			expected:    "",
		},
		{
			name:        "Whitespace-only merchant should return empty",
			description: "",
			merchant:    "   ",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.combineDescriptionAndMerchant(tt.description, tt.merchant)
			if result != tt.expected {
				t.Errorf("combineDescriptionAndMerchant() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilterAndSortWithSubjectFallback(t *testing.T) {
	factory := &carriers.ClientFactory{}
	config := &ExtractorConfig{
		MinConfidence: 0.5,
	}
	extractor := NewTrackingExtractor(factory, config, nil)

	content := &email.EmailContent{
		Subject: `Shipped: "Kuject 320PCS Heat Shrink..." and 1 more item`,
		From:    "shipment-tracking@amazon.com",
	}

	results := []email.TrackingInfo{
		{
			Number:      "11253893815053802",
			Carrier:     "amazon",
			Confidence:  0.8,
			Description: "", // Empty description to test fallback
			Merchant:    "",
		},
	}

	filtered := extractor.filterAndSort(results, content)

	if len(filtered) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(filtered))
	}

	expected := "Kuject 320PCS Heat Shrink"
	if filtered[0].Description != expected {
		t.Errorf("Expected description %q, got %q", expected, filtered[0].Description)
	}
}