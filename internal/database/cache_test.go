package database

import (
	"fmt"
	"testing"
	"time"
)

func TestRefreshCacheStore(t *testing.T) {
	// Create temporary database
	dbPath := ":memory:"
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a test shipment first
	shipment := &Shipment{
		TrackingNumber: "TEST123",
		Carrier:        "ups",
		Description:    "Test Package",
		Status:         "pending",
	}
	
	err = db.Shipments.Create(shipment)
	if err != nil {
		t.Fatalf("Failed to create test shipment: %v", err)
	}

	// Create test response
	testResponse := &RefreshResponse{
		ShipmentID:  shipment.ID,
		UpdatedAt:   time.Now(),
		EventsAdded: 2,
		TotalEvents: 5,
		Events: []TrackingEvent{
			{
				ShipmentID:  shipment.ID,
				Timestamp:   time.Now(),
				Location:    "Test Location",
				Status:      "in_transit",
				Description: "Package in transit",
			},
		},
	}

	t.Run("SetAndGet", func(t *testing.T) {
		// Cache miss initially
		cached, err := db.RefreshCache.Get(shipment.ID)
		if err != nil {
			t.Errorf("Expected no error on cache miss, got %v", err)
		}
		if cached != nil {
			t.Error("Expected cache miss, got response")
		}

		// Store in cache
		err = db.RefreshCache.Set(shipment.ID, testResponse, 5*time.Minute)
		if err != nil {
			t.Fatalf("Failed to store in cache: %v", err)
		}

		// Cache hit
		cached, err = db.RefreshCache.Get(shipment.ID)
		if err != nil {
			t.Errorf("Failed to get from cache: %v", err)
		}
		if cached == nil {
			t.Fatal("Expected cache hit, got nil")
		}

		if cached.ShipmentID != testResponse.ShipmentID {
			t.Errorf("Expected shipment ID %d, got %d", testResponse.ShipmentID, cached.ShipmentID)
		}
		if cached.EventsAdded != testResponse.EventsAdded {
			t.Errorf("Expected events added %d, got %d", testResponse.EventsAdded, cached.EventsAdded)
		}
		if cached.TotalEvents != testResponse.TotalEvents {
			t.Errorf("Expected total events %d, got %d", testResponse.TotalEvents, cached.TotalEvents)
		}
	})

	t.Run("Expiration", func(t *testing.T) {
		// Store with very short TTL
		err = db.RefreshCache.Set(shipment.ID, testResponse, 1*time.Millisecond)
		if err != nil {
			t.Fatalf("Failed to store in cache: %v", err)
		}

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		// Should be cache miss due to expiration
		cached, err := db.RefreshCache.Get(shipment.ID)
		if err != nil {
			t.Errorf("Expected no error on expired cache, got %v", err)
		}
		if cached != nil {
			t.Error("Expected cache miss due to expiration, got response")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		// Store in cache
		err = db.RefreshCache.Set(shipment.ID, testResponse, 5*time.Minute)
		if err != nil {
			t.Fatalf("Failed to store in cache: %v", err)
		}

		// Verify it's there
		cached, err := db.RefreshCache.Get(shipment.ID)
		if err != nil || cached == nil {
			t.Fatal("Expected cache hit before delete")
		}

		// Delete
		err = db.RefreshCache.Delete(shipment.ID)
		if err != nil {
			t.Fatalf("Failed to delete from cache: %v", err)
		}

		// Should be cache miss
		cached, err = db.RefreshCache.Get(shipment.ID)
		if err != nil {
			t.Errorf("Expected no error after delete, got %v", err)
		}
		if cached != nil {
			t.Error("Expected cache miss after delete, got response")
		}
	})

	t.Run("LoadAll", func(t *testing.T) {
		// Create additional test shipments
		var shipmentIDs []int
		for i := 1; i <= 3; i++ {
			testShipment := &Shipment{
				TrackingNumber: fmt.Sprintf("TEST%d", i),
				Carrier:        "ups",
				Description:    fmt.Sprintf("Test Package %d", i),
				Status:         "pending",
			}
			err = db.Shipments.Create(testShipment)
			if err != nil {
				t.Fatalf("Failed to create test shipment %d: %v", i, err)
			}
			shipmentIDs = append(shipmentIDs, testShipment.ID)
		}

		// Store multiple entries
		for i, shipmentID := range shipmentIDs {
			response := &RefreshResponse{
				ShipmentID:  shipmentID,
				UpdatedAt:   time.Now(),
				EventsAdded: i + 1,
				TotalEvents: (i + 1) * 2,
				Events:      []TrackingEvent{},
			}
			err = db.RefreshCache.Set(shipmentID, response, 5*time.Minute)
			if err != nil {
				t.Fatalf("Failed to store entry %d: %v", shipmentID, err)
			}
		}

		// Create expired shipment
		expiredShipment := &Shipment{
			TrackingNumber: "EXPIRED",
			Carrier:        "ups",
			Description:    "Expired Package",
			Status:         "pending",
		}
		err = db.Shipments.Create(expiredShipment)
		if err != nil {
			t.Fatalf("Failed to create expired shipment: %v", err)
		}

		// Add an expired entry
		expiredResponse := &RefreshResponse{
			ShipmentID:  expiredShipment.ID,
			UpdatedAt:   time.Now(),
			EventsAdded: 0,
			TotalEvents: 0,
			Events:      []TrackingEvent{},
		}
		err = db.RefreshCache.Set(expiredShipment.ID, expiredResponse, 1*time.Millisecond)
		if err != nil {
			t.Fatalf("Failed to store expired entry: %v", err)
		}

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		// Load all non-expired entries
		cache, err := db.RefreshCache.LoadAll()
		if err != nil {
			t.Fatalf("Failed to load cache: %v", err)
		}

		// Should have 3 entries (not the expired one)
		if len(cache) != 3 {
			t.Errorf("Expected 3 cache entries, got %d", len(cache))
		}

		// Verify entries
		for i, shipmentID := range shipmentIDs {
			if entry, exists := cache[shipmentID]; !exists {
				t.Errorf("Expected entry %d to exist", shipmentID)
			} else if entry.EventsAdded != i+1 {
				t.Errorf("Expected events added %d for entry %d, got %d", i+1, shipmentID, entry.EventsAdded)
			}
		}

		// Expired entry should not be present
		if _, exists := cache[expiredShipment.ID]; exists {
			t.Error("Expected expired entry to not be loaded")
		}
	})

	t.Run("DeleteExpired", func(t *testing.T) {
		// Create test shipments for this test
		validShipment := &Shipment{
			TrackingNumber: "VALID",
			Carrier:        "ups",
			Description:    "Valid Package",
			Status:         "pending",
		}
		err = db.Shipments.Create(validShipment)
		if err != nil {
			t.Fatalf("Failed to create valid shipment: %v", err)
		}

		expiringShipment := &Shipment{
			TrackingNumber: "EXPIRING",
			Carrier:        "ups",
			Description:    "Expiring Package",
			Status:         "pending",
		}
		err = db.Shipments.Create(expiringShipment)
		if err != nil {
			t.Fatalf("Failed to create expiring shipment: %v", err)
		}

		// Store entries with different expiration times
		err = db.RefreshCache.Set(validShipment.ID, testResponse, 5*time.Minute)  // Valid
		if err != nil {
			t.Fatalf("Failed to store valid entry: %v", err)
		}

		err = db.RefreshCache.Set(expiringShipment.ID, testResponse, 1*time.Millisecond)  // Will expire
		if err != nil {
			t.Fatalf("Failed to store expiring entry: %v", err)
		}

		// Wait for one to expire
		time.Sleep(10 * time.Millisecond)

		// Clean up expired entries
		err = db.RefreshCache.DeleteExpired()
		if err != nil {
			t.Fatalf("Failed to delete expired entries: %v", err)
		}

		// Valid entry should still be there
		cached, err := db.RefreshCache.Get(validShipment.ID)
		if err != nil || cached == nil {
			t.Error("Expected valid entry to remain after cleanup")
		}

		// Expired entry should be gone
		cached, err = db.RefreshCache.Get(expiringShipment.ID)
		if err != nil {
			t.Errorf("Unexpected error checking expired entry: %v", err)
		}
		if cached != nil {
			t.Error("Expected expired entry to be cleaned up")
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		// Clear cache first
		_, err := db.Exec("DELETE FROM refresh_cache")
		if err != nil {
			t.Fatalf("Failed to clear cache: %v", err)
		}

		// Create test shipments for stats test
		statsShipment1 := &Shipment{
			TrackingNumber: "STATS1",
			Carrier:        "ups",
			Description:    "Stats Package 1",
			Status:         "pending",
		}
		err = db.Shipments.Create(statsShipment1)
		if err != nil {
			t.Fatalf("Failed to create stats shipment 1: %v", err)
		}

		statsShipment2 := &Shipment{
			TrackingNumber: "STATS2",
			Carrier:        "ups",
			Description:    "Stats Package 2",
			Status:         "pending",
		}
		err = db.Shipments.Create(statsShipment2)
		if err != nil {
			t.Fatalf("Failed to create stats shipment 2: %v", err)
		}

		// Store some entries
		err = db.RefreshCache.Set(statsShipment1.ID, testResponse, 5*time.Minute)  // Valid
		if err != nil {
			t.Fatalf("Failed to store entry: %v", err)
		}

		err = db.RefreshCache.Set(statsShipment2.ID, testResponse, 1*time.Millisecond)  // Will expire
		if err != nil {
			t.Fatalf("Failed to store entry: %v", err)
		}

		// Wait for one to expire
		time.Sleep(10 * time.Millisecond)

		total, expired, err := db.RefreshCache.GetStats()
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		if total != 2 {
			t.Errorf("Expected 2 total entries, got %d", total)
		}
		if expired != 1 {
			t.Errorf("Expected 1 expired entry, got %d", expired)
		}
	})
}