package cache

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"package-tracking/internal/database"
)

// CachedResponse represents an in-memory cached response with expiry
type CachedResponse struct {
	Response  *database.RefreshResponse
	ExpiresAt time.Time
}

// IsExpired checks if the cached response has expired
func (c *CachedResponse) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// Manager manages both in-memory and persistent caching for refresh responses
type Manager struct {
	store    *database.RefreshCacheStore
	memory   sync.Map // map[int]*CachedResponse
	disabled bool
	ttl      time.Duration
	
	// Cleanup goroutine control
	ctx    context.Context
	cancel context.CancelFunc
}

// NewManager creates a new cache manager
func NewManager(store *database.RefreshCacheStore, disabled bool, ttl time.Duration) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &Manager{
		store:    store,
		disabled: disabled,
		ttl:      ttl,
		ctx:      ctx,
		cancel:   cancel,
	}
	
	if !disabled {
		// Load existing cache entries from database
		if err := manager.loadFromDatabase(); err != nil {
			log.Printf("WARN: Failed to load cache from database: %v", err)
		}
		
		// Start cleanup goroutine
		go manager.cleanupLoop()
	}
	
	return manager
}

// Get retrieves a cached refresh response
func (m *Manager) Get(shipmentID int) (*database.RefreshResponse, error) {
	if m.disabled {
		return nil, nil // Cache disabled, always miss
	}
	
	// Check in-memory cache first
	if value, ok := m.memory.Load(shipmentID); ok {
		cached := value.(*CachedResponse)
		if !cached.IsExpired() {
			return cached.Response, nil
		}
		// Remove expired entry from memory
		m.memory.Delete(shipmentID)
	}
	
	// Check database cache
	response, err := m.store.Get(shipmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get from database cache: %w", err)
	}
	
	if response != nil {
		// Store in memory for faster access next time
		cached := &CachedResponse{
			Response:  response,
			ExpiresAt: time.Now().Add(m.ttl),
		}
		m.memory.Store(shipmentID, cached)
	}
	
	return response, nil
}

// Set stores a refresh response in both memory and database
func (m *Manager) Set(shipmentID int, response *database.RefreshResponse) error {
	if m.disabled {
		return nil // Cache disabled, do nothing
	}
	
	// Store in database first
	if err := m.store.Set(shipmentID, response, m.ttl); err != nil {
		return fmt.Errorf("failed to store in database cache: %w", err)
	}
	
	// Store in memory
	cached := &CachedResponse{
		Response:  response,
		ExpiresAt: time.Now().Add(m.ttl),
	}
	m.memory.Store(shipmentID, cached)
	
	return nil
}

// Delete removes a cached response from both memory and database
func (m *Manager) Delete(shipmentID int) error {
	if m.disabled {
		return nil // Cache disabled, do nothing
	}
	
	// Remove from memory
	m.memory.Delete(shipmentID)
	
	// Remove from database
	if err := m.store.Delete(shipmentID); err != nil {
		return fmt.Errorf("failed to delete from database cache: %w", err)
	}
	
	return nil
}

// ForceInvalidate removes a cached response to force a fresh fetch
// Returns the age of the cache entry that was invalidated, or nil if no cache existed
func (m *Manager) ForceInvalidate(shipmentID int) (*time.Duration, error) {
	if m.disabled {
		return nil, nil // Cache disabled, nothing to invalidate
	}
	
	var cacheAge *time.Duration
	
	// Check if there was a cache entry and get its age
	if value, ok := m.memory.Load(shipmentID); ok {
		cached := value.(*CachedResponse)
		age := time.Since(cached.Response.UpdatedAt)
		cacheAge = &age
	} else {
		// Check database for cache age
		response, err := m.store.Get(shipmentID)
		if err != nil {
			return nil, fmt.Errorf("failed to check database cache age: %w", err)
		}
		if response != nil {
			age := time.Since(response.UpdatedAt)
			cacheAge = &age
		}
	}
	
	// Delete the cache entry
	if err := m.Delete(shipmentID); err != nil {
		return cacheAge, fmt.Errorf("failed to invalidate cache: %w", err)
	}
	
	return cacheAge, nil
}

// IsEnabled returns true if caching is enabled
func (m *Manager) IsEnabled() bool {
	return !m.disabled
}

// GetTTL returns the cache TTL duration
func (m *Manager) GetTTL() time.Duration {
	return m.ttl
}

// loadFromDatabase loads all non-expired cache entries from database into memory
func (m *Manager) loadFromDatabase() error {
	entries, err := m.store.LoadAll()
	if err != nil {
		return err
	}
	
	loaded := 0
	for shipmentID, response := range entries {
		cached := &CachedResponse{
			Response:  response,
			ExpiresAt: time.Now().Add(m.ttl), // Reset TTL from current time
		}
		m.memory.Store(shipmentID, cached)
		loaded++
	}
	
	if loaded > 0 {
		log.Printf("INFO: Loaded %d cache entries from database", loaded)
	}
	
	return nil
}

// cleanupLoop runs periodically to clean up expired entries
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute) // Cleanup every minute
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.cleanup()
		}
	}
}

// cleanup removes expired entries from both memory and database
func (m *Manager) cleanup() {
	// Clean up memory
	memoryCount := 0
	m.memory.Range(func(key, value interface{}) bool {
		cached := value.(*CachedResponse)
		if cached.IsExpired() {
			m.memory.Delete(key)
			memoryCount++
		}
		return true
	})
	
	// Clean up database
	if err := m.store.DeleteExpired(); err != nil {
		log.Printf("WARN: Failed to clean up expired database cache entries: %v", err)
	}
	
	if memoryCount > 0 {
		log.Printf("DEBUG: Cleaned up %d expired memory cache entries", memoryCount)
	}
}

// GetStats returns cache statistics
func (m *Manager) GetStats() (CacheStats, error) {
	stats := CacheStats{
		Disabled: m.disabled,
		TTL:      m.ttl,
	}
	
	if m.disabled {
		return stats, nil
	}
	
	// Count memory entries
	memoryTotal := 0
	memoryExpired := 0
	m.memory.Range(func(key, value interface{}) bool {
		memoryTotal++
		cached := value.(*CachedResponse)
		if cached.IsExpired() {
			memoryExpired++
		}
		return true
	})
	
	stats.MemoryTotal = memoryTotal
	stats.MemoryExpired = memoryExpired
	
	// Get database stats
	dbTotal, dbExpired, err := m.store.GetStats()
	if err != nil {
		return stats, fmt.Errorf("failed to get database stats: %w", err)
	}
	
	stats.DatabaseTotal = dbTotal
	stats.DatabaseExpired = dbExpired
	
	return stats, nil
}

// Close shuts down the cache manager and cleanup goroutine
func (m *Manager) Close() {
	if m.cancel != nil {
		m.cancel()
	}
}

// CacheStats represents cache statistics
type CacheStats struct {
	Disabled        bool          `json:"disabled"`
	TTL             time.Duration `json:"ttl"`
	MemoryTotal     int           `json:"memory_total"`
	MemoryExpired   int           `json:"memory_expired"`
	DatabaseTotal   int           `json:"database_total"`
	DatabaseExpired int           `json:"database_expired"`
}