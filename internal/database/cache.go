package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// RefreshCacheEntry represents a cached refresh response
type RefreshCacheEntry struct {
	ShipmentID   int       `json:"shipment_id"`
	ResponseData string    `json:"response_data"`
	CachedAt     time.Time `json:"cached_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// RefreshResponse represents the response from a manual refresh request
// This is duplicated from handlers package to avoid circular imports
type RefreshResponse struct {
	ShipmentID  int            `json:"shipment_id"`
	UpdatedAt   time.Time      `json:"updated_at"`
	EventsAdded int            `json:"events_added"`
	TotalEvents int            `json:"total_events"`
	Events      []TrackingEvent `json:"events"`
}

// RefreshCacheStore handles database operations for refresh cache
type RefreshCacheStore struct {
	db *sql.DB
}

// NewRefreshCacheStore creates a new refresh cache store
func NewRefreshCacheStore(db *sql.DB) *RefreshCacheStore {
	return &RefreshCacheStore{db: db}
}

// Get retrieves a cached refresh response for a shipment
func (r *RefreshCacheStore) Get(shipmentID int) (*RefreshResponse, error) {
	query := `SELECT response_data, expires_at FROM refresh_cache WHERE shipment_id = ?`
	
	var responseData string
	var expiresAt time.Time
	
	err := r.db.QueryRow(query, shipmentID).Scan(&responseData, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get cached response: %w", err)
	}
	
	// Check if entry has expired
	if time.Now().After(expiresAt) {
		// Delete expired entry and return cache miss
		r.Delete(shipmentID)
		return nil, nil
	}
	
	// Deserialize the response
	var response RefreshResponse
	if err := json.Unmarshal([]byte(responseData), &response); err != nil {
		return nil, fmt.Errorf("failed to deserialize cached response: %w", err)
	}
	
	return &response, nil
}

// Set stores a refresh response in the cache with the specified TTL
func (r *RefreshCacheStore) Set(shipmentID int, response *RefreshResponse, ttl time.Duration) error {
	// Serialize the response
	responseData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to serialize response: %w", err)
	}
	
	expiresAt := time.Now().Add(ttl)
	
	query := `INSERT OR REPLACE INTO refresh_cache (shipment_id, response_data, cached_at, expires_at) 
			  VALUES (?, ?, CURRENT_TIMESTAMP, ?)`
	
	_, err = r.db.Exec(query, shipmentID, string(responseData), expiresAt)
	if err != nil {
		return fmt.Errorf("failed to cache response: %w", err)
	}
	
	return nil
}

// Delete removes a cached entry for a shipment
func (r *RefreshCacheStore) Delete(shipmentID int) error {
	query := `DELETE FROM refresh_cache WHERE shipment_id = ?`
	
	_, err := r.db.Exec(query, shipmentID)
	if err != nil {
		return fmt.Errorf("failed to delete cached entry: %w", err)
	}
	
	return nil
}

// DeleteExpired removes all expired cache entries
func (r *RefreshCacheStore) DeleteExpired() error {
	query := `DELETE FROM refresh_cache WHERE expires_at <= ?`
	
	result, err := r.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired entries: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected > 0 {
		// Log cleanup if rows were deleted (optional)
		fmt.Printf("DEBUG: Cleaned up %d expired cache entries\n", rowsAffected)
	}
	
	return nil
}

// LoadAll loads all non-expired cache entries from the database
// Used for initializing in-memory cache on startup
func (r *RefreshCacheStore) LoadAll() (map[int]*RefreshResponse, error) {
	query := `SELECT shipment_id, response_data, expires_at FROM refresh_cache WHERE expires_at > ?`
	
	rows, err := r.db.Query(query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to load cache entries: %w", err)
	}
	defer rows.Close()
	
	cache := make(map[int]*RefreshResponse)
	
	for rows.Next() {
		var shipmentID int
		var responseData string
		var expiresAt time.Time
		
		err := rows.Scan(&shipmentID, &responseData, &expiresAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan cache entry: %w", err)
		}
		
		// Deserialize the response
		var response RefreshResponse
		if err := json.Unmarshal([]byte(responseData), &response); err != nil {
			// Log error and continue with other entries
			fmt.Printf("WARN: Failed to deserialize cached response for shipment %d: %v\n", shipmentID, err)
			continue
		}
		
		cache[shipmentID] = &response
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cache entries: %w", err)
	}
	
	return cache, nil
}

// GetStats returns cache statistics
func (r *RefreshCacheStore) GetStats() (int, int, error) {
	var total, expired int
	
	// Get total entries
	err := r.db.QueryRow("SELECT COUNT(*) FROM refresh_cache").Scan(&total)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get total cache entries: %w", err)
	}
	
	// Get expired entries
	err = r.db.QueryRow("SELECT COUNT(*) FROM refresh_cache WHERE expires_at <= ?", time.Now()).Scan(&expired)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get expired cache entries: %w", err)
	}
	
	return total, expired, nil
}