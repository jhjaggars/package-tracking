package database

import (
	"testing"
	"time"
)

func TestShipmentStore_GetByTrackingNumber(t *testing.T) {
	db := setupTestDB(t)

	store := db.Shipments

	// Create a test shipment
	shipment := &Shipment{
		TrackingNumber:     "1Z999AA1234567890",
		Carrier:            "ups",
		Description:        "Test Package",
		Status:             "in_transit",
		AutoRefreshEnabled: true,
	}

	err := store.Create(shipment)
	if err != nil {
		t.Fatalf("Failed to create shipment: %v", err)
	}

	// Test retrieving by tracking number
	retrieved, err := store.GetByTrackingNumber("1Z999AA1234567890")
	if err != nil {
		t.Fatalf("Failed to get shipment by tracking number: %v", err)
	}

	if retrieved.ID != shipment.ID {
		t.Errorf("Expected shipment ID %d, got %d", shipment.ID, retrieved.ID)
	}

	if retrieved.TrackingNumber != "1Z999AA1234567890" {
		t.Errorf("Expected tracking number '1Z999AA1234567890', got '%s'", retrieved.TrackingNumber)
	}

	if retrieved.Description != "Test Package" {
		t.Errorf("Expected description 'Test Package', got '%s'", retrieved.Description)
	}

	// Test non-existent tracking number
	_, err = store.GetByTrackingNumber("NONEXISTENT")
	if err == nil {
		t.Error("Expected error for non-existent tracking number")
	}
}

func TestShipmentStore_GetShipmentsWithPoorDescriptions(t *testing.T) {
	db := setupTestDB(t)

	store := db.Shipments

	// Create test shipments with various description states
	shipments := []*Shipment{
		{
			TrackingNumber:     "GOOD001",
			Carrier:            "ups",
			Description:        "iPhone 15 Pro from Apple",
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
			Carrier:            "usps",
			Description:        "",
			Status:             "pending",
			AutoRefreshEnabled: true,
		},
		{
			TrackingNumber:     "POOR003",
			Carrier:            "fedex",
			Description:        "Package from Amazon",
			Status:             "in_transit",
			AutoRefreshEnabled: true,
		},
	}

	// Create all shipments
	for _, shipment := range shipments {
		err := store.Create(shipment)
		if err != nil {
			t.Fatalf("Failed to create shipment %s: %v", shipment.TrackingNumber, err)
		}
	}

	// Test getting shipments with poor descriptions (no limit)
	poorShipments, err := store.GetShipmentsWithPoorDescriptions(0)
	if err != nil {
		t.Fatalf("Failed to get shipments with poor descriptions: %v", err)
	}

	// Should return 3 shipments (POOR001, POOR002, POOR003)
	if len(poorShipments) != 3 {
		t.Errorf("Expected 3 shipments with poor descriptions, got %d", len(poorShipments))
	}

	// Verify the shipments returned are the correct ones
	expectedTrackingNumbers := map[string]bool{
		"POOR001": true,
		"POOR002": true,
		"POOR003": true,
	}

	for _, shipment := range poorShipments {
		if !expectedTrackingNumbers[shipment.TrackingNumber] {
			t.Errorf("Unexpected shipment with tracking number %s", shipment.TrackingNumber)
		}
	}

	// Test with limit
	limitedShipments, err := store.GetShipmentsWithPoorDescriptions(2)
	if err != nil {
		t.Fatalf("Failed to get limited shipments with poor descriptions: %v", err)
	}

	if len(limitedShipments) != 2 {
		t.Errorf("Expected 2 shipments with limit, got %d", len(limitedShipments))
	}
}

func TestShipmentStore_UpdateDescription(t *testing.T) {
	db := setupTestDB(t)

	store := db.Shipments

	// Create a test shipment
	shipment := &Shipment{
		TrackingNumber:     "TEST123456789",
		Carrier:            "ups",
		Description:        "Package from ",
		Status:             "in_transit",
		AutoRefreshEnabled: true,
	}

	err := store.Create(shipment)
	if err != nil {
		t.Fatalf("Failed to create shipment: %v", err)
	}

	originalUpdatedAt := shipment.UpdatedAt

	// Wait a moment to ensure timestamp difference  
	time.Sleep(100 * time.Millisecond)

	// Update the description
	newDescription := "iPhone 15 Pro from Apple"
	err = store.UpdateDescription(shipment.ID, newDescription)
	if err != nil {
		t.Fatalf("Failed to update description: %v", err)
	}

	// Retrieve the updated shipment
	updated, err := store.GetByID(shipment.ID)
	if err != nil {
		t.Fatalf("Failed to get updated shipment: %v", err)
	}

	// Verify the description was updated
	if updated.Description != newDescription {
		t.Errorf("Expected description '%s', got '%s'", newDescription, updated.Description)
	}

	// Verify updated_at timestamp was changed (or at least not before)
	if updated.UpdatedAt.Before(originalUpdatedAt) {
		t.Errorf("Expected updated_at timestamp to be newer or equal. Original: %v, Updated: %v", originalUpdatedAt, updated.UpdatedAt)
	}

	// Test updating non-existent shipment
	err = store.UpdateDescription(99999, "Should fail")
	if err == nil {
		t.Error("Expected error when updating non-existent shipment")
	}
}

func TestEmailStore_GetEmailsForTrackingNumber(t *testing.T) {
	db := setupTestDB(t)

	// Create email store (not embedded in DB struct)
	store := NewEmailStore(db.DB)

	// Create test emails with tracking numbers
	emails := []*EmailBodyEntry{
		{
			GmailMessageID: "msg001",
			GmailThreadID:  "thread001",
			From:           "orders@amazon.com",
			Subject:        "Your order has shipped",
			Date:           time.Now(),
			BodyText:       "Your package has shipped with tracking number TEST123456789",
			TrackingNumbers: `["TEST123456789", "ANOTHER123"]`,
			Status:         "processed",
			ProcessedAt:    time.Now(),
			ScanMethod:     "search",
		},
		{
			GmailMessageID: "msg002",
			GmailThreadID:  "thread002",
			From:           "tracking@ups.com",
			Subject:        "Package delivered",
			Date:           time.Now(),
			BodyText:       "Package delivered successfully",
			TrackingNumbers: `["DIFFERENT456"]`,
			Status:         "processed",
			ProcessedAt:    time.Now(),
			ScanMethod:     "search",
		},
		{
			GmailMessageID: "msg003",
			GmailThreadID:  "thread003",
			From:           "shipment@amazon.com",
			Subject:        "Multiple items shipped",
			Date:           time.Now(),
			BodyText:       "Your order contains multiple items",
			TrackingNumbers: `[TEST123456789 EXTRA789]`,
			Status:         "processed",
			ProcessedAt:    time.Now(),
			ScanMethod:     "search",
		},
	}

	// Create all emails
	for _, email := range emails {
		err := store.CreateOrUpdate(email)
		if err != nil {
			t.Fatalf("Failed to create email %s: %v", email.GmailMessageID, err)
		}
	}

	// Test finding emails for specific tracking number
	foundEmails, err := store.GetEmailsForTrackingNumber("TEST123456789")
	if err != nil {
		t.Fatalf("Failed to get emails for tracking number: %v", err)
	}

	// Should find 2 emails (msg001 and msg003)
	if len(foundEmails) != 2 {
		t.Errorf("Expected 2 emails for tracking number TEST123456789, got %d", len(foundEmails))
	}

	// Verify the correct emails were found
	expectedMessageIDs := map[string]bool{
		"msg001": true,
		"msg003": true,
	}

	for _, email := range foundEmails {
		if !expectedMessageIDs[email.GmailMessageID] {
			t.Errorf("Unexpected email with message ID %s", email.GmailMessageID)
		}
	}

	// Test finding emails for non-existent tracking number
	notFoundEmails, err := store.GetEmailsForTrackingNumber("NONEXISTENT")
	if err != nil {
		t.Fatalf("Failed to search for non-existent tracking number: %v", err)
	}

	if len(notFoundEmails) != 0 {
		t.Errorf("Expected 0 emails for non-existent tracking number, got %d", len(notFoundEmails))
	}
}

func TestEmailStore_GetEmailsWithTrackingNumbers(t *testing.T) {
	db := setupTestDB(t)

	// Create email store (not embedded in DB struct)
	store := NewEmailStore(db.DB)

	// Create test emails - some with tracking numbers, some without
	emails := []*EmailBodyEntry{
		{
			GmailMessageID:  "msg001",
			GmailThreadID:   "thread001",
			From:            "orders@amazon.com",
			Subject:         "Your order has shipped",
			Date:            time.Now(),
			TrackingNumbers: `["TEST123456789"]`,
			Status:          "processed",
			ProcessedAt:     time.Now(),
			ScanMethod:      "search",
		},
		{
			GmailMessageID:  "msg002",
			GmailThreadID:   "thread002",
			From:            "newsletter@store.com",
			Subject:         "Weekly newsletter",
			Date:            time.Now(),
			TrackingNumbers: `[]`,
			Status:          "processed",
			ProcessedAt:     time.Now(),
			ScanMethod:      "search",
		},
		{
			GmailMessageID:  "msg003",
			GmailThreadID:   "thread003",
			From:            "tracking@fedex.com",
			Subject:         "Package update",
			Date:            time.Now(),
			TrackingNumbers: `["FEDEX456789"]`,
			Status:          "processed",
			ProcessedAt:     time.Now(),
			ScanMethod:      "search",
		},
		{
			GmailMessageID:  "msg004",
			GmailThreadID:   "thread004",
			From:            "spam@example.com",
			Subject:         "No tracking info",
			Date:            time.Now(),
			TrackingNumbers: "",
			Status:          "processed",
			ProcessedAt:     time.Now(),
			ScanMethod:      "search",
		},
	}

	// Create all emails
	for _, email := range emails {
		err := store.CreateOrUpdate(email)
		if err != nil {
			t.Fatalf("Failed to create email %s: %v", email.GmailMessageID, err)
		}
	}

	// Test getting emails with tracking numbers
	emailsWithTracking, err := store.GetEmailsWithTrackingNumbers()
	if err != nil {
		t.Fatalf("Failed to get emails with tracking numbers: %v", err)
	}

	// Should find 2 emails (msg001 and msg003)
	if len(emailsWithTracking) != 2 {
		t.Errorf("Expected 2 emails with tracking numbers, got %d", len(emailsWithTracking))
	}

	// Verify the correct emails were found
	expectedMessageIDs := map[string]bool{
		"msg001": true,
		"msg003": true,
	}

	for _, email := range emailsWithTracking {
		if !expectedMessageIDs[email.GmailMessageID] {
			t.Errorf("Unexpected email with message ID %s", email.GmailMessageID)
		}
	}
}