package database

import (
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
	
	// Test case 2: Deduplication - try to create the same event again
	duplicateEvent := TrackingEvent{
		ShipmentID:  shipment.ID,
		Timestamp:   event1.Timestamp, // Same timestamp
		Location:    "Different Location", // Different location shouldn't matter
		Status:      "different_status",   // Different status shouldn't matter
		Description: event1.Description,   // Same description
	}
	
	err = db.TrackingEvents.CreateEvent(&duplicateEvent)
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
	
	// Test case 3: Create event with different timestamp or description
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
	
	// Now we should have 2 events
	events, err = db.TrackingEvents.GetByShipmentID(shipment.ID)
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}
	
	if len(events) != 2 {
		t.Errorf("Expected 2 events after adding different event, got %d", len(events))
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