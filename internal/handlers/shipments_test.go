package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"package-tracking/internal/cache"
	"package-tracking/internal/database"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
)

// TestConfig implements the Config interface for testing
type TestConfig struct {
	DisableRateLimit bool
	DisableCache     bool
}

func (tc *TestConfig) GetDisableRateLimit() bool {
	return tc.DisableRateLimit
}

func (tc *TestConfig) GetDisableCache() bool {
	return tc.DisableCache
}

func (tc *TestConfig) GetFedExAPIKey() string {
	return ""
}

func (tc *TestConfig) GetFedExSecretKey() string {
	return ""
}

func (tc *TestConfig) GetFedExAPIURL() string {
	return "https://apis.fedex.com"
}

// setupTestHandler creates a shipment handler with disabled cache for testing
func setupTestHandler(db *database.DB) *ShipmentHandler {
	config := &TestConfig{DisableRateLimit: false, DisableCache: true}
	cacheManager := cache.NewManager(db.RefreshCache, true, 5*time.Minute)
	return NewShipmentHandler(db, config, cacheManager)
}

// Test database setup and teardown utilities
func setupTestDB(t *testing.T) *database.DB {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create tables
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
		auto_refresh_fail_count INTEGER DEFAULT 0,
		amazon_order_number TEXT,
		delegated_carrier TEXT,
		delegated_tracking_number TEXT,
		is_amazon_logistics BOOLEAN DEFAULT FALSE
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

	// Enable foreign key constraints
	if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	if _, err := sqlDB.Exec(schema); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	// Create the database wrapper
	db := &database.DB{
		DB:             sqlDB,
		Shipments:      database.NewShipmentStore(sqlDB),
		TrackingEvents: database.NewTrackingEventStore(sqlDB),
		Carriers:       database.NewCarrierStore(sqlDB),
		RefreshCache:   database.NewRefreshCacheStore(sqlDB),
	}

	return db
}

func teardownTestDB(db *database.DB) {
	db.Close()
}

func insertTestShipment(t *testing.T, db *database.DB, shipment database.Shipment) int {
	err := db.Shipments.Create(&shipment)
	if err != nil {
		t.Fatalf("Failed to insert test shipment: %v", err)
	}

	return shipment.ID
}

func insertTestTrackingEvent(t *testing.T, db *database.DB, event database.TrackingEvent) {
	query := `INSERT INTO tracking_events (shipment_id, timestamp, location, status, description) 
			  VALUES (?, ?, ?, ?, ?)`
	
	_, err := db.Exec(query, event.ShipmentID, event.Timestamp, event.Location, event.Status, event.Description)
	if err != nil {
		t.Fatalf("Failed to insert test tracking event: %v", err)
	}
}

// Test GET /api/shipments (list all)
func TestGetShipments(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(db)

	handler := setupTestHandler(db)

	// Test empty list
	t.Run("EmptyList", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/shipments", nil)
		w := httptest.NewRecorder()

		handler.GetShipments(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var shipments []database.Shipment
		if err := json.NewDecoder(w.Body).Decode(&shipments); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(shipments) != 0 {
			t.Errorf("Expected empty list, got %d shipments", len(shipments))
		}
	})

	// Test list with shipments
	t.Run("WithShipments", func(t *testing.T) {
		// Insert test data
		shipment1 := database.Shipment{
			TrackingNumber: "1Z999AA1234567890",
			Carrier:        "ups",
			Description:    "Test Package 1",
			Status:         "in_transit",
		}
		shipment2 := database.Shipment{
			TrackingNumber: "9400111899562867886926",
			Carrier:        "usps",
			Description:    "Test Package 2",
			Status:         "pending",
		}

		insertTestShipment(t, db, shipment1)
		insertTestShipment(t, db, shipment2)

		req := httptest.NewRequest("GET", "/api/shipments", nil)
		w := httptest.NewRecorder()

		handler.GetShipments(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var shipments []database.Shipment
		if err := json.NewDecoder(w.Body).Decode(&shipments); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(shipments) != 2 {
			t.Errorf("Expected 2 shipments, got %d", len(shipments))
			return
		}

		if shipments[0].TrackingNumber != "1Z999AA1234567890" {
			t.Errorf("Expected tracking number '1Z999AA1234567890', got '%s'", shipments[0].TrackingNumber)
		}
	})
}

// Test POST /api/shipments (create)
func TestCreateShipment(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(db)

	handler := setupTestHandler(db)

	t.Run("ValidShipment", func(t *testing.T) {
		shipment := database.Shipment{
			TrackingNumber: "1Z999AA1234567890",
			Carrier:        "ups",
			Description:    "Test Package",
			Status:         "pending",
		}

		jsonData, _ := json.Marshal(shipment)
		req := httptest.NewRequest("POST", "/api/shipments", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateShipment(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		var created database.Shipment
		if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if created.ID == 0 {
			t.Error("Expected non-zero ID")
		}
		if created.TrackingNumber != shipment.TrackingNumber {
			t.Errorf("Expected tracking number '%s', got '%s'", shipment.TrackingNumber, created.TrackingNumber)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/shipments", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateShipment(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		shipment := database.Shipment{
			Description: "Missing tracking number and carrier",
		}

		jsonData, _ := json.Marshal(shipment)
		req := httptest.NewRequest("POST", "/api/shipments", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateShipment(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("DuplicateTrackingNumber", func(t *testing.T) {
		shipment := database.Shipment{
			TrackingNumber: "DUPLICATE123",
			Carrier:        "ups",
			Description:    "First package",
		}

		// Insert first shipment
		insertTestShipment(t, db, shipment)

		// Try to insert duplicate
		jsonData, _ := json.Marshal(shipment)
		req := httptest.NewRequest("POST", "/api/shipments", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateShipment(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("Expected status 409, got %d", w.Code)
		}
	})
}

// Test GET /api/shipments/{id} (get by ID)
func TestGetShipmentByID(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(db)

	handler := setupTestHandler(db)

	t.Run("ExistingShipment", func(t *testing.T) {
		shipment := database.Shipment{
			TrackingNumber: "1Z999AA1234567777",
			Carrier:        "ups",
			Description:    "Test Package",
			Status:         "in_transit",
		}

		id := insertTestShipment(t, db, shipment)

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/shipments/%d", id), nil)
		w := httptest.NewRecorder()

		handler.GetShipmentByID(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var retrieved database.Shipment
		if err := json.NewDecoder(w.Body).Decode(&retrieved); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if retrieved.ID != id {
			t.Errorf("Expected ID %d, got %d", id, retrieved.ID)
		}
		if retrieved.TrackingNumber != "1Z999AA1234567777" {
			t.Errorf("Expected tracking number '1Z999AA1234567777', got '%s'", retrieved.TrackingNumber)
		}
	})

	t.Run("NonExistentShipment", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/shipments/999", nil)
		w := httptest.NewRecorder()

		handler.GetShipmentByID(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("InvalidID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/shipments/invalid", nil)
		w := httptest.NewRecorder()

		handler.GetShipmentByID(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// Test PUT /api/shipments/{id} (update)
func TestUpdateShipment(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(db)

	handler := setupTestHandler(db)

	t.Run("ValidUpdate", func(t *testing.T) {
		shipment := database.Shipment{
			TrackingNumber: "1Z999AA1234567999",
			Carrier:        "ups",
			Description:    "Original Description",
			Status:         "pending",
		}

		id := insertTestShipment(t, db, shipment)

		update := database.Shipment{
			TrackingNumber: "1Z999AA1234567999",
			Carrier:        "ups",
			Description:    "Updated Description",
			Status:         "in_transit",
			IsDelivered:    false,
		}

		jsonData, _ := json.Marshal(update)
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/shipments/%d", id), bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		
		// Add chi context to the request for URL parameter extraction
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", fmt.Sprintf("%d", id))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		handler.UpdateShipment(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var updated database.Shipment
		if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if updated.Description != "Updated Description" {
			t.Errorf("Expected description 'Updated Description', got '%s'", updated.Description)
		}
		if updated.Status != "in_transit" {
			t.Errorf("Expected status 'in_transit', got '%s'", updated.Status)
		}
	})

	t.Run("NonExistentShipment", func(t *testing.T) {
		update := database.Shipment{
			TrackingNumber: "1Z999AA1234567890",
			Carrier:        "ups",
			Description:    "Updated Description",
			Status:         "in_transit",
		}

		jsonData, _ := json.Marshal(update)
		req := httptest.NewRequest("PUT", "/api/shipments/999", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		
		// Add chi context to the request for URL parameter extraction
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "999")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		handler.UpdateShipment(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		shipment := database.Shipment{
			TrackingNumber: "1Z999AA1234567888",
			Carrier:        "ups",
			Description:    "Test Package",
		}

		id := insertTestShipment(t, db, shipment)

		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/shipments/%d", id), bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		
		// Add chi context to the request for URL parameter extraction
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", fmt.Sprintf("%d", id))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		handler.UpdateShipment(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// Test DELETE /api/shipments/{id} (delete)
func TestDeleteShipment(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(db)

	handler := setupTestHandler(db)

	t.Run("ExistingShipment", func(t *testing.T) {
		shipment := database.Shipment{
			TrackingNumber: "1Z999AA1234567666",
			Carrier:        "ups",
			Description:    "Test Package",
		}

		id := insertTestShipment(t, db, shipment)

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/shipments/%d", id), nil)
		w := httptest.NewRecorder()

		handler.DeleteShipment(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}

		// Verify shipment is deleted
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM shipments WHERE id = ?", id).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to check shipment deletion: %v", err)
		}
		if count != 0 {
			t.Error("Shipment was not deleted")
		}
	})

	t.Run("NonExistentShipment", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/shipments/999", nil)
		w := httptest.NewRecorder()

		handler.DeleteShipment(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("InvalidID", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/shipments/invalid", nil)
		w := httptest.NewRecorder()

		handler.DeleteShipment(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// Test GET /api/shipments/{id}/events (tracking events)
func TestGetShipmentEvents(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(db)

	handler := setupTestHandler(db)

	t.Run("WithEvents", func(t *testing.T) {
		shipment := database.Shipment{
			TrackingNumber: "1Z999AA1234567555",
			Carrier:        "ups",
			Description:    "Test Package",
		}

		shipmentID := insertTestShipment(t, db, shipment)

		// Insert tracking events
		event1 := database.TrackingEvent{
			ShipmentID:  shipmentID,
			Timestamp:   time.Now().Add(-2 * time.Hour),
			Location:    "Origin Facility",
			Status:      "picked_up",
			Description: "Package picked up",
		}
		event2 := database.TrackingEvent{
			ShipmentID:  shipmentID,
			Timestamp:   time.Now().Add(-1 * time.Hour),
			Location:    "Sorting Facility",
			Status:      "in_transit",
			Description: "Package in transit",
		}

		insertTestTrackingEvent(t, db, event1)
		insertTestTrackingEvent(t, db, event2)

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/shipments/%d/events", shipmentID), nil)
		w := httptest.NewRecorder()

		handler.GetShipmentEvents(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var events []database.TrackingEvent
		if err := json.NewDecoder(w.Body).Decode(&events); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(events) != 2 {
			t.Errorf("Expected 2 events, got %d", len(events))
		}
	})

	t.Run("NoEvents", func(t *testing.T) {
		shipment := database.Shipment{
			TrackingNumber: "NOEVENTS123",
			Carrier:        "ups",
			Description:    "No Events Package",
		}

		shipmentID := insertTestShipment(t, db, shipment)

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/shipments/%d/events", shipmentID), nil)
		w := httptest.NewRecorder()

		handler.GetShipmentEvents(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var events []database.TrackingEvent
		if err := json.NewDecoder(w.Body).Decode(&events); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(events) != 0 {
			t.Errorf("Expected 0 events, got %d", len(events))
		}
	})

	t.Run("NonExistentShipment", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/shipments/999/events", nil)
		w := httptest.NewRecorder()

		handler.GetShipmentEvents(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

// Test error cases and validation
func TestValidationAndErrors(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(db)

	handler := setupTestHandler(db)

	t.Run("EmptyTrackingNumber", func(t *testing.T) {
		shipment := database.Shipment{
			TrackingNumber: "",
			Carrier:        "ups",
			Description:    "Test Package",
		}

		jsonData, _ := json.Marshal(shipment)
		req := httptest.NewRequest("POST", "/api/shipments", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateShipment(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("InvalidCarrier", func(t *testing.T) {
		shipment := database.Shipment{
			TrackingNumber: "123456",
			Carrier:        "invalid_carrier",
			Description:    "Test Package",
		}

		jsonData, _ := json.Marshal(shipment)
		req := httptest.NewRequest("POST", "/api/shipments", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateShipment(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("EmptyDescription", func(t *testing.T) {
		shipment := database.Shipment{
			TrackingNumber: "123456",
			Carrier:        "ups",
			Description:    "",
		}

		jsonData, _ := json.Marshal(shipment)
		req := httptest.NewRequest("POST", "/api/shipments", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateShipment(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// Test Amazon-specific shipment creation and validation
func TestAmazonShipments(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(db)

	handler := setupTestHandler(db)

	t.Run("AmazonOrderNumber", func(t *testing.T) {
		shipment := database.Shipment{
			TrackingNumber:     "11312345671234567", // Cleaned format (no dashes)
			Carrier:            "amazon",
			Description:        "Amazon order shipment",
			Status:             "pending",
			AmazonOrderNumber:  stringPtr("113-1234567-1234567"),
			IsAmazonLogistics:  false,
		}

		jsonData, _ := json.Marshal(shipment)
		req := httptest.NewRequest("POST", "/api/shipments", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateShipment(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		var created database.Shipment
		if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if created.Carrier != "amazon" {
			t.Errorf("Expected carrier 'amazon', got '%s'", created.Carrier)
		}
		if created.AmazonOrderNumber == nil || *created.AmazonOrderNumber != "113-1234567-1234567" {
			t.Errorf("Expected Amazon order number '113-1234567-1234567', got %v", created.AmazonOrderNumber)
		}
		if created.IsAmazonLogistics {
			t.Error("Expected IsAmazonLogistics to be false")
		}
	})

	t.Run("AmazonLogisticsTracking", func(t *testing.T) {
		shipment := database.Shipment{
			TrackingNumber:    "TBA123456789012",
			Carrier:           "amazon",
			Description:       "Amazon Logistics shipment",
			Status:            "in_transit",
			IsAmazonLogistics: true,
		}

		jsonData, _ := json.Marshal(shipment)
		req := httptest.NewRequest("POST", "/api/shipments", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateShipment(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		var created database.Shipment
		if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if created.Carrier != "amazon" {
			t.Errorf("Expected carrier 'amazon', got '%s'", created.Carrier)
		}
		if !created.IsAmazonLogistics {
			t.Error("Expected IsAmazonLogistics to be true")
		}
		if created.TrackingNumber != "TBA123456789012" {
			t.Errorf("Expected tracking number 'TBA123456789012', got '%s'", created.TrackingNumber)
		}
	})

	t.Run("AmazonDelegationToUPS", func(t *testing.T) {
		shipment := database.Shipment{
			TrackingNumber:           "45612345671234567", // Cleaned Amazon order format
			Carrier:                  "amazon",
			Description:              "Amazon order shipped via UPS",
			Status:                   "in_transit",
			AmazonOrderNumber:        stringPtr("456-1234567-1234567"),
			DelegatedCarrier:         stringPtr("ups"),
			DelegatedTrackingNumber:  stringPtr("1Z999AA1234567890"),
			IsAmazonLogistics:        false,
		}

		jsonData, _ := json.Marshal(shipment)
		req := httptest.NewRequest("POST", "/api/shipments", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateShipment(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		var created database.Shipment
		if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if created.DelegatedCarrier == nil || *created.DelegatedCarrier != "ups" {
			t.Errorf("Expected delegated carrier 'ups', got %v", created.DelegatedCarrier)
		}
		if created.DelegatedTrackingNumber == nil || *created.DelegatedTrackingNumber != "1Z999AA1234567890" {
			t.Errorf("Expected delegated tracking number '1Z999AA1234567890', got %v", created.DelegatedTrackingNumber)
		}
	})

	t.Run("AmazonValidation", func(t *testing.T) {
		// Test invalid Amazon order format
		shipment := database.Shipment{
			TrackingNumber: "12345", // Too short for Amazon
			Carrier:        "amazon",
			Description:    "Invalid Amazon order",
		}

		jsonData, _ := json.Marshal(shipment)
		req := httptest.NewRequest("POST", "/api/shipments", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CreateShipment(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid Amazon tracking number, got %d", w.Code)
		}
	})

	t.Run("UpdateAmazonShipment", func(t *testing.T) {
		// Use a fresh database for this test to avoid conflicts
		freshDB := setupTestDB(t)
		defer teardownTestDB(freshDB)
		freshHandler := setupTestHandler(freshDB)

		// First create an Amazon shipment
		shipment := database.Shipment{
			TrackingNumber:    "78912345671234567",
			Carrier:           "amazon",
			Description:       "Original Amazon order",
			AmazonOrderNumber: stringPtr("789-1234567-1234567"),
		}

		id := insertTestShipment(t, freshDB, shipment)

		// Update with delegation info
		update := database.Shipment{
			TrackingNumber:          "78912345671234567",
			Carrier:                 "amazon",
			Description:             "Updated with UPS delegation",
			AmazonOrderNumber:       stringPtr("789-1234567-1234567"),
			DelegatedCarrier:        stringPtr("ups"),
			DelegatedTrackingNumber: stringPtr("1Z999AA1234567999"),
		}

		jsonData, _ := json.Marshal(update)
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/shipments/%d", id), bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		
		// Add chi context to the request for URL parameter extraction
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", fmt.Sprintf("%d", id))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		
		w := httptest.NewRecorder()

		freshHandler.UpdateShipment(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var updated database.Shipment
		if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
			t.Fatalf("Failed to decode response: %v. Body: %s", err, w.Body.String())
		}

		if updated.DelegatedCarrier == nil || *updated.DelegatedCarrier != "ups" {
			t.Errorf("Expected delegated carrier 'ups', got %v", updated.DelegatedCarrier)
		}
		if updated.DelegatedTrackingNumber == nil || *updated.DelegatedTrackingNumber != "1Z999AA1234567999" {
			t.Errorf("Expected delegated tracking '1Z999AA1234567999', got %v", updated.DelegatedTrackingNumber)
		}
	})

	t.Run("GetAmazonShipments", func(t *testing.T) {
		// Use a fresh database for this test to avoid conflicts
		freshDB := setupTestDB(t)
		defer teardownTestDB(freshDB)
		freshHandler := setupTestHandler(freshDB)

		// Insert multiple Amazon shipments with different types
		shipments := []database.Shipment{
			{
				TrackingNumber:    "12312345671234567",
				Carrier:           "amazon",
				Description:       "Amazon Order",
				AmazonOrderNumber: stringPtr("123-1234567-1234567"),
			},
			{
				TrackingNumber:    "TBA999888777666",
				Carrier:           "amazon",
				Description:       "Amazon Logistics",
				IsAmazonLogistics: true,
			},
			{
				TrackingNumber:           "55512345671234567",
				Carrier:                  "amazon",
				Description:              "Amazon via FedEx",
				AmazonOrderNumber:        stringPtr("555-1234567-1234567"),
				DelegatedCarrier:         stringPtr("fedex"),
				DelegatedTrackingNumber:  stringPtr("123456789012"),
			},
		}

		for _, shipment := range shipments {
			insertTestShipment(t, freshDB, shipment)
		}

		req := httptest.NewRequest("GET", "/api/shipments", nil)
		w := httptest.NewRecorder()

		freshHandler.GetShipments(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var retrieved []database.Shipment
		if err := json.NewDecoder(w.Body).Decode(&retrieved); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Count Amazon shipments
		amazonCount := 0
		for _, shipment := range retrieved {
			if shipment.Carrier == "amazon" {
				amazonCount++
			}
		}

		if amazonCount != 3 {
			t.Errorf("Expected 3 Amazon shipments, got %d", amazonCount)
		}
	})
}

// Helper function to create string pointers for optional fields
func stringPtr(s string) *string {
	return &s
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}