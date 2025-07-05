package database

import (
	"testing"
)

func TestAmazonFieldsMigration(t *testing.T) {
	db := setupTestDB(t)
	
	// After migration, check if Amazon fields exist in the schema
	var columnExists int
	
	// Check amazon_order_number column
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('shipments') 
		WHERE name = 'amazon_order_number'
	`).Scan(&columnExists)
	if err != nil {
		t.Fatalf("Failed to check amazon_order_number column: %v", err)
	}
	if columnExists != 1 {
		t.Error("amazon_order_number column should exist after migration")
	}
	
	// Check delegated_carrier column
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('shipments') 
		WHERE name = 'delegated_carrier'
	`).Scan(&columnExists)
	if err != nil {
		t.Fatalf("Failed to check delegated_carrier column: %v", err)
	}
	if columnExists != 1 {
		t.Error("delegated_carrier column should exist after migration")
	}
	
	// Check delegated_tracking_number column
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('shipments') 
		WHERE name = 'delegated_tracking_number'
	`).Scan(&columnExists)
	if err != nil {
		t.Fatalf("Failed to check delegated_tracking_number column: %v", err)
	}
	if columnExists != 1 {
		t.Error("delegated_tracking_number column should exist after migration")
	}
	
	// Check is_amazon_logistics column
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('shipments') 
		WHERE name = 'is_amazon_logistics'
	`).Scan(&columnExists)
	if err != nil {
		t.Fatalf("Failed to check is_amazon_logistics column: %v", err)
	}
	if columnExists != 1 {
		t.Error("is_amazon_logistics column should exist after migration")
	}
	
	// Check that Amazon carrier was added to default carriers
	var carrierExists int
	err = db.QueryRow("SELECT COUNT(*) FROM carriers WHERE code = 'amazon'").Scan(&carrierExists)
	if err != nil {
		t.Fatalf("Failed to check amazon carrier: %v", err)
	}
	if carrierExists != 1 {
		t.Error("Amazon carrier should exist after migration")
	}
	
	// Check if Amazon indexes were created
	var indexExists int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM sqlite_master 
		WHERE type = 'index' AND name = 'idx_shipments_amazon_order'
	`).Scan(&indexExists)
	if err != nil {
		t.Fatalf("Failed to check amazon_order index: %v", err)
	}
	if indexExists != 1 {
		t.Error("idx_shipments_amazon_order index should exist after migration")
	}
	
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM sqlite_master 
		WHERE type = 'index' AND name = 'idx_shipments_delegated_tracking'
	`).Scan(&indexExists)
	if err != nil {
		t.Fatalf("Failed to check delegated tracking index: %v", err)
	}
	if indexExists != 1 {
		t.Error("idx_shipments_delegated_tracking index should exist after migration")
	}
}

func TestAmazonShipmentCreation(t *testing.T) {
	db := setupTestDB(t)
	
	// Test creating Amazon shipment with order number
	amazonOrderNumber := "113-1234567-1234567"
	shipment := Shipment{
		TrackingNumber:    amazonOrderNumber,
		Carrier:           "amazon",
		Description:       "Amazon Package",
		Status:            "pending",
		IsDelivered:       false,
		AmazonOrderNumber: &amazonOrderNumber,
		IsAmazonLogistics: false,
	}
	
	err := db.Shipments.Create(&shipment)
	if err != nil {
		t.Fatalf("Failed to create Amazon shipment: %v", err)
	}
	
	// Verify the shipment was created correctly
	if shipment.ID == 0 {
		t.Error("Expected shipment ID to be set after creation")
	}
	
	// Retrieve the shipment and verify Amazon fields
	retrieved, err := db.Shipments.GetByID(shipment.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve Amazon shipment: %v", err)
	}
	
	if retrieved.Carrier != "amazon" {
		t.Errorf("Expected carrier 'amazon', got '%s'", retrieved.Carrier)
	}
	
	if retrieved.AmazonOrderNumber == nil {
		t.Error("Expected AmazonOrderNumber to be set")
	} else if *retrieved.AmazonOrderNumber != amazonOrderNumber {
		t.Errorf("Expected AmazonOrderNumber '%s', got '%s'", amazonOrderNumber, *retrieved.AmazonOrderNumber)
	}
	
	if retrieved.IsAmazonLogistics != false {
		t.Errorf("Expected IsAmazonLogistics to be false, got %v", retrieved.IsAmazonLogistics)
	}
	
	if retrieved.DelegatedCarrier != nil {
		t.Errorf("Expected DelegatedCarrier to be nil, got %v", retrieved.DelegatedCarrier)
	}
	
	if retrieved.DelegatedTrackingNumber != nil {
		t.Errorf("Expected DelegatedTrackingNumber to be nil, got %v", retrieved.DelegatedTrackingNumber)
	}
}

func TestAmazonShipmentWithDelegation(t *testing.T) {
	db := setupTestDB(t)
	
	// Test creating Amazon shipment with delegated carrier
	amazonOrderNumber := "113-1234567-1234567"
	delegatedCarrier := "ups"
	delegatedTrackingNumber := "1Z999AA1234567890"
	
	shipment := Shipment{
		TrackingNumber:           amazonOrderNumber,
		Carrier:                  "amazon",
		Description:              "Amazon Package via UPS",
		Status:                   "pending",
		IsDelivered:              false,
		AmazonOrderNumber:        &amazonOrderNumber,
		DelegatedCarrier:         &delegatedCarrier,
		DelegatedTrackingNumber:  &delegatedTrackingNumber,
		IsAmazonLogistics:        false,
	}
	
	err := db.Shipments.Create(&shipment)
	if err != nil {
		t.Fatalf("Failed to create Amazon shipment with delegation: %v", err)
	}
	
	// Retrieve and verify delegation fields
	retrieved, err := db.Shipments.GetByID(shipment.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve Amazon shipment with delegation: %v", err)
	}
	
	if retrieved.DelegatedCarrier == nil {
		t.Error("Expected DelegatedCarrier to be set")
	} else if *retrieved.DelegatedCarrier != delegatedCarrier {
		t.Errorf("Expected DelegatedCarrier '%s', got '%s'", delegatedCarrier, *retrieved.DelegatedCarrier)
	}
	
	if retrieved.DelegatedTrackingNumber == nil {
		t.Error("Expected DelegatedTrackingNumber to be set")
	} else if *retrieved.DelegatedTrackingNumber != delegatedTrackingNumber {
		t.Errorf("Expected DelegatedTrackingNumber '%s', got '%s'", delegatedTrackingNumber, *retrieved.DelegatedTrackingNumber)
	}
}

func TestAmazonLogisticsShipment(t *testing.T) {
	db := setupTestDB(t)
	
	// Test creating Amazon Logistics shipment
	amzlTrackingNumber := "TBA123456789012"
	amazonOrderNumber := "113-1234567-1234567"
	
	shipment := Shipment{
		TrackingNumber:    amzlTrackingNumber,
		Carrier:           "amazon",
		Description:       "Amazon Logistics Package",
		Status:            "pending",
		IsDelivered:       false,
		AmazonOrderNumber: &amazonOrderNumber,
		IsAmazonLogistics: true,
	}
	
	err := db.Shipments.Create(&shipment)
	if err != nil {
		t.Fatalf("Failed to create Amazon Logistics shipment: %v", err)
	}
	
	// Retrieve and verify AMZL fields
	retrieved, err := db.Shipments.GetByID(shipment.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve Amazon Logistics shipment: %v", err)
	}
	
	if retrieved.IsAmazonLogistics != true {
		t.Errorf("Expected IsAmazonLogistics to be true, got %v", retrieved.IsAmazonLogistics)
	}
	
	if retrieved.TrackingNumber != amzlTrackingNumber {
		t.Errorf("Expected TrackingNumber '%s', got '%s'", amzlTrackingNumber, retrieved.TrackingNumber)
	}
}

func TestAmazonShipmentQueries(t *testing.T) {
	db := setupTestDB(t)
	
	// Create test Amazon shipments
	shipments := []Shipment{
		{
			TrackingNumber:    "113-1234567-1234567",
			Carrier:           "amazon",
			Description:       "Amazon Order 1",
			Status:            "pending",
			IsDelivered:       false,
			AmazonOrderNumber: stringPtr("113-1234567-1234567"),
			IsAmazonLogistics: false,
		},
		{
			TrackingNumber:    "TBA123456789012",
			Carrier:           "amazon",
			Description:       "Amazon Logistics Package",
			Status:            "in_transit",
			IsDelivered:       false,
			AmazonOrderNumber: stringPtr("113-1234567-1234568"),
			IsAmazonLogistics: true,
		},
		{
			TrackingNumber:    "113-1234567-1234569",
			Carrier:           "amazon",
			Description:       "Amazon Order with UPS",
			Status:            "delivered",
			IsDelivered:       true,
			AmazonOrderNumber: stringPtr("113-1234567-1234569"),
			DelegatedCarrier:  stringPtr("ups"),
			DelegatedTrackingNumber: stringPtr("1Z999AA1234567890"),
			IsAmazonLogistics: false,
		},
	}
	
	for i := range shipments {
		err := db.Shipments.Create(&shipments[i])
		if err != nil {
			t.Fatalf("Failed to create test shipment %d: %v", i, err)
		}
	}
	
	// Test GetActiveByCarrier for Amazon
	activeAmazonShipments, err := db.Shipments.GetActiveByCarrier("amazon")
	if err != nil {
		t.Fatalf("GetActiveByCarrier failed: %v", err)
	}
	
	// Should return 2 active Amazon shipments (first two)
	if len(activeAmazonShipments) != 2 {
		t.Errorf("Expected 2 active Amazon shipments, got %d", len(activeAmazonShipments))
	}
	
	// Test GetAll includes Amazon shipments
	allShipments, err := db.Shipments.GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	
	// Count Amazon shipments
	amazonCount := 0
	for _, s := range allShipments {
		if s.Carrier == "amazon" {
			amazonCount++
		}
	}
	
	if amazonCount != 3 {
		t.Errorf("Expected 3 Amazon shipments in GetAll, got %d", amazonCount)
	}
}

func TestAmazonShipmentUpdate(t *testing.T) {
	db := setupTestDB(t)
	
	// Create Amazon shipment
	amazonOrderNumber := "113-1234567-1234567"
	shipment := Shipment{
		TrackingNumber:    amazonOrderNumber,
		Carrier:           "amazon",
		Description:       "Amazon Package",
		Status:            "pending",
		IsDelivered:       false,
		AmazonOrderNumber: &amazonOrderNumber,
		IsAmazonLogistics: false,
	}
	
	err := db.Shipments.Create(&shipment)
	if err != nil {
		t.Fatalf("Failed to create Amazon shipment: %v", err)
	}
	
	// Update with delegation info
	delegatedCarrier := "fedex"
	delegatedTrackingNumber := "1234567890"
	
	shipment.DelegatedCarrier = &delegatedCarrier
	shipment.DelegatedTrackingNumber = &delegatedTrackingNumber
	shipment.Status = "in_transit"
	
	err = db.Shipments.Update(shipment.ID, &shipment)
	if err != nil {
		t.Fatalf("Failed to update Amazon shipment: %v", err)
	}
	
	// Verify updates
	updated, err := db.Shipments.GetByID(shipment.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated Amazon shipment: %v", err)
	}
	
	if updated.Status != "in_transit" {
		t.Errorf("Expected status 'in_transit', got '%s'", updated.Status)
	}
	
	if updated.DelegatedCarrier == nil || *updated.DelegatedCarrier != delegatedCarrier {
		t.Errorf("Expected DelegatedCarrier '%s', got %v", delegatedCarrier, updated.DelegatedCarrier)
	}
	
	if updated.DelegatedTrackingNumber == nil || *updated.DelegatedTrackingNumber != delegatedTrackingNumber {
		t.Errorf("Expected DelegatedTrackingNumber '%s', got %v", delegatedTrackingNumber, updated.DelegatedTrackingNumber)
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}