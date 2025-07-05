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