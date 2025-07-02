package workers

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"package-tracking/internal/cache"
	"package-tracking/internal/carriers"
	"package-tracking/internal/config"
	"package-tracking/internal/database"
	"package-tracking/internal/ratelimit"

	_ "github.com/mattn/go-sqlite3"
)

// Test configuration with short timeouts for testing
func getTestConfig() *config.Config {
	return &config.Config{
		AutoUpdateEnabled:           true,
		AutoUpdateCutoffDays:        30,
		AutoUpdateBatchSize:         3, // Small batch for testing
		AutoUpdateMaxRetries:        5,
		AutoUpdateBatchTimeout:      5 * time.Second,
		AutoUpdateIndividualTimeout: 3 * time.Second,
	}
}

// setupTestDB creates a test database using the same approach as existing tests
func setupTestDB(t *testing.T) (*database.DB, func()) {
	// Create temporary file for test database
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()
	
	// Clean up the temp file when test completes
	cleanup := func() {
		os.Remove(tmpfile.Name())
	}
	
	db, err := database.Open(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
		cleanup()
		return nil, nil
	}

	return db, cleanup
}

// createTestShipment creates a test shipment in the database
func createTestShipment(t *testing.T, db *database.DB, trackingNumber string, lastManualRefresh *time.Time) *database.Shipment {
	shipment := &database.Shipment{
		TrackingNumber:      trackingNumber,
		Carrier:             "usps",
		Description:         "Test Package",
		Status:              "pending",
		AutoRefreshEnabled:  true,
		LastManualRefresh:   lastManualRefresh,
		AutoRefreshFailCount: 0,
	}

	err := db.Shipments.Create(shipment)
	if err != nil {
		t.Fatalf("Failed to create test shipment: %v", err)
	}

	// If lastManualRefresh is provided, update the shipment to set this field
	// since Create() doesn't handle this field
	if lastManualRefresh != nil {
		shipment.LastManualRefresh = lastManualRefresh
		err = db.Shipments.Update(shipment.ID, shipment)
		if err != nil {
			t.Fatalf("Failed to update test shipment with manual refresh time: %v", err)
		}
	}

	return shipment
}

// setupTestTrackingUpdater creates a test tracking updater with cache manager
func setupTestTrackingUpdater(t *testing.T, cfg *config.Config, db *database.DB) *TrackingUpdater {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	factory := carriers.NewClientFactory()
	cacheManager := cache.NewManager(db.RefreshCache, false, 5*time.Minute)
	
	return NewTrackingUpdater(cfg, db.Shipments, factory, cacheManager, logger)
}

func TestTrackingUpdater_UnifiedRateLimiting(t *testing.T) {
	cfg := getTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	now := time.Now()
	
	// Test the unified rate limiting logic
	recentRefresh := now.Add(-30 * time.Second) // Within 5-minute rate limit - should be blocked
	oldRefresh := now.Add(-6 * time.Minute)    // Outside 5-minute rate limit - should be allowed
	
	// Test recent refresh (should be blocked)
	result := ratelimit.CheckRefreshRateLimit(cfg, &recentRefresh, false)
	if !result.ShouldBlock {
		t.Error("Recent refresh should be blocked by rate limiting")
	}
	if result.Reason != "rate_limit_active" {
		t.Errorf("Expected reason 'rate_limit_active', got '%s'", result.Reason)
	}
	
	// Test old refresh (should be allowed)
	result = ratelimit.CheckRefreshRateLimit(cfg, &oldRefresh, false)
	if result.ShouldBlock {
		t.Error("Old refresh should not be blocked by rate limiting")
	}
	if result.Reason != "rate_limit_passed" {
		t.Errorf("Expected reason 'rate_limit_passed', got '%s'", result.Reason)
	}
	
	// Test no previous refresh (should be allowed)
	result = ratelimit.CheckRefreshRateLimit(cfg, nil, false)
	if result.ShouldBlock {
		t.Error("No previous refresh should not be blocked by rate limiting")
	}
	if result.Reason != "no_previous_refresh" {
		t.Errorf("Expected reason 'no_previous_refresh', got '%s'", result.Reason)
	}
	
	// Test forced refresh (should always be allowed)
	result = ratelimit.CheckRefreshRateLimit(cfg, &recentRefresh, true)
	if result.ShouldBlock {
		t.Error("Forced refresh should never be blocked by rate limiting")
	}
	if result.Reason != "forced_refresh" {
		t.Errorf("Expected reason 'forced_refresh', got '%s'", result.Reason)
	}
	
	t.Logf("Rate limit duration: %v", ratelimit.GetRateLimitDuration())
}

func TestTrackingUpdater_PauseResume(t *testing.T) {
	cfg := getTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Test initial state (should be running)
	if updater.IsPaused() {
		t.Error("Updater should not be paused initially")
	}
	
	// Test pause
	updater.Pause()
	if !updater.IsPaused() {
		t.Error("Updater should be paused after Pause()")
	}
	
	// Test resume
	updater.Resume()
	if updater.IsPaused() {
		t.Error("Updater should not be paused after Resume()")
	}
}

func TestTrackingUpdater_ConfigurableTimeouts(t *testing.T) {
	cfg := &config.Config{
		AutoUpdateEnabled:           true,
		AutoUpdateCutoffDays:        30,
		AutoUpdateBatchSize:         3,
		AutoUpdateMaxRetries:        5,
		AutoUpdateBatchTimeout:      100 * time.Millisecond, // Very short for testing
		AutoUpdateIndividualTimeout: 50 * time.Millisecond,  // Very short for testing
	}
	
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Test that the configuration values are used
	if updater.config.AutoUpdateBatchTimeout != 100*time.Millisecond {
		t.Errorf("Expected batch timeout 100ms, got %v", updater.config.AutoUpdateBatchTimeout)
	}
	
	if updater.config.AutoUpdateIndividualTimeout != 50*time.Millisecond {
		t.Errorf("Expected individual timeout 50ms, got %v", updater.config.AutoUpdateIndividualTimeout)
	}
}

func TestTrackingUpdater_CacheIntegration(t *testing.T) {
	cfg := getTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Create a test shipment
	shipment := createTestShipment(t, db, "TEST123", nil)

	// Create a cached response
	cachedResponse := &database.RefreshResponse{
		ShipmentID:      shipment.ID,
		UpdatedAt:       time.Now(),
		EventsAdded:     2,
		TotalEvents:     3,
		Events:          []database.TrackingEvent{},
	}

	// Cache the response
	err := updater.cache.Set(shipment.ID, cachedResponse)
	if err != nil {
		t.Fatalf("Failed to cache response: %v", err)
	}

	// Verify cache retrieval
	retrieved, err := updater.cache.Get(shipment.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve cached response: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected cached response, got nil")
	}
	if retrieved.ShipmentID != shipment.ID {
		t.Errorf("Expected shipment ID %d, got %d", shipment.ID, retrieved.ShipmentID)
	}

	t.Logf("Cache integration test passed: cached response retrieved successfully")
}

// Test context timeout handling indirectly by checking configuration
func TestTrackingUpdater_ContextConfiguration(t *testing.T) {
	cfg := &config.Config{
		AutoUpdateBatchTimeout:      2 * time.Second,
		AutoUpdateIndividualTimeout: 1 * time.Second,
	}
	
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Verify the updater uses the configured timeouts
	// This is an indirect test since the timeout usage is internal
	if updater.config.AutoUpdateBatchTimeout != 2*time.Second {
		t.Errorf("Expected batch timeout 2s, got %v", updater.config.AutoUpdateBatchTimeout)
	}
	
	if updater.config.AutoUpdateIndividualTimeout != 1*time.Second {
		t.Errorf("Expected individual timeout 1s, got %v", updater.config.AutoUpdateIndividualTimeout)
	}

	// Test that the context is properly set (non-nil and not background)
	if updater.ctx == nil {
		t.Error("Expected context to be set")
	}
}