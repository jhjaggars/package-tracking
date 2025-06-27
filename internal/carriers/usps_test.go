package carriers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUSPSClient_GetCarrierName(t *testing.T) {
	client := &USPSClient{}
	if got := client.GetCarrierName(); got != "usps" {
		t.Errorf("GetCarrierName() = %v, want %v", got, "usps")
	}
}

func TestUSPSClient_ValidateTrackingNumber(t *testing.T) {
	client := &USPSClient{}
	
	tests := []struct {
		name           string
		trackingNumber string
		want           bool
	}{
		{
			name:           "valid USPS tracking number",
			trackingNumber: "9400111699000367046792",
			want:           true,
		},
		{
			name:           "valid Priority Mail Express",
			trackingNumber: "EK123456789US",
			want:           true,
		},
		{
			name:           "valid Certified Mail",
			trackingNumber: "7012 3456 7890 1234 5678",
			want:           true,
		},
		{
			name:           "too short",
			trackingNumber: "123456",
			want:           false,
		},
		{
			name:           "empty string",
			trackingNumber: "",
			want:           false,
		},
		{
			name:           "non-USPS format (UPS)",
			trackingNumber: "1Z999AA1234567890",
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

func TestUSPSClient_Track_Success(t *testing.T) {
	// Mock USPS API response
	mockResponse := `<?xml version="1.0" encoding="UTF-8"?>
	<TrackResponse>
		<TrackInfo ID="9400111699000367046792">
			<TrackSummary>
				<EventTime>11:07 am</EventTime>
				<EventDate>May 11, 2016</EventDate>
				<Event>Delivered</Event>
				<EventCity>GREENSBORO</EventCity>
				<EventState>NC</EventState>
				<EventZIPCode>27401</EventZIPCode>
				<EventCountry></EventCountry>
				<FirmName></FirmName>
				<Name></Name>
				<AuthorizedAgent>false</AuthorizedAgent>
			</TrackSummary>
			<TrackDetail>
				<EventTime>6:00 am</EventTime>
				<EventDate>May 11, 2016</EventDate>
				<Event>Out for Delivery</Event>
				<EventCity>GREENSBORO</EventCity>
				<EventState>NC</EventState>
				<EventZIPCode>27401</EventZIPCode>
				<EventCountry></EventCountry>
				<FirmName></FirmName>
				<Name></Name>
				<AuthorizedAgent>false</AuthorizedAgent>
			</TrackDetail>
		</TrackInfo>
	</TrackResponse>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request format
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		
		queryParams := r.URL.Query()
		if queryParams.Get("API") != "TrackV2" {
			t.Errorf("Expected API=TrackV2, got %s", queryParams.Get("API"))
		}
		
		xml := queryParams.Get("XML")
		if !strings.Contains(xml, "9400111699000367046792") {
			t.Errorf("Expected tracking number in XML, got %s", xml)
		}
		
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := &USPSClient{
		userID:  "test_user_id",
		baseURL: server.URL,
		client:  server.Client(),
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"9400111699000367046792"},
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
	if result.TrackingNumber != "9400111699000367046792" {
		t.Errorf("Expected tracking number 9400111699000367046792, got %s", result.TrackingNumber)
	}

	if result.Status != StatusDelivered {
		t.Errorf("Expected status %s, got %s", StatusDelivered, result.Status)
	}

	if len(result.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(result.Events))
	}

	// Check first event (most recent - delivered)
	if result.Events[0].Status != StatusDelivered {
		t.Errorf("Expected first event status %s, got %s", StatusDelivered, result.Events[0].Status)
	}

	if result.Events[0].Location != "GREENSBORO, NC 27401" {
		t.Errorf("Expected location 'GREENSBORO, NC 27401', got '%s'", result.Events[0].Location)
	}
}

func TestUSPSClient_Track_Error(t *testing.T) {
	// Mock USPS API error response
	mockErrorResponse := `<?xml version="1.0" encoding="UTF-8"?>
	<TrackResponse>
		<TrackInfo ID="invalid_tracking">
			<Error>
				<Number>-2147219283</Number>
				<Description>The Postal Service could not locate the tracking information for your request. Please verify your tracking number and try again later.</Description>
				<HelpFile></HelpFile>
				<HelpContext></HelpContext>
			</Error>
		</TrackInfo>
	</TrackResponse>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockErrorResponse))
	}))
	defer server.Close()

	client := &USPSClient{
		userID:  "test_user_id",
		baseURL: server.URL,
		client:  server.Client(),
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

	if len(resp.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(resp.Errors))
	}

	carrierErr := resp.Errors[0]
	if carrierErr.Carrier != "usps" {
		t.Errorf("Expected carrier 'usps', got '%s'", carrierErr.Carrier)
	}

	if carrierErr.Code != "-2147219283" {
		t.Errorf("Expected error code '-2147219283', got '%s'", carrierErr.Code)
	}

	if !strings.Contains(carrierErr.Message, "could not locate") {
		t.Errorf("Expected error message to contain 'could not locate', got '%s'", carrierErr.Message)
	}
}

func TestUSPSClient_Track_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := &USPSClient{
		userID:  "test_user_id",
		baseURL: server.URL,
		client:  server.Client(),
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"9400111699000367046792"},
		Carrier:         "usps",
	}

	ctx := context.Background()
	_, err := client.Track(ctx, req)

	if err == nil {
		t.Fatal("Expected error for HTTP 500, got nil")
	}

	if !strings.Contains(err.Error(), "HTTP error") {
		t.Errorf("Expected error to contain 'HTTP error', got '%s'", err.Error())
	}
}

func TestUSPSClient_Track_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<TrackResponse></TrackResponse>`))
	}))
	defer server.Close()

	client := &USPSClient{
		userID:  "test_user_id",
		baseURL: server.URL,
		client:  &http.Client{Timeout: 100 * time.Millisecond},
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"9400111699000367046792"},
		Carrier:         "usps",
	}

	ctx := context.Background()
	_, err := client.Track(ctx, req)

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout error, got '%s'", err.Error())
	}
}

func TestUSPSClient_Track_InvalidXML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid xml response"))
	}))
	defer server.Close()

	client := &USPSClient{
		userID:  "test_user_id",
		baseURL: server.URL,
		client:  server.Client(),
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"9400111699000367046792"},
		Carrier:         "usps",
	}

	ctx := context.Background()
	_, err := client.Track(ctx, req)

	if err == nil {
		t.Fatal("Expected XML parsing error, got nil")
	}

	if !strings.Contains(err.Error(), "XML") && !strings.Contains(err.Error(), "parse") {
		t.Errorf("Expected XML parsing error, got '%s'", err.Error())
	}
}

func TestUSPSClient_Track_MultipleTrackingNumbers(t *testing.T) {
	mockResponse := `<?xml version="1.0" encoding="UTF-8"?>
	<TrackResponse>
		<TrackInfo ID="9400111699000367046792">
			<TrackSummary>
				<EventTime>11:07 am</EventTime>
				<EventDate>May 11, 2016</EventDate>
				<Event>Delivered</Event>
				<EventCity>GREENSBORO</EventCity>
				<EventState>NC</EventState>
				<EventZIPCode>27401</EventZIPCode>
			</TrackSummary>
		</TrackInfo>
		<TrackInfo ID="9400111699000367046793">
			<TrackSummary>
				<EventTime>2:00 pm</EventTime>
				<EventDate>May 10, 2016</EventDate>
				<Event>In Transit</Event>
				<EventCity>ATLANTA</EventCity>
				<EventState>GA</EventState>
				<EventZIPCode>30309</EventZIPCode>
			</TrackSummary>
		</TrackInfo>
	</TrackResponse>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xml := r.URL.Query().Get("XML")
		if !strings.Contains(xml, "9400111699000367046792") || !strings.Contains(xml, "9400111699000367046793") {
			t.Errorf("Expected both tracking numbers in XML, got %s", xml)
		}
		
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := &USPSClient{
		userID:  "test_user_id",
		baseURL: server.URL,
		client:  server.Client(),
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"9400111699000367046792", "9400111699000367046793"},
		Carrier:         "usps",
	}

	ctx := context.Background()
	resp, err := client.Track(ctx, req)

	if err != nil {
		t.Fatalf("Track() error = %v", err)
	}

	if len(resp.Results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(resp.Results))
	}

	// Check first result
	if resp.Results[0].TrackingNumber != "9400111699000367046792" {
		t.Errorf("Expected tracking number 9400111699000367046792, got %s", resp.Results[0].TrackingNumber)
	}
	if resp.Results[0].Status != StatusDelivered {
		t.Errorf("Expected status %s, got %s", StatusDelivered, resp.Results[0].Status)
	}

	// Check second result
	if resp.Results[1].TrackingNumber != "9400111699000367046793" {
		t.Errorf("Expected tracking number 9400111699000367046793, got %s", resp.Results[1].TrackingNumber)
	}
	if resp.Results[1].Status != StatusInTransit {
		t.Errorf("Expected status %s, got %s", StatusInTransit, resp.Results[1].Status)
	}
}