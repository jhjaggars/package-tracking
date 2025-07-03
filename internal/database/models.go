package database

import (
	"database/sql"
	"fmt"
	"time"
)

type Shipment struct {
	ID                  int        `json:"id"`
	TrackingNumber      string     `json:"tracking_number"`
	Carrier             string     `json:"carrier"`
	Description         string     `json:"description"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	ExpectedDelivery    *time.Time `json:"expected_delivery,omitempty"`
	IsDelivered         bool       `json:"is_delivered"`
	LastManualRefresh   *time.Time `json:"last_manual_refresh,omitempty"`
	ManualRefreshCount  int        `json:"manual_refresh_count"`
	LastAutoRefresh     *time.Time `json:"last_auto_refresh,omitempty"`
	AutoRefreshCount    int        `json:"auto_refresh_count"`
	AutoRefreshEnabled  bool       `json:"auto_refresh_enabled"`
	AutoRefreshError    *string    `json:"auto_refresh_error,omitempty"`
	AutoRefreshFailCount int       `json:"auto_refresh_fail_count"`
}

type TrackingEvent struct {
	ID          int       `json:"id"`
	ShipmentID  int       `json:"shipment_id"`
	Timestamp   time.Time `json:"timestamp"`
	Location    string    `json:"location"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Carrier struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	APIEndpoint string `json:"api_endpoint"`
	Active      bool   `json:"active"`
}

// ShipmentStore handles database operations for shipments
type ShipmentStore struct {
	db *sql.DB
}

func NewShipmentStore(db *sql.DB) *ShipmentStore {
	return &ShipmentStore{db: db}
}

// GetAll returns all shipments
func (s *ShipmentStore) GetAll() ([]Shipment, error) {
	query := `SELECT id, tracking_number, carrier, description, status, 
			  created_at, updated_at, expected_delivery, is_delivered,
			  last_manual_refresh, manual_refresh_count, last_auto_refresh,
			  auto_refresh_count, auto_refresh_enabled, auto_refresh_error,
			  auto_refresh_fail_count 
			  FROM shipments ORDER BY created_at DESC`
	
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shipments []Shipment
	for rows.Next() {
		var shipment Shipment
		err := rows.Scan(&shipment.ID, &shipment.TrackingNumber, &shipment.Carrier,
			&shipment.Description, &shipment.Status, &shipment.CreatedAt,
			&shipment.UpdatedAt, &shipment.ExpectedDelivery, &shipment.IsDelivered,
			&shipment.LastManualRefresh, &shipment.ManualRefreshCount,
			&shipment.LastAutoRefresh, &shipment.AutoRefreshCount,
			&shipment.AutoRefreshEnabled, &shipment.AutoRefreshError,
			&shipment.AutoRefreshFailCount)
		if err != nil {
			return nil, err
		}
		shipments = append(shipments, shipment)
	}

	return shipments, rows.Err()
}

// GetActiveByCarrier returns all active (non-delivered) shipments for a specific carrier
func (s *ShipmentStore) GetActiveByCarrier(carrier string) ([]Shipment, error) {
	query := `SELECT id, tracking_number, carrier, description, status, 
			  created_at, updated_at, expected_delivery, is_delivered,
			  last_manual_refresh, manual_refresh_count, last_auto_refresh,
			  auto_refresh_count, auto_refresh_enabled, auto_refresh_error,
			  auto_refresh_fail_count 
			  FROM shipments WHERE is_delivered = false AND carrier = ? ORDER BY created_at DESC`
	
	rows, err := s.db.Query(query, carrier)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shipments []Shipment
	for rows.Next() {
		var shipment Shipment
		err := rows.Scan(&shipment.ID, &shipment.TrackingNumber, &shipment.Carrier,
			&shipment.Description, &shipment.Status, &shipment.CreatedAt,
			&shipment.UpdatedAt, &shipment.ExpectedDelivery, &shipment.IsDelivered,
			&shipment.LastManualRefresh, &shipment.ManualRefreshCount,
			&shipment.LastAutoRefresh, &shipment.AutoRefreshCount,
			&shipment.AutoRefreshEnabled, &shipment.AutoRefreshError,
			&shipment.AutoRefreshFailCount)
		if err != nil {
			return nil, err
		}
		shipments = append(shipments, shipment)
	}

	return shipments, rows.Err()
}

// GetByID returns a shipment by ID
func (s *ShipmentStore) GetByID(id int) (*Shipment, error) {
	query := `SELECT id, tracking_number, carrier, description, status, 
			  created_at, updated_at, expected_delivery, is_delivered,
			  last_manual_refresh, manual_refresh_count, last_auto_refresh,
			  auto_refresh_count, auto_refresh_enabled, auto_refresh_error,
			  auto_refresh_fail_count 
			  FROM shipments WHERE id = ?`
	
	var shipment Shipment
	err := s.db.QueryRow(query, id).Scan(&shipment.ID, &shipment.TrackingNumber,
		&shipment.Carrier, &shipment.Description, &shipment.Status,
		&shipment.CreatedAt, &shipment.UpdatedAt, &shipment.ExpectedDelivery,
		&shipment.IsDelivered, &shipment.LastManualRefresh, &shipment.ManualRefreshCount,
		&shipment.LastAutoRefresh, &shipment.AutoRefreshCount,
		&shipment.AutoRefreshEnabled, &shipment.AutoRefreshError,
		&shipment.AutoRefreshFailCount)
	
	if err != nil {
		return nil, err
	}
	
	return &shipment, nil
}

// Create creates a new shipment
func (s *ShipmentStore) Create(shipment *Shipment) error {
	// Set default values for auto-refresh fields if not already set
	if !shipment.AutoRefreshEnabled {
		shipment.AutoRefreshEnabled = true // Default to enabled
	}
	
	query := `INSERT INTO shipments (tracking_number, carrier, description, status, expected_delivery, is_delivered, manual_refresh_count, auto_refresh_count, auto_refresh_enabled, auto_refresh_fail_count) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	result, err := s.db.Exec(query, shipment.TrackingNumber, shipment.Carrier,
		shipment.Description, shipment.Status, shipment.ExpectedDelivery,
		shipment.IsDelivered, shipment.ManualRefreshCount, shipment.AutoRefreshCount,
		shipment.AutoRefreshEnabled, shipment.AutoRefreshFailCount)
	if err != nil {
		return err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	
	shipment.ID = int(id)
	
	// Get the created shipment to populate timestamps
	created, err := s.GetByID(shipment.ID)
	if err != nil {
		return err
	}
	
	shipment.CreatedAt = created.CreatedAt
	shipment.UpdatedAt = created.UpdatedAt
	shipment.LastManualRefresh = created.LastManualRefresh
	shipment.ManualRefreshCount = created.ManualRefreshCount
	shipment.LastAutoRefresh = created.LastAutoRefresh
	shipment.AutoRefreshCount = created.AutoRefreshCount
	shipment.AutoRefreshEnabled = created.AutoRefreshEnabled
	shipment.AutoRefreshError = created.AutoRefreshError
	shipment.AutoRefreshFailCount = created.AutoRefreshFailCount
	
	return nil
}

// Update updates an existing shipment
func (s *ShipmentStore) Update(id int, shipment *Shipment) error {
	query := `UPDATE shipments SET tracking_number = ?, carrier = ?, description = ?, 
			  status = ?, expected_delivery = ?, is_delivered = ?, last_manual_refresh = ?, 
			  manual_refresh_count = ?, last_auto_refresh = ?, auto_refresh_count = ?,
			  auto_refresh_enabled = ?, auto_refresh_error = ?, auto_refresh_fail_count = ?,
			  updated_at = CURRENT_TIMESTAMP 
			  WHERE id = ?`
	
	result, err := s.db.Exec(query, shipment.TrackingNumber, shipment.Carrier,
		shipment.Description, shipment.Status, shipment.ExpectedDelivery,
		shipment.IsDelivered, shipment.LastManualRefresh, shipment.ManualRefreshCount,
		shipment.LastAutoRefresh, shipment.AutoRefreshCount, shipment.AutoRefreshEnabled,
		shipment.AutoRefreshError, shipment.AutoRefreshFailCount, id)
	
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
	
	// Update the shipment with new data
	updatedShipment, err := s.GetByID(id)
	if err != nil {
		return err
	}
	
	*shipment = *updatedShipment
	return nil
}

// Delete deletes a shipment by ID
func (s *ShipmentStore) Delete(id int) error {
	query := `DELETE FROM shipments WHERE id = ?`
	
	result, err := s.db.Exec(query, id)
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

// DashboardStats represents aggregated statistics for the dashboard
type DashboardStats struct {
	TotalShipments      int `json:"total_shipments"`
	ActiveShipments     int `json:"active_shipments"`
	InTransit           int `json:"in_transit"`
	Delivered           int `json:"delivered"`
	RequiringAttention  int `json:"requiring_attention"`
}

// GetStats returns aggregated statistics for the dashboard
func (s *ShipmentStore) GetStats() (*DashboardStats, error) {
	stats := &DashboardStats{}
	
	// Get total shipments
	err := s.db.QueryRow("SELECT COUNT(*) FROM shipments").Scan(&stats.TotalShipments)
	if err != nil {
		return nil, err
	}
	
	// Get active shipments (not delivered)
	err = s.db.QueryRow("SELECT COUNT(*) FROM shipments WHERE is_delivered = 0").Scan(&stats.ActiveShipments)
	if err != nil {
		return nil, err
	}
	
	// Get in transit shipments
	err = s.db.QueryRow("SELECT COUNT(*) FROM shipments WHERE status = 'in_transit'").Scan(&stats.InTransit)
	if err != nil {
		return nil, err
	}
	
	// Get delivered shipments
	err = s.db.QueryRow("SELECT COUNT(*) FROM shipments WHERE is_delivered = 1").Scan(&stats.Delivered)
	if err != nil {
		return nil, err
	}
	
	// Get shipments requiring attention (exceptions)
	err = s.db.QueryRow("SELECT COUNT(*) FROM shipments WHERE status = 'exception'").Scan(&stats.RequiringAttention)
	if err != nil {
		return nil, err
	}
	
	return stats, nil
}

// UpdateRefreshTracking updates the last_manual_refresh timestamp and increments the count
func (s *ShipmentStore) UpdateRefreshTracking(id int) error {
	query := `UPDATE shipments SET 
			  last_manual_refresh = CURRENT_TIMESTAMP,
			  manual_refresh_count = manual_refresh_count + 1,
			  updated_at = CURRENT_TIMESTAMP 
			  WHERE id = ?`
	
	result, err := s.db.Exec(query, id)
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

// GetActiveForAutoUpdate returns active shipments for auto-update within cutoff date
func (s *ShipmentStore) GetActiveForAutoUpdate(carrier string, cutoffDate time.Time, failureThreshold int) ([]Shipment, error) {
	query := `SELECT id, tracking_number, carrier, description, status, 
			  created_at, updated_at, expected_delivery, is_delivered,
			  last_manual_refresh, manual_refresh_count, last_auto_refresh,
			  auto_refresh_count, auto_refresh_enabled, auto_refresh_error,
			  auto_refresh_fail_count 
			  FROM shipments 
			  WHERE is_delivered = false 
			  AND carrier = ? 
			  AND created_at > ?
			  AND auto_refresh_enabled = true
			  AND auto_refresh_fail_count < ?
			  ORDER BY created_at DESC`
	
	rows, err := s.db.Query(query, carrier, cutoffDate, failureThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shipments []Shipment
	for rows.Next() {
		var shipment Shipment
		err := rows.Scan(&shipment.ID, &shipment.TrackingNumber, &shipment.Carrier,
			&shipment.Description, &shipment.Status, &shipment.CreatedAt,
			&shipment.UpdatedAt, &shipment.ExpectedDelivery, &shipment.IsDelivered,
			&shipment.LastManualRefresh, &shipment.ManualRefreshCount,
			&shipment.LastAutoRefresh, &shipment.AutoRefreshCount,
			&shipment.AutoRefreshEnabled, &shipment.AutoRefreshError,
			&shipment.AutoRefreshFailCount)
		if err != nil {
			return nil, err
		}
		shipments = append(shipments, shipment)
	}

	return shipments, rows.Err()
}

// UpdateAutoRefreshTracking updates auto-refresh tracking fields
func (s *ShipmentStore) UpdateAutoRefreshTracking(id int64, success bool, errorMsg string) error {
	var query string
	var args []interface{}
	
	if success {
		// Reset fail count on success
		query = `UPDATE shipments SET 
				 last_auto_refresh = CURRENT_TIMESTAMP,
				 auto_refresh_count = auto_refresh_count + 1,
				 auto_refresh_fail_count = 0,
				 auto_refresh_error = NULL,
				 updated_at = CURRENT_TIMESTAMP 
				 WHERE id = ?`
		args = []interface{}{id}
	} else {
		// Increment fail count on failure
		query = `UPDATE shipments SET 
				 auto_refresh_fail_count = auto_refresh_fail_count + 1,
				 auto_refresh_error = ?,
				 updated_at = CURRENT_TIMESTAMP 
				 WHERE id = ?`
		args = []interface{}{errorMsg, id}
	}
	
	result, err := s.db.Exec(query, args...)
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

// UpdateShipmentWithAutoRefresh atomically updates shipment data and auto-refresh tracking
// This prevents race conditions between the two separate update operations
func (s *ShipmentStore) UpdateShipmentWithAutoRefresh(id int, shipment *Shipment, success bool, errorMsg string) error {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Will be ignored if tx.Commit() succeeds

	// Update main shipment data
	updateQuery := `UPDATE shipments SET tracking_number = ?, carrier = ?, description = ?, 
			  status = ?, expected_delivery = ?, is_delivered = ?, last_manual_refresh = ?, 
			  manual_refresh_count = ?, last_auto_refresh = ?, auto_refresh_count = ?,
			  auto_refresh_enabled = ?, auto_refresh_error = ?, auto_refresh_fail_count = ?,
			  updated_at = CURRENT_TIMESTAMP 
			  WHERE id = ?`
	
	result, err := tx.Exec(updateQuery, shipment.TrackingNumber, shipment.Carrier,
		shipment.Description, shipment.Status, shipment.ExpectedDelivery,
		shipment.IsDelivered, shipment.LastManualRefresh, shipment.ManualRefreshCount,
		shipment.LastAutoRefresh, shipment.AutoRefreshCount, shipment.AutoRefreshEnabled,
		shipment.AutoRefreshError, shipment.AutoRefreshFailCount, id)
	
	if err != nil {
		return fmt.Errorf("failed to update shipment: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Update auto-refresh tracking fields
	var trackingQuery string
	var trackingArgs []interface{}
	
	if success {
		// Reset fail count on success
		trackingQuery = `UPDATE shipments SET 
				 last_auto_refresh = CURRENT_TIMESTAMP,
				 auto_refresh_count = auto_refresh_count + 1,
				 auto_refresh_fail_count = 0,
				 auto_refresh_error = NULL,
				 updated_at = CURRENT_TIMESTAMP 
				 WHERE id = ?`
		trackingArgs = []interface{}{id}
	} else {
		// Increment fail count on failure
		trackingQuery = `UPDATE shipments SET 
				 auto_refresh_fail_count = auto_refresh_fail_count + 1,
				 auto_refresh_error = ?,
				 updated_at = CURRENT_TIMESTAMP 
				 WHERE id = ?`
		trackingArgs = []interface{}{errorMsg, id}
	}
	
	_, err = tx.Exec(trackingQuery, trackingArgs...)
	if err != nil {
		return fmt.Errorf("failed to update auto-refresh tracking: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ResetAutoRefreshFailCount resets the auto-refresh fail count for a shipment
func (s *ShipmentStore) ResetAutoRefreshFailCount(id int64) error {
	query := `UPDATE shipments SET 
			  auto_refresh_fail_count = 0,
			  auto_refresh_error = NULL,
			  updated_at = CURRENT_TIMESTAMP 
			  WHERE id = ?`
	
	result, err := s.db.Exec(query, id)
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

// TrackingEventStore handles database operations for tracking events
type TrackingEventStore struct {
	db *sql.DB
}

func NewTrackingEventStore(db *sql.DB) *TrackingEventStore {
	return &TrackingEventStore{db: db}
}

// GetByShipmentID returns all tracking events for a shipment
func (t *TrackingEventStore) GetByShipmentID(shipmentID int) ([]TrackingEvent, error) {
	query := `SELECT id, shipment_id, timestamp, location, status, description, created_at 
			  FROM tracking_events WHERE shipment_id = ? ORDER BY timestamp ASC`
	
	rows, err := t.db.Query(query, shipmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []TrackingEvent
	for rows.Next() {
		var event TrackingEvent
		err := rows.Scan(&event.ID, &event.ShipmentID, &event.Timestamp,
			&event.Location, &event.Status, &event.Description, &event.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// CreateEvent creates a new tracking event if it doesn't already exist
func (t *TrackingEventStore) CreateEvent(event *TrackingEvent) error {
	// Use a transaction to make deduplication atomic
	tx, err := t.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Will be ignored if tx.Commit() succeeds
	
	// Check if event already exists (deduplication)
	var count int
	checkQuery := `SELECT COUNT(*) FROM tracking_events 
				   WHERE shipment_id = ? AND timestamp = ? AND description = ?`
	err = tx.QueryRow(checkQuery, event.ShipmentID, event.Timestamp, event.Description).Scan(&count)
	if err != nil {
		return err
	}
	
	// Skip if event already exists
	if count > 0 {
		return tx.Commit() // Commit empty transaction
	}
	
	// Insert new event
	query := `INSERT INTO tracking_events (shipment_id, timestamp, location, status, description, created_at) 
			  VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`
	
	result, err := tx.Exec(query, event.ShipmentID, event.Timestamp, 
		event.Location, event.Status, event.Description)
	if err != nil {
		return err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	
	event.ID = int(id)
	// Get the actual created_at timestamp from database
	err = tx.QueryRow("SELECT created_at FROM tracking_events WHERE id = ?", event.ID).Scan(&event.CreatedAt)
	if err != nil {
		return err
	}
	
	return tx.Commit()
}

// CarrierStore handles database operations for carriers
type CarrierStore struct {
	db *sql.DB
}

func NewCarrierStore(db *sql.DB) *CarrierStore {
	return &CarrierStore{db: db}
}

// GetAll returns all carriers
func (c *CarrierStore) GetAll(activeOnly bool) ([]Carrier, error) {
	query := `SELECT id, name, code, api_endpoint, active FROM carriers`
	if activeOnly {
		query += ` WHERE active = true`
	}
	query += ` ORDER BY name`
	
	rows, err := c.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var carriers []Carrier
	for rows.Next() {
		var carrier Carrier
		err := rows.Scan(&carrier.ID, &carrier.Name, &carrier.Code,
			&carrier.APIEndpoint, &carrier.Active)
		if err != nil {
			return nil, err
		}
		carriers = append(carriers, carrier)
	}

	return carriers, rows.Err()
}