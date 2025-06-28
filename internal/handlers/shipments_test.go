package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"package-tracking/internal/database"

	_ "github.com/mattn/go-sqlite3"
)

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
		is_delivered BOOLEAN DEFAULT FALSE
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

	handler := NewShipmentHandler(db)

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

	handler := NewShipmentHandler(db)

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

	handler := NewShipmentHandler(db)

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

	handler := NewShipmentHandler(db)

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

	handler := NewShipmentHandler(db)

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

	handler := NewShipmentHandler(db)

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

	handler := NewShipmentHandler(db)

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

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}