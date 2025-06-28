package carriers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDHLScrapingClient_GetCarrierName(t *testing.T) {
	client := NewDHLScrapingClient("test-agent")
	if got := client.GetCarrierName(); got != "dhl" {
		t.Errorf("GetCarrierName() = %v, want %v", got, "dhl")
	}
}

func TestDHLScrapingClient_ValidateTrackingNumber(t *testing.T) {
	client := NewDHLScrapingClient("test-agent")
	
	tests := []struct {
		name           string
		trackingNumber string
		want           bool
	}{
		{
			name:           "valid DHL Express 10 digits",
			trackingNumber: "1234567890",
			want:           true,
		},
		{
			name:           "valid DHL Express 11 digits with letters",
			trackingNumber: "JJD01234567890",
			want:           true,
		},
		{
			name:           "valid DHL eCommerce 14 digits",
			trackingNumber: "12345678901234",
			want:           true,
		},
		{
			name:           "valid DHL Global Mail 13 digits",
			trackingNumber: "1234567890123",
			want:           true,
		},
		{
			name:           "valid DHL Parcel UK format",
			trackingNumber: "12345678901234567890",
			want:           true,
		},
		{
			name:           "valid DHL with spaces",
			trackingNumber: "1234 5678 9012 34",
			want:           true,
		},
		{
			name:           "valid DHL mixed alphanumeric",
			trackingNumber: "ABC1234567890",
			want:           true,
		},
		{
			name:           "too short",
			trackingNumber: "123456789",
			want:           false,
		},
		{
			name:           "too long",
			trackingNumber: "123456789012345678901",
			want:           false,
		},
		{
			name:           "empty string",
			trackingNumber: "",
			want:           false,
		},
		{
			name:           "valid 15 digits",
			trackingNumber: "123456789012345",
			want:           true,
		},
		{
			name:           "valid 16 digits",
			trackingNumber: "1234567890123456",
			want:           true,
		},
		{
			name:           "valid 18 digits",
			trackingNumber: "123456789012345678",
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

func TestDHLScrapingClient_Track_Success(t *testing.T) {
	// Mock DHL tracking page HTML response
	mockHTML := `
<!DOCTYPE html>
<html>
<head><title>DHL Tracking</title></head>
<body>
	<div class="dhl-tracking">
		<div class="tracking-number">1234567890</div>
		<div class="shipment-status">
			<h2>Delivered</h2>
			<p>Your shipment has been delivered on Monday, May 15, 2023 at 2:15 PM to NEW YORK NY 10001</p>
		</div>
	</div>
	
	<div class="tracking-events">
		<div class="tracking-event">
			<div class="event-date">May 15, 2023</div>
			<div class="event-time">2:15 PM</div>
			<div class="event-status">Delivered</div>
			<div class="event-location">NEW YORK, NY 10001, US</div>
			<div class="event-description">Shipment delivered to recipient</div>
		</div>
		<div class="tracking-event">
			<div class="event-date">May 15, 2023</div>
			<div class="event-time">6:00 AM</div>
			<div class="event-status">Out for delivery</div>
			<div class="event-location">NEW YORK, NY 10001, US</div>
			<div class="event-description">With delivery courier</div>
		</div>
		<div class="tracking-event">
			<div class="event-date">May 14, 2023</div>
			<div class="event-time">8:45 PM</div>
			<div class="event-status">Processed at DHL facility</div>
			<div class="event-location">NEW YORK, NY 10001, US</div>
			<div class="event-description">Processed at DHL facility</div>
		</div>
		<div class="tracking-event">
			<div class="event-date">May 13, 2023</div>
			<div class="event-time">3:20 PM</div>
			<div class="event-status">In transit</div>
			<div class="event-location">ATLANTA, GA 30309, US</div>
			<div class="event-description">Shipment is in transit</div>
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
		trackingNumber := r.URL.Query().Get("tracking-id")
		if trackingNumber != "1234567890" {
			t.Errorf("Expected tracking-id=1234567890, got %s", trackingNumber)
		}
		
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	// Create test client with custom base URL
	client := &DHLScrapingClient{
		ScrapingClient: NewScrapingClient("dhl", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"1234567890"},
		Carrier:         "dhl",
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
	if result.TrackingNumber != "1234567890" {
		t.Errorf("Expected tracking number 1234567890, got %s", result.TrackingNumber)
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

	if result.Events[0].Location != "NEW YORK, NY 10001, US" {
		t.Errorf("Expected location 'NEW YORK, NY 10001, US', got '%s'", result.Events[0].Location)
	}

	if result.Events[0].Description != "Shipment delivered to recipient" {
		t.Errorf("Expected description 'Shipment delivered to recipient', got '%s'", result.Events[0].Description)
	}
	
	// Check second event (out for delivery)
	if result.Events[1].Status != StatusOutForDelivery {
		t.Errorf("Expected second event status %s, got %s", StatusOutForDelivery, result.Events[1].Status)
	}
	
	// Check fourth event (in transit from Atlanta)
	if result.Events[3].Status != StatusInTransit {
		t.Errorf("Expected fourth event status %s, got %s", StatusInTransit, result.Events[3].Status)
	}
	
	if result.Events[3].Location != "ATLANTA, GA 30309, US" {
		t.Errorf("Expected fourth event location 'ATLANTA, GA 30309, US', got '%s'", result.Events[3].Location)
	}
}

func TestDHLScrapingClient_Track_NotFound(t *testing.T) {
	mockHTML := `
<!DOCTYPE html>
<html>
<head><title>DHL Tracking</title></head>
<body>
	<div class="dhl-error">
		<h2>Tracking number not found</h2>
		<p>The tracking number you entered cannot be found. Please check the number and try again.</p>
	</div>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockHTML))
	}))
	defer server.Close()

	client := &DHLScrapingClient{
		ScrapingClient: NewScrapingClient("dhl", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"999999999999"},
		Carrier:         "dhl",
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
	if carrierErr.Carrier != "dhl" {
		t.Errorf("Expected carrier 'dhl', got '%s'", carrierErr.Carrier)
	}

	if carrierErr.Code != "NOT_FOUND" {
		t.Errorf("Expected error code 'NOT_FOUND', got '%s'", carrierErr.Code)
	}

	if !strings.Contains(carrierErr.Message, "not found") {
		t.Errorf("Expected error message to contain 'not found', got '%s'", carrierErr.Message)
	}
}

func TestDHLScrapingClient_Track_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := &DHLScrapingClient{
		ScrapingClient: NewScrapingClient("dhl", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"1234567890"},
		Carrier:         "dhl",
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

func TestDHLScrapingClient_Track_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("Too Many Requests"))
	}))
	defer server.Close()

	client := &DHLScrapingClient{
		ScrapingClient: NewScrapingClient("dhl", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"1234567890"},
		Carrier:         "dhl",
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

	if carrierErr.Carrier != "dhl" {
		t.Errorf("Expected carrier 'dhl', got '%s'", carrierErr.Carrier)
	}
}

func TestDHLScrapingClient_Track_MultiplePackages(t *testing.T) {
	mockHTML1 := `
<!DOCTYPE html>
<html>
<body>
	<div class="dhl-tracking">
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
	<div class="dhl-tracking">
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
			<div class="event-description">In transit to destination</div>
		</div>
	</div>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trackingNumber := r.URL.Query().Get("tracking-id")
		
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		
		if trackingNumber == "1234567890" {
			w.Write([]byte(mockHTML1))
		} else if trackingNumber == "1234567891" {
			w.Write([]byte(mockHTML2))
		} else {
			t.Errorf("Unexpected tracking number: %s", trackingNumber)
		}
	}))
	defer server.Close()

	client := &DHLScrapingClient{
		ScrapingClient: NewScrapingClient("dhl", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"1234567890", "1234567891"},
		Carrier:         "dhl",
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

	if trackingNumbers["1234567890"] != StatusDelivered {
		t.Errorf("Expected first package to be delivered, got %s", trackingNumbers["1234567890"])
	}

	if trackingNumbers["1234567891"] != StatusInTransit {
		t.Errorf("Expected second package to be in transit, got %s", trackingNumbers["1234567891"])
	}
}

func TestDHLScrapingClient_Track_AlternativeFormat(t *testing.T) {
	// Test alternative DHL page format
	mockHTML := `
<!DOCTYPE html>
<html>
<body>
	<div class="package-details">
		<div class="status-info">
			<span class="status-text">Shipment delivered</span>
			<span class="delivery-date">May 15, 2023 2:15 PM</span>
		</div>
	</div>
	
	<table class="tracking-table">
		<tr class="tracking-row">
			<td class="date-time">May 15, 2023 2:15 PM</td>
			<td class="status">Delivered</td>
			<td class="location">NEW YORK, NY 10001</td>
			<td class="details">Shipment delivered to recipient</td>
		</tr>
		<tr class="tracking-row">
			<td class="date-time">May 15, 2023 6:00 AM</td>
			<td class="status">Out for delivery</td>
			<td class="location">NEW YORK, NY 10001</td>
			<td class="details">With delivery courier</td>
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

	client := &DHLScrapingClient{
		ScrapingClient: NewScrapingClient("dhl", "test-agent"),
		baseURL:        server.URL,
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"1234567890"},
		Carrier:         "dhl",
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