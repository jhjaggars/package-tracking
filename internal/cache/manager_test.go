package cache

import (
	"testing"
	"time"

	"package-tracking/internal/database"
)

func TestCacheManager(t *testing.T) {
	// Create temporary database
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create a test shipment
	shipment := &database.Shipment{
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
	testResponse := &database.RefreshResponse{
		ShipmentID:  shipment.ID,
		UpdatedAt:   time.Now(),
		EventsAdded: 2,
		TotalEvents: 5,
		Events: []database.TrackingEvent{
			{
				ShipmentID:  shipment.ID,
				Timestamp:   time.Now(),
				Location:    "Test Location",
				Status:      "in_transit",
				Description: "Package in transit",
			},
		},
	}

	t.Run("EnabledCache", func(t *testing.T) {
		manager := NewManager(db.RefreshCache, false, 5*time.Minute)
		defer manager.Close()

		// Cache miss initially
		cached, err := manager.Get(shipment.ID)
		if err != nil {
			t.Errorf("Expected no error on cache miss, got %v", err)
		}
		if cached != nil {
			t.Error("Expected cache miss, got response")
		}

		// Store in cache
		err = manager.Set(shipment.ID, testResponse)
		if err != nil {
			t.Fatalf("Failed to store in cache: %v", err)
		}

		// Cache hit
		cached, err = manager.Get(shipment.ID)
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
	})

	t.Run("DisabledCache", func(t *testing.T) {
		manager := NewManager(db.RefreshCache, true, 5*time.Minute)
		defer manager.Close()

		// Cache should always miss when disabled
		cached, err := manager.Get(shipment.ID)
		if err != nil {
			t.Errorf("Expected no error when cache disabled, got %v", err)
		}
		if cached != nil {
			t.Error("Expected cache miss when disabled, got response")
		}

		// Store should be no-op when disabled
		err = manager.Set(shipment.ID, testResponse)
		if err != nil {
			t.Errorf("Expected no error when cache disabled, got %v", err)
		}

		// Should still be cache miss
		cached, err = manager.Get(shipment.ID)
		if err != nil {
			t.Errorf("Expected no error when cache disabled, got %v", err)
		}
		if cached != nil {
			t.Error("Expected cache miss when disabled, got response")
		}
	})

	t.Run("MemoryAndDatabaseSync", func(t *testing.T) {
		manager := NewManager(db.RefreshCache, false, 5*time.Minute)
		defer manager.Close()

		// Clear any existing cache
		err = manager.Delete(shipment.ID)
		if err != nil {
			t.Fatalf("Failed to clear cache: %v", err)
		}

		// Store in cache
		err = manager.Set(shipment.ID, testResponse)
		if err != nil {
			t.Fatalf("Failed to store in cache: %v", err)
		}

		// Create new manager to test database persistence
		manager2 := NewManager(db.RefreshCache, false, 5*time.Minute)
		defer manager2.Close()

		// Should load from database
		cached, err := manager2.Get(shipment.ID)
		if err != nil {
			t.Errorf("Failed to get from cache: %v", err)
		}
		if cached == nil {
			t.Fatal("Expected cache hit from database, got nil")
		}

		if cached.ShipmentID != testResponse.ShipmentID {
			t.Errorf("Expected shipment ID %d, got %d", testResponse.ShipmentID, cached.ShipmentID)
		}
	})

	t.Run("Expiration", func(t *testing.T) {
		manager := NewManager(db.RefreshCache, false, 10*time.Millisecond)
		defer manager.Close()

		// Store in cache with short TTL
		err = manager.Set(shipment.ID, testResponse)
		if err != nil {
			t.Fatalf("Failed to store in cache: %v", err)
		}

		// Should be cache hit initially
		cached, err := manager.Get(shipment.ID)
		if err != nil {
			t.Errorf("Failed to get from cache: %v", err)
		}
		if cached == nil {
			t.Fatal("Expected cache hit, got nil")
		}

		// Wait for expiration
		time.Sleep(50 * time.Millisecond)

		// Should be cache miss due to expiration
		cached, err = manager.Get(shipment.ID)
		if err != nil {
			t.Errorf("Expected no error on expired cache, got %v", err)
		}
		if cached != nil {
			t.Error("Expected cache miss due to expiration, got response")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		manager := NewManager(db.RefreshCache, false, 5*time.Minute)
		defer manager.Close()

		// Store in cache
		err = manager.Set(shipment.ID, testResponse)
		if err != nil {
			t.Fatalf("Failed to store in cache: %v", err)
		}

		// Verify it's there
		cached, err := manager.Get(shipment.ID)
		if err != nil || cached == nil {
			t.Fatal("Expected cache hit before delete")
		}

		// Delete
		err = manager.Delete(shipment.ID)
		if err != nil {
			t.Fatalf("Failed to delete from cache: %v", err)
		}

		// Should be cache miss
		cached, err = manager.Get(shipment.ID)
		if err != nil {
			t.Errorf("Expected no error after delete, got %v", err)
		}
		if cached != nil {
			t.Error("Expected cache miss after delete, got response")
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		manager := NewManager(db.RefreshCache, false, 5*time.Minute)
		defer manager.Close()

		// Clear cache first
		err = manager.Delete(shipment.ID)
		if err != nil {
			t.Fatalf("Failed to clear cache: %v", err)
		}

		// Get initial stats
		stats, err := manager.GetStats()
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		if stats.Disabled {
			t.Error("Expected cache to be enabled")
		}
		if stats.TTL != 5*time.Minute {
			t.Errorf("Expected TTL 5m, got %v", stats.TTL)
		}

		// Store some entries
		err = manager.Set(shipment.ID, testResponse)
		if err != nil {
			t.Fatalf("Failed to store in cache: %v", err)
		}

		// Get updated stats
		stats, err = manager.GetStats()
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		if stats.MemoryTotal == 0 {
			t.Error("Expected memory entries after storing")
		}
		if stats.DatabaseTotal == 0 {
			t.Error("Expected database entries after storing")
		}
	})

	t.Run("DisabledStats", func(t *testing.T) {
		manager := NewManager(db.RefreshCache, true, 5*time.Minute)
		defer manager.Close()

		stats, err := manager.GetStats()
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		if !stats.Disabled {
			t.Error("Expected cache to be disabled")
		}
		if stats.MemoryTotal != 0 {
			t.Error("Expected no memory entries when disabled")
		}
	})
}

func TestCachedResponse(t *testing.T) {
	t.Run("IsExpired", func(t *testing.T) {
		// Not expired
		cached := &CachedResponse{
			Response:  nil,
			ExpiresAt: time.Now().Add(1 * time.Minute),
		}
		if cached.IsExpired() {
			t.Error("Expected not expired")
		}

		// Expired
		cached.ExpiresAt = time.Now().Add(-1 * time.Minute)
		if !cached.IsExpired() {
			t.Error("Expected expired")
		}
	})
}