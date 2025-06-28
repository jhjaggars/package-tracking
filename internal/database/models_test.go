package database

import (
	"os"
	"testing"
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