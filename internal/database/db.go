package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the sql.DB connection and provides access to stores
type DB struct {
	*sql.DB
	Shipments      *ShipmentStore
	TrackingEvents *TrackingEventStore
	Carriers       *CarrierStore
}

// Open opens a database connection and initializes stores
func Open(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign key constraints in SQLite
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create the wrapper
	database := &DB{
		DB:             db,
		Shipments:      NewShipmentStore(db),
		TrackingEvents: NewTrackingEventStore(db),
		Carriers:       NewCarrierStore(db),
	}

	// Run migrations
	if err := database.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return database, nil
}

// migrate creates the database schema
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS shipments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tracking_number TEXT NOT NULL UNIQUE,
		carrier TEXT NOT NULL,
		description TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expected_delivery DATETIME,
		is_delivered BOOLEAN DEFAULT FALSE
	);

	CREATE TABLE IF NOT EXISTS tracking_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		shipment_id INTEGER NOT NULL,
		timestamp DATETIME NOT NULL,
		location TEXT,
		status TEXT NOT NULL,
		description TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (shipment_id) REFERENCES shipments(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS carriers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		code TEXT NOT NULL UNIQUE,
		api_endpoint TEXT,
		active BOOLEAN DEFAULT TRUE
	);

	CREATE INDEX IF NOT EXISTS idx_shipments_status ON shipments(status);
	CREATE INDEX IF NOT EXISTS idx_shipments_carrier ON shipments(carrier);
	CREATE INDEX IF NOT EXISTS idx_shipments_carrier_delivered ON shipments(carrier, is_delivered);
	CREATE INDEX IF NOT EXISTS idx_tracking_events_shipment ON tracking_events(shipment_id);
	CREATE INDEX IF NOT EXISTS idx_tracking_events_dedup ON tracking_events(shipment_id, timestamp, description);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Insert default carriers if they don't exist
	return db.insertDefaultCarriers()
}

// insertDefaultCarriers adds default carrier data
func (db *DB) insertDefaultCarriers() error {
	carriers := []struct {
		name        string
		code        string
		apiEndpoint string
		active      bool
	}{
		{"United Parcel Service", "ups", "https://api.ups.com/track", true},
		{"United States Postal Service", "usps", "https://api.usps.com/track", true},
		{"FedEx", "fedex", "https://api.fedex.com/track", true},
		{"DHL", "dhl", "https://api.dhl.com/track", false},
	}

	for _, carrier := range carriers {
		// Check if carrier already exists
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM carriers WHERE code = ?", carrier.code).Scan(&count)
		if err != nil {
			return err
		}

		// Insert if it doesn't exist
		if count == 0 {
			_, err := db.Exec(
				"INSERT INTO carriers (name, code, api_endpoint, active) VALUES (?, ?, ?, ?)",
				carrier.name, carrier.code, carrier.apiEndpoint, carrier.active,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// IsHealthy checks if the database connection is healthy
func (db *DB) IsHealthy() error {
	return db.Ping()
}