package carriers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUSPSScrapingClient_GetCarrierName(t *testing.T) {
	client := NewUSPSScrapingClient("test-agent")
	if got := client.GetCarrierName(); got != "usps" {
		t.Errorf("GetCarrierName() = %v, want %v", got, "usps")
	}
}

func TestUSPSScrapingClient_ValidateTrackingNumber(t *testing.T) {
	client := NewUSPSScrapingClient("test-agent")
	
	tests := []struct {
		name           string
		trackingNumber string
		want           bool
	}{
		{
			name:           "valid Priority Mail Express",
			trackingNumber: "9400111899562347123456",
			want:           true,
		},
		{
			name:           "valid Priority Mail",
			trackingNumber: "9505511899562347123456",
			want:           true,
		},
		{
			name:           "valid Certified Mail",
			trackingNumber: "9407300000000000000000",
			want:           true,
		},
		{
			name:           "valid Collect on Delivery",
			trackingNumber: "9303300000000000000000",
			want:           true,
		},
		{
			name:           "too short",
			trackingNumber: "123456789",
			want:           false,
		},
		{
			name:           "too long",
			trackingNumber: "123456789012345678901234567890",
			want:           false,
		},
		{
			name:           "empty string",
			trackingNumber: "",
			want:           false,
		},
		{
			name:           "contains letters",
			trackingNumber: "1234567890ABCDEF123456",
			want:           false,
		},
		{
			name:           "valid Global Express Guaranteed",
			trackingNumber: "82000000000",
			want:           true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := client.ValidateTrackingNumber(tt.trackingNumber); got != tt.want {
				t.Errorf("ValidateTrackingNumber(%v) = %v, want %v", tt.trackingNumber, got, tt.want)
			}
		})
	}
}

func TestUSPSScrapingClient_Track_Success(t *testing.T) {
	// Mock USPS tracking page HTML response
	mockHTML := `
<!DOCTYPE html>
<html>
<head><title>USPS Tracking</title></head>
<body>
	<div class="tracking-summary">
		<div class="tracking-number">9400111899562347123456</div>
		<div class="delivery-status">
			<h2>Delivered</h2>
			<p>Your item was delivered at 2:15 pm on May 15, 2023 in ATLANTA GA 30309.</p>
		</div>
	</div>
	
	<div class="tracking-history">
		<div class="tracking-event">
			<div class="event-timestamp">May 15, 2023 at 2:15 PM</div>
			<div class="event-status">Delivered</div>
			<div class="event-location">ATLANTA, GA 30309</div>
			<div class="event-description">Delivered, In/At Mailbox</div>
		</div>
		<div class="tracking-event">
			<div class="event-timestamp">May 15, 2023 at 6:00 AM</div>
			<div class="event-status">Out for Delivery</div>
			<div class="event-location">ATLANTA, GA 30309</div>
			<div class="event-description">Out for Delivery</div>
		</div>
		<div class="tracking-event">
			<div class="event-timestamp">May 14, 2023 at 8:45 PM</div>
			<div class="event-status">Arrived at Facility</div>
			<div class="event-location">ATLANTA, GA 30309</div>
			<div class="event-description">Arrived at USPS Facility</div>
		</div>
	</div>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "track") {
			t.Errorf("Expected path to contain 'track', got %s", r.URL.Path)
		}
		
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		
		// Check User-Agent header
		userAgent := r.Header.Get("User-Agent")
		if userAgent != "test-agent" {
			t.Errorf("Expected User-Agent 'test-agent', got '%s'", userAgent)
		}
		
		// Check tracking number in query parameters
		trackingNumber := r.URL.Query().Get("id")
		if trackingNumber != "9400111899562347123456" {
			t.Errorf("Expected tracking number=9400111899562347123456, got %s", trackingNumber)
		}
		
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	// Create test client with custom base URL
	client := &USPSScrapingClient{
		ScrapingClient: NewScrapingClient("usps", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"9400111899562347123456"},
		Carrier:         "usps",
	}

	ctx := context.Background()
	resp, err := client.Track(ctx, req)

	if err != nil {
		t.Fatalf("Track() error = %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(resp.Results))
	}

	result := resp.Results[0]
	if result.TrackingNumber != "9400111899562347123456" {
		t.Errorf("Expected tracking number 9400111899562347123456, got %s", result.TrackingNumber)
	}

	if result.Status != StatusDelivered {
		t.Errorf("Expected status %s, got %s", StatusDelivered, result.Status)
	}

	if len(result.Events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(result.Events))
	}

	// Check first event (most recent - delivered)
	if result.Events[0].Status != StatusDelivered {
		t.Errorf("Expected first event status %s, got %s", StatusDelivered, result.Events[0].Status)
	}

	if result.Events[0].Location != "ATLANTA, GA 30309" {
		t.Errorf("Expected location 'ATLANTA, GA 30309', got '%s'", result.Events[0].Location)
	}

	if result.Events[0].Description != "Delivered, In/At Mailbox" {
		t.Errorf("Expected description 'Delivered, In/At Mailbox', got '%s'", result.Events[0].Description)
	}
}

func TestUSPSScrapingClient_Track_NotFound(t *testing.T) {
	mockHTML := `
<!DOCTYPE html>
<html>
<head><title>USPS Tracking</title></head>
<body>
	<div class="tracking-summary">
		<div class="error-message">
			<h2>Status Not Available</h2>
			<p>We could not locate the tracking information for your request. Please verify your tracking number and try again.</p>
		</div>
	</div>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	client := &USPSScrapingClient{
		ScrapingClient: NewScrapingClient("usps", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"invalid_tracking"},
		Carrier:         "usps",
	}

	ctx := context.Background()
	resp, err := client.Track(ctx, req)

	if err != nil {
		t.Fatalf("Track() error = %v", err)
	}

	if len(resp.Results) != 0 {
		t.Errorf("Expected 0 results for not found, got %d", len(resp.Results))
	}

	if len(resp.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(resp.Errors))
	}

	carrierErr := resp.Errors[0]
	if carrierErr.Carrier != "usps" {
		t.Errorf("Expected carrier 'usps', got '%s'", carrierErr.Carrier)
	}

	if carrierErr.Code != "NOT_FOUND" {
		t.Errorf("Expected error code 'NOT_FOUND', got '%s'", carrierErr.Code)
	}

	if !strings.Contains(carrierErr.Message, "not found") {
		t.Errorf("Expected error message to contain 'not found', got '%s'", carrierErr.Message)
	}
}

func TestUSPSScrapingClient_Track_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := &USPSScrapingClient{
		ScrapingClient: NewScrapingClient("usps", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"9400111899562347123456"},
		Carrier:         "usps",
	}

	ctx := context.Background()
	_, err := client.Track(ctx, req)

	if err == nil {
		t.Fatal("Expected HTTP error, got nil")
	}

	if !strings.Contains(err.Error(), "HTTP error") {
		t.Errorf("Expected error to contain 'HTTP error', got '%s'", err.Error())
	}
}

func TestUSPSScrapingClient_Track_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("Too Many Requests"))
	}))
	defer server.Close()

	client := &USPSScrapingClient{
		ScrapingClient: NewScrapingClient("usps", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"9400111899562347123456"},
		Carrier:         "usps",
	}

	ctx := context.Background()
	_, err := client.Track(ctx, req)

	if err == nil {
		t.Fatal("Expected rate limit error, got nil")
	}

	carrierErr, ok := err.(*CarrierError)
	if !ok {
		t.Fatalf("Expected CarrierError, got %T", err)
	}

	if !carrierErr.RateLimit {
		t.Error("Expected RateLimit to be true")
	}

	if !carrierErr.Retryable {
		t.Error("Expected Retryable to be true for rate limit error")
	}

	if carrierErr.Carrier != "usps" {
		t.Errorf("Expected carrier 'usps', got '%s'", carrierErr.Carrier)
	}
}