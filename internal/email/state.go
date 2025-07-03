package email

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStateManager implements StateManager interface using SQLite
type SQLiteStateManager struct {
	db   *sql.DB
	path string
}

// NewSQLiteStateManager creates a new SQLite-based state manager
func NewSQLiteStateManager(dbPath string) (*SQLiteStateManager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}
	
	// Enable WAL mode for better concurrent access
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	
	// Set reasonable timeouts
	if _, err := db.Exec("PRAGMA busy_timeout=30000"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}
	
	manager := &SQLiteStateManager{
		db:   db,
		path: dbPath,
	}
	
	// Initialize database schema
	if err := manager.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	
	return manager, nil
}

// initSchema creates the necessary tables
func (s *SQLiteStateManager) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS processed_emails (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		gmail_message_id TEXT UNIQUE NOT NULL,
		gmail_thread_id TEXT,
		processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		tracking_numbers TEXT,
		status TEXT NOT NULL,
		sender TEXT,
		subject TEXT,
		error_message TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_gmail_message_id ON processed_emails(gmail_message_id);
	CREATE INDEX IF NOT EXISTS idx_processed_at ON processed_emails(processed_at);
	CREATE INDEX IF NOT EXISTS idx_status ON processed_emails(status);
	CREATE INDEX IF NOT EXISTS idx_sender ON processed_emails(sender);
	
	-- Add trigger to update updated_at
	CREATE TRIGGER IF NOT EXISTS update_processed_emails_updated_at
		AFTER UPDATE ON processed_emails
	BEGIN
		UPDATE processed_emails SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;
	`
	
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	
	return nil
}

// IsProcessed checks if an email has already been processed
func (s *SQLiteStateManager) IsProcessed(messageID string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM processed_emails WHERE gmail_message_id = ?"
	
	err := s.db.QueryRow(query, messageID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if email is processed: %w", err)
	}
	
	return count > 0, nil
}

// MarkProcessed marks an email as processed
func (s *SQLiteStateManager) MarkProcessed(entry *StateEntry) error {
	// Convert tracking numbers to JSON
	trackingJSON, err := json.Marshal(entry.TrackingNumbers)
	if err != nil {
		return fmt.Errorf("failed to marshal tracking numbers: %w", err)
	}
	
	query := `
		INSERT INTO processed_emails (
			gmail_message_id, gmail_thread_id, processed_at, 
			tracking_numbers, status, sender, subject, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(gmail_message_id) DO UPDATE SET
			gmail_thread_id = excluded.gmail_thread_id,
			processed_at = excluded.processed_at,
			tracking_numbers = excluded.tracking_numbers,
			status = excluded.status,
			sender = excluded.sender,
			subject = excluded.subject,
			error_message = excluded.error_message,
			updated_at = CURRENT_TIMESTAMP
	`
	
	_, err = s.db.Exec(query,
		entry.GmailMessageID,
		entry.GmailThreadID,
		entry.ProcessedAt,
		string(trackingJSON),
		entry.Status,
		entry.Sender,
		entry.Subject,
		entry.ErrorMessage,
	)
	
	if err != nil {
		return fmt.Errorf("failed to mark email as processed: %w", err)
	}
	
	return nil
}

// GetEntry retrieves a processed email entry
func (s *SQLiteStateManager) GetEntry(messageID string) (*StateEntry, error) {
	query := `
		SELECT id, gmail_message_id, gmail_thread_id, processed_at,
			   tracking_numbers, status, sender, subject, error_message
		FROM processed_emails 
		WHERE gmail_message_id = ?
	`
	
	var entry StateEntry
	var trackingJSON string
	
	err := s.db.QueryRow(query, messageID).Scan(
		&entry.ID,
		&entry.GmailMessageID,
		&entry.GmailThreadID,
		&entry.ProcessedAt,
		&trackingJSON,
		&entry.Status,
		&entry.Sender,
		&entry.Subject,
		&entry.ErrorMessage,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}
	
	// Parse tracking numbers JSON
	if trackingJSON != "" {
		if err := json.Unmarshal([]byte(trackingJSON), &entry.TrackingNumbers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tracking numbers: %w", err)
		}
	}
	
	return &entry, nil
}

// GetRecentEntries retrieves recent processed emails
func (s *SQLiteStateManager) GetRecentEntries(limit int) ([]StateEntry, error) {
	query := `
		SELECT id, gmail_message_id, gmail_thread_id, processed_at,
			   tracking_numbers, status, sender, subject, error_message
		FROM processed_emails 
		ORDER BY processed_at DESC
		LIMIT ?
	`
	
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent entries: %w", err)
	}
	defer rows.Close()
	
	var entries []StateEntry
	for rows.Next() {
		var entry StateEntry
		var trackingJSON string
		
		err := rows.Scan(
			&entry.ID,
			&entry.GmailMessageID,
			&entry.GmailThreadID,
			&entry.ProcessedAt,
			&trackingJSON,
			&entry.Status,
			&entry.Sender,
			&entry.Subject,
			&entry.ErrorMessage,
		)
		
		if err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}
		
		// Parse tracking numbers JSON
		if trackingJSON != "" {
			if err := json.Unmarshal([]byte(trackingJSON), &entry.TrackingNumbers); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tracking numbers: %w", err)
			}
		}
		
		entries = append(entries, entry)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	
	return entries, nil
}

// Cleanup removes old processed email entries
func (s *SQLiteStateManager) Cleanup(olderThan time.Time) error {
	query := "DELETE FROM processed_emails WHERE processed_at < ?"
	
	result, err := s.db.Exec(query, olderThan)
	if err != nil {
		return fmt.Errorf("failed to cleanup old entries: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected > 0 {
		// Run VACUUM to reclaim space
		if _, err := s.db.Exec("VACUUM"); err != nil {
			// Log but don't fail - VACUUM is optimization
			fmt.Printf("Warning: failed to vacuum database: %v\n", err)
		}
	}
	
	return nil
}

// GetStats returns processing statistics
func (s *SQLiteStateManager) GetStats() (*EmailMetrics, error) {
	metrics := &EmailMetrics{}
	
	// Get total processed emails
	err := s.db.QueryRow("SELECT COUNT(*) FROM processed_emails").Scan(&metrics.ProcessedEmails)
	if err != nil {
		return nil, fmt.Errorf("failed to get total processed emails: %w", err)
	}
	
	// Get count by status
	statusQuery := `
		SELECT status, COUNT(*) 
		FROM processed_emails 
		GROUP BY status
	`
	rows, err := s.db.Query(statusQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query status counts: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var status string
		var count int
		
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status count: %w", err)
		}
		
		switch status {
		case "processed":
			// Already counted above
		case "skipped":
			metrics.SkippedEmails = count
		case "error":
			metrics.ErrorEmails = count
		}
	}
	
	// Get tracking numbers found (approximate)
	trackingQuery := `
		SELECT COUNT(*) 
		FROM processed_emails 
		WHERE status = 'processed' AND tracking_numbers != '' AND tracking_numbers != '[]'
	`
	err = s.db.QueryRow(trackingQuery).Scan(&metrics.TrackingnumbersFound)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracking numbers count: %w", err)
	}
	
	// Get last processed time
	var lastProcessed sql.NullTime
	err = s.db.QueryRow("SELECT MAX(processed_at) FROM processed_emails").Scan(&lastProcessed)
	if err != nil {
		return nil, fmt.Errorf("failed to get last processed time: %w", err)
	}
	
	if lastProcessed.Valid {
		metrics.LastProcessed = lastProcessed.Time
	}
	
	// Total emails is processed + skipped + error
	metrics.TotalEmails = metrics.ProcessedEmails + metrics.SkippedEmails + metrics.ErrorEmails
	
	return metrics, nil
}

// GetStatsByDateRange returns statistics for a specific date range
func (s *SQLiteStateManager) GetStatsByDateRange(start, end time.Time) (*EmailMetrics, error) {
	metrics := &EmailMetrics{}
	
	// Base query with date filter
	baseQuery := `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN status = 'processed' THEN 1 ELSE 0 END) as processed,
			SUM(CASE WHEN status = 'skipped' THEN 1 ELSE 0 END) as skipped,
			SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as errors,
			SUM(CASE WHEN status = 'processed' AND tracking_numbers != '' AND tracking_numbers != '[]' THEN 1 ELSE 0 END) as with_tracking
		FROM processed_emails 
		WHERE processed_at BETWEEN ? AND ?
	`
	
	err := s.db.QueryRow(baseQuery, start, end).Scan(
		&metrics.TotalEmails,
		&metrics.ProcessedEmails,
		&metrics.SkippedEmails,
		&metrics.ErrorEmails,
		&metrics.TrackingnumbersFound,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get stats by date range: %w", err)
	}
	
	return metrics, nil
}

// UpdateEntry updates an existing entry
func (s *SQLiteStateManager) UpdateEntry(messageID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	
	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	
	for field, value := range updates {
		setParts = append(setParts, field+" = ?")
		args = append(args, value)
	}
	
	// Add message ID for WHERE clause
	args = append(args, messageID)
	
	query := fmt.Sprintf(
		"UPDATE processed_emails SET %s, updated_at = CURRENT_TIMESTAMP WHERE gmail_message_id = ?",
		fmt.Sprintf("%s", setParts[0]), // Handle single element
	)
	
	if len(setParts) > 1 {
		query = fmt.Sprintf(
			"UPDATE processed_emails SET %s, updated_at = CURRENT_TIMESTAMP WHERE gmail_message_id = ?",
			fmt.Sprintf("%s", setParts),
		)
	}
	
	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update entry: %w", err)
	}
	
	return nil
}

// Close closes the database connection
func (s *SQLiteStateManager) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// GetDatabasePath returns the database file path
func (s *SQLiteStateManager) GetDatabasePath() string {
	return s.path
}

// ExportEntries exports processed emails to JSON
func (s *SQLiteStateManager) ExportEntries(filename string) error {
	entries, err := s.GetRecentEntries(10000) // Get all entries
	if err != nil {
		return fmt.Errorf("failed to get entries for export: %w", err)
	}
	
	_, err = json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entries: %w", err)
	}
	
	// Write to file (using fmt for simplicity)
	return fmt.Errorf("file writing not implemented in this example")
}

// Optimize runs database optimization
func (s *SQLiteStateManager) Optimize() error {
	// Analyze tables for query optimization
	if _, err := s.db.Exec("ANALYZE"); err != nil {
		return fmt.Errorf("failed to analyze database: %w", err)
	}
	
	// Check integrity
	var integrity string
	err := s.db.QueryRow("PRAGMA integrity_check").Scan(&integrity)
	if err != nil {
		return fmt.Errorf("failed to check integrity: %w", err)
	}
	
	if integrity != "ok" {
		return fmt.Errorf("database integrity check failed: %s", integrity)
	}
	
	return nil
}