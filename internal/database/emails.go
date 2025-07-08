package database

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"time"
)

// EmailBodyEntry represents a stored email body in the database
type EmailBodyEntry struct {
	ID                   int       `json:"id"`
	GmailMessageID       string    `json:"gmail_message_id"`
	GmailThreadID        string    `json:"gmail_thread_id"`
	From                 string    `json:"from"`
	Subject              string    `json:"subject"`
	Date                 time.Time `json:"date"`
	BodyText             string    `json:"body_text"`
	BodyHTML             string    `json:"body_html"`
	BodyCompressed       []byte    `json:"body_compressed,omitempty"`
	InternalTimestamp    time.Time `json:"internal_timestamp"`
	ScanMethod           string    `json:"scan_method"` // "search" or "time-based"
	ProcessedAt          time.Time `json:"processed_at"`
	Status               string    `json:"status"`
	TrackingNumbers      string    `json:"tracking_numbers"` // JSON encoded
	ErrorMessage         string    `json:"error_message,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	
	// Two-phase processing fields
	ProcessingPhase      string     `json:"processing_phase"`      // "metadata_only", "content_extracted", "legacy"
	RelevanceScore       float64    `json:"relevance_score"`       // 0.0-1.0 score for shipping relevance
	Snippet              string     `json:"snippet"`               // Email snippet/preview text
	HasContent           bool       `json:"has_content"`           // Whether full content has been downloaded
	MetadataExtractedAt  *time.Time `json:"metadata_extracted_at"` // When metadata was extracted
	ContentExtractedAt   *time.Time `json:"content_extracted_at"`  // When full content was extracted
}

// EmailThread represents a Gmail thread/conversation
type EmailThread struct {
	ID               int       `json:"id"`
	GmailThreadID    string    `json:"gmail_thread_id"`
	Subject          string    `json:"subject"`
	Participants     string    `json:"participants"` // JSON encoded array of email addresses
	MessageCount     int       `json:"message_count"`
	FirstMessageDate time.Time `json:"first_message_date"`
	LastMessageDate  time.Time `json:"last_message_date"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// EmailShipmentLink represents the many-to-many relationship between emails and shipments
type EmailShipmentLink struct {
	ID             int       `json:"id"`
	EmailID        int       `json:"email_id"`
	ShipmentID     int       `json:"shipment_id"`
	LinkType       string    `json:"link_type"`       // "automatic" or "manual"
	TrackingNumber string    `json:"tracking_number"` // The tracking number that caused the link
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      string    `json:"created_by"` // "system" or user identifier
}

// EmailStore handles database operations for emails
type EmailStore struct {
	db *sql.DB
}

func NewEmailStore(db *sql.DB) *EmailStore {
	return &EmailStore{db: db}
}

// GetByGmailMessageID retrieves an email by Gmail message ID
func (e *EmailStore) GetByGmailMessageID(gmailMessageID string) (*EmailBodyEntry, error) {
	query := `SELECT id, gmail_message_id, gmail_thread_id, sender, subject, date, 
			  body_text, body_html, body_compressed, internal_timestamp, scan_method,
			  processed_at, status, tracking_numbers, error_message, created_at, updated_at,
			  COALESCE(processing_phase, 'legacy') as processing_phase,
			  COALESCE(relevance_score, 0.0) as relevance_score,
			  COALESCE(snippet, '') as snippet,
			  COALESCE(has_content, FALSE) as has_content,
			  metadata_extracted_at, content_extracted_at
			  FROM processed_emails WHERE gmail_message_id = ?`
	
	var email EmailBodyEntry
	err := e.db.QueryRow(query, gmailMessageID).Scan(
		&email.ID, &email.GmailMessageID, &email.GmailThreadID, &email.From,
		&email.Subject, &email.Date, &email.BodyText, &email.BodyHTML,
		&email.BodyCompressed, &email.InternalTimestamp, &email.ScanMethod,
		&email.ProcessedAt, &email.Status, &email.TrackingNumbers,
		&email.ErrorMessage, &email.CreatedAt, &email.UpdatedAt,
		&email.ProcessingPhase, &email.RelevanceScore, &email.Snippet,
		&email.HasContent, &email.MetadataExtractedAt, &email.ContentExtractedAt)
	
	if err != nil {
		return nil, err
	}
	
	return &email, nil
}

// GetByShipmentID retrieves all emails linked to a shipment
func (e *EmailStore) GetByShipmentID(shipmentID int) ([]EmailBodyEntry, error) {
	query := `SELECT pe.id, pe.gmail_message_id, pe.gmail_thread_id, pe.sender, 
			  pe.subject, pe.date, pe.body_text, pe.body_html, pe.body_compressed,
			  pe.internal_timestamp, pe.scan_method, pe.processed_at, pe.status,
			  pe.tracking_numbers, pe.error_message, pe.created_at, pe.updated_at
			  FROM processed_emails pe
			  JOIN email_shipments es ON pe.id = es.email_id
			  WHERE es.shipment_id = ?
			  ORDER BY pe.date DESC`
	
	rows, err := e.db.Query(query, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var emails []EmailBodyEntry
	for rows.Next() {
		var email EmailBodyEntry
		err := rows.Scan(
			&email.ID, &email.GmailMessageID, &email.GmailThreadID, &email.From,
			&email.Subject, &email.Date, &email.BodyText, &email.BodyHTML,
			&email.BodyCompressed, &email.InternalTimestamp, &email.ScanMethod,
			&email.ProcessedAt, &email.Status, &email.TrackingNumbers,
			&email.ErrorMessage, &email.CreatedAt, &email.UpdatedAt)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	
	return emails, rows.Err()
}

// GetByShipmentIDPaginated retrieves emails linked to a shipment with pagination
func (e *EmailStore) GetByShipmentIDPaginated(shipmentID int, limit, offset int) ([]EmailBodyEntry, error) {
	query := `SELECT pe.id, pe.gmail_message_id, pe.gmail_thread_id, pe.sender, 
		  pe.subject, pe.date, pe.body_text, pe.body_html, pe.body_compressed,
		  pe.internal_timestamp, pe.scan_method, pe.processed_at, pe.status,
		  pe.tracking_numbers, pe.error_message, pe.created_at, pe.updated_at
		  FROM processed_emails pe
		  JOIN email_shipments es ON pe.id = es.email_id
		  WHERE es.shipment_id = ?
		  ORDER BY pe.date DESC`
	
	// Add pagination if limit is specified
	args := []interface{}{shipmentID}
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}
	
	rows, err := e.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var emails []EmailBodyEntry
	for rows.Next() {
		var email EmailBodyEntry
		err := rows.Scan(
			&email.ID, &email.GmailMessageID, &email.GmailThreadID, &email.From,
			&email.Subject, &email.Date, &email.BodyText, &email.BodyHTML,
			&email.BodyCompressed, &email.InternalTimestamp, &email.ScanMethod,
			&email.ProcessedAt, &email.Status, &email.TrackingNumbers,
			&email.ErrorMessage, &email.CreatedAt, &email.UpdatedAt)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	
	return emails, rows.Err()
}

// CreateOrUpdate creates or updates an email entry
func (e *EmailStore) CreateOrUpdate(email *EmailBodyEntry) error {
	// Check if email already exists
	existing, err := e.GetByGmailMessageID(email.GmailMessageID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	
	if existing != nil {
		// Update existing email
		return e.update(email)
	} else {
		// Create new email
		return e.create(email)
	}
}

// create creates a new email entry
func (e *EmailStore) create(email *EmailBodyEntry) error {
	query := `INSERT INTO processed_emails (gmail_message_id, gmail_thread_id, sender, 
			  subject, date, body_text, body_html, body_compressed, internal_timestamp, 
			  scan_method, processed_at, status, tracking_numbers, error_message,
			  processing_phase, relevance_score, snippet, has_content, 
			  metadata_extracted_at, content_extracted_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	result, err := e.db.Exec(query, email.GmailMessageID, email.GmailThreadID, 
		email.From, email.Subject, email.Date, email.BodyText, email.BodyHTML,
		email.BodyCompressed, email.InternalTimestamp, email.ScanMethod,
		email.ProcessedAt, email.Status, email.TrackingNumbers, email.ErrorMessage,
		email.ProcessingPhase, email.RelevanceScore, email.Snippet, email.HasContent,
		email.MetadataExtractedAt, email.ContentExtractedAt)
	
	if err != nil {
		return err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	
	email.ID = int(id)
	return nil
}

// update updates an existing email entry
func (e *EmailStore) update(email *EmailBodyEntry) error {
	query := `UPDATE processed_emails SET gmail_thread_id = ?, sender = ?, 
			  subject = ?, date = ?, body_text = ?, body_html = ?, body_compressed = ?,
			  internal_timestamp = ?, scan_method = ?, processed_at = ?, status = ?,
			  tracking_numbers = ?, error_message = ?, processing_phase = ?, 
			  relevance_score = ?, snippet = ?, has_content = ?, 
			  metadata_extracted_at = ?, content_extracted_at = ?,
			  updated_at = CURRENT_TIMESTAMP
			  WHERE gmail_message_id = ?`
	
	result, err := e.db.Exec(query, email.GmailThreadID, email.From, email.Subject,
		email.Date, email.BodyText, email.BodyHTML, email.BodyCompressed,
		email.InternalTimestamp, email.ScanMethod, email.ProcessedAt, email.Status,
		email.TrackingNumbers, email.ErrorMessage, email.ProcessingPhase,
		email.RelevanceScore, email.Snippet, email.HasContent,
		email.MetadataExtractedAt, email.ContentExtractedAt, email.GmailMessageID)
	
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}

// GetThreadByGmailThreadID retrieves a thread by Gmail thread ID
func (e *EmailStore) GetThreadByGmailThreadID(gmailThreadID string) (*EmailThread, error) {
	query := `SELECT id, gmail_thread_id, subject, participants, message_count,
			  first_message_date, last_message_date, created_at, updated_at
			  FROM email_threads WHERE gmail_thread_id = ?`
	
	var thread EmailThread
	err := e.db.QueryRow(query, gmailThreadID).Scan(
		&thread.ID, &thread.GmailThreadID, &thread.Subject, &thread.Participants,
		&thread.MessageCount, &thread.FirstMessageDate, &thread.LastMessageDate,
		&thread.CreatedAt, &thread.UpdatedAt)
	
	if err != nil {
		return nil, err
	}
	
	return &thread, nil
}

// CreateOrUpdateThread creates or updates a thread entry
func (e *EmailStore) CreateOrUpdateThread(thread *EmailThread) error {
	existing, err := e.GetThreadByGmailThreadID(thread.GmailThreadID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	
	if existing != nil {
		// Update existing thread
		query := `UPDATE email_threads SET subject = ?, participants = ?, message_count = ?,
				  first_message_date = ?, last_message_date = ?, updated_at = CURRENT_TIMESTAMP
				  WHERE gmail_thread_id = ?`
		
		result, err := e.db.Exec(query, thread.Subject, thread.Participants,
			thread.MessageCount, thread.FirstMessageDate, thread.LastMessageDate,
			thread.GmailThreadID)
		
		if err != nil {
			return err
		}
		
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		
		if rowsAffected == 0 {
			return sql.ErrNoRows
		}
		
		thread.ID = existing.ID
		return nil
	} else {
		// Create new thread
		query := `INSERT INTO email_threads (gmail_thread_id, subject, participants, 
				  message_count, first_message_date, last_message_date)
				  VALUES (?, ?, ?, ?, ?, ?)`
		
		result, err := e.db.Exec(query, thread.GmailThreadID, thread.Subject,
			thread.Participants, thread.MessageCount, thread.FirstMessageDate,
			thread.LastMessageDate)
		
		if err != nil {
			return err
		}
		
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		
		thread.ID = int(id)
		return nil
	}
}

// LinkEmailToShipment creates a link between an email and a shipment
func (e *EmailStore) LinkEmailToShipment(emailID, shipmentID int, linkType, trackingNumber, createdBy string) error {
	// Check if link already exists
	var count int
	checkQuery := `SELECT COUNT(*) FROM email_shipments WHERE email_id = ? AND shipment_id = ?`
	err := e.db.QueryRow(checkQuery, emailID, shipmentID).Scan(&count)
	if err != nil {
		return err
	}
	
	if count > 0 {
		return nil // Link already exists
	}
	
	// Create new link
	query := `INSERT INTO email_shipments (email_id, shipment_id, link_type, tracking_number, created_by)
			  VALUES (?, ?, ?, ?, ?)`
	
	_, err = e.db.Exec(query, emailID, shipmentID, linkType, trackingNumber, createdBy)
	return err
}

// UnlinkEmailFromShipment removes the link between an email and a shipment
func (e *EmailStore) UnlinkEmailFromShipment(emailID, shipmentID int) error {
	query := `DELETE FROM email_shipments WHERE email_id = ? AND shipment_id = ?`
	
	result, err := e.db.Exec(query, emailID, shipmentID)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}

// GetEmailsByThreadID retrieves all emails in a thread
func (e *EmailStore) GetEmailsByThreadID(gmailThreadID string) ([]EmailBodyEntry, error) {
	query := `SELECT id, gmail_message_id, gmail_thread_id, sender, subject, date, 
			  body_text, body_html, body_compressed, internal_timestamp, scan_method,
			  processed_at, status, tracking_numbers, error_message, created_at, updated_at
			  FROM processed_emails WHERE gmail_thread_id = ?
			  ORDER BY date ASC`
	
	rows, err := e.db.Query(query, gmailThreadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var emails []EmailBodyEntry
	for rows.Next() {
		var email EmailBodyEntry
		err := rows.Scan(
			&email.ID, &email.GmailMessageID, &email.GmailThreadID, &email.From,
			&email.Subject, &email.Date, &email.BodyText, &email.BodyHTML,
			&email.BodyCompressed, &email.InternalTimestamp, &email.ScanMethod,
			&email.ProcessedAt, &email.Status, &email.TrackingNumbers,
			&email.ErrorMessage, &email.CreatedAt, &email.UpdatedAt)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	
	return emails, rows.Err()
}

// GetEmailsSince retrieves emails processed since a specific timestamp
func (e *EmailStore) GetEmailsSince(since time.Time) ([]EmailBodyEntry, error) {
	query := `SELECT id, gmail_message_id, gmail_thread_id, sender, subject, date, 
			  body_text, body_html, body_compressed, internal_timestamp, scan_method,
			  processed_at, status, tracking_numbers, error_message, created_at, updated_at
			  FROM processed_emails WHERE internal_timestamp >= ?
			  ORDER BY internal_timestamp DESC`
	
	rows, err := e.db.Query(query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var emails []EmailBodyEntry
	for rows.Next() {
		var email EmailBodyEntry
		err := rows.Scan(
			&email.ID, &email.GmailMessageID, &email.GmailThreadID, &email.From,
			&email.Subject, &email.Date, &email.BodyText, &email.BodyHTML,
			&email.BodyCompressed, &email.InternalTimestamp, &email.ScanMethod,
			&email.ProcessedAt, &email.Status, &email.TrackingNumbers,
			&email.ErrorMessage, &email.CreatedAt, &email.UpdatedAt)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	
	return emails, rows.Err()
}

// CleanupOldEmails removes email bodies older than the specified date
func (e *EmailStore) CleanupOldEmails(olderThan time.Time) error {
	query := `UPDATE processed_emails SET body_text = '', body_html = '', 
			  body_compressed = NULL WHERE processed_at < ?`
	
	result, err := e.db.Exec(query, olderThan)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected > 0 {
		// Log cleanup operation
		fmt.Printf("Cleaned up email bodies for %d emails older than %s\n", rowsAffected, olderThan.Format("2006-01-02"))
	}
	
	return nil
}

// IsProcessed checks if an email has been processed (for backward compatibility)
func (e *EmailStore) IsProcessed(gmailMessageID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM processed_emails WHERE gmail_message_id = ?`
	err := e.db.QueryRow(query, gmailMessageID).Scan(&count)
	if err != nil {
		return false, err
	}
	
	return count > 0, nil
}

// GetEmailsForTrackingNumber finds emails that contain a specific tracking number
func (e *EmailStore) GetEmailsForTrackingNumber(trackingNumber string) ([]EmailBodyEntry, error) {
	query := `SELECT id, gmail_message_id, gmail_thread_id, sender, subject, date, 
			  body_text, body_html, body_compressed, internal_timestamp, scan_method,
			  processed_at, status, tracking_numbers, error_message, created_at, updated_at
			  FROM processed_emails 
			  WHERE tracking_numbers LIKE ? OR tracking_numbers LIKE ? OR tracking_numbers LIKE ?
			  ORDER BY date DESC`
	
	// Create search patterns for JSON array containing the tracking number
	pattern1 := `%"` + trackingNumber + `"%`           // "tracking_number"
	pattern2 := `%[` + trackingNumber + `%`             // [tracking_number
	pattern3 := `% ` + trackingNumber + `%`             // space tracking_number
	
	rows, err := e.db.Query(query, pattern1, pattern2, pattern3)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var emails []EmailBodyEntry
	for rows.Next() {
		var email EmailBodyEntry
		err := rows.Scan(
			&email.ID, &email.GmailMessageID, &email.GmailThreadID, &email.From,
			&email.Subject, &email.Date, &email.BodyText, &email.BodyHTML,
			&email.BodyCompressed, &email.InternalTimestamp, &email.ScanMethod,
			&email.ProcessedAt, &email.Status, &email.TrackingNumbers,
			&email.ErrorMessage, &email.CreatedAt, &email.UpdatedAt)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	
	return emails, rows.Err()
}

// GetEmailsWithTrackingNumbers retrieves all emails that have tracking numbers
func (e *EmailStore) GetEmailsWithTrackingNumbers() ([]EmailBodyEntry, error) {
	query := `SELECT id, gmail_message_id, gmail_thread_id, sender, subject, date, 
			  body_text, body_html, body_compressed, internal_timestamp, scan_method,
			  processed_at, status, tracking_numbers, error_message, created_at, updated_at
			  FROM processed_emails 
			  WHERE tracking_numbers IS NOT NULL 
			  AND tracking_numbers != '' 
			  AND tracking_numbers != '[]'
			  AND tracking_numbers != 'null'
			  ORDER BY date DESC`
	
	rows, err := e.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var emails []EmailBodyEntry
	for rows.Next() {
		var email EmailBodyEntry
		err := rows.Scan(
			&email.ID, &email.GmailMessageID, &email.GmailThreadID, &email.From,
			&email.Subject, &email.Date, &email.BodyText, &email.BodyHTML,
			&email.BodyCompressed, &email.InternalTimestamp, &email.ScanMethod,
			&email.ProcessedAt, &email.Status, &email.TrackingNumbers,
			&email.ErrorMessage, &email.CreatedAt, &email.UpdatedAt)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	
	return emails, rows.Err()
}

// CompressEmailBody compresses email body text for efficient storage
func CompressEmailBody(text string) ([]byte, error) {
	if text == "" {
		return nil, nil
	}
	
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	
	if _, err := gz.Write([]byte(text)); err != nil {
		return nil, fmt.Errorf("failed to write to gzip: %w", err)
	}
	
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip: %w", err)
	}
	
	return buf.Bytes(), nil
}

// DecompressEmailBody decompresses compressed email body text
func DecompressEmailBody(compressed []byte) (string, error) {
	if len(compressed) == 0 {
		return "", nil
	}
	
	buf := bytes.NewReader(compressed)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()
	
	decompressed, err := io.ReadAll(gz)
	if err != nil {
		return "", fmt.Errorf("failed to read from gzip: %w", err)
	}
	
	return string(decompressed), nil
}

// CreateMetadataEntry creates an email entry with metadata only (no content)
func (e *EmailStore) CreateMetadataEntry(email *EmailBodyEntry) error {
	// Ensure this is marked as metadata-only phase
	email.ProcessingPhase = "metadata_only"
	email.HasContent = false
	now := time.Now()
	email.MetadataExtractedAt = &now
	
	return e.create(email)
}

// UpdateWithContent updates an existing metadata-only entry with full email content
func (e *EmailStore) UpdateWithContent(gmailMessageID string, bodyText, bodyHTML string, compressed []byte) error {
	now := time.Now()
	query := `UPDATE processed_emails SET 
			  body_text = ?, body_html = ?, body_compressed = ?,
			  processing_phase = 'content_extracted', has_content = TRUE,
			  content_extracted_at = ?, updated_at = CURRENT_TIMESTAMP
			  WHERE gmail_message_id = ?`
	
	result, err := e.db.Exec(query, bodyText, bodyHTML, compressed, now, gmailMessageID)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}

// GetMetadataOnlyEmails retrieves emails that only have metadata (no content downloaded)
func (e *EmailStore) GetMetadataOnlyEmails(limit int) ([]EmailBodyEntry, error) {
	query := `SELECT id, gmail_message_id, gmail_thread_id, sender, subject, date, 
			  body_text, body_html, body_compressed, internal_timestamp, scan_method,
			  processed_at, status, tracking_numbers, error_message, created_at, updated_at,
			  COALESCE(processing_phase, 'legacy') as processing_phase,
			  COALESCE(relevance_score, 0.0) as relevance_score,
			  COALESCE(snippet, '') as snippet,
			  COALESCE(has_content, FALSE) as has_content,
			  metadata_extracted_at, content_extracted_at
			  FROM processed_emails 
			  WHERE processing_phase = 'metadata_only' AND has_content = FALSE
			  ORDER BY relevance_score DESC, date DESC`
	
	args := []interface{}{}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	
	rows, err := e.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var emails []EmailBodyEntry
	for rows.Next() {
		var email EmailBodyEntry
		err := rows.Scan(
			&email.ID, &email.GmailMessageID, &email.GmailThreadID, &email.From,
			&email.Subject, &email.Date, &email.BodyText, &email.BodyHTML,
			&email.BodyCompressed, &email.InternalTimestamp, &email.ScanMethod,
			&email.ProcessedAt, &email.Status, &email.TrackingNumbers,
			&email.ErrorMessage, &email.CreatedAt, &email.UpdatedAt,
			&email.ProcessingPhase, &email.RelevanceScore, &email.Snippet,
			&email.HasContent, &email.MetadataExtractedAt, &email.ContentExtractedAt)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	
	return emails, rows.Err()
}

// GetEmailsByRelevanceScore retrieves emails above a certain relevance threshold
func (e *EmailStore) GetEmailsByRelevanceScore(minScore float64, limit int) ([]EmailBodyEntry, error) {
	query := `SELECT id, gmail_message_id, gmail_thread_id, sender, subject, date, 
			  body_text, body_html, body_compressed, internal_timestamp, scan_method,
			  processed_at, status, tracking_numbers, error_message, created_at, updated_at,
			  COALESCE(processing_phase, 'legacy') as processing_phase,
			  COALESCE(relevance_score, 0.0) as relevance_score,
			  COALESCE(snippet, '') as snippet,
			  COALESCE(has_content, FALSE) as has_content,
			  metadata_extracted_at, content_extracted_at
			  FROM processed_emails 
			  WHERE relevance_score >= ?
			  ORDER BY relevance_score DESC, date DESC`
	
	args := []interface{}{minScore}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	
	rows, err := e.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var emails []EmailBodyEntry
	for rows.Next() {
		var email EmailBodyEntry
		err := rows.Scan(
			&email.ID, &email.GmailMessageID, &email.GmailThreadID, &email.From,
			&email.Subject, &email.Date, &email.BodyText, &email.BodyHTML,
			&email.BodyCompressed, &email.InternalTimestamp, &email.ScanMethod,
			&email.ProcessedAt, &email.Status, &email.TrackingNumbers,
			&email.ErrorMessage, &email.CreatedAt, &email.UpdatedAt,
			&email.ProcessingPhase, &email.RelevanceScore, &email.Snippet,
			&email.HasContent, &email.MetadataExtractedAt, &email.ContentExtractedAt)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	
	return emails, rows.Err()
}

// UpdateRelevanceScore updates the relevance score for an email
func (e *EmailStore) UpdateRelevanceScore(gmailMessageID string, score float64) error {
	query := `UPDATE processed_emails SET relevance_score = ?, updated_at = CURRENT_TIMESTAMP
			  WHERE gmail_message_id = ?`
	
	result, err := e.db.Exec(query, score, gmailMessageID)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}