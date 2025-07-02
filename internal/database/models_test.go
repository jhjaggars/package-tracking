package database

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) *DB {
	// Create temporary file for test database
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()
	
	// Clean up the temp file when test completes
	t.Cleanup(func() {
		os.Remove(tmpfile.Name())
	})
	
	db, err := Open(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	
	t.Cleanup(func() {
		db.Close()
	})
	
	return db
}

func TestShipmentStore_GetActiveByCarrier(t *testing.T) {
	db := setupTestDB(t)
	
	// Create test shipments - mix of active and delivered, different carriers
	testShipments := []Shipment{
		{
			TrackingNumber: "123456789012",
			Carrier:        "fedex",
			Description:    "Active FedEx Package",
			Status:         "in_transit",
			IsDelivered:    false,
		},
		{
			TrackingNumber: "123456789013",
			Carrier:        "fedex",
			Description:    "Delivered FedEx Package",
			Status:         "delivered",
			IsDelivered:    true, // Should not be returned
		},
		{
			TrackingNumber: "123456789014",
			Carrier:        "ups",
			Description:    "Active UPS Package",
			Status:         "pending",
			IsDelivered:    false, // Different carrier, should not be returned
		},
		{
			TrackingNumber: "123456789015",
			Carrier:        "fedex",
			Description:    "Another Active FedEx Package",
			Status:         "pending",
			IsDelivered:    false,
		},
	}
	
	// Create shipments in database
	for i := range testShipments {
		if err := db.Shipments.Create(&testShipments[i]); err != nil {
			t.Fatalf("Failed to create test shipment: %v", err)
		}
	}
	
	// Test GetActiveByCarrier for FedEx
	activeFedExShipments, err := db.Shipments.GetActiveByCarrier("fedex")
	if err != nil {
		t.Fatalf("GetActiveByCarrier failed: %v", err)
	}
	
	// Should return 2 active FedEx shipments (index 0 and 3)
	if len(activeFedExShipments) != 2 {
		t.Errorf("Expected 2 active FedEx shipments, got %d", len(activeFedExShipments))
	}
	
	// Verify the correct shipments were returned
	foundTrackingNumbers := make(map[string]bool)
	for _, shipment := range activeFedExShipments {
		if shipment.Carrier != "fedex" {
			t.Errorf("Expected carrier 'fedex', got '%s'", shipment.Carrier)
		}
		if shipment.IsDelivered {
			t.Errorf("Expected non-delivered shipment, got delivered shipment %s", shipment.TrackingNumber)
		}
		foundTrackingNumbers[shipment.TrackingNumber] = true
	}
	
	// Check that we got the expected tracking numbers
	expectedTrackingNumbers := []string{"123456789012", "123456789015"}
	for _, expected := range expectedTrackingNumbers {
		if !foundTrackingNumbers[expected] {
			t.Errorf("Expected to find tracking number %s in results", expected)
		}
	}
	
	// Test with carrier that has no active shipments
	activeUPSShipments, err := db.Shipments.GetActiveByCarrier("ups")
	if err != nil {
		t.Fatalf("GetActiveByCarrier for UPS failed: %v", err)
	}
	
	// Should return 1 active UPS shipment
	if len(activeUPSShipments) != 1 {
		t.Errorf("Expected 1 active UPS shipment, got %d", len(activeUPSShipments))
	}
	
	// Test with carrier that doesn't exist
	activeDHLShipments, err := db.Shipments.GetActiveByCarrier("dhl")
	if err != nil {
		t.Fatalf("GetActiveByCarrier for DHL failed: %v", err)
	}
	
	// Should return empty slice
	if len(activeDHLShipments) != 0 {
		t.Errorf("Expected 0 active DHL shipments, got %d", len(activeDHLShipments))
	}
}

func TestTrackingEventStore_CreateEvent(t *testing.T) {
	db := setupTestDB(t)
	
	// First create a shipment to associate events with
	shipment := Shipment{
		TrackingNumber: "123456789012",
		Carrier:        "fedex",
		Description:    "Test Package",
		Status:         "pending",
		IsDelivered:    false,
	}
	if err := db.Shipments.Create(&shipment); err != nil {
		t.Fatalf("Failed to create test shipment: %v", err)
	}
	
	// Test case 1: Create new event successfully
	event1 := TrackingEvent{
		ShipmentID:  shipment.ID,
		Timestamp:   time.Now().Add(-2 * time.Hour),
		Location:    "Memphis, TN",
		Status:      "in_transit",
		Description: "Package in transit",
	}
	
	err := db.TrackingEvents.CreateEvent(&event1)
	if err != nil {
		t.Fatalf("Failed to create tracking event: %v", err)
	}
	
	// Verify event was created with ID
	if event1.ID == 0 {
		t.Error("Expected event ID to be set after creation")
	}
	
	// Test case 2: Deduplication - exact duplicate should be prevented
	// Deduplication is based on: shipment_id + timestamp + description ONLY
	exactDuplicate := TrackingEvent{
		ShipmentID:  shipment.ID,         // Same shipment
		Timestamp:   event1.Timestamp,   // Same timestamp
		Location:    "Different Location", // Different location (doesn't affect deduplication)
		Status:      "different_status",   // Different status (doesn't affect deduplication)  
		Description: event1.Description, // Same description
	}
	
	err = db.TrackingEvents.CreateEvent(&exactDuplicate)
	if err != nil {
		t.Fatalf("Deduplication failed, got error: %v", err)
	}
	
	// Verify only one event exists (deduplication worked)
	events, err := db.TrackingEvents.GetByShipmentID(shipment.ID)
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}
	
	if len(events) != 1 {
		t.Errorf("Expected 1 event after deduplication, got %d", len(events))
	}
	
	// Test case 2b: Different location/status with same timestamp/description should be deduplicated
	anotherDuplicate := TrackingEvent{
		ShipmentID:  shipment.ID,
		Timestamp:   event1.Timestamp,   // Same timestamp
		Location:    "Yet Another Location", // Different location again
		Status:      "another_status",   // Different status again
		Description: event1.Description, // Same description
	}
	
	err = db.TrackingEvents.CreateEvent(&anotherDuplicate)
	if err != nil {
		t.Fatalf("Deduplication failed for second duplicate, got error: %v", err)
	}
	
	// Still should be only one event
	events, err = db.TrackingEvents.GetByShipmentID(shipment.ID)
	if err != nil {
		t.Fatalf("Failed to get events after second duplicate: %v", err)
	}
	
	if len(events) != 1 {
		t.Errorf("Expected 1 event after second deduplication, got %d", len(events))
	}
	
	// Test case 2c: Same timestamp but different description should NOT be deduplicated
	differentDescription := TrackingEvent{
		ShipmentID:  shipment.ID,
		Timestamp:   event1.Timestamp,      // Same timestamp
		Location:    "Same location as first", // Location doesn't matter
		Status:      "same_status",          // Status doesn't matter
		Description: "Different description", // Different description - should create new event
	}
	
	err = db.TrackingEvents.CreateEvent(&differentDescription)
	if err != nil {
		t.Fatalf("Failed to create event with different description: %v", err)
	}
	
	// Now should have 2 events (original + different description)
	events, err = db.TrackingEvents.GetByShipmentID(shipment.ID)
	if err != nil {
		t.Fatalf("Failed to get events after different description: %v", err)
	}
	
	if len(events) != 2 {
		t.Errorf("Expected 2 events after different description, got %d", len(events))
	}
	
	// Test case 3: Create event with different timestamp (should create new event)
	event2 := TrackingEvent{
		ShipmentID:  shipment.ID,
		Timestamp:   time.Now().Add(-1 * time.Hour), // Different timestamp
		Location:    "Atlanta, GA",
		Status:      "out_for_delivery",
		Description: "Out for delivery",
	}
	
	err = db.TrackingEvents.CreateEvent(&event2)
	if err != nil {
		t.Fatalf("Failed to create second tracking event: %v", err)
	}
	
	// Now we should have 3 events (original + different description + different timestamp)
	events, err = db.TrackingEvents.GetByShipmentID(shipment.ID)
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}
	
	if len(events) != 3 {
		t.Errorf("Expected 3 events after adding different timestamp, got %d", len(events))
	}
	
	// Test case 4: Create event for non-existent shipment
	invalidEvent := TrackingEvent{
		ShipmentID:  999999, // Non-existent shipment
		Timestamp:   time.Now(),
		Location:    "Nowhere",
		Status:      "unknown",
		Description: "Invalid shipment",
	}
	
	err = db.TrackingEvents.CreateEvent(&invalidEvent)
	if err == nil {
		t.Error("Expected error when creating event for non-existent shipment")
	}
}

func TestTrackingEventStore_CreateEvent_Concurrent(t *testing.T) {
	db := setupTestDB(t)
	
	// Create a shipment
	shipment := Shipment{
		TrackingNumber: "123456789012",
		Carrier:        "fedex",
		Description:    "Test Package",
		Status:         "pending",
		IsDelivered:    false,
	}
	if err := db.Shipments.Create(&shipment); err != nil {
		t.Fatalf("Failed to create test shipment: %v", err)
	}
	
	// Test concurrent creation of the same event
	timestamp := time.Now()
	description := "Concurrent test event"
	
	// Use a wait group to ensure all goroutines start at the same time
	var wg sync.WaitGroup
	var startSignal sync.WaitGroup
	startSignal.Add(1)
	
	concurrency := 10
	errors := make([]error, concurrency)
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			// Wait for start signal
			startSignal.Wait()
			
			event := TrackingEvent{
				ShipmentID:  shipment.ID,
				Timestamp:   timestamp,
				Location:    "Test Location",
				Status:      "test_status",
				Description: description,
			}
			
			errors[index] = db.TrackingEvents.CreateEvent(&event)
		}(i)
	}
	
	// Start all goroutines
	startSignal.Done()
	
	// Wait for all to complete
	wg.Wait()
	
	// All operations should succeed (no errors)
	for i, err := range errors {
		if err != nil {
			t.Errorf("Goroutine %d got error: %v", i, err)
		}
	}
	
	// But only one event should exist
	events, err := db.TrackingEvents.GetByShipmentID(shipment.ID)
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}
	
	if len(events) != 1 {
		t.Errorf("Expected 1 event after concurrent creation, got %d", len(events))
	}
}

// Test our new atomic transaction method for race condition fix
func TestShipmentStore_UpdateShipmentWithAutoRefresh_Success(t *testing.T) {
	db := setupTestDB(t)

	// Create test shipment
	shipment := Shipment{
		TrackingNumber:      "TEST123456",
		Carrier:             "usps",
		Description:         "Test Package",
		Status:              "pending",
		AutoRefreshEnabled:  true,
		AutoRefreshFailCount: 0,
	}

	err := db.Shipments.Create(&shipment)
	if err != nil {
		t.Fatalf("Failed to create test shipment: %v", err)
	}

	originalID := shipment.ID

	// Modify shipment data
	shipment.Status = "in_transit"
	shipment.Description = "Updated Package"

	// Test successful atomic update
	err = db.Shipments.UpdateShipmentWithAutoRefresh(originalID, &shipment, true, "")
	if err != nil {
		t.Fatalf("UpdateShipmentWithAutoRefresh failed: %v", err)
	}

	// Verify shipment data was updated
	updated, err := db.Shipments.GetByID(originalID)
	if err != nil {
		t.Fatalf("Failed to get updated shipment: %v", err)
	}

	if updated.Status != "in_transit" {
		t.Errorf("Expected status 'in_transit', got '%s'", updated.Status)
	}

	if updated.Description != "Updated Package" {
		t.Errorf("Expected description 'Updated Package', got '%s'", updated.Description)
	}

	// Verify auto-refresh tracking was updated atomically
	if updated.AutoRefreshCount != 1 {
		t.Errorf("Expected auto refresh count 1, got %d", updated.AutoRefreshCount)
	}

	if updated.AutoRefreshFailCount != 0 {
		t.Errorf("Expected auto refresh fail count 0, got %d", updated.AutoRefreshFailCount)
	}

	if updated.LastAutoRefresh == nil {
		t.Error("Expected LastAutoRefresh to be set")
	}

	if updated.AutoRefreshError != nil {
		t.Errorf("Expected AutoRefreshError to be nil, got %v", updated.AutoRefreshError)
	}
}

func TestShipmentStore_UpdateShipmentWithAutoRefresh_Failure(t *testing.T) {
	db := setupTestDB(t)

	// Create test shipment
	shipment := Shipment{
		TrackingNumber:      "TEST123456",
		Carrier:             "usps",
		Description:         "Test Package",
		Status:              "pending",
		AutoRefreshEnabled:  true,
		AutoRefreshFailCount: 0,
	}

	err := db.Shipments.Create(&shipment)
	if err != nil {
		t.Fatalf("Failed to create test shipment: %v", err)
	}

	originalID := shipment.ID

	// Test failure case
	errorMsg := "Carrier API temporarily unavailable"
	err = db.Shipments.UpdateShipmentWithAutoRefresh(originalID, &shipment, false, errorMsg)
	if err != nil {
		t.Fatalf("UpdateShipmentWithAutoRefresh failed: %v", err)
	}

	// Verify error tracking was updated
	updated, err := db.Shipments.GetByID(originalID)
	if err != nil {
		t.Fatalf("Failed to get updated shipment: %v", err)
	}

	// Auto refresh count should not change on failure
	if updated.AutoRefreshCount != 0 {
		t.Errorf("Expected auto refresh count 0, got %d", updated.AutoRefreshCount)
	}

	// Fail count should increment
	if updated.AutoRefreshFailCount != 1 {
		t.Errorf("Expected auto refresh fail count 1, got %d", updated.AutoRefreshFailCount)
	}

	// Error message should be recorded
	if updated.AutoRefreshError == nil {
		t.Error("Expected AutoRefreshError to be set")
	} else if *updated.AutoRefreshError != errorMsg {
		t.Errorf("Expected error message '%s', got '%s'", errorMsg, *updated.AutoRefreshError)
	}
}

func TestShipmentStore_UpdateShipmentWithAutoRefresh_AtomicTransaction(t *testing.T) {
	db := setupTestDB(t)

	// Create test shipment
	shipment := Shipment{
		TrackingNumber:      "TEST123456",
		Carrier:             "usps",
		Description:         "Test Package",
		Status:              "pending",
		AutoRefreshEnabled:  true,
		AutoRefreshFailCount: 0,
	}

	err := db.Shipments.Create(&shipment)
	if err != nil {
		t.Fatalf("Failed to create test shipment: %v", err)
	}

	originalID := shipment.ID

	// Test atomic updates to verify transaction consistency
	// The UpdateShipmentWithAutoRefresh method combines shipment updates with auto-refresh tracking
	shipment.Status = "in_transit"
	
	// Perform multiple sequential successful updates 
	expectedCount := 5
	for i := 0; i < expectedCount; i++ {
		// Get current shipment state for each update
		current, err := db.Shipments.GetByID(originalID)
		if err != nil {
			t.Fatalf("Failed to get current shipment for update %d: %v", i, err)
		}
		
		// Modify the current shipment data
		current.Description = fmt.Sprintf("Updated Package %d", i)
		current.Status = "in_transit"
		
		// Update with success=true, which should increment auto_refresh_count
		err = db.Shipments.UpdateShipmentWithAutoRefresh(originalID, current, true, "")
		if err != nil {
			t.Fatalf("Update %d failed: %v", i, err)
		}
	}
	
	// Verify final state after successful updates
	final, err := db.Shipments.GetByID(originalID)
	if err != nil {
		t.Fatalf("Failed to get final shipment: %v", err)
	}
	
	// Auto refresh count should equal number of successful updates
	if final.AutoRefreshCount != expectedCount {
		t.Errorf("Expected auto refresh count %d, got %d", expectedCount, final.AutoRefreshCount)
	}
	
	// Fail count should be 0 since all updates were successful
	if final.AutoRefreshFailCount != 0 {
		t.Errorf("Expected auto refresh fail count 0, got %d", final.AutoRefreshFailCount)
	}
	
	// Test one failure scenario to verify atomic error handling
	current, err := db.Shipments.GetByID(originalID)
	if err != nil {
		t.Fatalf("Failed to get current shipment for error test: %v", err)
	}
	
	// Update with success=false, which should increment fail count
	err = db.Shipments.UpdateShipmentWithAutoRefresh(originalID, current, false, "Test error")
	if err != nil {
		t.Fatalf("Failed update failed: %v", err)
	}
	
	// Verify error tracking was updated atomically
	finalWithError, err := db.Shipments.GetByID(originalID)
	if err != nil {
		t.Fatalf("Failed to get shipment after error: %v", err)
	}
	
	// Success count should remain the same (fail operations don't change it)
	if finalWithError.AutoRefreshCount != expectedCount {
		t.Errorf("Expected auto refresh count %d after error, got %d", expectedCount, finalWithError.AutoRefreshCount)
	}
	
	// Fail count should increment
	if finalWithError.AutoRefreshFailCount != 1 {
		t.Errorf("Expected auto refresh fail count 1 after error, got %d", finalWithError.AutoRefreshFailCount)
	}
	
	// Error message should be set
	if finalWithError.AutoRefreshError == nil || *finalWithError.AutoRefreshError != "Test error" {
		t.Errorf("Expected error message 'Test error', got %v", finalWithError.AutoRefreshError)
	}
	
	t.Logf("Atomicity test: %d successful + 1 failed update resulted in success count %d, fail count %d", 
		expectedCount, finalWithError.AutoRefreshCount, finalWithError.AutoRefreshFailCount)
}