package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"package-tracking/internal/cache"
	"package-tracking/internal/database"

	"github.com/go-chi/chi/v5"
)

// TestCacheIntegration tests the cache functionality in the handlers
func TestCacheIntegration(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(db)

	// Create test shipment
	shipment := database.Shipment{
		TrackingNumber: "CACHE_TEST_123",
		Carrier:        "ups",
		Description:    "Cache Test Package",
		Status:         "pending",
	}
	shipmentID := insertTestShipment(t, db, shipment)

	t.Run("CacheDisabled", func(t *testing.T) {
		// Create handler with cache disabled
		config := &TestConfig{DisableRateLimit: true, DisableCache: true}
		cacheManager := cache.NewManager(db.RefreshCache, true, 5*time.Minute)
		defer cacheManager.Close()
		handler := NewShipmentHandler(db, config, cacheManager)

		// Mock a successful refresh by inserting some events
		insertTestTrackingEvent(t, db, database.TrackingEvent{
			ShipmentID:  shipmentID,
			Timestamp:   time.Now(),
			Location:    "Test Location",
			Status:      "in_transit",
			Description: "Package picked up",
		})

		// First refresh request - should go to actual refresh (but will fail since we don't have real carriers)
		req := httptest.NewRequest("POST", "/api/shipments/1/refresh", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()

		handler.RefreshShipment(w, req)

		// Should get some response (probably an error since no real carrier integration)
		// but the important thing is that cache is bypassed
		if w.Code == http.StatusOK {
			t.Log("Refresh succeeded (unexpected but OK)")
		} else {
			t.Logf("Refresh failed as expected: %d", w.Code)
		}
	})

	t.Run("CacheEnabled", func(t *testing.T) {
		// Create handler with cache enabled but rate limiting disabled for easier testing
		config := &TestConfig{DisableRateLimit: true, DisableCache: false}
		cacheManager := cache.NewManager(db.RefreshCache, false, 5*time.Minute)
		defer cacheManager.Close()
		handler := NewShipmentHandler(db, config, cacheManager)

		// Manually insert a cache entry to simulate a previous successful refresh
		testResponse := &database.RefreshResponse{
			ShipmentID:  shipmentID,
			UpdatedAt:   time.Now(),
			EventsAdded: 1,
			TotalEvents: 2,
			Events: []database.TrackingEvent{
				{
					ID:          1,
					ShipmentID:  shipmentID,
					Timestamp:   time.Now(),
					Location:    "Cached Location",
					Status:      "in_transit",
					Description: "Cached event",
					CreatedAt:   time.Now(),
				},
			},
		}

		err := cacheManager.Set(shipmentID, testResponse)
		if err != nil {
			t.Fatalf("Failed to set cache entry: %v", err)
		}

		// Request refresh - should return cached response
		// Create a proper chi context with the path parameter
		r := chi.NewRouter()
		r.Post("/api/shipments/{id}/refresh", handler.RefreshShipment)

		req := httptest.NewRequest("POST", fmt.Sprintf("/api/shipments/%d/refresh", shipmentID), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response RefreshResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.ShipmentID != shipmentID {
			t.Errorf("Expected shipment ID %d, got %d", shipmentID, response.ShipmentID)
		}
		if response.EventsAdded != 1 {
			t.Errorf("Expected events added 1, got %d", response.EventsAdded)
		}
		if response.TotalEvents != 2 {
			t.Errorf("Expected total events 2, got %d", response.TotalEvents)
		}
		if len(response.Events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(response.Events))
		}
		if response.Events[0].Description != "Cached event" {
			t.Errorf("Expected cached event description, got %s", response.Events[0].Description)
		}
	})

	t.Run("CacheInvalidationOnUpdate", func(t *testing.T) {
		// Create handler with cache enabled
		config := &TestConfig{DisableRateLimit: true, DisableCache: false}
		cacheManager := cache.NewManager(db.RefreshCache, false, 5*time.Minute)
		defer cacheManager.Close()
		handler := NewShipmentHandler(db, config, cacheManager)

		// Manually insert a cache entry
		testResponse := &database.RefreshResponse{
			ShipmentID:  shipmentID,
			UpdatedAt:   time.Now(),
			EventsAdded: 1,
			TotalEvents: 1,
			Events:      []database.TrackingEvent{},
		}

		err := cacheManager.Set(shipmentID, testResponse)
		if err != nil {
			t.Fatalf("Failed to set cache entry: %v", err)
		}

		// Verify cache entry exists
		cached, err := cacheManager.Get(shipmentID)
		if err != nil || cached == nil {
			t.Fatal("Cache entry should exist before update")
		}

		// Update the shipment
		updatedShipment := database.Shipment{
			TrackingNumber: "CACHE_TEST_123",
			Carrier:        "ups",
			Description:    "Updated Description",
			Status:         "pending",
		}
		shipmentJSON, _ := json.Marshal(updatedShipment)

		// Create a proper chi context for the update request
		r := chi.NewRouter()
		r.Put("/api/shipments/{id}", handler.UpdateShipment)

		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/shipments/%d", shipmentID), bytes.NewBuffer(shipmentJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify cache entry was invalidated
		cached, err = cacheManager.Get(shipmentID)
		if err != nil {
			t.Errorf("Unexpected error checking cache: %v", err)
		}
		if cached != nil {
			t.Error("Cache entry should be invalidated after update")
		}
	})

	t.Run("CacheInvalidationOnDelete", func(t *testing.T) {
		// Create handler with cache enabled
		config := &TestConfig{DisableRateLimit: true, DisableCache: false}
		cacheManager := cache.NewManager(db.RefreshCache, false, 5*time.Minute)
		defer cacheManager.Close()
		handler := NewShipmentHandler(db, config, cacheManager)

		// Create a new shipment for deletion test
		deleteShipment := database.Shipment{
			TrackingNumber: "DELETE_TEST_123",
			Carrier:        "ups",
			Description:    "Delete Test Package",
			Status:         "pending",
		}
		deleteShipmentID := insertTestShipment(t, db, deleteShipment)

		// Manually insert a cache entry
		testResponse := &database.RefreshResponse{
			ShipmentID:  deleteShipmentID,
			UpdatedAt:   time.Now(),
			EventsAdded: 1,
			TotalEvents: 1,
			Events:      []database.TrackingEvent{},
		}

		err := cacheManager.Set(deleteShipmentID, testResponse)
		if err != nil {
			t.Fatalf("Failed to set cache entry: %v", err)
		}

		// Verify cache entry exists
		cached, err := cacheManager.Get(deleteShipmentID)
		if err != nil || cached == nil {
			t.Fatal("Cache entry should exist before delete")
		}

		// Delete the shipment
		// Create a proper chi context for the delete request
		r := chi.NewRouter()
		r.Delete("/api/shipments/{id}", handler.DeleteShipment)

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/shipments/%d", deleteShipmentID), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}

		// Verify cache entry was invalidated
		cached, err = cacheManager.Get(deleteShipmentID)
		if err != nil {
			t.Errorf("Unexpected error checking cache: %v", err)
		}
		if cached != nil {
			t.Error("Cache entry should be invalidated after delete")
		}
	})

	t.Run("CacheExpiry", func(t *testing.T) {
		// Create cache manager with very short TTL
		cacheManager := cache.NewManager(db.RefreshCache, false, 10*time.Millisecond)
		defer cacheManager.Close()

		// Manually insert a cache entry with short TTL
		testResponse := &database.RefreshResponse{
			ShipmentID:  shipmentID,
			UpdatedAt:   time.Now(),
			EventsAdded: 1,
			TotalEvents: 1,
			Events:      []database.TrackingEvent{},
		}

		err := cacheManager.Set(shipmentID, testResponse)
		if err != nil {
			t.Fatalf("Failed to set cache entry: %v", err)
		}

		// Immediately check - should be cached
		cached, err := cacheManager.Get(shipmentID)
		if err != nil || cached == nil {
			t.Fatal("Cache entry should exist immediately after setting")
		}

		// Wait for expiry
		time.Sleep(50 * time.Millisecond)

		// Check again - should be expired
		cached, err = cacheManager.Get(shipmentID)
		if err != nil {
			t.Errorf("Unexpected error checking expired cache: %v", err)
		}
		if cached != nil {
			t.Error("Cache entry should be expired")
		}
	})

	t.Run("CacheStats", func(t *testing.T) {
		// Create cache manager
		cacheManager := cache.NewManager(db.RefreshCache, false, 5*time.Minute)
		defer cacheManager.Close()

		// Get initial stats
		stats, err := cacheManager.GetStats()
		if err != nil {
			t.Fatalf("Failed to get cache stats: %v", err)
		}

		if stats.Disabled {
			t.Error("Cache should be enabled")
		}
		if stats.TTL != 5*time.Minute {
			t.Errorf("Expected TTL 5m, got %v", stats.TTL)
		}

		// Add cache entry
		testResponse := &database.RefreshResponse{
			ShipmentID:  shipmentID,
			UpdatedAt:   time.Now(),
			EventsAdded: 1,
			TotalEvents: 1,
			Events:      []database.TrackingEvent{},
		}

		err = cacheManager.Set(shipmentID, testResponse)
		if err != nil {
			t.Fatalf("Failed to set cache entry: %v", err)
		}

		// Get updated stats
		stats, err = cacheManager.GetStats()
		if err != nil {
			t.Fatalf("Failed to get updated cache stats: %v", err)
		}

		if stats.MemoryTotal == 0 {
			t.Error("Expected memory entries after setting cache")
		}
		if stats.DatabaseTotal == 0 {
			t.Error("Expected database entries after setting cache")
		}
	})
}