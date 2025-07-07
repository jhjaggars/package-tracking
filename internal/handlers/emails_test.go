package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"package-tracking/internal/database"
)

func setupEmailTestDB(t *testing.T) *database.DB {
	// Create in-memory test database
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create test shipment for linking tests
	shipment := &database.Shipment{
		TrackingNumber: "TEST123456789",
		Carrier:        "ups",
		Description:    "Test shipment",
		Status:         "in_transit",
	}
	
	err = db.Shipments.Create(shipment)
	if err != nil {
		t.Fatalf("Failed to create test shipment: %v", err)
	}

	return db
}

func TestGetShipmentEmails(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	handler := NewEmailHandler(db)

	// Create test emails
	testEmails := []*database.EmailBodyEntry{
		{
			GmailMessageID:    "email-1",
			GmailThreadID:     "thread-1",
			From:              "sender@example.com",
			Subject:           "Package shipped",
			Date:              time.Now().Add(-time.Hour),
			BodyText:          "Your package TEST123456789 has been shipped",
			InternalTimestamp: time.Now().Add(-time.Hour),
			ScanMethod:        "time-based",
			ProcessedAt:       time.Now(),
			Status:            "processed",
			TrackingNumbers:   `["TEST123456789"]`,
		},
		{
			GmailMessageID:    "email-2",
			GmailThreadID:     "thread-1",
			From:              "carrier@example.com",
			Subject:           "Re: Package shipped",
			Date:              time.Now().Add(-30 * time.Minute),
			BodyText:          "Package TEST123456789 is out for delivery",
			InternalTimestamp: time.Now().Add(-30 * time.Minute),
			ScanMethod:        "time-based",
			ProcessedAt:       time.Now(),
			Status:            "processed",
			TrackingNumbers:   `["TEST123456789"]`,
		},
	}

	// Store emails
	for _, email := range testEmails {
		err := db.Emails.CreateOrUpdate(email)
		if err != nil {
			t.Fatalf("Failed to create test email: %v", err)
		}
	}

	// Link emails to shipment (shipment ID should be 1 from setup)
	shipmentID := 1
	for _, email := range testEmails {
		err := db.Emails.LinkEmailToShipment(email.ID, shipmentID, "automatic", "TEST123456789", "system")
		if err != nil {
			t.Fatalf("Failed to link email to shipment: %v", err)
		}
	}

	// Test getting emails for shipment
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/shipments/%d/emails", shipmentID), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.GetShipmentEmails(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response []database.EmailBodyEntry
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 emails, got %d", len(response))
	}

	// Verify email order (should be ordered by date DESC)
	if response[0].Date.Before(response[1].Date) {
		t.Error("Expected emails to be ordered by date DESC")
	}
}

func TestGetEmailThread(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	handler := NewEmailHandler(db)

	threadID := "thread-conversation"

	// Create test thread
	thread := &database.EmailThread{
		GmailThreadID:    threadID,
		Subject:          "Package conversation",
		Participants:     `["sender@example.com", "recipient@example.com"]`,
		MessageCount:     2,
		FirstMessageDate: time.Now().Add(-2 * time.Hour),
		LastMessageDate:  time.Now().Add(-time.Hour),
	}

	err := db.Emails.CreateOrUpdateThread(thread)
	if err != nil {
		t.Fatalf("Failed to create test thread: %v", err)
	}

	// Create emails in the thread
	threadEmails := []*database.EmailBodyEntry{
		{
			GmailMessageID:    "thread-email-1",
			GmailThreadID:     threadID,
			From:              "sender@example.com",
			Subject:           "Package order",
			Date:              time.Now().Add(-2 * time.Hour),
			BodyText:          "I'd like to order a package",
			InternalTimestamp: time.Now().Add(-2 * time.Hour),
			ScanMethod:        "time-based",
			ProcessedAt:       time.Now(),
			Status:            "processed",
		},
		{
			GmailMessageID:    "thread-email-2",
			GmailThreadID:     threadID,
			From:              "recipient@example.com",
			Subject:           "Re: Package order",
			Date:              time.Now().Add(-time.Hour),
			BodyText:          "Your package TEST123456789 is on the way",
			InternalTimestamp: time.Now().Add(-time.Hour),
			ScanMethod:        "time-based",
			ProcessedAt:       time.Now(),
			Status:            "processed",
			TrackingNumbers:   `["TEST123456789"]`,
		},
	}

	for _, email := range threadEmails {
		err := db.Emails.CreateOrUpdate(email)
		if err != nil {
			t.Fatalf("Failed to create thread email: %v", err)
		}
	}

	// Test getting thread
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/emails/%s/thread", threadID), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.GetEmailThread(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response struct {
		Thread database.EmailThread           `json:"thread"`
		Emails []database.EmailBodyEntry `json:"emails"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Thread.GmailThreadID != threadID {
		t.Errorf("Expected thread ID %s, got %s", threadID, response.Thread.GmailThreadID)
	}

	if len(response.Emails) != 2 {
		t.Errorf("Expected 2 emails in thread, got %d", len(response.Emails))
	}

	// Verify emails are ordered by date ASC (chronological order)
	if response.Emails[0].Date.After(response.Emails[1].Date) {
		t.Error("Expected thread emails to be ordered chronologically (date ASC)")
	}
}

func TestGetEmailBody(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	handler := NewEmailHandler(db)

	// Create test email with body content
	emailID := "email-with-body"
	testEmail := &database.EmailBodyEntry{
		GmailMessageID:    emailID,
		GmailThreadID:     "thread-body-test",
		From:              "test@example.com",
		Subject:           "Email with body content",
		Date:              time.Now(),
		BodyText:          "This is the plain text body content",
		BodyHTML:          "<p>This is the <strong>HTML body</strong> content</p>",
		InternalTimestamp: time.Now(),
		ScanMethod:        "time-based",
		ProcessedAt:       time.Now(),
		Status:            "processed",
	}

	err := db.Emails.CreateOrUpdate(testEmail)
	if err != nil {
		t.Fatalf("Failed to create test email: %v", err)
	}

	// Test getting email body
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/emails/%s/body", emailID), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.GetEmailBody(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response struct {
		PlainText string `json:"plain_text"`
		HTMLText  string `json:"html_text"`
		Subject   string `json:"subject"`
		From      string `json:"from"`
		Date      string `json:"date"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.PlainText != testEmail.BodyText {
		t.Errorf("Expected plain text '%s', got '%s'", testEmail.BodyText, response.PlainText)
	}

	if response.HTMLText != testEmail.BodyHTML {
		t.Errorf("Expected HTML text '%s', got '%s'", testEmail.BodyHTML, response.HTMLText)
	}

	if response.Subject != testEmail.Subject {
		t.Errorf("Expected subject '%s', got '%s'", testEmail.Subject, response.Subject)
	}
}

func TestLinkEmailToShipment(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	handler := NewEmailHandler(db)

	// Create test email
	testEmail := &database.EmailBodyEntry{
		GmailMessageID:    "email-to-link",
		GmailThreadID:     "thread-link-test",
		From:              "test@example.com",
		Subject:           "Email to link",
		Date:              time.Now(),
		BodyText:          "This email should be linked to a shipment",
		InternalTimestamp: time.Now(),
		ScanMethod:        "time-based",
		ProcessedAt:       time.Now(),
		Status:            "processed",
	}

	err := db.Emails.CreateOrUpdate(testEmail)
	if err != nil {
		t.Fatalf("Failed to create test email: %v", err)
	}

	// Test linking email to shipment
	shipmentID := 1 // From setup
	linkData := map[string]interface{}{
		"link_type":       "manual",
		"tracking_number": "TEST123456789",
		"created_by":      "user",
	}

	jsonData, err := json.Marshal(linkData)
	if err != nil {
		t.Fatalf("Failed to marshal link data: %v", err)
	}

	req, err := http.NewRequest("POST", 
		fmt.Sprintf("/api/emails/%d/link/%d", testEmail.ID, shipmentID), 
		bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.LinkEmailToShipment(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Verify link was created by checking if email appears in shipment emails
	req2, err := http.NewRequest("GET", fmt.Sprintf("/api/shipments/%d/emails", shipmentID), nil)
	if err != nil {
		t.Fatalf("Failed to create verification request: %v", err)
	}

	rr2 := httptest.NewRecorder()
	handler.GetShipmentEmails(rr2, req2)

	if status := rr2.Code; status != http.StatusOK {
		t.Errorf("Verification request returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var emails []database.EmailBodyEntry
	if err := json.Unmarshal(rr2.Body.Bytes(), &emails); err != nil {
		t.Fatalf("Failed to unmarshal verification response: %v", err)
	}

	// Should now have the linked email plus any existing ones
	found := false
	for _, email := range emails {
		if email.GmailMessageID == testEmail.GmailMessageID {
			found = true
			break
		}
	}

	if !found {
		t.Error("Linked email not found in shipment emails")
	}
}

func TestUnlinkEmailFromShipment(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	handler := NewEmailHandler(db)

	// Create test email
	testEmail := &database.EmailBodyEntry{
		GmailMessageID:    "email-to-unlink",
		GmailThreadID:     "thread-unlink-test",
		From:              "test@example.com",
		Subject:           "Email to unlink",
		Date:              time.Now(),
		BodyText:          "This email will be unlinked",
		InternalTimestamp: time.Now(),
		ScanMethod:        "time-based",
		ProcessedAt:       time.Now(),
		Status:            "processed",
	}

	err := db.Emails.CreateOrUpdate(testEmail)
	if err != nil {
		t.Fatalf("Failed to create test email: %v", err)
	}

	// Link email to shipment first
	shipmentID := 1 // From setup
	err = db.Emails.LinkEmailToShipment(testEmail.ID, shipmentID, "manual", "TEST123456789", "user")
	if err != nil {
		t.Fatalf("Failed to create initial link: %v", err)
	}

	// Test unlinking email from shipment
	req, err := http.NewRequest("DELETE", 
		fmt.Sprintf("/api/emails/%d/link/%d", testEmail.ID, shipmentID), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.UnlinkEmailFromShipment(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}

	// Verify link was removed by checking shipment emails
	req2, err := http.NewRequest("GET", fmt.Sprintf("/api/shipments/%d/emails", shipmentID), nil)
	if err != nil {
		t.Fatalf("Failed to create verification request: %v", err)
	}

	rr2 := httptest.NewRecorder()
	handler.GetShipmentEmails(rr2, req2)

	if status := rr2.Code; status != http.StatusOK {
		t.Errorf("Verification request returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var emails []database.EmailBodyEntry
	if err := json.Unmarshal(rr2.Body.Bytes(), &emails); err != nil {
		t.Fatalf("Failed to unmarshal verification response: %v", err)
	}

	// Should not find the unlinked email
	found := false
	for _, email := range emails {
		if email.GmailMessageID == testEmail.GmailMessageID {
			found = true
			break
		}
	}

	if found {
		t.Error("Unlinked email still found in shipment emails")
	}
}

func TestEmailEndpointsErrorHandling(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	handler := NewEmailHandler(db)

	// Test non-existent shipment
	req, err := http.NewRequest("GET", "/api/shipments/999/emails", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.GetShipmentEmails(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler should return OK even for non-existent shipment, got %v", status)
	}

	// Should return empty array
	var response []database.EmailBodyEntry
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 0 {
		t.Errorf("Expected empty array for non-existent shipment, got %d emails", len(response))
	}

	// Test non-existent email
	req2, err := http.NewRequest("GET", "/api/emails/non-existent/body", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr2 := httptest.NewRecorder()
	handler.GetEmailBody(rr2, req2)

	if status := rr2.Code; status != http.StatusNotFound {
		t.Errorf("Handler should return 404 for non-existent email, got %v", status)
	}
}

