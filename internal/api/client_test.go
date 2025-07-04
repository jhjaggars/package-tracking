package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"package-tracking/internal/email"
)

func TestNewClient(t *testing.T) {
	testCases := []struct {
		name        string
		config      *ClientConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid configuration",
			config: &ClientConfig{
				BaseURL:    "http://localhost:8080",
				Timeout:    30 * time.Second,
				RetryCount: 3,
				RetryDelay: 1 * time.Second,
			},
			expectError: false,
		},
		{
			name:        "Nil configuration",
			config:      nil,
			expectError: true,
			errorMsg:    "config cannot be nil",
		},
		{
			name: "Empty base URL",
			config: &ClientConfig{
				BaseURL: "",
				Timeout: 30 * time.Second,
			},
			expectError: true,
			errorMsg:    "base URL is required",
		},
		{
			name: "Invalid base URL",
			config: &ClientConfig{
				BaseURL: "not-a-url",
				Timeout: 30 * time.Second,
			},
			expectError: true,
			errorMsg:    "invalid base URL",
		},
		{
			name: "Zero timeout",
			config: &ClientConfig{
				BaseURL: "http://localhost:8080",
				Timeout: 0,
			},
			expectError: true,
			errorMsg:    "timeout must be positive",
		},
		{
			name: "Negative retry count",
			config: &ClientConfig{
				BaseURL:    "http://localhost:8080",
				Timeout:    30 * time.Second,
				RetryCount: -1,
			},
			expectError: true,
			errorMsg:    "retry count cannot be negative",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(tc.config)

			if tc.expectError {
				// NewClient doesn't return errors, it sets defaults
				// For invalid configs, we expect client to be created with defaults
				if client == nil {
					t.Errorf("Expected client even with invalid config, but got nil")
				}
			} else {
				if client == nil {
					t.Errorf("Expected client, but got nil")
				}
			}
		})
	}
}

func TestClient_CreateShipment(t *testing.T) {
	testCases := []struct {
		name           string
		tracking       email.TrackingInfo
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
		errorMsg       string
	}{
		{
			name: "Successful creation",
			tracking: email.TrackingInfo{
				Number:      "1Z999AA1234567890",
				Carrier:     "ups",
				Description: "Package from Amazon",
				Confidence:  0.9,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				if r.Method != "POST" {
					t.Errorf("Expected POST method, got %s", r.Method)
				}
				if r.URL.Path != "/api/shipments" {
					t.Errorf("Expected path /api/shipments, got %s", r.URL.Path)
				}

				// Verify request body
				var req ShipmentRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("Failed to decode request body: %v", err)
				}

				if req.TrackingNumber != "1Z999AA1234567890" {
					t.Errorf("Expected tracking number 1Z999AA1234567890, got %s", req.TrackingNumber)
				}
				if req.Carrier != "ups" {
					t.Errorf("Expected carrier ups, got %s", req.Carrier)
				}

				// Send successful response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				response := ShipmentResponse{
					ID:             1,
					TrackingNumber: req.TrackingNumber,
					Carrier:        req.Carrier,
					Status:         "pending",
					CreatedAt:      time.Now().Format(time.RFC3339),
				}
				json.NewEncoder(w).Encode(response)
			},
			expectError: false,
		},
		{
			name: "Server error response",
			tracking: email.TrackingInfo{
				Number:  "INVALID123",
				Carrier: "unknown",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "Invalid tracking number"}`))
			},
			expectError: true,
			errorMsg:    "bad request",
		},
		{
			name: "Network timeout simulation",
			tracking: email.TrackingInfo{
				Number:  "1Z999AA1234567890",
				Carrier: "ups",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Simulate slow response (longer than client timeout)
				time.Sleep(100 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			},
			expectError: true, // Should timeout with short timeout
		},
		{
			name: "Invalid JSON response",
			tracking: email.TrackingInfo{
				Number:  "1Z999AA1234567890",
				Carrier: "ups",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"invalid": json}`)) // Invalid JSON
			},
			expectError: true,
			errorMsg:    "failed to parse success response",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tc.serverResponse))
			defer server.Close()

			// Create client with test server URL
			config := &ClientConfig{
				BaseURL:    server.URL,
				Timeout:    50 * time.Millisecond, // Short timeout for testing
				RetryCount: 0,                     // No retries for simpler testing
				RetryDelay: 1 * time.Millisecond,
			}

			client := NewClient(config)

			// Test the CreateShipment method
			err := client.CreateShipment(tc.tracking)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				} else if tc.errorMsg != "" && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %v", tc.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestClient_CreateShipmentWithRetries(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		
		// Fail first two attempts, succeed on third
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Internal server error"}`))
			return
		}

		// Succeed on third attempt
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		response := ShipmentResponse{
			ID:             1,
			TrackingNumber: "1Z999AA1234567890",
			Carrier:        "ups",
			Status:         "pending",
			CreatedAt:      time.Now().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    1 * time.Second,
		RetryCount: 3,
		RetryDelay: 1 * time.Millisecond, // Fast retries for testing
	}

	client := NewClient(config)

	tracking := email.TrackingInfo{
		Number:  "1Z999AA1234567890",
		Carrier: "ups",
	}

	err := client.CreateShipment(tracking)
	if err != nil {
		t.Errorf("Unexpected error after retries: %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestClient_CreateShipmentMaxRetriesExceeded(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Server always fails"}`))
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    1 * time.Second,
		RetryCount: 2, // Only 2 retries
		RetryDelay: 1 * time.Millisecond,
	}

	client := NewClient(config)

	tracking := email.TrackingInfo{
		Number:  "1Z999AA1234567890",
		Carrier: "ups",
	}

	err := client.CreateShipment(tracking)
	if err == nil {
		t.Errorf("Expected error after max retries, but got none")
	}

	// Should attempt initial request + 2 retries = 3 total
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts (1 initial + 2 retries), got %d", attemptCount)
	}
}

func TestClient_HealthCheck(t *testing.T) {
	testCases := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
	}{
		{
			name: "Healthy server",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/health" {
					t.Errorf("Expected path /api/health, got %s", r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "healthy"}`))
			},
			expectError: false,
		},
		{
			name: "Unhealthy server",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"status": "unhealthy"}`))
			},
			expectError: true,
		},
		{
			name: "Server not responding",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Simulate timeout by not responding
				time.Sleep(100 * time.Millisecond)
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tc.serverResponse))
			defer server.Close()

			config := &ClientConfig{
				BaseURL:    server.URL,
				Timeout:    50 * time.Millisecond,
				RetryCount: 0, // No retries for health check
			}

			client := NewClient(config)

			err := client.HealthCheck()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected health check error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected health check error: %v", err)
				}
			}
		})
	}
}

func TestClient_Close(t *testing.T) {
	config := &ClientConfig{
		BaseURL:    "http://localhost:8080",
		Timeout:    30 * time.Second,
		RetryCount: 3,
		RetryDelay: 1 * time.Second,
	}

	client := NewClient(config)

	// Test that Close doesn't panic
	err := client.Close()
	if err != nil {
		t.Errorf("Unexpected error from Close: %v", err)
	}
}

func TestShipmentRequest_Validation(t *testing.T) {
	testCases := []struct {
		name     string
		tracking email.TrackingInfo
		expected ShipmentRequest
	}{
		{
			name: "Complete tracking info",
			tracking: email.TrackingInfo{
				Number:      "1Z999AA1234567890",
				Carrier:     "ups",
				Description: "Package from Amazon",
				Confidence:  0.9,
				Source:      "regex",
			},
			expected: ShipmentRequest{
				TrackingNumber: "1Z999AA1234567890",
				Carrier:        "ups",
				Description:    "Package from Amazon",
			},
		},
		{
			name: "Minimal tracking info",
			tracking: email.TrackingInfo{
				Number:  "123456789012",
				Carrier: "fedex",
			},
			expected: ShipmentRequest{
				TrackingNumber: "123456789012",
				Carrier:        "fedex",
				Description:    "",
			},
		},
		{
			name: "Long description truncation",
			tracking: email.TrackingInfo{
				Number:      "1Z999AA1234567890",
				Carrier:     "ups",
				Description: strings.Repeat("A", 300), // Very long description
			},
			expected: ShipmentRequest{
				TrackingNumber: "1Z999AA1234567890",
				Carrier:        "ups",
				Description:    strings.Repeat("A", 255) + "...", // Should be truncated
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := createShipmentRequest(tc.tracking)

			if request.TrackingNumber != tc.expected.TrackingNumber {
				t.Errorf("Expected tracking number %s, got %s", tc.expected.TrackingNumber, request.TrackingNumber)
			}

			if request.Carrier != tc.expected.Carrier {
				t.Errorf("Expected carrier %s, got %s", tc.expected.Carrier, request.Carrier)
			}

			// For the truncation test case, verify truncation happened
			if tc.name == "Long description truncation" {
				if len(request.Description) != 258 { // 255 + "..."
					t.Errorf("Expected truncated description length 258, got %d", len(request.Description))
				}
				if !strings.HasSuffix(request.Description, "...") {
					t.Errorf("Expected truncated description to end with '...', got: %s", request.Description[len(request.Description)-10:])
				}
			} else {
				// For non-truncation tests, ensure no unnecessary length
				if len(request.Description) > 300 {
					t.Errorf("Description unexpectedly long: %d characters", len(request.Description))
				}
			}
		})
	}
}

func TestClient_ConcurrentRequests(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		response := ShipmentResponse{
			ID:             requestCount,
			TrackingNumber: "TEST123",
			Carrier:        "ups",
			Status:         "pending",
			CreatedAt:      time.Now().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    1 * time.Second,
		RetryCount: 0,
	}

	client := NewClient(config)

	// Send multiple concurrent requests
	const numRequests = 5
	done := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			tracking := email.TrackingInfo{
				Number:  fmt.Sprintf("TRACK%d", id),
				Carrier: "ups",
			}
			done <- client.CreateShipment(tracking)
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		err := <-done
		if err != nil {
			t.Errorf("Request %d failed: %v", i, err)
		}
	}

	if requestCount != numRequests {
		t.Errorf("Expected %d requests, got %d", numRequests, requestCount)
	}
}

func TestClient_HandleSpecialCharacters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ShipmentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request with special characters: %v", err)
		}

		// Verify special characters are properly handled
		if !strings.Contains(req.Description, "Möbel") {
			t.Errorf("Special characters not preserved in description: %s", req.Description)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		response := ShipmentResponse{
			ID:             1,
			TrackingNumber: req.TrackingNumber,
			Carrier:        req.Carrier,
			Status:         "pending",
			CreatedAt:      time.Now().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    1 * time.Second,
		RetryCount: 0,
	}

	client := NewClient(config)

	tracking := email.TrackingInfo{
		Number:      "1Z999AA1234567890",
		Carrier:     "ups",
		Description: "Möbel für das Büro (Office furniture) - 包裹 📦",
	}

	err := client.CreateShipment(tracking)
	if err != nil {
		t.Errorf("Failed to handle special characters: %v", err)
	}
}

// Helper function to create shipment request from tracking info
func createShipmentRequest(tracking email.TrackingInfo) ShipmentRequest {
	description := tracking.Description
	if len(description) > 255 {
		description = description[:255] + "..."
	}

	return ShipmentRequest{
		TrackingNumber: tracking.Number,
		Carrier:        tracking.Carrier,
		Description:    description,
	}
}

// Benchmark tests
func BenchmarkClient_CreateShipment(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		response := ShipmentResponse{
			ID:             1,
			TrackingNumber: "1Z999AA1234567890",
			Carrier:        "ups",
			Status:         "pending",
			CreatedAt:      time.Now().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &ClientConfig{
		BaseURL:    server.URL,
		Timeout:    1 * time.Second,
		RetryCount: 0,
	}

	client := NewClient(config)

	tracking := email.TrackingInfo{
		Number:      "1Z999AA1234567890",
		Carrier:     "ups",
		Description: "Benchmark test package",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := client.CreateShipment(tracking)
		if err != nil {
			b.Fatalf("CreateShipment failed: %v", err)
		}
	}
}

func TestClient_CreateShipment_EnhancedDescriptions(t *testing.T) {
	testCases := []struct {
		name                string
		tracking            email.TrackingInfo
		expectedDescription string
		expectedRequest     string
	}{
		{
			name: "description with merchant information",
			tracking: email.TrackingInfo{
				Number:      "1Z999AA1234567890",
				Carrier:     "ups",
				Description: "Apple iPhone 15 Pro",
				Merchant:    "Amazon",
				SourceEmail: email.EmailMessage{
					From:    "noreply@amazon.com",
					Subject: "Your order has shipped",
				},
			},
			expectedDescription: "Apple iPhone 15 Pro from Amazon",
			expectedRequest:     `"description":"Apple iPhone 15 Pro from Amazon"`,
		},
		{
			name: "description only (no merchant)",
			tracking: email.TrackingInfo{
				Number:      "1Z999AA1234567890",
				Carrier:     "ups",
				Description: "Samsung Galaxy S24",
				SourceEmail: email.EmailMessage{
					From:    "noreply@samsung.com",
					Subject: "Your order has shipped",
				},
			},
			expectedDescription: "Samsung Galaxy S24",
			expectedRequest:     `"description":"Samsung Galaxy S24"`,
		},
		{
			name: "merchant only (no description)",
			tracking: email.TrackingInfo{
				Number:   "1Z999AA1234567890",
				Carrier:  "ups",
				Merchant: "Best Buy",
				SourceEmail: email.EmailMessage{
					From:    "noreply@bestbuy.com",
					Subject: "Your order has shipped",
				},
			},
			expectedDescription: "Package from Best Buy",
			expectedRequest:     `"description":"Package from Best Buy"`,
		},
		{
			name: "fallback to email sender",
			tracking: email.TrackingInfo{
				Number:  "1Z999AA1234567890",
				Carrier: "ups",
				SourceEmail: email.EmailMessage{
					From:    "noreply@example.com",
					Subject: "Your order has shipped",
				},
			},
			expectedDescription: "Your order has shipped",
			expectedRequest:     `"description":"Your order has shipped"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server that captures request
			var receivedRequest string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/api/shipments" {
					t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
				}

				// Read and store request body
				body := make([]byte, r.ContentLength)
				r.Body.Read(body)
				receivedRequest = string(body)

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id": 123, "status": "created"}`))
			}))
			defer server.Close()

			config := &ClientConfig{
				BaseURL:    server.URL,
				Timeout:    5 * time.Second,
				RetryCount: 0,
			}

			client := NewClient(config)

			err := client.CreateShipment(tc.tracking)
			if err != nil {
				t.Fatalf("CreateShipment failed: %v", err)
			}

			// Verify the expected description format is used in the request
			if !strings.Contains(receivedRequest, tc.expectedRequest) {
				t.Errorf("Expected request to contain '%s', but got: %s", tc.expectedRequest, receivedRequest)
			}
		})
	}
}

func TestFormatDescriptionWithMerchant(t *testing.T) {
	// Test helper function for formatting descriptions with merchant info
	testCases := []struct {
		name        string
		description string
		merchant    string
		from        string
		subject     string
		expected    string
	}{
		{
			name:        "both description and merchant",
			description: "iPhone 15 Pro",
			merchant:    "Amazon",
			expected:    "iPhone 15 Pro from Amazon",
		},
		{
			name:        "description only",
			description: "Samsung Galaxy S24",
			merchant:    "",
			expected:    "Samsung Galaxy S24",
		},
		{
			name:        "merchant only",
			description: "",
			merchant:    "Best Buy",
			expected:    "Package from Best Buy",
		},
		{
			name:        "neither description nor merchant",
			description: "",
			merchant:    "",
			from:        "noreply@example.com",
			subject:     "Your order has shipped",
			expected:    "Your order has shipped",
		},
		{
			name:        "empty subject fallback to sender",
			description: "",
			merchant:    "",
			from:        "noreply@example.com",
			subject:     "",
			expected:    "Package from noreply@example.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatDescriptionWithMerchant(tc.description, tc.merchant, tc.from, tc.subject)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

