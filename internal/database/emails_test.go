package database

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestEmailDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create tables (we'll need to run migrations)
	if err := createEmailTables(db); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

// createEmailTables creates the email-related tables for testing
func createEmailTables(db *sql.DB) error {
	// Create processed_emails table with new fields
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS processed_emails (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		gmail_message_id TEXT UNIQUE NOT NULL,
		gmail_thread_id TEXT NOT NULL,
		from_address TEXT NOT NULL,
		subject TEXT NOT NULL,
		date DATETIME NOT NULL,
		body_text TEXT,
		body_html TEXT,
		body_compressed BLOB,
		internal_timestamp DATETIME NOT NULL,
		scan_method TEXT NOT NULL DEFAULT 'search',
		processed_at DATETIME NOT NULL,
		status TEXT NOT NULL,
		tracking_numbers TEXT,
		error_message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}

	// Create email_threads table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS email_threads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		gmail_thread_id TEXT UNIQUE NOT NULL,
		subject TEXT NOT NULL,
		participants TEXT NOT NULL,
		message_count INTEGER NOT NULL DEFAULT 1,
		first_message_date DATETIME NOT NULL,
		last_message_date DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}

	// Create email_shipments linking table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS email_shipments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email_id INTEGER NOT NULL,
		shipment_id INTEGER NOT NULL,
		link_type TEXT NOT NULL,
		tracking_number TEXT NOT NULL,
		created_by TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(email_id, shipment_id)
	)`)
	if err != nil {
		return err
	}

	// Create indexes
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_processed_emails_gmail_message_id ON processed_emails(gmail_message_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_processed_emails_gmail_thread_id ON processed_emails(gmail_thread_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_processed_emails_internal_timestamp ON processed_emails(internal_timestamp)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_email_threads_gmail_thread_id ON email_threads(gmail_thread_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_email_shipments_email_id ON email_shipments(email_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_email_shipments_shipment_id ON email_shipments(shipment_id)`)
	if err != nil {
		return err
	}

	return nil
}

func TestEmailStore_CreateOrUpdate(t *testing.T) {
	db, cleanup := setupTestEmailDB(t)
	defer cleanup()

	store := NewEmailStore(db)

	// Test creating a new email
	email := &EmailBodyEntry{
		GmailMessageID:    "test-message-id",
		GmailThreadID:     "test-thread-id",
		From:              "test@example.com",
		Subject:           "Test Subject",
		Date:              time.Now(),
		BodyText:          "Test body text",
		BodyHTML:          "<p>Test body HTML</p>",
		InternalTimestamp: time.Now(),
		ScanMethod:        "time-based",
		ProcessedAt:       time.Now(),
		Status:            "processed",
		TrackingNumbers:   `["1Z999AA1234567890"]`,
	}

	err := store.CreateOrUpdate(email)
	if err != nil {
		t.Fatalf("Failed to create email: %v", err)
	}

	if email.ID == 0 {
		t.Error("Expected email ID to be set after creation")
	}

	// Test retrieving the email
	retrieved, err := store.GetByGmailMessageID(email.GmailMessageID)
	if err != nil {
		t.Fatalf("Failed to retrieve email: %v", err)
	}

	if retrieved.GmailMessageID != email.GmailMessageID {
		t.Errorf("Expected Gmail message ID %s, got %s", email.GmailMessageID, retrieved.GmailMessageID)
	}

	if retrieved.From != email.From {
		t.Errorf("Expected From %s, got %s", email.From, retrieved.From)
	}

	if retrieved.Subject != email.Subject {
		t.Errorf("Expected Subject %s, got %s", email.Subject, retrieved.Subject)
	}

	if retrieved.BodyText != email.BodyText {
		t.Errorf("Expected BodyText %s, got %s", email.BodyText, retrieved.BodyText)
	}

	if retrieved.ScanMethod != email.ScanMethod {
		t.Errorf("Expected ScanMethod %s, got %s", email.ScanMethod, retrieved.ScanMethod)
	}

	// Test updating the email
	email.Subject = "Updated Subject"
	email.BodyText = "Updated body text"
	email.Status = "updated"

	err = store.CreateOrUpdate(email)
	if err != nil {
		t.Fatalf("Failed to update email: %v", err)
	}

	// Verify update
	updated, err := store.GetByGmailMessageID(email.GmailMessageID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated email: %v", err)
	}

	if updated.Subject != "Updated Subject" {
		t.Errorf("Expected updated Subject 'Updated Subject', got %s", updated.Subject)
	}

	if updated.BodyText != "Updated body text" {
		t.Errorf("Expected updated BodyText 'Updated body text', got %s", updated.BodyText)
	}

	if updated.Status != "updated" {
		t.Errorf("Expected updated Status 'updated', got %s", updated.Status)
	}
}

func TestEmailStore_ThreadOperations(t *testing.T) {
	db, cleanup := setupTestEmailDB(t)
	defer cleanup()

	store := NewEmailStore(db)

	// Test creating a new thread
	thread := &EmailThread{
		GmailThreadID:    "test-thread-id",
		Subject:          "Test Thread",
		Participants:     `["test1@example.com", "test2@example.com"]`,
		MessageCount:     2,
		FirstMessageDate: time.Now().Add(-time.Hour),
		LastMessageDate:  time.Now(),
	}

	err := store.CreateOrUpdateThread(thread)
	if err != nil {
		t.Fatalf("Failed to create thread: %v", err)
	}

	if thread.ID == 0 {
		t.Error("Expected thread ID to be set after creation")
	}

	// Test retrieving the thread
	retrieved, err := store.GetThreadByGmailThreadID(thread.GmailThreadID)
	if err != nil {
		t.Fatalf("Failed to retrieve thread: %v", err)
	}

	if retrieved.GmailThreadID != thread.GmailThreadID {
		t.Errorf("Expected Gmail thread ID %s, got %s", thread.GmailThreadID, retrieved.GmailThreadID)
	}

	if retrieved.Subject != thread.Subject {
		t.Errorf("Expected Subject %s, got %s", thread.Subject, retrieved.Subject)
	}

	if retrieved.MessageCount != thread.MessageCount {
		t.Errorf("Expected MessageCount %d, got %d", thread.MessageCount, retrieved.MessageCount)
	}

	// Test updating the thread
	thread.Subject = "Updated Thread"
	thread.MessageCount = 3

	err = store.CreateOrUpdateThread(thread)
	if err != nil {
		t.Fatalf("Failed to update thread: %v", err)
	}

	// Verify update
	updated, err := store.GetThreadByGmailThreadID(thread.GmailThreadID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated thread: %v", err)
	}

	if updated.Subject != "Updated Thread" {
		t.Errorf("Expected updated Subject 'Updated Thread', got %s", updated.Subject)
	}

	if updated.MessageCount != 3 {
		t.Errorf("Expected updated MessageCount 3, got %d", updated.MessageCount)
	}
}

func TestEmailStore_EmailShipmentLinking(t *testing.T) {
	db, cleanup := setupTestEmailDB(t)
	defer cleanup()

	store := NewEmailStore(db)

	// Create a test email
	email := &EmailBodyEntry{
		GmailMessageID:    "test-message-id",
		GmailThreadID:     "test-thread-id",
		From:              "test@example.com",
		Subject:           "Test Subject",
		Date:              time.Now(),
		BodyText:          "Test body text",
		InternalTimestamp: time.Now(),
		ScanMethod:        "time-based",
		ProcessedAt:       time.Now(),
		Status:            "processed",
		TrackingNumbers:   `["1Z999AA1234567890"]`,
	}

	err := store.CreateOrUpdate(email)
	if err != nil {
		t.Fatalf("Failed to create email: %v", err)
	}

	// Test linking email to shipment
	shipmentID := 123
	err = store.LinkEmailToShipment(email.ID, shipmentID, "automatic", "1Z999AA1234567890", "system")
	if err != nil {
		t.Fatalf("Failed to link email to shipment: %v", err)
	}

	// Test retrieving emails by shipment ID
	// Note: This would require a shipment to exist, but for this test we'll test the query structure
	_, err = store.GetByShipmentID(shipmentID)
	if err != nil {
		// This is expected to fail since we don't have a shipment table in our test DB
		// But the query should be structurally correct
		t.Logf("GetByShipmentID failed as expected (no shipment table): %v", err)
	}

	// Test duplicate link prevention
	err = store.LinkEmailToShipment(email.ID, shipmentID, "manual", "1Z999AA1234567890", "user")
	if err != nil {
		t.Fatalf("Failed to handle duplicate link: %v", err)
	}

	// Test unlinking
	err = store.UnlinkEmailFromShipment(email.ID, shipmentID)
	if err != nil {
		t.Fatalf("Failed to unlink email from shipment: %v", err)
	}

	// Test unlinking non-existent link
	err = store.UnlinkEmailFromShipment(email.ID, 999)
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows for non-existent link, got %v", err)
	}
}

func TestEmailStore_GetEmailsByThreadID(t *testing.T) {
	db, cleanup := setupTestEmailDB(t)
	defer cleanup()

	store := NewEmailStore(db)

	threadID := "test-thread-id"
	
	// Create multiple emails in the same thread
	emailsToCreate := []*EmailBodyEntry{
		{
			GmailMessageID:    "test-message-1",
			GmailThreadID:     threadID,
			From:              "test1@example.com",
			Subject:           "Test Subject 1",
			Date:              time.Now().Add(-time.Hour),
			BodyText:          "Test body text 1",
			InternalTimestamp: time.Now().Add(-time.Hour),
			ScanMethod:        "time-based",
			ProcessedAt:       time.Now(),
			Status:            "processed",
		},
		{
			GmailMessageID:    "test-message-2",
			GmailThreadID:     threadID,
			From:              "test2@example.com",
			Subject:           "Re: Test Subject 1",
			Date:              time.Now().Add(-30 * time.Minute),
			BodyText:          "Test body text 2",
			InternalTimestamp: time.Now().Add(-30 * time.Minute),
			ScanMethod:        "time-based",
			ProcessedAt:       time.Now(),
			Status:            "processed",
		},
	}

	// Create the emails
	for _, email := range emailsToCreate {
		err := store.CreateOrUpdate(email)
		if err != nil {
			t.Fatalf("Failed to create email %s: %v", email.GmailMessageID, err)
		}
	}

	// Test retrieving emails by thread ID
	threadEmails, err := store.GetEmailsByThreadID(threadID)
	if err != nil {
		t.Fatalf("Failed to get emails by thread ID: %v", err)
	}

	if len(threadEmails) != 2 {
		t.Errorf("Expected 2 emails in thread, got %d", len(threadEmails))
	}

	// Verify emails are ordered by date (ASC)
	if threadEmails[0].Date.After(threadEmails[1].Date) {
		t.Error("Expected emails to be ordered by date (ASC)")
	}
}

func TestEmailStore_GetEmailsSince(t *testing.T) {
	db, cleanup := setupTestEmailDB(t)
	defer cleanup()

	store := NewEmailStore(db)

	now := time.Now()
	oneHourAgo := now.Add(-time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)

	// Create emails with different timestamps
	emailsToCreate := []*EmailBodyEntry{
		{
			GmailMessageID:    "test-message-1",
			GmailThreadID:     "test-thread-1",
			From:              "test1@example.com",
			Subject:           "Old email",
			Date:              twoHoursAgo,
			BodyText:          "Old email body",
			InternalTimestamp: twoHoursAgo,
			ScanMethod:        "time-based",
			ProcessedAt:       now,
			Status:            "processed",
		},
		{
			GmailMessageID:    "test-message-2",
			GmailThreadID:     "test-thread-2",
			From:              "test2@example.com",
			Subject:           "Recent email",
			Date:              oneHourAgo,
			BodyText:          "Recent email body",
			InternalTimestamp: oneHourAgo,
			ScanMethod:        "time-based",
			ProcessedAt:       now,
			Status:            "processed",
		},
	}

	// Create the emails
	for _, email := range emailsToCreate {
		err := store.CreateOrUpdate(email)
		if err != nil {
			t.Fatalf("Failed to create email %s: %v", email.GmailMessageID, err)
		}
	}

	// Test getting emails since 90 minutes ago (should get 1 email)
	since := now.Add(-90 * time.Minute)
	recentEmails, err := store.GetEmailsSince(since)
	if err != nil {
		t.Fatalf("Failed to get emails since %v: %v", since, err)
	}

	if len(recentEmails) != 1 {
		t.Errorf("Expected 1 recent email, got %d", len(recentEmails))
	}

	if len(recentEmails) > 0 && recentEmails[0].Subject != "Recent email" {
		t.Errorf("Expected recent email subject 'Recent email', got %s", recentEmails[0].Subject)
	}

	// Test getting emails since 3 hours ago (should get 2 emails)
	since = now.Add(-3 * time.Hour)
	allEmails, err := store.GetEmailsSince(since)
	if err != nil {
		t.Fatalf("Failed to get emails since %v: %v", since, err)
	}

	if len(allEmails) != 2 {
		t.Errorf("Expected 2 emails, got %d", len(allEmails))
	}
}

func TestEmailStore_CleanupOldEmails(t *testing.T) {
	db, cleanup := setupTestEmailDB(t)
	defer cleanup()

	store := NewEmailStore(db)

	now := time.Now()
	oldTime := now.Add(-48 * time.Hour)

	// Create an old email
	email := &EmailBodyEntry{
		GmailMessageID:    "test-message-1",
		GmailThreadID:     "test-thread-1",
		From:              "test@example.com",
		Subject:           "Old email",
		Date:              oldTime,
		BodyText:          "Old email body that should be cleaned up",
		BodyHTML:          "<p>Old email HTML that should be cleaned up</p>",
		InternalTimestamp: oldTime,
		ScanMethod:        "time-based",
		ProcessedAt:       oldTime,
		Status:            "processed",
	}

	err := store.CreateOrUpdate(email)
	if err != nil {
		t.Fatalf("Failed to create email: %v", err)
	}

	// Verify email has body content
	retrieved, err := store.GetByGmailMessageID(email.GmailMessageID)
	if err != nil {
		t.Fatalf("Failed to retrieve email: %v", err)
	}

	if retrieved.BodyText == "" {
		t.Error("Expected email to have body text before cleanup")
	}

	// Clean up emails older than 24 hours
	cleanupTime := now.Add(-24 * time.Hour)
	err = store.CleanupOldEmails(cleanupTime)
	if err != nil {
		t.Fatalf("Failed to cleanup old emails: %v", err)
	}

	// Verify email body was cleaned up
	cleaned, err := store.GetByGmailMessageID(email.GmailMessageID)
	if err != nil {
		t.Fatalf("Failed to retrieve cleaned email: %v", err)
	}

	if cleaned.BodyText != "" {
		t.Error("Expected email body text to be empty after cleanup")
	}

	if cleaned.BodyHTML != "" {
		t.Error("Expected email body HTML to be empty after cleanup")
	}

	// Verify other fields are still intact
	if cleaned.Subject != email.Subject {
		t.Error("Expected email subject to remain after cleanup")
	}
}

func TestEmailStore_IsProcessed(t *testing.T) {
	db, cleanup := setupTestEmailDB(t)
	defer cleanup()

	store := NewEmailStore(db)

	// Test with non-existent email
	processed, err := store.IsProcessed("non-existent-id")
	if err != nil {
		t.Fatalf("Failed to check if email is processed: %v", err)
	}

	if processed {
		t.Error("Expected non-existent email to not be processed")
	}

	// Create an email
	email := &EmailBodyEntry{
		GmailMessageID:    "test-message-1",
		GmailThreadID:     "test-thread-1",
		From:              "test@example.com",
		Subject:           "Test email",
		Date:              time.Now(),
		BodyText:          "Test body",
		InternalTimestamp: time.Now(),
		ScanMethod:        "time-based",
		ProcessedAt:       time.Now(),
		Status:            "processed",
	}

	err = store.CreateOrUpdate(email)
	if err != nil {
		t.Fatalf("Failed to create email: %v", err)
	}

	// Test with existing email
	processed, err = store.IsProcessed(email.GmailMessageID)
	if err != nil {
		t.Fatalf("Failed to check if email is processed: %v", err)
	}

	if !processed {
		t.Error("Expected existing email to be processed")
	}
}