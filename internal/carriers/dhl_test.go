package carriers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDHLClient_GetCarrierName(t *testing.T) {
	client := &DHLClient{}
	if got := client.GetCarrierName(); got != "dhl" {
		t.Errorf("GetCarrierName() = %v, want %v", got, "dhl")
	}
}

func TestDHLClient_ValidateTrackingNumber(t *testing.T) {
	client := &DHLClient{}
	
	tests := []struct {
		name           string
		trackingNumber string
		want           bool
	}{
		{
			name:           "valid DHL tracking number 10 digits",
			trackingNumber: "1234567890",
			want:           true,
		},
		{
			name:           "valid DHL tracking number 11 digits",
			trackingNumber: "12345678901",
			want:           true,
		},
		{
			name:           "valid DHL waybill",
			trackingNumber: "1234567890",
			want:           true,
		},
		{
			name:           "too short",
			trackingNumber: "123456789",
			want:           false,
		},
		{
			name:           "valid 12 digits (DHL eCommerce)",
			trackingNumber: "123456789012",
			want:           true,
		},
		{
			name:           "empty string",
			trackingNumber: "",
			want:           false,
		},
		{
			name:           "valid alphanumeric (DHL service)",
			trackingNumber: "1234567ABC",
			want:           true,
		},
		{
			name:           "no digits (letters only)",
			trackingNumber: "INFORMATION",
			want:           false,
		},
		{
			name:           "too long (21+ characters)",
			trackingNumber: "123456789012345678901",
			want:           false,
		},
		{
			name:           "valid DHL Express format",
			trackingNumber: "1234567890",
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

func TestDHLClient_Track_Success(t *testing.T) {
	mockResponse := `{
		"shipments": [{
			"id": "1234567890",
			"service": "express",
			"origin": {
				"address": {
					"countryCode": "DE",
					"postalCode": "53113",
					"addressLocality": "Bonn"
				}
			},
			"destination": {
				"address": {
					"countryCode": "US",
					"postalCode": "10001",
					"addressLocality": "New York"
				}
			},
			"status": {
				"timestamp": "2023-05-15T14:45:00.000+02:00",
				"location": {
					"address": {
						"countryCode": "US",
						"postalCode": "10001",
						"addressLocality": "New York"
					}
				},
				"statusCode": "delivered",
				"status": "delivered",
				"description": "Delivered"
			},
			"estimatedTimeOfDelivery": "2023-05-15T18:00:00.000+02:00",
			"estimatedDeliveryTimeFrame": {
				"estimatedFrom": "2023-05-15T09:00:00.000+02:00",
				"estimatedThrough": "2023-05-15T18:00:00.000+02:00"
			},
			"estimatedTimeOfDeliveryRemark": "By 6:00 pm",
			"serviceUrl": "https://www.dhl.com/shipmentTracking?AWB=1234567890",
			"rerouteUrl": "https://www.dhl.com/reroute?AWB=1234567890",
			"details": {
				"carrier": {
					"id": "dhl-express",
					"name": "DHL Express"
				},
				"product": {
					"productName": "DHL Express Worldwide"
				},
				"receiver": {
					"name": "John Doe",
					"organizationName": "Test Company"
				},
				"sender": {
					"name": "Jane Smith",
					"organizationName": "Sender Company"
				},
				"proofOfDelivery": {
					"timestamp": "2023-05-15T14:45:00.000+02:00",
					"signedBy": "J. DOE"
				},
				"totalNumberOfPieces": 1,
				"pieceIds": ["1234567890"],
				"weight": {
					"value": 2.5,
					"unitText": "kg"
				},
				"volume": {
					"value": 0.001,
					"unitText": "m3"
				}
			},
			"events": [{
				"timestamp": "2023-05-15T14:45:00.000+02:00",
				"location": {
					"address": {
						"countryCode": "US",
						"postalCode": "10001",
						"addressLocality": "New York",
						"streetAddress": ""
					}
				},
				"statusCode": "delivered",
				"status": "delivered",
				"description": "Delivered",
				"remark": "Delivered to J. DOE"
			}, {
				"timestamp": "2023-05-15T07:00:00.000+02:00",
				"location": {
					"address": {
						"countryCode": "US",
						"postalCode": "10001",
						"addressLocality": "New York"
					}
				},
				"statusCode": "with-delivery-courier",
				"status": "transit",
				"description": "With delivery courier",
				"remark": "Out for delivery"
			}]
		}]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "track/shipments") {
			t.Errorf("Expected path to contain 'track/shipments', got %s", r.URL.Path)
		}
		
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		
		// Check DHL API Key header
		apiKey := r.Header.Get("DHL-API-Key")
		if apiKey != "test_api_key" {
			t.Errorf("Expected DHL-API-Key 'test_api_key', got '%s'", apiKey)
		}
		
		// Check Accept header
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept application/json, got %s", r.Header.Get("Accept"))
		}
		
		// Check tracking number in query parameters
		trackingNumber := r.URL.Query().Get("trackingNumber")
		if trackingNumber != "1234567890" {
			t.Errorf("Expected trackingNumber=1234567890, got %s", trackingNumber)
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := &DHLClient{
		apiKey:  "test_api_key",
		baseURL: server.URL,
		client:  server.Client(),
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

	if len(result.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(result.Events))
	}

	// Check first event (most recent - delivered)
	if result.Events[0].Status != StatusDelivered {
		t.Errorf("Expected first event status %s, got %s", StatusDelivered, result.Events[0].Status)
	}

	if result.Events[0].Location != "New York, 10001, US" {
		t.Errorf("Expected location 'New York, 10001, US', got '%s'", result.Events[0].Location)
	}

	if result.Events[0].Description != "Delivered" {
		t.Errorf("Expected description 'Delivered', got '%s'", result.Events[0].Description)
	}

	if result.Events[0].Details != "Delivered to J. DOE" {
		t.Errorf("Expected details 'Delivered to J. DOE', got '%s'", result.Events[0].Details)
	}

	// Check service type and weight
	if result.ServiceType != "DHL Express Worldwide" {
		t.Errorf("Expected service type 'DHL Express Worldwide', got '%s'", result.ServiceType)
	}

	if result.Weight != "2.5 kg" {
		t.Errorf("Expected weight '2.5 kg', got '%s'", result.Weight)
	}
}

func TestDHLClient_Track_Error(t *testing.T) {
	mockErrorResponse := `{
		"title": "Bad request",
		"status": 400,
		"detail": "Tracking number not found or invalid format",
		"instance": "/track/shipments?trackingNumber=invalid_tracking"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(mockErrorResponse))
	}))
	defer server.Close()

	client := &DHLClient{
		apiKey:  "test_api_key",
		baseURL: server.URL,
		client:  server.Client(),
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"invalid_tracking"},
		Carrier:         "dhl",
	}

	ctx := context.Background()
	_, err := client.Track(ctx, req)

	if err == nil {
		t.Fatal("Expected error for invalid tracking number, got nil")
	}

	carrierErr, ok := err.(*CarrierError)
	if !ok {
		t.Fatalf("Expected CarrierError, got %T", err)
	}

	if carrierErr.Carrier != "dhl" {
		t.Errorf("Expected carrier 'dhl', got '%s'", carrierErr.Carrier)
	}

	if carrierErr.Code != "400" {
		t.Errorf("Expected error code '400', got '%s'", carrierErr.Code)
	}

	if !strings.Contains(carrierErr.Message, "Tracking number not found") {
		t.Errorf("Expected error message to contain 'Tracking number not found', got '%s'", carrierErr.Message)
	}
}

func TestDHLClient_Track_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "250")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1234567890")
		w.Header().Set("Retry-After", "300")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{
			"title": "Too Many Requests",
			"status": 429,
			"detail": "Rate limit exceeded. Maximum 250 requests per day."
		}`))
	}))
	defer server.Close()

	client := &DHLClient{
		apiKey:  "test_api_key",
		baseURL: server.URL,
		client:  server.Client(),
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

	if carrierErr.Code != "429" {
		t.Errorf("Expected error code '429', got '%s'", carrierErr.Code)
	}
}

func TestDHLClient_Track_Unauthorized(t *testing.T) {
	mockErrorResponse := `{
		"title": "Unauthorized",
		"status": 401,
		"detail": "Invalid API key"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(mockErrorResponse))
	}))
	defer server.Close()

	client := &DHLClient{
		apiKey:  "invalid_api_key",
		baseURL: server.URL,
		client:  server.Client(),
	}

	req := &TrackingRequest{
		TrackingNumbers: []string{"1234567890"},
		Carrier:         "dhl",
	}

	ctx := context.Background()
	_, err := client.Track(ctx, req)

	if err == nil {
		t.Fatal("Expected unauthorized error, got nil")
	}

	carrierErr, ok := err.(*CarrierError)
	if !ok {
		t.Fatalf("Expected CarrierError, got %T", err)
	}

	if carrierErr.Code != "401" {
		t.Errorf("Expected error code '401', got '%s'", carrierErr.Code)
	}

	if !strings.Contains(carrierErr.Message, "Invalid API key") {
		t.Errorf("Expected error message to contain 'Invalid API key', got '%s'", carrierErr.Message)
	}

	if carrierErr.Retryable {
		t.Error("Expected Retryable to be false for authentication error")
	}
}

func TestDHLClient_Track_MultiplePackages(t *testing.T) {
	mockResponse1 := `{
		"shipments": [{
			"id": "1234567890",
			"status": {
				"statusCode": "delivered",
				"status": "delivered",
				"description": "Delivered"
			},
			"events": [{
				"statusCode": "delivered",
				"status": "delivered",
				"description": "Delivered",
				"location": {
					"address": {
						"countryCode": "US",
						"addressLocality": "New York"
					}
				}
			}]
		}]
	}`

	mockResponse2 := `{
		"shipments": [{
			"id": "1234567891",
			"status": {
				"statusCode": "transit",
				"status": "transit",
				"description": "In transit"
			},
			"events": [{
				"statusCode": "transit",
				"status": "transit",
				"description": "In transit",
				"location": {
					"address": {
						"countryCode": "DE",
						"addressLocality": "Frankfurt"
					}
				}
			}]
		}]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trackingNumber := r.URL.Query().Get("trackingNumber")
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		if trackingNumber == "1234567890" {
			w.Write([]byte(mockResponse1))
		} else if trackingNumber == "1234567891" {
			w.Write([]byte(mockResponse2))
		} else {
			t.Errorf("Unexpected tracking number: %s", trackingNumber)
		}
	}))
	defer server.Close()

	client := &DHLClient{
		apiKey:  "test_api_key",
		baseURL: server.URL,
		client:  server.Client(),
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

func TestDHLClient_Track_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := &DHLClient{
		apiKey:  "test_api_key",
		baseURL: server.URL,
		client:  server.Client(),
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