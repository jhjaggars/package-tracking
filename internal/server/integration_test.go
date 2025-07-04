package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"package-tracking/internal/database"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestServer creates a test server with in-memory database
func setupTestServer(t *testing.T) *httptest.Server {
	// Create in-memory database
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create schema
	schema := `
	CREATE TABLE shipments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tracking_number TEXT NOT NULL UNIQUE,
		carrier TEXT NOT NULL,
		description TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expected_delivery DATETIME,
		is_delivered BOOLEAN DEFAULT FALSE,
		last_manual_refresh DATETIME,
		manual_refresh_count INTEGER DEFAULT 0,
		last_auto_refresh DATETIME,
		auto_refresh_count INTEGER DEFAULT 0,
		auto_refresh_enabled BOOLEAN DEFAULT TRUE,
		auto_refresh_error TEXT,
		auto_refresh_fail_count INTEGER DEFAULT 0
	);

	CREATE TABLE tracking_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		shipment_id INTEGER NOT NULL,
		timestamp DATETIME NOT NULL,
		location TEXT,
		status TEXT NOT NULL,
		description TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (shipment_id) REFERENCES shipments(id) ON DELETE CASCADE
	);

	CREATE TABLE refresh_cache (
		shipment_id INTEGER PRIMARY KEY,
		response_data TEXT NOT NULL,
		cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (shipment_id) REFERENCES shipments(id) ON DELETE CASCADE
	);

	CREATE TABLE carriers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		code TEXT NOT NULL UNIQUE,
		api_endpoint TEXT,
		active BOOLEAN DEFAULT TRUE
	);

	CREATE INDEX idx_shipments_status ON shipments(status);
	CREATE INDEX idx_shipments_carrier ON shipments(carrier);
	CREATE INDEX idx_shipments_carrier_delivered ON shipments(carrier, is_delivered);
	CREATE INDEX idx_tracking_events_shipment ON tracking_events(shipment_id);
	CREATE INDEX idx_tracking_events_dedup ON tracking_events(shipment_id, timestamp, description);
	`

	if _, err := sqlDB.Exec(schema); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	// Create database wrapper
	db := &database.DB{
		DB:             sqlDB,
		Shipments:      database.NewShipmentStore(sqlDB),
		TrackingEvents: database.NewTrackingEventStore(sqlDB),
		Carriers:       database.NewCarrierStore(sqlDB),
	}

	// Insert default carriers
	carriers := []struct {
		name, code, apiEndpoint string
		active                  bool
	}{
		{"United Parcel Service", "ups", "https://api.ups.com/track", true},
		{"United States Postal Service", "usps", "https://api.usps.com/track", true},
		{"FedEx", "fedex", "https://api.fedex.com/track", true},
		{"DHL", "dhl", "https://api.dhl.com/track", false},
	}

	for _, carrier := range carriers {
		_, err := sqlDB.Exec(
			"INSERT INTO carriers (name, code, api_endpoint, active) VALUES (?, ?, ?, ?)",
			carrier.name, carrier.code, carrier.apiEndpoint, carrier.active,
		)
		if err != nil {
			t.Fatalf("Failed to insert test carrier: %v", err)
		}
	}

	// Create chi router like production
	r := chi.NewRouter()

	// Add middleware like production
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(CORSMiddleware)
	r.Use(ContentTypeMiddleware)
	r.Use(SecurityMiddleware)

	// Create handlers
	handlerWrappers := NewHandlerWrappers(db)
	handlerWrappers.RegisterChiRoutes(r)

	handler := r

	return httptest.NewServer(handler)
}

func TestIntegrationWorkflow(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	client := server.Client()

	t.Run("CompleteShipmentWorkflow", func(t *testing.T) {
		// 1. Check initial empty shipments list
		resp, err := client.Get(server.URL + "/api/shipments")
		if err != nil {
			t.Fatalf("Failed to get shipments: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var shipments []database.Shipment
		if err := json.NewDecoder(resp.Body).Decode(&shipments); err != nil {
			t.Fatalf("Failed to decode shipments: %v", err)
		}

		if len(shipments) != 0 {
			t.Errorf("Expected 0 shipments initially, got %d", len(shipments))
		}

		// 2. Create a new shipment
		newShipment := database.Shipment{
			TrackingNumber: "1Z999AA1234567890",
			Carrier:        "ups",
			Description:    "Test Package",
			Status:         "pending",
		}

		jsonData, _ := json.Marshal(newShipment)
		resp, err = client.Post(server.URL+"/api/shipments", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			t.Fatalf("Failed to create shipment: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}

		var createdShipment database.Shipment
		if err := json.NewDecoder(resp.Body).Decode(&createdShipment); err != nil {
			t.Fatalf("Failed to decode created shipment: %v", err)
		}

		if createdShipment.ID == 0 {
			t.Error("Expected non-zero ID for created shipment")
		}

		if createdShipment.TrackingNumber != newShipment.TrackingNumber {
			t.Errorf("Expected tracking number %s, got %s", newShipment.TrackingNumber, createdShipment.TrackingNumber)
		}

		shipmentID := createdShipment.ID

		// 3. Get the shipment by ID
		resp, err = client.Get(server.URL + "/api/shipments/" + fmt.Sprintf("%d", shipmentID))
		if err != nil {
			t.Fatalf("Failed to get shipment by ID: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var retrievedShipment database.Shipment
		if err := json.NewDecoder(resp.Body).Decode(&retrievedShipment); err != nil {
			t.Fatalf("Failed to decode retrieved shipment: %v", err)
		}

		if retrievedShipment.ID != shipmentID {
			t.Errorf("Expected ID %d, got %d", shipmentID, retrievedShipment.ID)
		}

		// 4. Update the shipment
		updateShipment := database.Shipment{
			TrackingNumber: "1Z999AA1234567890",
			Carrier:        "ups",
			Description:    "Updated Test Package",
			Status:         "in_transit",
		}

		jsonData, _ = json.Marshal(updateShipment)
		req, _ := http.NewRequest("PUT", server.URL+"/api/shipments/"+fmt.Sprintf("%d", shipmentID), bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Failed to update shipment: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// 5. Get shipment events (should be empty initially)
		resp, err = client.Get(server.URL + "/api/shipments/" + fmt.Sprintf("%d", shipmentID) + "/events")
		if err != nil {
			t.Fatalf("Failed to get shipment events: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var events []database.TrackingEvent
		if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
			t.Fatalf("Failed to decode events: %v", err)
		}

		if len(events) != 0 {
			t.Errorf("Expected 0 events initially, got %d", len(events))
		}

		// 6. Delete the shipment
		req, _ = http.NewRequest("DELETE", server.URL+"/api/shipments/"+fmt.Sprintf("%d", shipmentID), nil)
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Failed to delete shipment: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", resp.StatusCode)
		}

		// 7. Verify shipment is deleted
		resp, err = client.Get(server.URL + "/api/shipments/" + fmt.Sprintf("%d", shipmentID))
		if err != nil {
			t.Fatalf("Failed to check deleted shipment: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 for deleted shipment, got %d", resp.StatusCode)
		}
	})

	t.Run("HealthCheck", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/api/health")
		if err != nil {
			t.Fatalf("Failed to get health: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var health struct {
			Status   string `json:"status"`
			Database string `json:"database"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to decode health response: %v", err)
		}

		if health.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", health.Status)
		}

		if health.Database != "ok" {
			t.Errorf("Expected database 'ok', got '%s'", health.Database)
		}
	})

	t.Run("GetCarriers", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/api/carriers")
		if err != nil {
			t.Fatalf("Failed to get carriers: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var carriers []database.Carrier
		if err := json.NewDecoder(resp.Body).Decode(&carriers); err != nil {
			t.Fatalf("Failed to decode carriers: %v", err)
		}

		if len(carriers) != 4 {
			t.Errorf("Expected 4 carriers, got %d", len(carriers))
		}

		// Test active carriers filter
		resp, err = client.Get(server.URL + "/api/carriers?active=true")
		if err != nil {
			t.Fatalf("Failed to get active carriers: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var activeCarriers []database.Carrier
		if err := json.NewDecoder(resp.Body).Decode(&activeCarriers); err != nil {
			t.Fatalf("Failed to decode active carriers: %v", err)
		}

		if len(activeCarriers) != 3 {
			t.Errorf("Expected 3 active carriers, got %d", len(activeCarriers))
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		// Test 404 for non-existent shipment
		resp, err := client.Get(server.URL + "/api/shipments/999")
		if err != nil {
			t.Fatalf("Failed to get non-existent shipment: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}

		// Test invalid JSON
		resp, err = client.Post(server.URL+"/api/shipments", "application/json", bytes.NewBufferString("invalid json"))
		if err != nil {
			t.Fatalf("Failed to post invalid JSON: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}

		// Test missing required fields
		invalidShipment := map[string]string{"description": "Missing required fields"}
		jsonData, _ := json.Marshal(invalidShipment)
		resp, err = client.Post(server.URL+"/api/shipments", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			t.Fatalf("Failed to post invalid shipment: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestMiddlewareIntegration(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	client := server.Client()

	t.Run("CORSHeaders", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/api/health")
		if err != nil {
			t.Fatalf("Failed to get health: %v", err)
		}
		defer resp.Body.Close()

		if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected CORS origin header")
		}
	})

	t.Run("SecurityHeaders", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/api/health")
		if err != nil {
			t.Fatalf("Failed to get health: %v", err)
		}
		defer resp.Body.Close()

		expectedHeaders := map[string]string{
			"X-Content-Type-Options": "nosniff",
			"X-Frame-Options":        "DENY",
			"X-XSS-Protection":       "1; mode=block",
		}

		for header, expectedValue := range expectedHeaders {
			if resp.Header.Get(header) != expectedValue {
				t.Errorf("Expected header %s to be '%s', got '%s'", header, expectedValue, resp.Header.Get(header))
			}
		}
	})

	t.Run("JSONContentType", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/api/health")
		if err != nil {
			t.Fatalf("Failed to get health: %v", err)
		}
		defer resp.Body.Close()

		if resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected JSON content type, got '%s'", resp.Header.Get("Content-Type"))
		}
	})
}