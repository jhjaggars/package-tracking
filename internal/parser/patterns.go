package parser

import (
	"regexp"
	"strings"

	"package-tracking/internal/email"
)

// PatternManager handles carrier-specific regex patterns for tracking number extraction
type PatternManager struct {
	upsPatterns    []*PatternEntry
	uspsPatterns   []*PatternEntry
	fedexPatterns  []*PatternEntry
	dhlPatterns    []*PatternEntry
	genericPatterns []*PatternEntry
}

// PatternEntry represents a regex pattern with metadata
type PatternEntry struct {
	Regex       *regexp.Regexp
	Carrier     string
	Format      string
	Confidence  float64
	Context     string
	Description string
}

// NewPatternManager creates a new pattern manager with all carrier patterns
func NewPatternManager() *PatternManager {
	pm := &PatternManager{}
	pm.initializePatterns()
	return pm
}

// initializePatterns sets up all the regex patterns for each carrier
func (pm *PatternManager) initializePatterns() {
	pm.initUPSPatterns()
	pm.initUSPSPatterns()
	pm.initFedExPatterns()
	pm.initDHLPatterns()
	pm.initGenericPatterns()
}

// initUPSPatterns initializes UPS tracking number patterns
func (pm *PatternManager) initUPSPatterns() {
	pm.upsPatterns = []*PatternEntry{
		// Direct UPS pattern - most reliable
		{
			Regex:       regexp.MustCompile(`\b1Z[A-Z0-9]{6}\d{2}\d{7}\b`),
			Carrier:     "ups",
			Format:      "standard",
			Confidence:  0.9,
			Context:     "direct",
			Description: "Standard UPS tracking number format",
		},
		// Labeled context patterns - more precise to avoid capturing surrounding words
		{
			Regex:       regexp.MustCompile(`(?i)(?:tracking\s*(?:number|#|id)?|shipment\s*(?:id|number)?)\s*:?\s*(1Z[A-Z0-9]{6}\d{2}\d{7})\b`),
			Carrier:     "ups",
			Format:      "labeled",
			Confidence:  0.8,
			Context:     "labeled",
			Description: "UPS number with tracking label",
		},
		// Table/structured data - more precise pattern
		{
			Regex:       regexp.MustCompile(`<td[^>]*>(1Z[A-Z0-9]{6}\d{2}\d{7})</td>`),
			Carrier:     "ups",
			Format:      "table",
			Confidence:  0.7,
			Context:     "table",
			Description: "UPS number in HTML table",
		},
		// Spaced format (common in emails)
		{
			Regex:       regexp.MustCompile(`\b1Z\s?[A-Z0-9]{3}\s?[A-Z0-9]{3}\s?\d{2}\s?\d{4}\s?\d{3}\b`),
			Carrier:     "ups",
			Format:      "spaced",
			Confidence:  0.8,
			Context:     "formatted",
			Description: "UPS number with spacing",
		},
	}
}

// initUSPSPatterns initializes USPS tracking number patterns
func (pm *PatternManager) initUSPSPatterns() {
	pm.uspsPatterns = []*PatternEntry{
		// Priority Mail patterns
		{
			Regex:       regexp.MustCompile(`\b94\d{20}\b`),
			Carrier:     "usps",
			Format:      "priority_mail",
			Confidence:  0.9,
			Context:     "direct",
			Description: "USPS Priority Mail 22-digit",
		},
		{
			Regex:       regexp.MustCompile(`\b93\d{20}\b`),
			Carrier:     "usps",
			Format:      "signature_confirmation",
			Confidence:  0.9,
			Context:     "direct",
			Description: "USPS Signature Confirmation",
		},
		{
			Regex:       regexp.MustCompile(`\b92\d{20}\b`),
			Carrier:     "usps",
			Format:      "certified_mail",
			Confidence:  0.9,
			Context:     "direct",
			Description: "USPS Certified Mail",
		},
		{
			Regex:       regexp.MustCompile(`\b91\d{20}\b`),
			Carrier:     "usps",
			Format:      "signature_confirmation",
			Confidence:  0.9,
			Context:     "direct",
			Description: "USPS Signature Confirmation",
		},
		// Certified Mail
		{
			Regex:       regexp.MustCompile(`\b7\d{19}\b`),
			Carrier:     "usps",
			Format:      "certified_mail",
			Confidence:  0.9,
			Context:     "direct",
			Description: "USPS Certified Mail 20-digit",
		},
		// International patterns
		{
			Regex:       regexp.MustCompile(`\b[A-Z]{2}\d{9}US\b`),
			Carrier:     "usps",
			Format:      "international",
			Confidence:  0.9,
			Context:     "direct",
			Description: "USPS International format",
		},
		{
			Regex:       regexp.MustCompile(`\b(LC|LK|EA|CP|RA|RB|RC|RD)\d{9}US\b`),
			Carrier:     "usps",
			Format:      "international_specific",
			Confidence:  0.95,
			Context:     "direct",
			Description: "USPS International specific services",
		},
		// Express Mail International
		{
			Regex:       regexp.MustCompile(`\b82\d{8}\b`),
			Carrier:     "usps",
			Format:      "express_international",
			Confidence:  0.8,
			Context:     "direct",
			Description: "USPS Express Mail International",
		},
		// Labeled context patterns
		{
			Regex:       regexp.MustCompile(`(?i)(?:tracking\s*(?:number|#)?|usps)\s*:?\s*([94][0-9\s]{20,25})`),
			Carrier:     "usps",
			Format:      "labeled_priority",
			Confidence:  0.8,
			Context:     "labeled",
			Description: "USPS Priority Mail with label",
		},
		{
			Regex:       regexp.MustCompile(`(?i)(?:tracking\s*(?:number|#)?|usps)\s*:?\s*([A-Z]{2}[0-9]{9}US)`),
			Carrier:     "usps",
			Format:      "labeled_international",
			Confidence:  0.8,
			Context:     "labeled",
			Description: "USPS International with label",
		},
		// Spaced formats
		{
			Regex:       regexp.MustCompile(`\b94\d{2}\s?\d{4}\s?\d{4}\s?\d{4}\s?\d{4}\s?\d{4}\b`),
			Carrier:     "usps",
			Format:      "spaced_priority",
			Confidence:  0.8,
			Context:     "formatted",
			Description: "USPS Priority Mail with spacing",
		},
	}
}

// initFedExPatterns initializes FedEx tracking number patterns
func (pm *PatternManager) initFedExPatterns() {
	pm.fedexPatterns = []*PatternEntry{
		// Direct numeric patterns (FedEx uses only digits)
		{
			Regex:       regexp.MustCompile(`\b\d{12}\b`),
			Carrier:     "fedex",
			Format:      "express_12",
			Confidence:  0.6, // Lower confidence due to ambiguity
			Context:     "direct",
			Description: "FedEx Express 12-digit",
		},
		{
			Regex:       regexp.MustCompile(`\b\d{14}\b`),
			Carrier:     "fedex",
			Format:      "ground_14",
			Confidence:  0.7,
			Context:     "direct",
			Description: "FedEx Ground 14-digit",
		},
		{
			Regex:       regexp.MustCompile(`\b\d{15}\b`),
			Carrier:     "fedex",
			Format:      "ground_15",
			Confidence:  0.7,
			Context:     "direct",
			Description: "FedEx Ground 15-digit",
		},
		{
			Regex:       regexp.MustCompile(`\b\d{18}\b`),
			Carrier:     "fedex",
			Format:      "ground_18",
			Confidence:  0.8,
			Context:     "direct",
			Description: "FedEx Ground 18-digit",
		},
		{
			Regex:       regexp.MustCompile(`\b\d{20}\b`),
			Carrier:     "fedex",
			Format:      "ground_20",
			Confidence:  0.8,
			Context:     "direct",
			Description: "FedEx Ground 20-digit",
		},
		{
			Regex:       regexp.MustCompile(`\b\d{22}\b`),
			Carrier:     "fedex",
			Format:      "ground_22",
			Confidence:  0.8,
			Context:     "direct",
			Description: "FedEx Ground 22-digit",
		},
		// Labeled context patterns (higher confidence)
		{
			Regex:       regexp.MustCompile(`(?i)(?:fedex|tracking\s*(?:number|#)?)\s*:?\s*(\d{12,22})`),
			Carrier:     "fedex",
			Format:      "labeled",
			Confidence:  0.9,
			Context:     "labeled",
			Description: "FedEx number with label",
		},
		// Spaced formats
		{
			Regex:       regexp.MustCompile(`\b\d{4}\s?\d{4}\s?\d{4}\b`),
			Carrier:     "fedex",
			Format:      "spaced_12",
			Confidence:  0.7,
			Context:     "formatted",
			Description: "FedEx 12-digit with spacing",
		},
		{
			Regex:       regexp.MustCompile(`\b\d{4}\s?\d{4}\s?\d{4}\s?\d{2}\b`),
			Carrier:     "fedex",
			Format:      "spaced_14",
			Confidence:  0.7,
			Context:     "formatted",
			Description: "FedEx 14-digit with spacing",
		},
	}
}

// initDHLPatterns initializes DHL tracking number patterns
func (pm *PatternManager) initDHLPatterns() {
	pm.dhlPatterns = []*PatternEntry{
		// Only use labeled patterns for DHL to avoid false positives
		// Direct numeric patterns are too ambiguous and match common words
		// Labeled context patterns (much higher confidence)
		{
			Regex:       regexp.MustCompile(`(?i)(?:dhl|tracking\s*(?:number|#)?)\s*:?\s*(\d{10,11})`),
			Carrier:     "dhl",
			Format:      "labeled",
			Confidence:  0.9,
			Context:     "labeled",
			Description: "DHL number with label",
		},
		// Waybill format
		{
			Regex:       regexp.MustCompile(`(?i)waybill\s*(?:number|#)?\s*:?\s*(\d{10,11})`),
			Carrier:     "dhl",
			Format:      "waybill",
			Confidence:  0.9,
			Context:     "labeled",
			Description: "DHL waybill number",
		},
	}
}

// initGenericPatterns initializes generic patterns for any carrier
func (pm *PatternManager) initGenericPatterns() {
	pm.genericPatterns = []*PatternEntry{
		// Generic tracking number patterns - more flexible but still targeted
		{
			Regex:       regexp.MustCompile(`(?i)tracking\s*(?:number|#|id)\s*(?::|is)?\s*([A-Z0-9]{10,25})`),
			Carrier:     "unknown",
			Format:      "generic_labeled",
			Confidence:  0.6,
			Context:     "labeled",
			Description: "Generic tracking number with explicit label",
		},
		{
			Regex:       regexp.MustCompile(`(?i)shipment\s*(?:id|number)\s*:?\s*([A-Z0-9]{10,25})`),
			Carrier:     "unknown",
			Format:      "generic_shipment", 
			Confidence:  0.5,
			Context:     "labeled",
			Description: "Generic shipment number with explicit label",
		},
		// Simple tracking pattern for emails with minimal context
		{
			Regex:       regexp.MustCompile(`(?i)tracking:\s*([A-Z0-9]{10,25})`),
			Carrier:     "unknown",
			Format:      "simple_colon",
			Confidence:  0.7,
			Context:     "labeled",
			Description: "Simple tracking: format",
		},
		// Removed overly broad package pattern to reduce false positives
	}
}

// ExtractForCarrier extracts tracking candidates for a specific carrier
func (pm *PatternManager) ExtractForCarrier(text, carrier string) []email.TrackingCandidate {
	var patterns []*PatternEntry
	
	switch carrier {
	case "ups":
		patterns = pm.upsPatterns
	case "usps":
		patterns = pm.uspsPatterns
	case "fedex":
		patterns = pm.fedexPatterns
	case "dhl":
		patterns = pm.dhlPatterns
	default:
		return nil
	}
	
	return pm.extractWithPatterns(text, patterns)
}

// ExtractGeneric extracts tracking candidates using generic patterns
func (pm *PatternManager) ExtractGeneric(text string) []email.TrackingCandidate {
	return pm.extractWithPatterns(text, pm.genericPatterns)
}

// extractWithPatterns applies a set of patterns to extract candidates
func (pm *PatternManager) extractWithPatterns(text string, patterns []*PatternEntry) []email.TrackingCandidate {
	var candidates []email.TrackingCandidate
	
	for _, pattern := range patterns {
		matches := pattern.Regex.FindAllStringSubmatch(text, -1)
		indices := pattern.Regex.FindAllStringIndex(text, -1)
		
		for i, match := range matches {
			var trackingNumber string
			var position int
			
			if len(match) > 1 {
				// Use captured group
				trackingNumber = strings.TrimSpace(match[1])
				// Find position of the captured group
				position = indices[i][0]
			} else {
				// Use full match
				trackingNumber = strings.TrimSpace(match[0])
				position = indices[i][0]
			}
			
			if trackingNumber == "" {
				continue
			}
			
			// Extract context around the match
			context := pm.extractContext(text, position, 50)
			
			candidate := email.TrackingCandidate{
				Text:       trackingNumber,
				Position:   position,
				Context:    context,
				Carrier:    pattern.Carrier,
				Confidence: pattern.Confidence,
				Method:     pattern.Context,
			}
			
			candidates = append(candidates, candidate)
		}
	}
	
	return candidates
}

// extractContext extracts surrounding text for context
func (pm *PatternManager) extractContext(text string, position, radius int) string {
	start := position - radius
	if start < 0 {
		start = 0
	}
	
	end := position + radius
	if end > len(text) {
		end = len(text)
	}
	
	context := text[start:end]
	
	// Clean up context
	context = strings.ReplaceAll(context, "\n", " ")
	context = strings.ReplaceAll(context, "\t", " ")
	
	// Normalize whitespace
	re := regexp.MustCompile(`\s+`)
	context = re.ReplaceAllString(context, " ")
	
	return strings.TrimSpace(context)
}

// GetAllPatterns returns all patterns for debugging/testing
func (pm *PatternManager) GetAllPatterns() map[string][]*PatternEntry {
	return map[string][]*PatternEntry{
		"ups":     pm.upsPatterns,
		"usps":    pm.uspsPatterns,
		"fedex":   pm.fedexPatterns,
		"dhl":     pm.dhlPatterns,
		"generic": pm.genericPatterns,
	}
}

// ValidatePattern tests if a pattern works correctly
func (pm *PatternManager) ValidatePattern(pattern *PatternEntry, testString string) bool {
	matches := pattern.Regex.FindStringSubmatch(testString)
	return len(matches) > 0
}