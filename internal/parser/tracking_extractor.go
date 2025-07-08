package parser

import (
	"regexp"
	"strings"
)

// SimplifiedTrackingExtractor represents the simplified tracking number extractor
type SimplifiedTrackingExtractor struct {
	patterns map[string]*regexp.Regexp
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

// NewSimplifiedTrackingExtractor creates a new simplified tracking extractor
func NewSimplifiedTrackingExtractor() SimplifiedTrackingExtractorInterface {
	// Compile regex patterns for each carrier
	patterns := map[string]*regexp.Regexp{
		"ups":    regexp.MustCompile(`\b1Z[A-Z0-9]{15}\b`),
		"usps":   regexp.MustCompile(`\b(9[0-9]{21}|9[4-6][0-9]{20}|82[0-9]{8}|[A-Z]{2}[0-9]{9}US)\b`),
		"fedex":  regexp.MustCompile(`\b[0-9]{12,20}\b`),
		"dhl":    regexp.MustCompile(`\b[0-9]{10,11}\b`),
		"amazon": regexp.MustCompile(`\bTBA[0-9A-Z]{12}\b`),
	}

	return &SimplifiedTrackingExtractor{
		patterns: patterns,
	}
}

// ExtractTrackingNumbers extracts tracking numbers from content using regex patterns
func (s *SimplifiedTrackingExtractor) ExtractTrackingNumbers(content string) ([]TrackingResult, error) {
	// Initialize with empty slice, not nil
	results := []TrackingResult{}
	seen := make(map[string]bool) // To avoid duplicates

	// Normalize content for better matching
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\r", " ")
	
	// Try each carrier pattern
	for carrier, pattern := range s.patterns {
		matches := pattern.FindAllString(content, -1)
		
		for _, match := range matches {
			// Skip if we've already found this tracking number
			if seen[match] {
				continue
			}
			
			// Additional validation for specific carriers
			if s.isValidTrackingNumber(match, carrier) {
				results = append(results, TrackingResult{
					Number:  match,
					Carrier: carrier,
					Valid:   true,
				})
				seen[match] = true
			}
		}
	}

	return results, nil
}

// isValidTrackingNumber performs additional validation for specific carrier patterns
func (s *SimplifiedTrackingExtractor) isValidTrackingNumber(number, carrier string) bool {
	switch carrier {
	case "ups":
		// UPS tracking numbers: 1Z + 15 alphanumeric characters = 17 total
		return len(number) == 17 && strings.HasPrefix(number, "1Z")
		
	case "usps":
		// USPS patterns are more complex, basic length checks
		return len(number) >= 10 && len(number) <= 22
		
	case "fedex":
		// FedEx: 12, 14, or 20 digits
		length := len(number)
		return length == 12 || length == 14 || length == 20
		
	case "dhl":
		// DHL: 10 or 11 digits
		length := len(number)
		return length == 10 || length == 11
		
	case "amazon":
		// Amazon: TBA + 12 alphanumeric characters = 15 total
		return len(number) == 15 && strings.HasPrefix(number, "TBA")
		
	default:
		return true
	}
}

// Helper function to detect carrier hints from email metadata
func (s *SimplifiedTrackingExtractor) detectCarrierFromContent(content string) string {
	content = strings.ToLower(content)
	
	// Look for carrier indicators in the content
	if strings.Contains(content, "ups.com") || strings.Contains(content, "ups") {
		return "ups"
	}
	if strings.Contains(content, "usps.com") || strings.Contains(content, "usps") || strings.Contains(content, "postal") {
		return "usps"
	}
	if strings.Contains(content, "fedex.com") || strings.Contains(content, "fedex") {
		return "fedex"
	}
	if strings.Contains(content, "dhl.com") || strings.Contains(content, "dhl") {
		return "dhl"
	}
	if strings.Contains(content, "amazon.com") || strings.Contains(content, "amazon") {
		return "amazon"
	}
	
	return ""
}