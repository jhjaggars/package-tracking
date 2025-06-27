package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"package-tracking/internal/database"
)

func insertTestCarrier(t *testing.T, db *database.DB, carrier database.Carrier) int {
	query := `INSERT INTO carriers (name, code, api_endpoint, active) VALUES (?, ?, ?, ?)`
	
	result, err := db.Exec(query, carrier.Name, carrier.Code, carrier.APIEndpoint, carrier.Active)
	if err != nil {
		t.Fatalf("Failed to insert test carrier: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last insert ID: %v", err)
	}

	return int(id)
}

func setupCarrierTestDB(t *testing.T) *database.DB {
	db := setupTestDB(t)

	// Insert default carriers
	carriers := []database.Carrier{
		{Name: "United Parcel Service", Code: "ups", APIEndpoint: "https://api.ups.com/track", Active: true},
		{Name: "United States Postal Service", Code: "usps", APIEndpoint: "https://api.usps.com/track", Active: true},
		{Name: "FedEx", Code: "fedex", APIEndpoint: "https://api.fedex.com/track", Active: true},
		{Name: "DHL", Code: "dhl", APIEndpoint: "https://api.dhl.com/track", Active: false},
	}

	for _, carrier := range carriers {
		insertTestCarrier(t, db, carrier)
	}

	return db
}

func TestGetCarriers(t *testing.T) {
	t.Run("AllCarriers", func(t *testing.T) {
		db := setupCarrierTestDB(t)
		defer teardownTestDB(db)

		handler := NewCarrierHandler(db)

		req := httptest.NewRequest("GET", "/api/carriers", nil)
		w := httptest.NewRecorder()

		handler.GetCarriers(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var carriers []database.Carrier
		if err := json.NewDecoder(w.Body).Decode(&carriers); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(carriers) != 4 {
			t.Errorf("Expected 4 carriers, got %d", len(carriers))
		}

		// Check if UPS is in the list
		found := false
		for _, carrier := range carriers {
			if carrier.Code == "ups" {
				found = true
				if carrier.Name != "United Parcel Service" {
					t.Errorf("Expected UPS name 'United Parcel Service', got '%s'", carrier.Name)
				}
				if !carrier.Active {
					t.Error("Expected UPS to be active")
				}
				break
			}
		}
		if !found {
			t.Error("UPS carrier not found in response")
		}
	})

	t.Run("ActiveCarriersOnly", func(t *testing.T) {
		db := setupCarrierTestDB(t)
		defer teardownTestDB(db)

		handler := NewCarrierHandler(db)

		req := httptest.NewRequest("GET", "/api/carriers?active=true", nil)
		w := httptest.NewRecorder()

		handler.GetCarriers(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var carriers []database.Carrier
		if err := json.NewDecoder(w.Body).Decode(&carriers); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(carriers) != 3 {
			t.Errorf("Expected 3 active carriers, got %d", len(carriers))
		}

		// Verify all returned carriers are active
		for _, carrier := range carriers {
			if !carrier.Active {
				t.Errorf("Expected all carriers to be active, but %s is not", carrier.Name)
			}
		}
	})

	t.Run("EmptyDatabase", func(t *testing.T) {
		db := setupTestDB(t) // Empty database
		defer teardownTestDB(db)

		handler := NewCarrierHandler(db)

		req := httptest.NewRequest("GET", "/api/carriers", nil)
		w := httptest.NewRecorder()

		handler.GetCarriers(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var carriers []database.Carrier
		if err := json.NewDecoder(w.Body).Decode(&carriers); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(carriers) != 0 {
			t.Errorf("Expected 0 carriers, got %d", len(carriers))
		}
	})
}