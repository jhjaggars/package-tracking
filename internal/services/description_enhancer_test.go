package services

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"package-tracking/internal/carriers"
	"package-tracking/internal/database"
	"package-tracking/internal/parser"
)

func setupTestEnhancer(t *testing.T) (*DescriptionEnhancer, *database.DB) {
	// Create test database
	db := setupTestDB(t)
	
	// Create stores
	shipmentStore := db.Shipments
	emailStore := database.NewEmailStore(db.DB)
	
	// Create basic extractor (no LLM for tests)
	carrierFactory := &carriers.ClientFactory{}
	extractorConfig := &parser.ExtractorConfig{
		EnableLLM:           false,
		MinConfidence:       0.5,
		MaxCandidates:       10,
		UseHybridValidation: true,
		DebugMode:           false,
	}
	extractor := parser.NewTrackingExtractor(carrierFactory, extractorConfig, nil)
	
	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	
	// Create enhancer
	enhancer := NewDescriptionEnhancer(shipmentStore, emailStore, extractor, logger)
	
	return enhancer, db
}

// setupTestDB mimics the one from database package for our test
func setupTestDB(t *testing.T) *database.DB {
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
	
	db, err := database.Open(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	
	t.Cleanup(func() {
		db.Close()
	})
	
	return db
}

func TestDescriptionEnhancer_EnhanceSpecificShipment(t *testing.T) {
	enhancer, db := setupTestEnhancer(t)
	
	// Create a test shipment with poor description
	shipment := &database.Shipment{
		TrackingNumber:     "TEST123456789",
		Carrier:            "amazon",
		Description:        "Package from ",
		Status:             "delivered",
		AutoRefreshEnabled: true,
	}
	
	err := db.Shipments.Create(shipment)
	if err != nil {
		t.Fatalf("Failed to create test shipment: %v", err)
	}
	
	// Create an associated email with tracking number
	emailStore := database.NewEmailStore(db.DB)
	email := &database.EmailBodyEntry{
		GmailMessageID:  "test-msg-123",
		GmailThreadID:   "test-thread-123",
		From:            "orders@amazon.com",
		Subject:         `Shipped: "iPhone 15 Pro 256GB" from Amazon`,
		Date:            time.Now(),
		BodyText:        "Your order has shipped with tracking number TEST123456789",
		TrackingNumbers: `["TEST123456789"]`,
		Status:          "processed",
		ProcessedAt:     time.Now(),
		ScanMethod:      "search",
	}
	
	err = emailStore.CreateOrUpdate(email)
	if err != nil {
		t.Fatalf("Failed to create test email: %v", err)
	}
	
	// Test enhancing the shipment description
	result, err := enhancer.EnhanceSpecificShipment(shipment.ID, false)
	if err != nil {
		t.Fatalf("Failed to enhance shipment description: %v", err)
	}
	
	// Verify the result
	if !result.Success {
		t.Errorf("Expected enhancement to succeed, got error: %s", result.Error)
	}
	
	if result.ShipmentID != shipment.ID {
		t.Errorf("Expected shipment ID %d, got %d", shipment.ID, result.ShipmentID)
	}
	
	if result.TrackingNumber != "TEST123456789" {
		t.Errorf("Expected tracking number TEST123456789, got %s", result.TrackingNumber)
	}
	
	if result.OldDescription != "Package from " {
		t.Errorf("Expected old description 'Package from ', got '%s'", result.OldDescription)
	}
	
	if result.EmailsFound != 1 {
		t.Errorf("Expected 1 email found, got %d", result.EmailsFound)
	}
	
	// The new description should be extracted from the subject line
	if result.NewDescription == "" || result.NewDescription == "Package from " {
		t.Errorf("Expected improved description, got '%s'", result.NewDescription)
	}
	
	// Verify the shipment was actually updated in the database
	updatedShipment, err := db.Shipments.GetByID(shipment.ID)
	if err != nil {
		t.Fatalf("Failed to get updated shipment: %v", err)
	}
	
	if updatedShipment.Description == "Package from " {
		t.Error("Shipment description was not updated in database")
	}
}

func TestDescriptionEnhancer_EnhanceSpecificShipment_DryRun(t *testing.T) {
	enhancer, db := setupTestEnhancer(t)
	
	// Create a test shipment with poor description
	shipment := &database.Shipment{
		TrackingNumber:     "DRYRUN123456789",
		Carrier:            "amazon",
		Description:        "Package from ",
		Status:             "delivered",
		AutoRefreshEnabled: true,
	}
	
	err := db.Shipments.Create(shipment)
	if err != nil {
		t.Fatalf("Failed to create test shipment: %v", err)
	}
	
	// Create an associated email with tracking number
	emailStore := database.NewEmailStore(db.DB)
	email := &database.EmailBodyEntry{
		GmailMessageID:  "dryrun-msg-123",
		GmailThreadID:   "dryrun-thread-123",
		From:            "orders@amazon.com",
		Subject:         `Shipped: "MacBook Pro 14-inch" from Amazon`,
		Date:            time.Now(),
		BodyText:        "Your order has shipped with tracking number DRYRUN123456789",
		TrackingNumbers: `["DRYRUN123456789"]`,
		Status:          "processed",
		ProcessedAt:     time.Now(),
		ScanMethod:      "search",
	}
	
	err = emailStore.CreateOrUpdate(email)
	if err != nil {
		t.Fatalf("Failed to create test email: %v", err)
	}
	
	originalDescription := shipment.Description
	
	// Test dry run enhancement
	result, err := enhancer.EnhanceSpecificShipment(shipment.ID, true)
	if err != nil {
		t.Fatalf("Failed to enhance shipment description in dry run: %v", err)
	}
	
	// Verify the result shows what would happen
	if !result.Success {
		t.Errorf("Expected dry run to succeed, got error: %s", result.Error)
	}
	
	// Verify the shipment was NOT updated in the database
	unchangedShipment, err := db.Shipments.GetByID(shipment.ID)
	if err != nil {
		t.Fatalf("Failed to get shipment after dry run: %v", err)
	}
	
	if unchangedShipment.Description != originalDescription {
		t.Errorf("Dry run should not modify database. Expected '%s', got '%s'", 
			originalDescription, unchangedShipment.Description)
	}
}

func TestDescriptionEnhancer_GetShipmentsWithPoorDescriptions_Integration(t *testing.T) {
	enhancer, db := setupTestEnhancer(t)
	
	// Create multiple shipments with various description qualities
	shipments := []*database.Shipment{
		{
			TrackingNumber:     "GOOD001",
			Carrier:            "ups",
			Description:        "iPhone 15 Pro from Apple Store",
			Status:             "delivered",
			AutoRefreshEnabled: true,
		},
		{
			TrackingNumber:     "POOR001",
			Carrier:            "amazon",
			Description:        "Package from ",
			Status:             "in_transit",
			AutoRefreshEnabled: true,
		},
		{
			TrackingNumber:     "POOR002",
			Carrier:            "fedex",
			Description:        "",
			Status:             "pending",
			AutoRefreshEnabled: true,
		},
	}
	
	// Create all shipments
	for _, shipment := range shipments {
		err := db.Shipments.Create(shipment)
		if err != nil {
			t.Fatalf("Failed to create shipment %s: %v", shipment.TrackingNumber, err)
		}
	}
	
	// Test getting all shipments with poor descriptions (limit 0 = no limit)
	summary, err := enhancer.EnhanceAllShipmentsWithPoorDescriptions(0, true) // dry run
	if err != nil {
		t.Fatalf("Failed to enhance shipments with poor descriptions: %v", err)
	}
	
	// Should find 2 shipments with poor descriptions
	if summary.TotalShipments != 2 {
		t.Errorf("Expected 2 shipments with poor descriptions, got %d", summary.TotalShipments)
	}
	
	// Should have results for each shipment
	if len(summary.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(summary.Results))
	}
	
	// Test with limit
	limitedSummary, err := enhancer.EnhanceAllShipmentsWithPoorDescriptions(1, true)
	if err != nil {
		t.Fatalf("Failed to enhance limited shipments: %v", err)
	}
	
	if limitedSummary.TotalShipments != 1 {
		t.Errorf("Expected 1 shipment with limit, got %d", limitedSummary.TotalShipments)
	}
}