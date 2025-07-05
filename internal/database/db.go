// Copyright 2024 Package Tracking System
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	RefreshCache   *RefreshCacheStore
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
		RefreshCache:   NewRefreshCacheStore(db),
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
		is_delivered BOOLEAN DEFAULT FALSE,
		last_manual_refresh DATETIME,
		manual_refresh_count INTEGER DEFAULT 0
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

	CREATE TABLE IF NOT EXISTS refresh_cache (
		shipment_id INTEGER PRIMARY KEY,
		response_data TEXT NOT NULL,
		cached_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (shipment_id) REFERENCES shipments(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_shipments_status ON shipments(status);
	CREATE INDEX IF NOT EXISTS idx_shipments_carrier ON shipments(carrier);
	CREATE INDEX IF NOT EXISTS idx_shipments_carrier_delivered ON shipments(carrier, is_delivered);
	CREATE INDEX IF NOT EXISTS idx_tracking_events_shipment ON tracking_events(shipment_id);
	CREATE INDEX IF NOT EXISTS idx_tracking_events_dedup ON tracking_events(shipment_id, timestamp, description);
	CREATE INDEX IF NOT EXISTS idx_refresh_cache_expires ON refresh_cache(expires_at);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Insert default carriers if they don't exist
	if err := db.insertDefaultCarriers(); err != nil {
		return err
	}

	// Run additional migrations for new fields
	if err := db.migrateRefreshFields(); err != nil {
		return err
	}

	// Run auto-refresh field migrations
	if err := db.migrateAutoRefreshFields(); err != nil {
		return err
	}

	// Run Amazon fields migration
	return db.migrateAmazonFields()
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
		// DHL is inactive by default due to strict rate limiting (250 requests/day)
		// and limited geographical coverage compared to other carriers
		{"DHL", "dhl", "https://api.dhl.com/track", false},
		{"Amazon", "amazon", "", true},
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

// migrateRefreshFields adds refresh-related fields to existing databases
func (db *DB) migrateRefreshFields() error {
	// Check if columns already exist
	var columnExists int
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('shipments') 
		WHERE name = 'last_manual_refresh'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check column existence: %w", err)
	}

	// If columns don't exist, add them
	if columnExists == 0 {
		alterQueries := []string{
			"ALTER TABLE shipments ADD COLUMN last_manual_refresh DATETIME",
			"ALTER TABLE shipments ADD COLUMN manual_refresh_count INTEGER DEFAULT 0",
		}

		for _, query := range alterQueries {
			if _, err := db.Exec(query); err != nil {
				return fmt.Errorf("failed to execute migration query '%s': %w", query, err)
			}
		}
	}

	return nil
}

// migrateAutoRefreshFields adds auto-refresh fields to existing databases
func (db *DB) migrateAutoRefreshFields() error {
	// Check if auto-refresh columns already exist
	var columnExists int
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('shipments') 
		WHERE name = 'last_auto_refresh'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check auto-refresh column existence: %w", err)
	}

	// If columns don't exist, add them
	if columnExists == 0 {
		alterQueries := []string{
			"ALTER TABLE shipments ADD COLUMN last_auto_refresh DATETIME",
			"ALTER TABLE shipments ADD COLUMN auto_refresh_count INTEGER DEFAULT 0",
			"ALTER TABLE shipments ADD COLUMN auto_refresh_enabled BOOLEAN DEFAULT TRUE",
			"ALTER TABLE shipments ADD COLUMN auto_refresh_error TEXT",
			"ALTER TABLE shipments ADD COLUMN auto_refresh_fail_count INTEGER DEFAULT 0",
		}

		for _, query := range alterQueries {
			if _, err := db.Exec(query); err != nil {
				return fmt.Errorf("failed to execute auto-refresh migration query '%s': %w", query, err)
			}
		}

		// Add index for auto-update queries
		indexQueries := []string{
			"CREATE INDEX IF NOT EXISTS idx_shipments_auto_update ON shipments(carrier, is_delivered, auto_refresh_enabled, auto_refresh_fail_count, created_at)",
		}

		for _, query := range indexQueries {
			if _, err := db.Exec(query); err != nil {
				return fmt.Errorf("failed to create auto-refresh index '%s': %w", query, err)
			}
		}
	}

	return nil
}

// migrateAmazonFields adds Amazon-related fields to existing databases
func (db *DB) migrateAmazonFields() error {
	// Check if Amazon columns already exist
	var columnExists int
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('shipments') 
		WHERE name = 'amazon_order_number'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check amazon_order_number column existence: %w", err)
	}

	// If columns don't exist, add them
	if columnExists == 0 {
		alterQueries := []string{
			"ALTER TABLE shipments ADD COLUMN amazon_order_number TEXT",
			"ALTER TABLE shipments ADD COLUMN delegated_carrier TEXT",
			"ALTER TABLE shipments ADD COLUMN delegated_tracking_number TEXT",
			"ALTER TABLE shipments ADD COLUMN is_amazon_logistics BOOLEAN DEFAULT FALSE",
		}

		for _, query := range alterQueries {
			if _, err := db.Exec(query); err != nil {
				return fmt.Errorf("failed to execute Amazon migration query '%s': %w", query, err)
			}
		}

		// Add indexes for Amazon fields
		indexQueries := []string{
			"CREATE INDEX IF NOT EXISTS idx_shipments_amazon_order ON shipments(amazon_order_number)",
			"CREATE INDEX IF NOT EXISTS idx_shipments_delegated_tracking ON shipments(delegated_carrier, delegated_tracking_number)",
		}

		for _, query := range indexQueries {
			if _, err := db.Exec(query); err != nil {
				return fmt.Errorf("failed to create Amazon index '%s': %w", query, err)
			}
		}
	}

	return nil
}

// IsHealthy checks if the database connection is healthy
func (db *DB) IsHealthy() error {
	return db.Ping()
}