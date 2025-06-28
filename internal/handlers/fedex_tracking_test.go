package handlers

import (
	"context"
	"testing"

	"package-tracking/internal/carriers"
	"package-tracking/internal/database"
)

// TestFedExAutomaticTrackingUpdate tests the automatic tracking update functionality
// This is an integration test that verifies FedEx shipments can be automatically updated
func TestFedExAutomaticTrackingUpdate(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	
	// Create a test FedEx shipment that needs updating
	testShipment := database.Shipment{
		TrackingNumber: "123456789012",
		Carrier:        "fedex",
		Description:    "Test FedEx Package",
		Status:         "pending",
		IsDelivered:    false,
	}
	
	if err := db.Shipments.Create(&testShipment); err != nil {
		t.Fatalf("Failed to create test shipment: %v", err)
	}
	
	// Create FedEx client factory
	factory := carriers.NewClientFactory()
	
	// Create and validate the updater
	updater := &FedExTrackingUpdater{
		DB:      db,
		Factory: factory,
	}
	
	// Validate that the updater is properly initialized
	if updater.DB == nil {
		t.Fatal("DB not properly initialized")
	}
	if updater.Factory == nil {
		t.Fatal("Factory not properly initialized")
	}
	
	// This should update the FedEx shipment with real tracking data
	err := updater.UpdateFedExShipments(context.Background())
	if err != nil {
		t.Fatalf("UpdateFedExShipments failed: %v", err)
	}
	
	// Verify the shipment state after update attempt
	updatedShipment, err := db.Shipments.GetByID(testShipment.ID)
	if err != nil {
		t.Fatalf("Failed to get updated shipment: %v", err)
	}
	
	// The function should have run without error, indicating the basic flow works
	// For a test tracking number with web scraping, we expect either:
	// 1. Status remains "pending" if tracking number is not found (web scraping fails)
	// 2. Status is updated if web scraping succeeds
	
	// Verify shipment was not corrupted
	if updatedShipment.TrackingNumber != testShipment.TrackingNumber {
		t.Errorf("Tracking number changed unexpectedly: %s -> %s", 
			testShipment.TrackingNumber, updatedShipment.TrackingNumber)
	}
	
	if updatedShipment.Carrier != testShipment.Carrier {
		t.Errorf("Carrier changed unexpectedly: %s -> %s", 
			testShipment.Carrier, updatedShipment.Carrier)
	}
	
	// Check if any tracking events were created (they might not be for invalid tracking numbers)
	events, err := db.TrackingEvents.GetByShipmentID(testShipment.ID)
	if err != nil {
		t.Fatalf("Failed to get tracking events: %v", err)
	}
	
	// For a test tracking number, we might get 0 events (tracking not found)
	// This is acceptable - we're testing the flow, not the external API
	t.Logf("Created %d tracking events for test shipment", len(events))
}


// FedExTrackingUpdater is the component that will update FedEx shipments
type FedExTrackingUpdater struct {
	DB      *database.DB
	Factory *carriers.ClientFactory
}

// UpdateFedExShipments updates all active FedEx shipments with latest tracking data
func (u *FedExTrackingUpdater) UpdateFedExShipments(ctx context.Context) error {
	// 1. Get all active FedEx shipments from database
	activeShipments, err := u.DB.Shipments.GetActiveByCarrier("fedex")
	if err != nil {
		return err
	}
	
	if len(activeShipments) == 0 {
		return nil // No active FedEx shipments to update
	}
	
	// 2. Create FedEx client using factory
	client, _, err := u.Factory.CreateClient("fedex")
	if err != nil {
		return err
	}
	
	// 3. Call FedEx API to get tracking information
	var trackingNumbers []string
	shipmentMap := make(map[string]*database.Shipment)
	
	for i := range activeShipments {
		trackingNumber := activeShipments[i].TrackingNumber
		
		// Validate tracking number format before adding to request
		if !client.ValidateTrackingNumber(trackingNumber) {
			// Log invalid tracking number but continue processing others
			continue
		}
		
		trackingNumbers = append(trackingNumbers, trackingNumber)
		shipmentMap[trackingNumber] = &activeShipments[i]
	}
	
	// Skip API call if no valid tracking numbers
	if len(trackingNumbers) == 0 {
		return nil // No valid tracking numbers to process
	}
	
	req := &carriers.TrackingRequest{
		TrackingNumbers: trackingNumbers,
	}
	
	resp, err := client.Track(ctx, req)
	if err != nil {
		return err
	}
	
	// 4. Update shipments and create tracking events
	for _, result := range resp.Results {
		shipment := shipmentMap[result.TrackingNumber]
		if shipment == nil {
			continue
		}
		
		// Update shipment status if it changed
		newStatus := string(result.Status)
		if shipment.Status != newStatus {
			shipment.Status = newStatus
			shipment.IsDelivered = (result.Status == carriers.StatusDelivered)
			
			if err := u.DB.Shipments.Update(shipment.ID, shipment); err != nil {
				return err
			}
		}
		
		// Create tracking events
		for _, event := range result.Events {
			dbEvent := database.TrackingEvent{
				ShipmentID:  shipment.ID,
				Timestamp:   event.Timestamp,
				Location:    event.Location,
				Status:      string(event.Status),
				Description: event.Description,
			}
			
			if err := u.DB.TrackingEvents.CreateEvent(&dbEvent); err != nil {
				return err
			}
		}
	}
	
	return nil
}

func TestFedExAutomaticTrackingUpdate_NoActiveShipments(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	
	// Create FedEx client factory (no active shipments in database)
	factory := carriers.NewClientFactory()
	
	updater := &FedExTrackingUpdater{
		DB:      db,
		Factory: factory,
	}
	
	// Should handle empty shipments gracefully
	err := updater.UpdateFedExShipments(context.Background())
	if err != nil {
		t.Fatalf("UpdateFedExShipments should handle no active shipments gracefully: %v", err)
	}
}

func TestFedExAutomaticTrackingUpdate_InvalidTrackingNumbers(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	
	// Create a shipment with invalid tracking number
	invalidShipment := database.Shipment{
		TrackingNumber: "INVALID123", // Invalid FedEx format
		Carrier:        "fedex",
		Description:    "Invalid tracking number test",
		Status:         "pending",
		IsDelivered:    false,
	}
	
	if err := db.Shipments.Create(&invalidShipment); err != nil {
		t.Fatalf("Failed to create test shipment: %v", err)
	}
	
	factory := carriers.NewClientFactory()
	
	updater := &FedExTrackingUpdater{
		DB:      db,
		Factory: factory,
	}
	
	// Should handle invalid tracking numbers gracefully
	err := updater.UpdateFedExShipments(context.Background())
	if err != nil {
		t.Fatalf("UpdateFedExShipments should handle invalid tracking numbers gracefully: %v", err)
	}
	
	// Verify shipment was not corrupted
	updatedShipment, err := db.Shipments.GetByID(invalidShipment.ID)
	if err != nil {
		t.Fatalf("Failed to get updated shipment: %v", err)
	}
	
	if updatedShipment.Status != "pending" {
		t.Errorf("Expected status to remain 'pending' for invalid tracking number, got '%s'", updatedShipment.Status)
	}
}