package carriers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFedExScrapingClient_GetCarrierName(t *testing.T) {
	client := NewFedExScrapingClient("test-agent")
	if got := client.GetCarrierName(); got != "fedex" {
		t.Errorf("GetCarrierName() = %v, want %v", got, "fedex")
	}
}

func TestFedExScrapingClient_ValidateTrackingNumber(t *testing.T) {
	client := NewFedExScrapingClient("test-agent")
	
	tests := []struct {
		name           string
		trackingNumber string
		want           bool
	}{
		{
			name:           "valid FedEx Express 12 digits",
			trackingNumber: "123456789012",
			want:           true,
		},
		{
			name:           "valid FedEx Ground 14 digits",
			trackingNumber: "12345678901234",
			want:           true,
		},
		{
			name:           "valid FedEx SmartPost 15 digits",
			trackingNumber: "123456789012345",
			want:           true,
		},
		{
			name:           "valid FedEx Express 18 digits",
			trackingNumber: "123456789012345678",
			want:           true,
		},
		{
			name:           "valid FedEx Ground 20 digits",
			trackingNumber: "12345678901234567890",
			want:           true,
		},
		{
			name:           "valid FedEx 22 digits",
			trackingNumber: "1234567890123456789012",
			want:           true,
		},
		{
			name:           "with spaces",
			trackingNumber: "1234 5678 9012 3456",
			want:           true,
		},
		{
			name:           "too short",
			trackingNumber: "12345678901",
			want:           false,
		},
		{
			name:           "too long",
			trackingNumber: "12345678901234567890123",
			want:           false,
		},
		{
			name:           "empty string",
			trackingNumber: "",
			want:           false,
		},
		{
			name:           "contains letters",
			trackingNumber: "123456789ABC456",
			want:           false,
		},
		{
			name:           "13 digits (invalid length)",
			trackingNumber: "1234567890123",
			want:           false,
		},
		{
			name:           "valid 16 digits",
			trackingNumber: "1234567890123456",
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

func TestFedExScrapingClient_Track_Success(t *testing.T) {
	// Mock FedEx tracking page HTML response
	mockHTML := `
<!DOCTYPE html>
<html>
<head><title>FedEx Tracking</title></head>
<body>
	<div class="fedex-tracking">
		<div class="tracking-number">123456789012</div>
		<div class="shipment-status">
			<h2>Delivered</h2>
			<p>Package delivered on Monday, May 15, 2023 at 2:15 PM to ATLANTA GA 30309</p>
		</div>
	</div>
	
	<div class="tracking-events">
		<div class="tracking-event">
			<div class="event-date">May 15, 2023</div>
			<div class="event-time">2:15 PM</div>
			<div class="event-status">Delivered</div>
			<div class="event-location">ATLANTA, GA 30309, US</div>
			<div class="event-description">Package delivered to recipient address</div>
		</div>
		<div class="tracking-event">
			<div class="event-date">May 15, 2023</div>
			<div class="event-time">6:00 AM</div>
			<div class="event-status">On FedEx vehicle for delivery</div>
			<div class="event-location">ATLANTA, GA 30309, US</div>
			<div class="event-description">On FedEx vehicle for delivery</div>
		</div>
		<div class="tracking-event">
			<div class="event-date">May 14, 2023</div>
			<div class="event-time">8:45 PM</div>
			<div class="event-status">At local FedEx facility</div>
			<div class="event-location">ATLANTA, GA 30309, US</div>
			<div class="event-description">At local FedEx facility</div>
		</div>
		<div class="tracking-event">
			<div class="event-date">May 13, 2023</div>
			<div class="event-time">3:20 PM</div>
			<div class="event-status">In transit</div>
			<div class="event-location">MEMPHIS, TN 38118, US</div>
			<div class="event-description">In transit</div>
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
		trackingNumber := r.URL.Query().Get("trackingnumber")
		if trackingNumber != "123456789012" {
			t.Errorf("Expected trackingnumber=123456789012, got %s", trackingNumber)
		}
		
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	// Create test client with custom base URL
	client := &FedExScrapingClient{
		ScrapingClient: NewScrapingClient("fedex", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"123456789012"},
		Carrier:         "fedex",
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
	if result.TrackingNumber != "123456789012" {
		t.Errorf("Expected tracking number 123456789012, got %s", result.TrackingNumber)
	}

	if result.Status != StatusDelivered {
		t.Errorf("Expected status %s, got %s", StatusDelivered, result.Status)
	}

	if len(result.Events) != 4 {
		t.Errorf("Expected 4 events, got %d", len(result.Events))
	}

	// Check first event (most recent - delivered)
	if result.Events[0].Status != StatusDelivered {
		t.Errorf("Expected first event status %s, got %s", StatusDelivered, result.Events[0].Status)
	}

	if result.Events[0].Location != "ATLANTA, GA 30309, US" {
		t.Errorf("Expected location 'ATLANTA, GA 30309, US', got '%s'", result.Events[0].Location)
	}

	if result.Events[0].Description != "Package delivered to recipient address" {
		t.Errorf("Expected description 'Package delivered to recipient address', got '%s'", result.Events[0].Description)
	}
	
	// Check second event (out for delivery)
	if result.Events[1].Status != StatusOutForDelivery {
		t.Errorf("Expected second event status %s, got %s", StatusOutForDelivery, result.Events[1].Status)
	}
	
	// Check fourth event (in transit from Memphis)
	if result.Events[3].Status != StatusInTransit {
		t.Errorf("Expected fourth event status %s, got %s", StatusInTransit, result.Events[3].Status)
	}
	
	if result.Events[3].Location != "MEMPHIS, TN 38118, US" {
		t.Errorf("Expected fourth event location 'MEMPHIS, TN 38118, US', got '%s'", result.Events[3].Location)
	}
}

func TestFedExScrapingClient_Track_NotFound(t *testing.T) {
	mockHTML := `
<!DOCTYPE html>
<html>
<head><title>FedEx Tracking</title></head>
<body>
	<div class="fedex-error">
		<h2>Tracking number not found</h2>
		<p>We cannot locate the shipment details for this tracking number. Please check the number and try again.</p>
	</div>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	client := &FedExScrapingClient{
		ScrapingClient: NewScrapingClient("fedex", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"999999999999"},
		Carrier:         "fedex",
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
	if carrierErr.Carrier != "fedex" {
		t.Errorf("Expected carrier 'fedex', got '%s'", carrierErr.Carrier)
	}

	if carrierErr.Code != "NOT_FOUND" {
		t.Errorf("Expected error code 'NOT_FOUND', got '%s'", carrierErr.Code)
	}

	if !strings.Contains(carrierErr.Message, "not found") {
		t.Errorf("Expected error message to contain 'not found', got '%s'", carrierErr.Message)
	}
}

func TestFedExScrapingClient_Track_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := &FedExScrapingClient{
		ScrapingClient: NewScrapingClient("fedex", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"123456789012"},
		Carrier:         "fedex",
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

func TestFedExScrapingClient_Track_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("Too Many Requests"))
	}))
	defer server.Close()

	client := &FedExScrapingClient{
		ScrapingClient: NewScrapingClient("fedex", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"123456789012"},
		Carrier:         "fedex",
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

	if carrierErr.Carrier != "fedex" {
		t.Errorf("Expected carrier 'fedex', got '%s'", carrierErr.Carrier)
	}
}

func TestFedExScrapingClient_Track_MultiplePackages(t *testing.T) {
	mockHTML1 := `
<!DOCTYPE html>
<html>
<body>
	<div class="fedex-tracking">
		<div class="shipment-status">
			<h2>Delivered</h2>
		</div>
	</div>
	<div class="tracking-events">
		<div class="tracking-event">
			<div class="event-date">May 15, 2023</div>
			<div class="event-time">2:15 PM</div>
			<div class="event-status">Delivered</div>
			<div class="event-location">NEW YORK, NY 10001, US</div>
			<div class="event-description">Delivered</div>
		</div>
	</div>
</body>
</html>`

	mockHTML2 := `
<!DOCTYPE html>
<html>
<body>
	<div class="fedex-tracking">
		<div class="shipment-status">
			<h2>In transit</h2>
		</div>
	</div>
	<div class="tracking-events">
		<div class="tracking-event">
			<div class="event-date">May 15, 2023</div>
			<div class="event-time">10:30 AM</div>
			<div class="event-status">In transit</div>
			<div class="event-location">CHICAGO, IL 60601, US</div>
			<div class="event-description">In transit</div>
		</div>
	</div>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trackingNumber := r.URL.Query().Get("trackingnumber")
		
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		
		if trackingNumber == "123456789012" {
			w.Write([]byte(mockHTML1))
		} else if trackingNumber == "123456789013" {
			w.Write([]byte(mockHTML2))
		} else {
			t.Errorf("Unexpected tracking number: %s", trackingNumber)
		}
	}))
	defer server.Close()

	client := &FedExScrapingClient{
		ScrapingClient: NewScrapingClient("fedex", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"123456789012", "123456789013"},
		Carrier:         "fedex",
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

	if trackingNumbers["123456789012"] != StatusDelivered {
		t.Errorf("Expected first package to be delivered, got %s", trackingNumbers["123456789012"])
	}

	if trackingNumbers["123456789013"] != StatusInTransit {
		t.Errorf("Expected second package to be in transit, got %s", trackingNumbers["123456789013"])
	}
}

func TestFedExScrapingClient_Track_AlternativeFormat(t *testing.T) {
	// Test alternative FedEx page format
	mockHTML := `
<!DOCTYPE html>
<html>
<body>
	<div class="package-details">
		<div class="status-info">
			<span class="status-text">Package delivered</span>
			<span class="delivery-date">May 15, 2023 2:15 PM</span>
		</div>
	</div>
	
	<table class="tracking-table">
		<tr class="tracking-row">
			<td class="date-time">May 15, 2023 2:15 PM</td>
			<td class="status">Delivered</td>
			<td class="location">ATLANTA, GA 30309</td>
			<td class="details">Package delivered to recipient</td>
		</tr>
		<tr class="tracking-row">
			<td class="date-time">May 15, 2023 6:00 AM</td>
			<td class="status">Out for delivery</td>
			<td class="location">ATLANTA, GA 30309</td>
			<td class="details">On FedEx vehicle for delivery</td>
		</tr>
	</table>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	client := &FedExScrapingClient{
		ScrapingClient: NewScrapingClient("fedex", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"123456789012"},
		Carrier:         "fedex",
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
	if result.Status != StatusDelivered {
		t.Errorf("Expected status %s, got %s", StatusDelivered, result.Status)
	}

	// Should have at least 1 event (might extract more from table)
	if len(result.Events) < 1 {
		t.Errorf("Expected at least 1 event, got %d", len(result.Events))
	}
}