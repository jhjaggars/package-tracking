package parser

import (
	"testing"

	"package-tracking/internal/email"
)

func TestNewAmazonPatterns(t *testing.T) {
	pm := NewPatternManager()
	
	tests := []struct {
		name     string
		text     string
		expected []email.TrackingCandidate
	}{
		// Test new Amazon contextual reference pattern
		{
			name: "Amazon contextual reference - amazon reference code",
			text: "Your Amazon reference code: AMZ123DEF456 has been processed.",
			expected: []email.TrackingCandidate{
				{
					Text:       "AMZ123DEF456",
					Carrier:    "amazon",
					Confidence: 0.7,
					Method:     "contextual",
				},
			},
		},
		{
			name: "Amazon contextual reference - amazon id",
			text: "Amazon ID: REF789GHI012 for your shipment.",
			expected: []email.TrackingCandidate{
				{
					Text:       "REF789GHI012",
					Carrier:    "amazon",
					Confidence: 0.7,
					Method:     "contextual",
				},
			},
		},
		{
			name: "Amazon contextual reference - amazon code",
			text: "Use Amazon code BqPz3RXRS to track your package.",
			expected: []email.TrackingCandidate{
				{
					Text:       "BqPz3RXRS",
					Carrier:    "amazon",
					Confidence: 0.7,
					Method:     "contextual",
				},
			},
		},
		
		// Test new Amazon shipment reference pattern
		{
			name: "Amazon shipment reference",
			text: "Amazon shipment reference: SHIP123ABC456 is now in transit.",
			expected: []email.TrackingCandidate{
				{
					Text:       "SHIP123ABC456",
					Carrier:    "amazon",
					Confidence: 0.7,
					Method:     "contextual",
				},
				{
					Text:       "SHIP123ABC456",
					Carrier:    "amazon",
					Confidence: 0.8,
					Method:     "labeled",
				},
			},
		},
		{
			name: "Amazon package reference",
			text: "Your Amazon package number: PKG789XYZ012.",
			expected: []email.TrackingCandidate{
				{
					Text:       "PKG789XYZ012",
					Carrier:    "amazon",
					Confidence: 0.8,
					Method:     "labeled",
				},
			},
		},
		{
			name: "Amazon order reference",
			text: "Amazon order reference code FBA456DEF is ready for pickup.",
			expected: []email.TrackingCandidate{
				{
					Text:       "FBA456DEF",
					Carrier:    "amazon",
					Confidence: 0.7,
					Method:     "contextual",
				},
			},
		},
		
		// Test that overly broad patterns are NOT matching
		{
			name: "Should not match generic alphanumeric without Amazon context",
			text: "Your tracking number is ABC123DEF456 from UPS.",
			expected: []email.TrackingCandidate{}, // Should not match Amazon patterns
		},
		{
			name: "Should not match in non-Amazon context",
			text: "Reference code XYZ789 is available at customer service.",
			expected: []email.TrackingCandidate{}, // Should not match Amazon patterns
		},
		{
			name: "Should not match UPS numbers as Amazon",
			text: "Amazon uses UPS tracking: 1Z999AA1234567890 for this delivery.",
			expected: []email.TrackingCandidate{}, // UPS pattern should be filtered out by other validation
		},
		
		// Case sensitivity tests
		{
			name: "Case insensitive Amazon reference",
			text: "AMAZON REFERENCE: AMZ123lower for your delivery.",
			expected: []email.TrackingCandidate{
				{
					Text:       "AMZ123lower",
					Carrier:    "amazon",
					Confidence: 0.7,
					Method:     "contextual",
				},
			},
		},
		{
			name: "Mixed case Amazon shipment",
			text: "amazon Shipment Reference: mixedCASE123 is processed.",
			expected: []email.TrackingCandidate{
				{
					Text:       "mixedCASE123",
					Carrier:    "amazon",
					Confidence: 0.7,
					Method:     "contextual",
				},
				{
					Text:       "mixedCASE123",
					Carrier:    "amazon",
					Confidence: 0.8,
					Method:     "labeled",
				},
			},
		},
		
		// Edge cases for patterns
		{
			name: "Minimum length Amazon reference",
			text: "Amazon code: MIN6CH works fine.",
			expected: []email.TrackingCandidate{
				{
					Text:       "MIN6CH",
					Carrier:    "amazon",
					Confidence: 0.7,
					Method:     "contextual",
				},
			},
		},
		{
			name: "Maximum length Amazon reference",
			text: "Amazon reference: MAXLENGTHREFERENCE20 is valid.",
			expected: []email.TrackingCandidate{
				{
					Text:       "MAXLENGTHREFERENCE20",
					Carrier:    "amazon",
					Confidence: 0.7,
					Method:     "contextual",
				},
			},
		},
		{
			name: "Too short for Amazon patterns",
			text: "Amazon code: X1 is too short.",
			expected: []email.TrackingCandidate{}, // Too short to match
		},
		{
			name: "Too long for Amazon patterns",
			text: "Amazon reference: VERYLONGCODETHATEXCEEDSTWENTYCHARACTERS should not match.",
			expected: []email.TrackingCandidate{
				{
					Text:       "VERYLONGCODETHATEXCE", // Pattern only captures first 20 chars
					Carrier:    "amazon",
					Confidence: 0.7,
					Method:     "contextual",
				},
			},
		},
		
		// Multiple patterns in same text
		{
			name: "Multiple Amazon references",
			text: "Amazon reference: REF123ABC and Amazon shipment code: SHIP456DEF are both valid.",
			expected: []email.TrackingCandidate{
				{
					Text:       "REF123ABC",
					Carrier:    "amazon",
					Confidence: 0.7,
					Method:     "contextual",
				},
				{
					Text:       "SHIP456DEF",
					Carrier:    "amazon",
					Confidence: 0.7,
					Method:     "contextual",
				},
				{
					Text:       "SHIP456DEF",
					Carrier:    "amazon",
					Confidence: 0.8,
					Method:     "labeled",
				},
			},
		},
		
		// Test with special characters and spacing
		{
			name: "Amazon reference with colon spacing",
			text: "Amazon reference   :   SPACED123 tracking update.",
			expected: []email.TrackingCandidate{
				{
					Text:       "SPACED123",
					Carrier:    "amazon",
					Confidence: 0.7,
					Method:     "contextual",
				},
			},
		},
		{
			name: "Amazon shipment without colon",
			text: "Amazon shipment reference NOCOLON456 is in transit.",
			expected: []email.TrackingCandidate{
				{
					Text:       "NOCOLON456",
					Carrier:    "amazon",
					Confidence: 0.7,
					Method:     "contextual",
				},
				{
					Text:       "NOCOLON456",
					Carrier:    "amazon",
					Confidence: 0.8,
					Method:     "labeled",
				},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := pm.ExtractForCarrier(tt.text, "amazon")
			
			if len(candidates) != len(tt.expected) {
				t.Errorf("Expected %d candidates, got %d\nCandidates: %+v\nExpected: %+v", 
					len(tt.expected), len(candidates), candidates, tt.expected)
				return
			}
			
			for i, candidate := range candidates {
				if i >= len(tt.expected) {
					t.Errorf("Unexpected extra candidate: %+v", candidate)
					continue
				}
				
				expected := tt.expected[i]
				
				if candidate.Text != expected.Text {
					t.Errorf("Candidate[%d].Text = %s, want %s", i, candidate.Text, expected.Text)
				}
				if candidate.Carrier != expected.Carrier {
					t.Errorf("Candidate[%d].Carrier = %s, want %s", i, candidate.Carrier, expected.Carrier)
				}
				if candidate.Confidence != expected.Confidence {
					t.Errorf("Candidate[%d].Confidence = %f, want %f", i, candidate.Confidence, expected.Confidence)
				}
				if candidate.Method != expected.Method {
					t.Errorf("Candidate[%d].Method = %s, want %s", i, candidate.Method, expected.Method)
				}
			}
		})
	}
}

func TestAmazonEnhancedPatternValidation(t *testing.T) {
	pm := NewPatternManager()
	
	// Test specific pattern validation
	patterns := pm.GetAllPatterns()["amazon"]
	
	tests := []struct {
		name        string
		patternDesc string
		testString  string
		shouldMatch bool
	}{
		// Test Amazon contextual reference pattern
		{
			name:        "Contextual pattern - valid",
			patternDesc: "Amazon reference code in Amazon context",
			testString:  "Amazon reference code: BqPz3RXRS",
			shouldMatch: true,
		},
		{
			name:        "Contextual pattern - case insensitive",
			patternDesc: "Amazon reference code in Amazon context",
			testString:  "amazon CODE: TEST123",
			shouldMatch: true,
		},
		{
			name:        "Contextual pattern - without Amazon prefix",
			patternDesc: "Amazon reference code in Amazon context",
			testString:  "Reference code: TEST123",
			shouldMatch: false, // Should require Amazon context
		},
		
		// Test Amazon shipment reference pattern
		{
			name:        "Shipment pattern - valid",
			patternDesc: "Amazon shipment reference with label",
			testString:  "Amazon shipment reference: SHIP123ABC",
			shouldMatch: true,
		},
		{
			name:        "Package pattern - valid",
			patternDesc: "Amazon shipment reference with label",
			testString:  "Amazon package number: PKG456DEF",
			shouldMatch: true,
		},
		{
			name:        "Order pattern - valid",
			patternDesc: "Amazon shipment reference with label",
			testString:  "Amazon order reference: ORD789GHI",
			shouldMatch: true,
		},
		{
			name:        "Shipment pattern - without Amazon prefix",
			patternDesc: "Amazon shipment reference with label",
			testString:  "Shipment reference: SHIP123ABC",
			shouldMatch: false, // Should require Amazon prefix
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for _, pattern := range patterns {
				if pattern.Description == tt.patternDesc {
					matches := pattern.Regex.FindStringSubmatch(tt.testString)
					if (len(matches) > 0) != tt.shouldMatch {
						t.Errorf("Pattern %q with test %q: got match=%v, want match=%v", 
							tt.patternDesc, tt.testString, len(matches) > 0, tt.shouldMatch)
					}
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Pattern with description %q not found", tt.patternDesc)
			}
		})
	}
}

func TestAmazonPatternPerformance(t *testing.T) {
	pm := NewPatternManager()
	
	// Test with a large text block to ensure patterns don't cause performance issues
	largeText := `
	Dear Customer,
	
	Your Amazon order 123-4567890-1234567 has been shipped.
	
	Tracking information:
	- Amazon reference code: BqPz3RXRS
	- Amazon shipment reference: SHIP123ABC456
	- Amazon package number: PKG789DEF012
	- Amazon Logistics tracking: TBA123456789012
	
	Additional references:
	Amazon ID: AMZ111222333
	Amazon code: REF444555666
	
	This package is being delivered by Amazon Logistics.
	For more information, visit amazon.com/orders
	
	Thank you for shopping with Amazon!
	`
	
	// Run extraction multiple times to test performance
	for i := 0; i < 100; i++ {
		candidates := pm.ExtractForCarrier(largeText, "amazon")
		
		if len(candidates) == 0 {
			t.Error("Expected to find Amazon tracking candidates in large text")
			break
		}
		
		// Verify we found the expected patterns without performance degradation
		foundOriginal := false
		for _, candidate := range candidates {
			if candidate.Text == "BqPz3RXRS" {
				foundOriginal = true
				break
			}
		}
		
		if !foundOriginal {
			t.Error("Failed to find the original failing case BqPz3RXRS in performance test")
			break
		}
	}
}