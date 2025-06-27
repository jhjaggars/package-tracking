package carriers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFedExClient_GetCarrierName(t *testing.T) {
	client := &FedExClient{}
	if got := client.GetCarrierName(); got != "fedex" {
		t.Errorf("GetCarrierName() = %v, want %v", got, "fedex")
	}
}

func TestFedExClient_ValidateTrackingNumber(t *testing.T) {
	client := &FedExClient{}
	
	tests := []struct {
		name           string
		trackingNumber string
		want           bool
	}{
		{
			name:           "valid FedEx Express 12-digit",
			trackingNumber: "123456789012",
			want:           true,
		},
		{
			name:           "valid FedEx Ground 14-digit",
			trackingNumber: "12345678901234",
			want:           true,
		},
		{
			name:           "valid FedEx Ground 15-digit",
			trackingNumber: "123456789012345",
			want:           true,
		},
		{
			name:           "valid FedEx Ground 18-digit",
			trackingNumber: "123456789012345678",
			want:           true,
		},
		{
			name:           "valid FedEx Ground 20-digit",
			trackingNumber: "12345678901234567890",
			want:           true,
		},
		{
			name:           "valid FedEx Ground 22-digit",
			trackingNumber: "1234567890123456789012",
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
			name:           "invalid length (13 digits)",
			trackingNumber: "1234567890123",
			want:           false,
		},
		{
			name:           "empty string",
			trackingNumber: "",
			want:           false,
		},
		{
			name:           "contains letters",
			trackingNumber: "1234567890AB",
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

func TestFedExClient_OAuth_Success(t *testing.T) {
	mockTokenResponse := `{
		"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
		"token_type": "bearer",
		"expires_in": 3600,
		"scope": "CXS"
	}`

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Errorf("Expected path /oauth/token, got %s", r.URL.Path)
		}
		
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		// Check Content-Type
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected Content-Type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
		}
		
		// Parse form data
		r.ParseForm()
		if r.Form.Get("grant_type") != "client_credentials" {
			t.Errorf("Expected grant_type=client_credentials, got %s", r.Form.Get("grant_type"))
		}
		
		if r.Form.Get("client_id") != "test_client_id" {
			t.Errorf("Expected client_id=test_client_id, got %s", r.Form.Get("client_id"))
		}
		
		if r.Form.Get("client_secret") != "test_client_secret" {
			t.Errorf("Expected client_secret=test_client_secret, got %s", r.Form.Get("client_secret"))
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockTokenResponse))
	}))
	defer tokenServer.Close()

	client := &FedExClient{
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

	// Token should expire in ~60 minutes (3600 seconds)
	expectedExpiry := time.Now().Add(3600 * time.Second)
	if client.tokenExpiry.Before(expectedExpiry.Add(-5*time.Second)) || client.tokenExpiry.After(expectedExpiry.Add(5*time.Second)) {
		t.Errorf("Expected token expiry around %v, got %v", expectedExpiry, client.tokenExpiry)
	}
}

func TestFedExClient_OAuth_Error(t *testing.T) {
	mockErrorResponse := `{
		"error": "invalid_client",
		"error_description": "Client authentication failed"
	}`

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(mockErrorResponse))
	}))
	defer tokenServer.Close()

	client := &FedExClient{
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

func TestFedExClient_Track_Success(t *testing.T) {
	mockTokenResponse := `{
		"access_token": "test_token",
		"token_type": "bearer",
		"expires_in": 3600
	}`

	mockTrackResponse := `{
		"output": {
			"completeTrackResults": [{
				"trackingNumber": "123456789012",
				"trackResults": [{
					"trackingNumberInfo": {
						"trackingNumber": "123456789012",
						"trackingNumberUniqueId": "",
						"carrierCode": "FDXE"
					},
					"additionalTrackingInfo": {
						"nickname": "",
						"packageIdentifiers": [{
							"type": "TRACKING_NUMBER_OR_DOORTAG",
							"value": "123456789012"
						}],
						"hasAssociatedShipments": false
					},
					"shipmentDetails": {
						"contents": [{
							"itemNumber": "",
							"receivedDateTime": "",
							"description": "",
							"partNumber": ""
						}],
						"beforePossessionStatus": false,
						"weight": [{
							"value": "1.0",
							"unit": "LB"
						}],
						"contentPieceCount": 1,
						"packagingDescription": "FedEx Pak",
						"physicalPackagingType": "FEDEX_PAK",
						"sequenceNumber": "1"
					},
					"scanEvents": [{
						"date": "2023-05-15T14:45:00-05:00",
						"eventType": "DL",
						"eventDescription": "Delivered",
						"exceptionCode": "",
						"exceptionDescription": "",
						"scanLocation": {
							"streetLines": [""],
							"city": "ATLANTA",
							"stateOrProvinceCode": "GA",
							"postalCode": "30309",
							"countryCode": "US",
							"residential": false
						},
						"locationId": "5531",
						"locationType": "CUSTOMER"
					}, {
						"date": "2023-05-15T07:00:00-05:00",
						"eventType": "OD",
						"eventDescription": "On FedEx vehicle for delivery",
						"exceptionCode": "",
						"exceptionDescription": "",
						"scanLocation": {
							"streetLines": [""],
							"city": "ATLANTA",
							"stateOrProvinceCode": "GA",
							"postalCode": "30309",
							"countryCode": "US",
							"residential": false
						},
						"locationId": "5531",
						"locationType": "VEHICLE"
					}],
					"availableImages": [],
					"specialHandlings": [],
					"packageDetails": {
						"physicalPackagingType": "FEDEX_PAK",
						"sequenceNumber": "1",
						"count": "1",
						"weightAndDimensions": {
							"weight": [{
								"value": "1.0",
								"unit": "LB"
							}]
						},
						"packageContent": []
					},
					"goodsClassificationCode": "",
					"holdAtLocationEligible": false,
					"customDeliveryOptions": [],
					"estimatedDeliveryTimeWindow": {
						"description": ""
					},
					"pieceCounts": [{
						"count": "1",
						"description": "TOTAL_PIECES"
					}],
					"dateAndTimes": [{
						"type": "ACTUAL_DELIVERY",
						"dateTime": "2023-05-15T14:45:00-05:00"
					}, {
						"type": "ESTIMATED_DELIVERY",
						"dateTime": "2023-05-15T23:59:59-05:00"
					}],
					"availableNotifications": [],
					"error": {
						"code": "",
						"message": ""
					}
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
		
		if strings.Contains(r.URL.Path, "track/v1/trackingnumbers") {
			// Verify authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test_token" {
				t.Errorf("Expected Authorization 'Bearer test_token', got '%s'", authHeader)
			}
			
			// Verify Content-Type
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
			}
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockTrackResponse))
			return
		}
		
		t.Errorf("Unexpected request path: %s", r.URL.Path)
	}))
	defer server.Close()

	client := &FedExClient{
		clientID:     "test_client_id",
		clientSecret: "test_client_secret",
		baseURL:      server.URL,
		client:       server.Client(),
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

	// Check service type and weight
	if result.ServiceType != "FedEx Pak" {
		t.Errorf("Expected service type 'FedEx Pak', got '%s'", result.ServiceType)
	}

	if result.Weight != "1.0 LB" {
		t.Errorf("Expected weight '1.0 LB', got '%s'", result.Weight)
	}
}

func TestFedExClient_Track_RateLimit(t *testing.T) {
	mockTokenResponse := `{
		"access_token": "test_token",
		"token_type": "bearer",
		"expires_in": 3600
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "oauth/token") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockTokenResponse))
			return
		}
		
		if strings.Contains(r.URL.Path, "track/v1/trackingnumbers") {
			w.Header().Set("X-RateLimit-Limit", "1400")
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", "1234567890")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{
				"errors": [{
					"code": "RATE.LIMIT.EXCEEDED",
					"message": "Rate limit exceeded. Maximum 1400 requests per 10 seconds."
				}]
			}`))
			return
		}
	}))
	defer server.Close()

	client := &FedExClient{
		clientID:     "test_client_id",
		clientSecret: "test_client_secret",
		baseURL:      server.URL,
		client:       server.Client(),
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

	if carrierErr.Code != "RATE.LIMIT.EXCEEDED" {
		t.Errorf("Expected error code 'RATE.LIMIT.EXCEEDED', got '%s'", carrierErr.Code)
	}
}

func TestFedExClient_Track_TokenExpired(t *testing.T) {
	callCount := 0
	mockTokenResponse := `{
		"access_token": "new_test_token",
		"token_type": "bearer",
		"expires_in": 3600
	}`

	mockTrackResponse := `{
		"output": {
			"completeTrackResults": [{
				"trackingNumber": "123456789012",
				"trackResults": [{
					"trackingNumberInfo": {"trackingNumber": "123456789012"},
					"scanEvents": [{
						"date": "2023-05-15T14:45:00-05:00",
						"eventType": "DL",
						"eventDescription": "Delivered",
						"scanLocation": {
							"city": "ATLANTA",
							"stateOrProvinceCode": "GA"
						}
					}],
					"dateAndTimes": [{
						"type": "ACTUAL_DELIVERY",
						"dateTime": "2023-05-15T14:45:00-05:00"
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
		
		if strings.Contains(r.URL.Path, "track/v1/trackingnumbers") {
			callCount++
			authHeader := r.Header.Get("Authorization")
			
			// First call with expired token should return 401
			if callCount == 1 && authHeader == "Bearer expired_token" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{
					"errors": [{
						"code": "UNAUTHORIZED",
						"message": "Authentication failed"
					}]
				}`))
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

	client := &FedExClient{
		clientID:     "test_client_id",
		clientSecret: "test_client_secret",
		baseURL:      server.URL,
		client:       server.Client(),
		accessToken:  "expired_token",
		tokenExpiry:  time.Now().Add(-1 * time.Hour), // Expired
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

	// Verify token was refreshed
	if client.accessToken != "new_test_token" {
		t.Errorf("Expected token to be refreshed to 'new_test_token', got '%s'", client.accessToken)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 track API calls (retry after token refresh), got %d", callCount)
	}
}

func TestFedExClient_Track_MultiplePackages(t *testing.T) {
	mockTokenResponse := `{
		"access_token": "test_token",
		"token_type": "bearer",
		"expires_in": 3600
	}`

	mockTrackResponse := `{
		"output": {
			"completeTrackResults": [{
				"trackingNumber": "123456789012",
				"trackResults": [{
					"trackingNumberInfo": {"trackingNumber": "123456789012"},
					"scanEvents": [{
						"eventType": "DL",
						"eventDescription": "Delivered",
						"scanLocation": {"city": "ATLANTA", "stateOrProvinceCode": "GA"}
					}]
				}]
			}, {
				"trackingNumber": "123456789013",
				"trackResults": [{
					"trackingNumberInfo": {"trackingNumber": "123456789013"},
					"scanEvents": [{
						"eventType": "IT",
						"eventDescription": "In transit",
						"scanLocation": {"city": "CHICAGO", "stateOrProvinceCode": "IL"}
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
		
		if strings.Contains(r.URL.Path, "track/v1/trackingnumbers") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockTrackResponse))
			return
		}
	}))
	defer server.Close()

	client := &FedExClient{
		clientID:     "test_client_id",
		clientSecret: "test_client_secret",
		baseURL:      server.URL,
		client:       server.Client(),
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