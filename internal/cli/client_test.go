package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"package-tracking/internal/database"
)

func TestNewClient(t *testing.T) {
	baseURL := "http://example.com"
	client := NewClient(baseURL)
	
	if client.baseURL != baseURL {
		t.Errorf("Expected baseURL to be '%s', got '%s'", baseURL, client.baseURL)
	}
	
	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout to be 30s, got %v", client.httpClient.Timeout)
	}
}

func TestNewClient_RemovesTrailingSlash(t *testing.T) {
	baseURL := "http://example.com/"
	client := NewClient(baseURL)
	
	expected := "http://example.com"
	if client.baseURL != expected {
		t.Errorf("Expected baseURL to be '%s', got '%s'", expected, client.baseURL)
	}
}

func TestNewClientWithTimeout(t *testing.T) {
	baseURL := "http://example.com"
	timeout := 60 * time.Second
	client := NewClientWithTimeout(baseURL, timeout)
	
	if client.baseURL != baseURL {
		t.Errorf("Expected baseURL to be '%s', got '%s'", baseURL, client.baseURL)
	}
	
	if client.httpClient.Timeout != timeout {
		t.Errorf("Expected timeout to be %v, got %v", timeout, client.httpClient.Timeout)
	}
}

func TestHealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/api/health" {
			t.Errorf("Expected path '/api/health', got '%s'", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()
	
	client := NewClient(server.URL)
	err := client.HealthCheck()
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestHealthCheck_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":500,"message":"Internal server error"}`))
	}))
	defer server.Close()
	
	client := NewClient(server.URL)
	err := client.HealthCheck()
	
	if err == nil {
		t.Error("Expected error, got nil")
	}
	
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Expected APIError, got %T", err)
	}
	
	if apiErr.Code != 500 {
		t.Errorf("Expected error code 500, got %d", apiErr.Code)
	}
}

func TestCreateShipment_Success(t *testing.T) {
	expectedShipment := database.Shipment{
		ID:             1,
		TrackingNumber: "1Z999AA1234567890",
		Carrier:        "ups",
		Description:    "Test package",
		Status:         "pending",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/shipments" {
			t.Errorf("Expected path '/api/shipments', got '%s'", r.URL.Path)
		}
		
		var req CreateShipmentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}
		
		if req.TrackingNumber != "1Z999AA1234567890" {
			t.Errorf("Expected tracking number '1Z999AA1234567890', got '%s'", req.TrackingNumber)
		}
		
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(expectedShipment)
	}))
	defer server.Close()
	
	client := NewClient(server.URL)
	req := &CreateShipmentRequest{
		TrackingNumber: "1Z999AA1234567890",
		Carrier:        "ups",
		Description:    "Test package",
	}
	
	shipment, err := client.CreateShipment(req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if shipment.ID != expectedShipment.ID {
		t.Errorf("Expected shipment ID %d, got %d", expectedShipment.ID, shipment.ID)
	}
}

func TestCreateShipment_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"code":400,"message":"Invalid tracking number"}`))
	}))
	defer server.Close()
	
	client := NewClient(server.URL)
	req := &CreateShipmentRequest{
		TrackingNumber: "",
		Carrier:        "ups",
		Description:    "Test package",
	}
	
	_, err := client.CreateShipment(req)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Expected APIError, got %T", err)
	}
	
	if apiErr.Code != 400 {
		t.Errorf("Expected error code 400, got %d", apiErr.Code)
	}
}

func TestGetShipments_Success(t *testing.T) {
	expectedShipments := []database.Shipment{
		{
			ID:             1,
			TrackingNumber: "1Z999AA1234567890",
			Carrier:        "ups",
			Description:    "Test package 1",
			Status:         "pending",
		},
		{
			ID:             2,
			TrackingNumber: "1234567890",
			Carrier:        "fedex",
			Description:    "Test package 2",
			Status:         "delivered",
		},
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/api/shipments" {
			t.Errorf("Expected path '/api/shipments', got '%s'", r.URL.Path)
		}
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedShipments)
	}))
	defer server.Close()
	
	client := NewClient(server.URL)
	shipments, err := client.GetShipments()
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(shipments) != 2 {
		t.Errorf("Expected 2 shipments, got %d", len(shipments))
	}
	
	if shipments[0].ID != 1 {
		t.Errorf("Expected first shipment ID 1, got %d", shipments[0].ID)
	}
}

func TestGetShipment_Success(t *testing.T) {
	expectedShipment := database.Shipment{
		ID:             1,
		TrackingNumber: "1Z999AA1234567890",
		Carrier:        "ups",
		Description:    "Test package",
		Status:         "pending",
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/api/shipments/1" {
			t.Errorf("Expected path '/api/shipments/1', got '%s'", r.URL.Path)
		}
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedShipment)
	}))
	defer server.Close()
	
	client := NewClient(server.URL)
	shipment, err := client.GetShipment(1)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if shipment.ID != 1 {
		t.Errorf("Expected shipment ID 1, got %d", shipment.ID)
	}
}

func TestGetShipment_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code":404,"message":"Shipment not found"}`))
	}))
	defer server.Close()
	
	client := NewClient(server.URL)
	_, err := client.GetShipment(999)
	
	if err == nil {
		t.Error("Expected error, got nil")
	}
	
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Expected APIError, got %T", err)
	}
	
	if apiErr.Code != 404 {
		t.Errorf("Expected error code 404, got %d", apiErr.Code)
	}
}

func TestUpdateShipment_Success(t *testing.T) {
	expectedShipment := database.Shipment{
		ID:             1,
		TrackingNumber: "1Z999AA1234567890",
		Carrier:        "ups",
		Description:    "Updated description",
		Status:         "pending",
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT request, got %s", r.Method)
		}
		if r.URL.Path != "/api/shipments/1" {
			t.Errorf("Expected path '/api/shipments/1', got '%s'", r.URL.Path)
		}
		
		var req UpdateShipmentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}
		
		if req.Description != "Updated description" {
			t.Errorf("Expected description 'Updated description', got '%s'", req.Description)
		}
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedShipment)
	}))
	defer server.Close()
	
	client := NewClient(server.URL)
	req := &UpdateShipmentRequest{
		Description: "Updated description",
	}
	
	shipment, err := client.UpdateShipment(1, req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if shipment.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", shipment.Description)
	}
}

func TestDeleteShipment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}
		if r.URL.Path != "/api/shipments/1" {
			t.Errorf("Expected path '/api/shipments/1', got '%s'", r.URL.Path)
		}
		
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	
	client := NewClient(server.URL)
	err := client.DeleteShipment(1)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestGetEvents_Success(t *testing.T) {
	expectedEvents := []database.TrackingEvent{
		{
			ID:          1,
			ShipmentID:  1,
			Timestamp:   time.Now(),
			Location:    "Origin facility",
			Status:      "picked_up",
			Description: "Package picked up",
		},
		{
			ID:          2,
			ShipmentID:  1,
			Timestamp:   time.Now(),
			Location:    "Sorting facility",
			Status:      "in_transit",
			Description: "In transit",
		},
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/api/shipments/1/events" {
			t.Errorf("Expected path '/api/shipments/1/events', got '%s'", r.URL.Path)
		}
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedEvents)
	}))
	defer server.Close()
	
	client := NewClient(server.URL)
	events, err := client.GetEvents(1)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}
	
	if events[0].ID != 1 {
		t.Errorf("Expected first event ID 1, got %d", events[0].ID)
	}
}

func TestAPIError_Error(t *testing.T) {
	apiErr := &APIError{
		Code:    404,
		Message: "Not found",
	}
	
	expected := "API error 404: Not found"
	if apiErr.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, apiErr.Error())
	}
}

func TestDoRequest_NetworkError(t *testing.T) {
	// Use an invalid URL to trigger a network error
	client := NewClient("http://invalid-url-that-does-not-exist.test")
	
	_, err := client.doRequest("GET", "/api/health", nil)
	if err == nil {
		t.Error("Expected network error, got nil")
	}
	
	// Should be an APIError with Code 0 for network errors
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Expected APIError, got %T", err)
	}
	
	if apiErr.Code != 0 {
		t.Errorf("Expected network error code 0, got %d", apiErr.Code)
	}
	
	if !strings.Contains(err.Error(), "Network error") {
		t.Errorf("Expected error to contain 'Network error', got '%s'", err.Error())
	}
}

func TestDoRequest_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()
	
	client := NewClient(server.URL)
	_, err := client.doRequest("GET", "/api/health", nil)
	
	if err == nil {
		t.Error("Expected error, got nil")
	}
	
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Expected APIError, got %T", err)
	}
	
	// Should create a generic error when JSON decode fails
	if apiErr.Code != 400 {
		t.Errorf("Expected error code 400, got %d", apiErr.Code)
	}
}