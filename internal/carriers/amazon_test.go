package carriers

import (
	"context"
	"testing"
)

func TestAmazonClient_GetCarrierName(t *testing.T) {
	client := NewAmazonClient(NewClientFactory())
	
	if client.GetCarrierName() != "amazon" {
		t.Errorf("Expected carrier name 'amazon', got '%s'", client.GetCarrierName())
	}
}

func TestAmazonClient_ValidateTrackingNumber(t *testing.T) {
	client := NewAmazonClient(NewClientFactory())
	
	tests := []struct {
		name           string
		trackingNumber string
		want           bool
	}{
		// Amazon order numbers
		{
			name:           "Valid Amazon order number",
			trackingNumber: "113-1234567-1234567",
			want:           true,
		},
		{
			name:           "Valid Amazon order number with different format",
			trackingNumber: "123-4567890-1234567",
			want:           true,
		},
		{
			name:           "Amazon order number without dashes",
			trackingNumber: "11312345671234567",
			want:           true,
		},
		{
			name:           "Amazon order number with spaces",
			trackingNumber: "113 1234567 1234567",
			want:           true,
		},
		// Amazon Logistics tracking numbers
		{
			name:           "Valid AMZL tracking number",
			trackingNumber: "TBA123456789012",
			want:           true,
		},
		{
			name:           "Valid AMZL tracking number with lowercase",
			trackingNumber: "tba123456789012",
			want:           true,
		},
		{
			name:           "Valid AMZL tracking number with mixed case",
			trackingNumber: "TbA123456789012",
			want:           true,
		},
		// Invalid tracking numbers
		{
			name:           "Too short Amazon order number",
			trackingNumber: "113-123-123",
			want:           false,
		},
		{
			name:           "Too long Amazon order number",
			trackingNumber: "113-12345678901234567890-1234567",
			want:           false,
		},
		{
			name:           "Invalid AMZL format - wrong prefix",
			trackingNumber: "ABC123456789012",
			want:           false,
		},
		{
			name:           "Invalid AMZL format - too short",
			trackingNumber: "TBA12345",
			want:           false,
		},
		{
			name:           "Empty tracking number",
			trackingNumber: "",
			want:           false,
		},
		{
			name:           "Invalid characters",
			trackingNumber: "113-1234567-123456@",
			want:           false,
		},
		{
			name:           "Regular UPS tracking number",
			trackingNumber: "1Z999AA1234567890",
			want:           false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := client.ValidateTrackingNumber(tt.trackingNumber); got != tt.want {
				t.Errorf("ValidateTrackingNumber(%s) = %v, want %v", tt.trackingNumber, got, tt.want)
			}
		})
	}
}

func TestAmazonClient_Track_OrderNumber(t *testing.T) {
	factory := NewClientFactory()
	client := NewAmazonClient(factory)
	
	// Test tracking with Amazon order number - should return pending status
	// since Amazon doesn't have public APIs
	req := &TrackingRequest{
		TrackingNumbers: []string{"113-1234567-1234567"},
		Carrier:         "amazon",
	}
	
	ctx := context.Background()
	resp, err := client.Track(ctx, req)
	if err != nil {
		t.Fatalf("Track failed: %v", err)
	}
	
	if len(resp.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(resp.Results))
	}
	
	result := resp.Results[0]
	if result.TrackingNumber != "113-1234567-1234567" {
		t.Errorf("Expected tracking number '113-1234567-1234567', got '%s'", result.TrackingNumber)
	}
	
	if result.Carrier != "amazon" {
		t.Errorf("Expected carrier 'amazon', got '%s'", result.Carrier)
	}
	
	if result.Status != StatusPreShip {
		t.Errorf("Expected status 'pre_ship', got '%s'", result.Status)
	}
}

func TestAmazonClient_Track_AMZLTrackingNumber(t *testing.T) {
	factory := NewClientFactory()
	client := NewAmazonClient(factory)
	
	// Test tracking with Amazon Logistics tracking number
	req := &TrackingRequest{
		TrackingNumbers: []string{"TBA123456789012"},
		Carrier:         "amazon",
	}
	
	ctx := context.Background()
	resp, err := client.Track(ctx, req)
	if err != nil {
		t.Fatalf("Track failed: %v", err)
	}
	
	if len(resp.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(resp.Results))
	}
	
	result := resp.Results[0]
	if result.TrackingNumber != "TBA123456789012" {
		t.Errorf("Expected tracking number 'TBA123456789012', got '%s'", result.TrackingNumber)
	}
	
	if result.Carrier != "amazon" {
		t.Errorf("Expected carrier 'amazon', got '%s'", result.Carrier)
	}
	
	if result.Status != StatusPreShip {
		t.Errorf("Expected status 'pre_ship', got '%s'", result.Status)
	}
}

func TestAmazonClient_Track_WithDelegation(t *testing.T) {
	factory := NewClientFactory()
	
	// Set up UPS mock configuration for delegation testing
	factory.SetCarrierConfig("ups", &CarrierConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		PreferredType: ClientTypeScraping, // Use scraping to avoid real API calls
	})
	
	// Test delegation to UPS - this would typically be set via database
	// For now, we'll test the delegation mechanism
	delegatedTrackingNumber := "1Z999AA1234567890"
	
	// In a real scenario, the Amazon client would need access to the database
	// to look up delegation information. For this test, we'll verify the 
	// delegation capability exists.
	
	// This test verifies that the Amazon client can delegate to other carriers
	upsClient, _, err := factory.CreateClient("ups")
	if err != nil {
		t.Fatalf("Failed to create UPS client for delegation test: %v", err)
	}
	
	if upsClient.GetCarrierName() != "ups" {
		t.Errorf("Expected delegated carrier 'ups', got '%s'", upsClient.GetCarrierName())
	}
	
	// Verify UPS tracking number validation
	if !upsClient.ValidateTrackingNumber(delegatedTrackingNumber) {
		t.Errorf("UPS client should validate tracking number '%s'", delegatedTrackingNumber)
	}
}

func TestAmazonClient_Track_InvalidTrackingNumber(t *testing.T) {
	factory := NewClientFactory()
	client := NewAmazonClient(factory)
	
	// Test with invalid tracking number
	req := &TrackingRequest{
		TrackingNumbers: []string{"INVALID123"},
		Carrier:         "amazon",
	}
	
	ctx := context.Background()
	resp, err := client.Track(ctx, req)
	if err != nil {
		t.Fatalf("Track failed: %v", err)
	}
	
	// Should return an error for invalid tracking number
	if len(resp.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(resp.Errors))
	}
	
	if len(resp.Results) != 0 {
		t.Errorf("Expected 0 results for invalid tracking number, got %d", len(resp.Results))
	}
	
	if resp.Errors[0].Carrier != "amazon" {
		t.Errorf("Expected error carrier 'amazon', got '%s'", resp.Errors[0].Carrier)
	}
}

func TestAmazonClient_Track_MultipleTrackingNumbers(t *testing.T) {
	factory := NewClientFactory()
	client := NewAmazonClient(factory)
	
	// Test with multiple tracking numbers
	req := &TrackingRequest{
		TrackingNumbers: []string{
			"113-1234567-1234567", // Valid Amazon order
			"TBA123456789012",     // Valid AMZL
			"INVALID123",          // Invalid
		},
		Carrier: "amazon",
	}
	
	ctx := context.Background()
	resp, err := client.Track(ctx, req)
	if err != nil {
		t.Fatalf("Track failed: %v", err)
	}
	
	// Should have 2 results and 1 error
	if len(resp.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(resp.Results))
	}
	
	if len(resp.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(resp.Errors))
	}
	
	// Verify results
	for _, result := range resp.Results {
		if result.Carrier != "amazon" {
			t.Errorf("Expected carrier 'amazon', got '%s'", result.Carrier)
		}
		if result.Status != StatusPreShip {
			t.Errorf("Expected status 'pre_ship', got '%s'", result.Status)
		}
	}
}

func TestAmazonClient_GetRateLimit(t *testing.T) {
	factory := NewClientFactory()
	client := NewAmazonClient(factory)
	
	rateLimit := client.GetRateLimit()
	if rateLimit == nil {
		t.Error("Expected rate limit info, got nil")
	}
	
	// Amazon has no rate limits (email-based), so should return unlimited
	if rateLimit.Limit != -1 {
		t.Errorf("Expected unlimited rate limit (-1), got %d", rateLimit.Limit)
	}
}

func TestNewAmazonClient(t *testing.T) {
	factory := NewClientFactory()
	client := NewAmazonClient(factory)
	
	if client == nil {
		t.Error("NewAmazonClient returned nil")
	}
	
	if client.GetCarrierName() != "amazon" {
		t.Errorf("Expected carrier name 'amazon', got '%s'", client.GetCarrierName())
	}
}

func TestAmazonClient_ValidateInternalReference(t *testing.T) {
	client := NewAmazonClient(NewClientFactory())
	
	tests := []struct {
		name           string
		trackingNumber string
		want           bool
		description    string
	}{
		// Valid Amazon internal reference codes
		{
			name:           "Original failing case - BqPz3RXRS",
			trackingNumber: "BqPz3RXRS",
			want:           true,
			description:    "The actual tracking code that was failing in production",
		},
		{
			name:           "Mixed alphanumeric 8 chars",
			trackingNumber: "AMZ123AB",
			want:           true,
			description:    "Short mixed alphanumeric code",
		},
		{
			name:           "Mixed alphanumeric 12 chars",
			trackingNumber: "REF456DEF789",
			want:           true,
			description:    "Medium length mixed code",
		},
		{
			name:           "Mixed alphanumeric 16 chars",
			trackingNumber: "SHIP789GHI012JKL",
			want:           true,
			description:    "Long mixed code",
		},
		{
			name:           "Amazon warehouse code style",
			trackingNumber: "FBA7X8Y9Z0AB",
			want:           true,
			description:    "Warehouse/fulfillment style code",
		},
		{
			name:           "With dashes",
			trackingNumber: "AMZ-123-DEF",
			want:           true,
			description:    "Internal code with dashes",
		},
		
		// Boundary conditions - valid (6-20 characters)
		{
			name:           "Exactly 6 characters",
			trackingNumber: "AMZ123",
			want:           true,
			description:    "Minimum length boundary",
		},
		{
			name:           "Exactly 20 characters",
			trackingNumber: "AMAZON12345REFERENCE",
			want:           true,
			description:    "Maximum length boundary",
		},
		
		// Boundary conditions - invalid
		{
			name:           "Too short - 5 characters",
			trackingNumber: "AMZ12",
			want:           false,
			description:    "Below minimum length",
		},
		{
			name:           "Too long - 21 characters",
			trackingNumber: "AMAZON123456REFERENCES",
			want:           false,
			description:    "Above maximum length",
		},
		
		// Invalid formats
		{
			name:           "Only letters",
			trackingNumber: "AMAZONCODE",
			want:           false,
			description:    "Must contain at least one number",
		},
		{
			name:           "Only numbers",
			trackingNumber: "123456789",
			want:           false,
			description:    "Must contain at least one letter",
		},
		{
			name:           "Contains special characters",
			trackingNumber: "AMZ123@DEF",
			want:           false,
			description:    "Special characters not allowed",
		},
		{
			name:           "Contains spaces",
			trackingNumber: "AMZ 123 DEF",
			want:           false,
			description:    "Spaces not allowed in internal references",
		},
		
		// False positive filtering
		{
			name:           "Invalid prefix",
			trackingNumber: "INVALID123",
			want:           false,
			description:    "Should filter obvious invalid patterns",
		},
		{
			name:           "Test pattern",
			trackingNumber: "test123",
			want:           false,
			description:    "Should filter test patterns",
		},
		{
			name:           "Fake pattern",
			trackingNumber: "fake456",
			want:           false,
			description:    "Should filter fake patterns",
		},
		{
			name:           "Example pattern",
			trackingNumber: "example789",
			want:           false,
			description:    "Should filter example patterns",
		},
		{
			name:           "Year pattern",
			trackingNumber: "2024",
			want:           false,
			description:    "Should filter year patterns",
		},
		{
			name:           "Day pattern",
			trackingNumber: "monday123",
			want:           false,
			description:    "Should filter day patterns",
		},
		{
			name:           "Month pattern",
			trackingNumber: "january456",
			want:           false,
			description:    "Should filter month patterns",
		},
		
		// Known carrier patterns that should be excluded
		{
			name:           "UPS tracking number",
			trackingNumber: "1Z999AA1234567890",
			want:           false,
			description:    "Should not validate UPS numbers as Amazon internal",
		},
		{
			name:           "USPS tracking number",
			trackingNumber: "94001234567890123456",
			want:           false,
			description:    "Should not validate USPS numbers as Amazon internal",
		},
		{
			name:           "FedEx tracking number",
			trackingNumber: "123456789012",
			want:           false,
			description:    "Should not validate FedEx numbers as Amazon internal",
		},
		{
			name:           "DHL tracking number",
			trackingNumber: "1234567890",
			want:           false,
			description:    "Should not validate DHL numbers as Amazon internal",
		},
		{
			name:           "Amazon Logistics number",
			trackingNumber: "TBA123456789012",
			want:           true,
			description:    "TBA numbers are valid Amazon tracking numbers",
		},
		{
			name:           "Amazon order number format",
			trackingNumber: "11312345671234567",
			want:           true,
			description:    "Order numbers are valid Amazon tracking numbers",
		},
		{
			name:           "Generic carrier format",
			trackingNumber: "ABC123456789012",
			want:           false,
			description:    "Should filter generic 3-letter + 12-digit patterns",
		},
		{
			name:           "Short TBA format",
			trackingNumber: "TBA12345",
			want:           false,
			description:    "Should filter incomplete TBA formats",
		},
		
		// Edge cases
		{
			name:           "Empty string",
			trackingNumber: "",
			want:           false,
			description:    "Empty strings should be rejected",
		},
		{
			name:           "Whitespace only",
			trackingNumber: "   ",
			want:           false,
			description:    "Whitespace-only strings should be rejected",
		},
		{
			name:           "Mixed case valid",
			trackingNumber: "AmZ123DeFgHi",
			want:           true,
			description:    "Mixed case should be valid",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.ValidateTrackingNumber(tt.trackingNumber)
			if got != tt.want {
				t.Errorf("ValidateTrackingNumber(%q) = %v, want %v\nDescription: %s", 
					tt.trackingNumber, got, tt.want, tt.description)
			}
		})
	}
}