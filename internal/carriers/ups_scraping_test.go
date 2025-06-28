package carriers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUPSScrapingClient_GetCarrierName(t *testing.T) {
	client := NewUPSScrapingClient("test-agent")
	if got := client.GetCarrierName(); got != "ups" {
		t.Errorf("GetCarrierName() = %v, want %v", got, "ups")
	}
}

func TestUPSScrapingClient_ValidateTrackingNumber(t *testing.T) {
	client := NewUPSScrapingClient("test-agent")
	
	tests := []struct {
		name           string
		trackingNumber string
		want           bool
	}{
		{
			name:           "valid UPS tracking number",
			trackingNumber: "1Z999AA1234567890",
			want:           true,
		},
		{
			name:           "valid UPS tracking number lowercase",
			trackingNumber: "1z999aa1234567890",
			want:           true,
		},
		{
			name:           "valid UPS tracking number with spaces",
			trackingNumber: "1Z 999 AA1 234 567 890",
			want:           true,
		},
		{
			name:           "too short",
			trackingNumber: "1Z999AA123456789",
			want:           false,
		},
		{
			name:           "too long",
			trackingNumber: "1Z999AA12345678901",
			want:           false,
		},
		{
			name:           "wrong prefix",
			trackingNumber: "2Z999AA1234567890",
			want:           false,
		},
		{
			name:           "empty string",
			trackingNumber: "",
			want:           false,
		},
		{
			name:           "invalid characters",
			trackingNumber: "1Z999@@1234567890",
			want:           false,
		},
		{
			name:           "wrong format",
			trackingNumber: "1234567890123456",
			want:           false,
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

func TestUPSScrapingClient_Track_Success(t *testing.T) {
	// Mock UPS tracking page HTML response
	mockHTML := `
<!DOCTYPE html>
<html>
<head><title>UPS Tracking</title></head>
<body>
	<div class="ups-public_trackingDetails">
		<div class="tracking-number">1Z999AA1234567890</div>
		<div class="package-status">
			<h2>Delivered</h2>
			<p>Your package was delivered on Monday 05/15/2023 at 2:15 PM.</p>
		</div>
	</div>
	
	<div class="ups-public_trackingProgressBarContainer">
		<div class="progress-step delivered">
			<div class="step-date">May 15, 2023</div>
			<div class="step-time">2:15 PM</div>
			<div class="step-status">Delivered</div>
			<div class="step-location">ATLANTA, GA 30309, US</div>
			<div class="step-description">Package delivered to recipient</div>
		</div>
		<div class="progress-step">
			<div class="step-date">May 15, 2023</div>
			<div class="step-time">6:00 AM</div>
			<div class="step-status">Out For Delivery</div>
			<div class="step-location">ATLANTA, GA 30309, US</div>
			<div class="step-description">On UPS vehicle for delivery</div>
		</div>
		<div class="progress-step">
			<div class="step-date">May 14, 2023</div>
			<div class="step-time">8:45 PM</div>
			<div class="step-status">Arrival Scan</div>
			<div class="step-location">ATLANTA, GA 30309, US</div>
			<div class="step-description">Arrived at facility</div>
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
		trackingNumber := r.URL.Query().Get("tracknum")
		if trackingNumber != "1Z999AA1234567890" {
			t.Errorf("Expected tracknum=1Z999AA1234567890, got %s", trackingNumber)
		}
		
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	// Create test client with custom base URL
	client := &UPSScrapingClient{
		ScrapingClient: NewScrapingClient("ups", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"1Z999AA1234567890"},
		Carrier:         "ups",
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
	if result.TrackingNumber != "1Z999AA1234567890" {
		t.Errorf("Expected tracking number 1Z999AA1234567890, got %s", result.TrackingNumber)
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

	if result.Events[0].Location != "ATLANTA, GA 30309, US" {
		t.Errorf("Expected location 'ATLANTA, GA 30309, US', got '%s'", result.Events[0].Location)
	}

	if result.Events[0].Description != "Package delivered to recipient" {
		t.Errorf("Expected description 'Package delivered to recipient', got '%s'", result.Events[0].Description)
	}
	
	// Check second event (out for delivery)
	if result.Events[1].Status != StatusOutForDelivery {
		t.Errorf("Expected second event status %s, got %s", StatusOutForDelivery, result.Events[1].Status)
	}
}

func TestUPSScrapingClient_Track_NotFound(t *testing.T) {
	mockHTML := `
<!DOCTYPE html>
<html>
<head><title>UPS Tracking</title></head>
<body>
	<div class="ups-error">
		<h2>Tracking Information Not Found</h2>
		<p>We could not locate the shipment details for this tracking number. Please check the number and try again.</p>
	</div>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	client := &UPSScrapingClient{
		ScrapingClient: NewScrapingClient("ups", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"1Z999AA0000000000"},
		Carrier:         "ups",
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
	if carrierErr.Carrier != "ups" {
		t.Errorf("Expected carrier 'ups', got '%s'", carrierErr.Carrier)
	}

	if carrierErr.Code != "NOT_FOUND" {
		t.Errorf("Expected error code 'NOT_FOUND', got '%s'", carrierErr.Code)
	}

	if !strings.Contains(carrierErr.Message, "not found") {
		t.Errorf("Expected error message to contain 'not found', got '%s'", carrierErr.Message)
	}
}

func TestUPSScrapingClient_Track_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := &UPSScrapingClient{
		ScrapingClient: NewScrapingClient("ups", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"1Z999AA1234567890"},
		Carrier:         "ups",
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

func TestUPSScrapingClient_Track_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("Too Many Requests"))
	}))
	defer server.Close()

	client := &UPSScrapingClient{
		ScrapingClient: NewScrapingClient("ups", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"1Z999AA1234567890"},
		Carrier:         "ups",
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

	if carrierErr.Carrier != "ups" {
		t.Errorf("Expected carrier 'ups', got '%s'", carrierErr.Carrier)
	}
}

func TestUPSScrapingClient_Track_MultiplePackages(t *testing.T) {
	mockHTML1 := `
<!DOCTYPE html>
<html>
<body>
	<div class="ups-public_trackingDetails">
		<div class="package-status">
			<h2>Delivered</h2>
		</div>
	</div>
	<div class="ups-public_trackingProgressBarContainer">
		<div class="progress-step delivered">
			<div class="step-status">Delivered</div>
			<div class="step-location">NEW YORK, NY 10001, US</div>
		</div>
	</div>
</body>
</html>`

	mockHTML2 := `
<!DOCTYPE html>
<html>
<body>
	<div class="ups-public_trackingDetails">
		<div class="package-status">
			<h2>In Transit</h2>
		</div>
	</div>
	<div class="ups-public_trackingProgressBarContainer">
		<div class="progress-step">
			<div class="step-status">In Transit</div>
			<div class="step-location">CHICAGO, IL 60601, US</div>
		</div>
	</div>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trackingNumber := r.URL.Query().Get("tracknum")
		
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		
		if trackingNumber == "1Z999AA1234567890" {
			w.Write([]byte(mockHTML1))
		} else if trackingNumber == "1Z999AA1234567891" {
			w.Write([]byte(mockHTML2))
		} else {
			t.Errorf("Unexpected tracking number: %s", trackingNumber)
		}
	}))
	defer server.Close()

	client := &UPSScrapingClient{
		ScrapingClient: NewScrapingClient("ups", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"1Z999AA1234567890", "1Z999AA1234567891"},
		Carrier:         "ups",
	}

	ctx := context.Background()
	resp, err := client.Track(ctx, req)

	if err != nil {
		t.Fatalf("Track() error = %v", err)
	}

	if len(resp.Results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(resp.Results))
	}

	// Check both results
	trackingNumbers := make(map[string]TrackingStatus)
	for _, result := range resp.Results {
		trackingNumbers[result.TrackingNumber] = result.Status
	}

	if trackingNumbers["1Z999AA1234567890"] != StatusDelivered {
		t.Errorf("Expected first package to be delivered, got %s", trackingNumbers["1Z999AA1234567890"])
	}

	if trackingNumbers["1Z999AA1234567891"] != StatusInTransit {
		t.Errorf("Expected second package to be in transit, got %s", trackingNumbers["1Z999AA1234567891"])
	}
}