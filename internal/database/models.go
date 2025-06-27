package database

import (
	"database/sql"
	"time"
)

type Shipment struct {
	ID               int        `json:"id"`
	TrackingNumber   string     `json:"tracking_number"`
	Carrier          string     `json:"carrier"`
	Description      string     `json:"description"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ExpectedDelivery *time.Time `json:"expected_delivery,omitempty"`
	IsDelivered      bool       `json:"is_delivered"`
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
			  created_at, updated_at, expected_delivery, is_delivered 
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
			&shipment.UpdatedAt, &shipment.ExpectedDelivery, &shipment.IsDelivered)
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
			  created_at, updated_at, expected_delivery, is_delivered 
			  FROM shipments WHERE id = ?`
	
	var shipment Shipment
	err := s.db.QueryRow(query, id).Scan(&shipment.ID, &shipment.TrackingNumber,
		&shipment.Carrier, &shipment.Description, &shipment.Status,
		&shipment.CreatedAt, &shipment.UpdatedAt, &shipment.ExpectedDelivery,
		&shipment.IsDelivered)
	
	if err != nil {
		return nil, err
	}
	
	return &shipment, nil
}

// Create creates a new shipment
func (s *ShipmentStore) Create(shipment *Shipment) error {
	query := `INSERT INTO shipments (tracking_number, carrier, description, status, expected_delivery, is_delivered) 
			  VALUES (?, ?, ?, ?, ?, ?)`
	
	result, err := s.db.Exec(query, shipment.TrackingNumber, shipment.Carrier,
		shipment.Description, shipment.Status, shipment.ExpectedDelivery,
		shipment.IsDelivered)
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
	
	return nil
}

// Update updates an existing shipment
func (s *ShipmentStore) Update(id int, shipment *Shipment) error {
	query := `UPDATE shipments SET tracking_number = ?, carrier = ?, description = ?, 
			  status = ?, expected_delivery = ?, is_delivered = ?, updated_at = CURRENT_TIMESTAMP 
			  WHERE id = ?`
	
	result, err := s.db.Exec(query, shipment.TrackingNumber, shipment.Carrier,
		shipment.Description, shipment.Status, shipment.ExpectedDelivery,
		shipment.IsDelivered, id)
	
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