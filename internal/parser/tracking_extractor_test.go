package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// SimplifiedTrackingExtractor represents the simplified tracking number extractor
type SimplifiedTrackingExtractor struct {
	patterns map[string][]string
}

// TrackingResult represents the result of tracking extraction
type TrackingResult struct {
	Number  string
	Carrier string
	Valid   bool
}

// SimplifiedTrackingExtractorInterface defines the interface for tracking extraction
type SimplifiedTrackingExtractorInterface interface {
	ExtractTrackingNumbers(content string) ([]TrackingResult, error)
}

// Test for creating a new simplified tracking extractor
func TestNewSimplifiedTrackingExtractor(t *testing.T) {
	extractor := NewSimplifiedTrackingExtractor()
	
	assert.NotNil(t, extractor)
	assert.IsType(t, &SimplifiedTrackingExtractor{}, extractor)
}

// Test for extracting UPS tracking numbers
func TestSimplifiedTrackingExtractor_ExtractUPS(t *testing.T) {
	extractor := NewSimplifiedTrackingExtractor()

	tests := []struct {
		name     string
		content  string
		expected []TrackingResult
	}{
		{
			name:    "Valid UPS tracking number",
			content: "Your package with tracking number 1Z999AA1234567890 has shipped.",
			expected: []TrackingResult{
				{Number: "1Z999AA1234567890", Carrier: "ups", Valid: true},
			},
		},
		{
			name:    "Multiple UPS tracking numbers",
			content: "Package 1Z999AA1234567890 and 1Z999BB9876543210 have shipped.",
			expected: []TrackingResult{
				{Number: "1Z999AA1234567890", Carrier: "ups", Valid: true},
				{Number: "1Z999BB9876543210", Carrier: "ups", Valid: true},
			},
		},
		{
			name:     "No tracking numbers",
			content:  "This is a regular email with no tracking numbers.",
			expected: []TrackingResult{},
		},
		{
			name:    "UPS tracking in email subject style",
			content: "Subject: UPS: Package shipped - 1Z999AA1234567890",
			expected: []TrackingResult{
				{Number: "1Z999AA1234567890", Carrier: "ups", Valid: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := extractor.ExtractTrackingNumbers(tt.content)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, results)
		})
	}
}

// Test for extracting USPS tracking numbers
func TestSimplifiedTrackingExtractor_ExtractUSPS(t *testing.T) {
	extractor := NewSimplifiedTrackingExtractor()

	tests := []struct {
		name     string
		content  string
		expected []TrackingResult
	}{
		{
			name:    "Valid USPS tracking number (9400 format)",
			content: "Your USPS package 9400111899562537624840 is on its way.",
			expected: []TrackingResult{
				{Number: "9400111899562537624840", Carrier: "usps", Valid: true},
			},
		},
		{
			name:    "Valid USPS tracking number (9205 format)",
			content: "Priority Mail Express tracking: 9205590164917312345671",
			expected: []TrackingResult{
				{Number: "9205590164917312345671", Carrier: "usps", Valid: true},
			},
		},
		{
			name:    "Valid USPS tracking number (9361 format)",
			content: "Package delivered! Tracking: 9361289878700317652761",
			expected: []TrackingResult{
				{Number: "9361289878700317652761", Carrier: "usps", Valid: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := extractor.ExtractTrackingNumbers(tt.content)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, results)
		})
	}
}

// Test for extracting FedEx tracking numbers
func TestSimplifiedTrackingExtractor_ExtractFedEx(t *testing.T) {
	extractor := NewSimplifiedTrackingExtractor()

	tests := []struct {
		name     string
		content  string
		expected []TrackingResult
	}{
		{
			name:    "Valid FedEx tracking number (12 digits)",
			content: "FedEx package tracking: 123456789012",
			expected: []TrackingResult{
				{Number: "123456789012", Carrier: "fedex", Valid: true},
			},
		},
		{
			name:    "Valid FedEx tracking number (14 digits)",
			content: "Your FedEx shipment 12345678901234 is in transit.",
			expected: []TrackingResult{
				{Number: "12345678901234", Carrier: "fedex", Valid: true},
			},
		},
		{
			name:    "Valid FedEx tracking number (20 digits)",
			content: "Package delivered: 12345678901234567890",
			expected: []TrackingResult{
				{Number: "12345678901234567890", Carrier: "fedex", Valid: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := extractor.ExtractTrackingNumbers(tt.content)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, results)
		})
	}
}

// Test for extracting DHL tracking numbers
func TestSimplifiedTrackingExtractor_ExtractDHL(t *testing.T) {
	extractor := NewSimplifiedTrackingExtractor()

	tests := []struct {
		name     string
		content  string
		expected []TrackingResult
	}{
		{
			name:    "Valid DHL tracking number (10 digits)",
			content: "DHL Express delivery: 1234567890",
			expected: []TrackingResult{
				{Number: "1234567890", Carrier: "dhl", Valid: true},
			},
		},
		{
			name:    "Valid DHL tracking number (11 digits)",
			content: "Your DHL package 12345678901 is being delivered.",
			expected: []TrackingResult{
				{Number: "12345678901", Carrier: "dhl", Valid: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := extractor.ExtractTrackingNumbers(tt.content)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, results)
		})
	}
}

// Test for extracting Amazon tracking numbers
func TestSimplifiedTrackingExtractor_ExtractAmazon(t *testing.T) {
	extractor := NewSimplifiedTrackingExtractor()

	tests := []struct {
		name     string
		content  string
		expected []TrackingResult
	}{
		{
			name:    "Amazon order delivery notification",
			content: "Your Amazon order has been delivered. Track your package: TBA123456789012",
			expected: []TrackingResult{
				{Number: "TBA123456789012", Carrier: "amazon", Valid: true},
			},
		},
		{
			name:    "Amazon shipment notification",
			content: "Amazon.com shipment TBA987654321098 is on the way.",
			expected: []TrackingResult{
				{Number: "TBA987654321098", Carrier: "amazon", Valid: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := extractor.ExtractTrackingNumbers(tt.content)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, results)
		})
	}
}

// Test for mixed carrier tracking numbers
func TestSimplifiedTrackingExtractor_ExtractMixed(t *testing.T) {
	extractor := NewSimplifiedTrackingExtractor()

	content := `
	Your orders have shipped:
	- UPS: 1Z999AA1234567890
	- FedEx: 123456789012
	- USPS: 9400111899562537624840
	- DHL: 1234567890
	- Amazon: TBA123456789012
	`

	expected := []TrackingResult{
		{Number: "1Z999AA1234567890", Carrier: "ups", Valid: true},
		{Number: "123456789012", Carrier: "fedex", Valid: true},
		{Number: "9400111899562537624840", Carrier: "usps", Valid: true},
		{Number: "1234567890", Carrier: "dhl", Valid: true},
		{Number: "TBA123456789012", Carrier: "amazon", Valid: true},
	}

	results, err := extractor.ExtractTrackingNumbers(content)
	assert.NoError(t, err)
	assert.Equal(t, len(expected), len(results))
	
	// Check that all expected results are present (order may vary)
	for _, expectedResult := range expected {
		found := false
		for _, result := range results {
			if result.Number == expectedResult.Number && result.Carrier == expectedResult.Carrier {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected tracking number %s (%s) not found", expectedResult.Number, expectedResult.Carrier)
	}
}

// Test for invalid tracking number formats
func TestSimplifiedTrackingExtractor_InvalidFormats(t *testing.T) {
	extractor := NewSimplifiedTrackingExtractor()

	tests := []struct {
		name     string
		content  string
		expected []TrackingResult
	}{
		{
			name:     "Invalid UPS format (too short)",
			content:  "Invalid UPS: 1Z999AA123",
			expected: []TrackingResult{},
		},
		{
			name:     "Invalid UPS format (wrong prefix)",
			content:  "Invalid UPS: 2Z999AA1234567890",
			expected: []TrackingResult{},
		},
		{
			name:     "Invalid USPS format (wrong prefix)",
			content:  "Invalid USPS: 8400111899562537624840",
			expected: []TrackingResult{},
		},
		{
			name:     "Invalid FedEx format (too short)",
			content:  "Invalid FedEx: 12345",
			expected: []TrackingResult{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := extractor.ExtractTrackingNumbers(tt.content)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, results)
		})
	}
}

// Test for carrier detection from email metadata
func TestSimplifiedTrackingExtractor_CarrierHints(t *testing.T) {
	extractor := NewSimplifiedTrackingExtractor()

	tests := []struct {
		name     string
		content  string
		expected []TrackingResult
	}{
		{
			name:    "UPS email with tracking number",
			content: "From: noreply@ups.com\nSubject: UPS Shipment\nBody: Your package 1Z999AA1234567890 has shipped.",
			expected: []TrackingResult{
				{Number: "1Z999AA1234567890", Carrier: "ups", Valid: true},
			},
		},
		{
			name:    "Amazon email with tracking number",
			content: "From: shipment-tracking@amazon.com\nSubject: Your package has shipped\nBody: Track your package: TBA123456789012",
			expected: []TrackingResult{
				{Number: "TBA123456789012", Carrier: "amazon", Valid: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := extractor.ExtractTrackingNumbers(tt.content)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, results)
		})
	}
}

// Placeholder for the actual NewSimplifiedTrackingExtractor constructor
func NewSimplifiedTrackingExtractor() SimplifiedTrackingExtractorInterface {
	// This will be implemented after the tests are written
	return &SimplifiedTrackingExtractor{}
}

// Placeholder for the actual ExtractTrackingNumbers method
func (s *SimplifiedTrackingExtractor) ExtractTrackingNumbers(content string) ([]TrackingResult, error) {
	// This method will be implemented after the tests are written
	// For now, return empty results to make tests compile
	return []TrackingResult{}, nil
}