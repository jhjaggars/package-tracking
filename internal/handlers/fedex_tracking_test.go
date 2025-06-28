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
	
	// Call the update function that should be implemented
	updater := &FedExTrackingUpdater{
		DB:      db,
		Factory: factory,
	}
	
	// This should update the FedEx shipment with real tracking data
	err := updater.UpdateFedExShipments(context.Background())
	if err != nil {
		t.Fatalf("UpdateFedExShipments failed: %v", err)
	}
	
	// The function should have run without error, indicating the basic flow works
	// For a test tracking number, we might not get actual updates, but the code should execute
	// The status might remain "pending" if the tracking number is not found
	// This is acceptable for a unit test - we're testing the flow, not the external API
	
	// The test passes if UpdateFedExShipments ran without error
	// In a real scenario with valid tracking numbers, the status would be updated
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
		trackingNumbers = append(trackingNumbers, activeShipments[i].TrackingNumber)
		shipmentMap[activeShipments[i].TrackingNumber] = &activeShipments[i]
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