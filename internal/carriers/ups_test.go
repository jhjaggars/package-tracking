package carriers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUPSClient_GetCarrierName(t *testing.T) {
	client := &UPSClient{}
	if got := client.GetCarrierName(); got != "ups" {
		t.Errorf("GetCarrierName() = %v, want %v", got, "ups")
	}
}

func TestUPSClient_ValidateTrackingNumber(t *testing.T) {
	client := &UPSClient{}
	
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
			name:           "invalid format",
			trackingNumber: "2Z999AA1234567890",
			want:           false,
		},
		{
			name:           "empty string",
			trackingNumber: "",
			want:           false,
		},
		{
			name:           "non-UPS format (USPS)",
			trackingNumber: "9400111699000367046792",
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

func TestUPSClient_OAuth_Success(t *testing.T) {
	mockTokenResponse := `{
		"access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
		"token_type": "Bearer",
		"expires_in": 14400,
		"scope": "read"
	}`

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/security/v1/oauth/token" {
			t.Errorf("Expected path /security/v1/oauth/token, got %s", r.URL.Path)
		}
		
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		// Check Content-Type
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected Content-Type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
		}
		
		// Check Authorization header (Basic auth)
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Basic ") {
			t.Errorf("Expected Basic auth header, got %s", authHeader)
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockTokenResponse))
	}))
	defer tokenServer.Close()

	client := &UPSClient{
		clientID:     "test_client_id",
		clientSecret: "test_client_secret",
		baseURL:      tokenServer.URL,
		client:       tokenServer.Client(),
	}

	ctx := context.Background()
	err := client.authenticate(ctx)

	if err != nil {
		t.Fatalf("authenticate() error = %v", err)
	}

	if client.accessToken == "" {
		t.Error("Expected access token to be set")
	}

	if client.tokenExpiry.IsZero() {
		t.Error("Expected token expiry to be set")
	}
}

func TestUPSClient_OAuth_Error(t *testing.T) {
	mockErrorResponse := `{
		"error": "invalid_client",
		"error_description": "The client credentials are invalid"
	}`

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(mockErrorResponse))
	}))
	defer tokenServer.Close()

	client := &UPSClient{
		clientID:     "invalid_client_id",
		clientSecret: "invalid_client_secret",
		baseURL:      tokenServer.URL,
		client:       tokenServer.Client(),
	}

	ctx := context.Background()
	err := client.authenticate(ctx)

	if err == nil {
		t.Fatal("Expected authentication error, got nil")
	}

	if !strings.Contains(err.Error(), "invalid_client") {
		t.Errorf("Expected error to contain 'invalid_client', got '%s'", err.Error())
	}
}

func TestUPSClient_Track_Success(t *testing.T) {
	mockTokenResponse := `{
		"access_token": "test_token",
		"token_type": "Bearer",
		"expires_in": 14400
	}`

	mockTrackResponse := `{
		"trackResponse": {
			"shipment": [{
				"package": [{
					"trackingNumber": "1Z999AA1234567890",
					"deliveryDate": [{
						"date": "20230515"
					}],
					"activity": [{
						"date": "20230515",
						"time": "144500",
						"status": {
							"type": "D",
							"description": "Delivered",
							"code": "KB"
						},
						"location": {
							"address": {
								"city": "ATLANTA",
								"stateProvinceCode": "GA",
								"postalCode": "30309",
								"country": "US"
							}
						}
					}, {
						"date": "20230515",
						"time": "070000",
						"status": {
							"type": "I",
							"description": "Out For Delivery",
							"code": "OFD"
						},
						"location": {
							"address": {
								"city": "ATLANTA",
								"stateProvinceCode": "GA",
								"postalCode": "30309",
								"country": "US"
							}
						}
					}]
				}]
			}]
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "oauth/token") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockTokenResponse))
			return
		}
		
		if strings.Contains(r.URL.Path, "track/v1/details") {
			// Verify authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test_token" {
				t.Errorf("Expected Authorization 'Bearer test_token', got '%s'", authHeader)
			}
			
			// Verify tracking number in URL
			if !strings.Contains(r.URL.Path, "1Z999AA1234567890") {
				t.Errorf("Expected tracking number in URL path, got %s", r.URL.Path)
			}
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockTrackResponse))
			return
		}
		
		t.Errorf("Unexpected request path: %s", r.URL.Path)
	}))
	defer server.Close()

	client := &UPSClient{
		clientID:     "test_client_id",
		clientSecret: "test_client_secret",
		baseURL:      server.URL,
		client:       server.Client(),
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

	if len(result.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(result.Events))
	}

	// Check first event (most recent - delivered)
	if result.Events[0].Status != StatusDelivered {
		t.Errorf("Expected first event status %s, got %s", StatusDelivered, result.Events[0].Status)
	}

	if result.Events[0].Location != "ATLANTA, GA 30309, US" {
		t.Errorf("Expected location 'ATLANTA, GA 30309, US', got '%s'", result.Events[0].Location)
	}
}

func TestUPSClient_Track_TokenExpired(t *testing.T) {
	callCount := 0
	mockTokenResponse := `{
		"access_token": "new_test_token",
		"token_type": "Bearer",
		"expires_in": 14400
	}`

	mockTrackResponse := `{
		"trackResponse": {
			"shipment": [{
				"package": [{
					"trackingNumber": "1Z999AA1234567890",
					"activity": [{
						"date": "20230515",
						"time": "144500",
						"status": {
							"type": "D",
							"description": "Delivered",
							"code": "KB"
						},
						"location": {
							"address": {
								"city": "ATLANTA",
								"stateProvinceCode": "GA"
							}
						}
					}]
				}]
			}]
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "oauth/token") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockTokenResponse))
			return
		}
		
		if strings.Contains(r.URL.Path, "track/v1/details") {
			callCount++
			authHeader := r.Header.Get("Authorization")
			
			// First call with expired token should return 401
			if callCount == 1 && authHeader == "Bearer expired_token" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "invalid_token"}`))
				return
			}
			
			// Second call with new token should succeed
			if callCount == 2 && authHeader == "Bearer new_test_token" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(mockTrackResponse))
				return
			}
			
			t.Errorf("Unexpected authorization header: %s", authHeader)
		}
	}))
	defer server.Close()

	client := &UPSClient{
		clientID:     "test_client_id",
		clientSecret: "test_client_secret",
		baseURL:      server.URL,
		client:       server.Client(),
		accessToken:  "expired_token",
		tokenExpiry:  time.Now().Add(-1 * time.Hour), // Expired
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

	// Verify token was refreshed
	if client.accessToken != "new_test_token" {
		t.Errorf("Expected token to be refreshed to 'new_test_token', got '%s'", client.accessToken)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 track API calls (retry after token refresh), got %d", callCount)
	}
}

func TestUPSClient_Track_RateLimit(t *testing.T) {
	mockTokenResponse := `{
		"access_token": "test_token",
		"token_type": "Bearer",
		"expires_in": 14400
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "oauth/token") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockTokenResponse))
			return
		}
		
		if strings.Contains(r.URL.Path, "track/v1/details") {
			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", "1234567890")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate_limit_exceeded"}`))
			return
		}
	}))
	defer server.Close()

	client := &UPSClient{
		clientID:     "test_client_id",
		clientSecret: "test_client_secret",
		baseURL:      server.URL,
		client:       server.Client(),
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
}

func TestUPSClient_Track_MultiplePackages(t *testing.T) {
	mockTokenResponse := `{
		"access_token": "test_token",
		"token_type": "Bearer",
		"expires_in": 14400
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "oauth/token") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockTokenResponse))
			return
		}
		
		// UPS API handles one tracking number per request
		if strings.Contains(r.URL.Path, "1Z999AA1234567890") {
			mockResponse := `{
				"trackResponse": {
					"shipment": [{
						"package": [{
							"trackingNumber": "1Z999AA1234567890",
							"activity": [{
								"status": {"type": "D", "description": "Delivered"},
								"location": {"address": {"city": "ATLANTA", "stateProvinceCode": "GA"}}
							}]
						}]
					}]
				}
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
			return
		}
		
		if strings.Contains(r.URL.Path, "1Z999AA1234567891") {
			mockResponse := `{
				"trackResponse": {
					"shipment": [{
						"package": [{
							"trackingNumber": "1Z999AA1234567891",
							"activity": [{
								"status": {"type": "I", "description": "In Transit"},
								"location": {"address": {"city": "CHICAGO", "stateProvinceCode": "IL"}}
							}]
						}]
					}]
				}
			}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
			return
		}
		
		t.Errorf("Unexpected request path: %s", r.URL.Path)
	}))
	defer server.Close()

	client := &UPSClient{
		clientID:     "test_client_id",
		clientSecret: "test_client_secret",
		baseURL:      server.URL,
		client:       server.Client(),
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
	trackingNumbers := make(map[string]bool)
	for _, result := range resp.Results {
		trackingNumbers[result.TrackingNumber] = true
	}

	if !trackingNumbers["1Z999AA1234567890"] {
		t.Error("Expected first tracking number in results")
	}

	if !trackingNumbers["1Z999AA1234567891"] {
		t.Error("Expected second tracking number in results")
	}
}