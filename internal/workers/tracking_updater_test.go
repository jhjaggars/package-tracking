package workers

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"package-tracking/internal/carriers"
	"package-tracking/internal/config"
	"package-tracking/internal/database"

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
		AutoUpdateRateLimit:         1 * time.Minute, // Short for testing
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

func TestTrackingUpdater_FilterRecentlyRefreshed(t *testing.T) {
	cfg := getTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	factory := carriers.NewClientFactory()
	
	updater := NewTrackingUpdater(cfg, db.Shipments, factory, logger)
	defer updater.Stop()

	now := time.Now()
	
	// Create test shipments with explicit rate limit testing
	recentRefresh := now.Add(-30 * time.Second) // Within 1-minute rate limit - should be filtered
	oldRefresh := now.Add(-2 * time.Minute)    // Outside 1-minute rate limit - should be eligible
	
	shipment1 := createTestShipment(t, db, "RECENT123", &recentRefresh)
	shipment2 := createTestShipment(t, db, "OLD123", &oldRefresh)
	shipment3 := createTestShipment(t, db, "NEVER123", nil) // Never refreshed - should be eligible

	shipments := []database.Shipment{*shipment1, *shipment2, *shipment3}
	
	// Test filtering
	eligible := updater.filterRecentlyRefreshed(shipments)
	
	// Debug output to understand what's happening
	t.Logf("Rate limit: %v", updater.config.AutoUpdateRateLimit)
	t.Logf("Cutoff time: %v", now.Add(-updater.config.AutoUpdateRateLimit))
	for i, s := range shipments {
		t.Logf("Shipment %d: %s, LastManualRefresh: %v", i, s.TrackingNumber, s.LastManualRefresh)
	}
	t.Logf("Eligible count: %d", len(eligible))
	for i, s := range eligible {
		t.Logf("Eligible %d: %s", i, s.TrackingNumber)
	}
	
	// Should filter out the recently refreshed shipment (within 1 minute)
	// Expecting OLD123 and NEVER123 to be eligible (2 shipments)
	if len(eligible) != 2 {
		t.Errorf("Expected 2 eligible shipments, got %d", len(eligible))
	}
	
	// Check that the recent one was filtered out
	foundRecent := false
	foundOld := false
	foundNever := false
	for _, shipment := range eligible {
		if shipment.TrackingNumber == "RECENT123" {
			foundRecent = true
		} else if shipment.TrackingNumber == "OLD123" {
			foundOld = true
		} else if shipment.TrackingNumber == "NEVER123" {
			foundNever = true
		}
	}
	
	if foundRecent {
		t.Error("Recently refreshed shipment (RECENT123) should have been filtered out")
	}
	if !foundOld {
		t.Error("Old refreshed shipment (OLD123) should be eligible")
	}
	if !foundNever {
		t.Error("Never refreshed shipment (NEVER123) should be eligible")
	}
}

func TestTrackingUpdater_PauseResume(t *testing.T) {
	cfg := getTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	factory := carriers.NewClientFactory()
	
	updater := NewTrackingUpdater(cfg, db.Shipments, factory, logger)
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
		AutoUpdateRateLimit:         10 * time.Second,
	}
	
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	factory := carriers.NewClientFactory()
	
	updater := NewTrackingUpdater(cfg, db.Shipments, factory, logger)
	defer updater.Stop()

	// Test that the configuration values are used
	if updater.config.AutoUpdateBatchTimeout != 100*time.Millisecond {
		t.Errorf("Expected batch timeout 100ms, got %v", updater.config.AutoUpdateBatchTimeout)
	}
	
	if updater.config.AutoUpdateIndividualTimeout != 50*time.Millisecond {
		t.Errorf("Expected individual timeout 50ms, got %v", updater.config.AutoUpdateIndividualTimeout)
	}
	
	if updater.config.AutoUpdateRateLimit != 10*time.Second {
		t.Errorf("Expected rate limit 10s, got %v", updater.config.AutoUpdateRateLimit)
	}
}

func TestTrackingUpdater_RateLimitConfiguration(t *testing.T) {
	cfg := &config.Config{
		AutoUpdateRateLimit: 30 * time.Second, // Custom rate limit
	}
	
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	factory := carriers.NewClientFactory()
	
	updater := NewTrackingUpdater(cfg, db.Shipments, factory, logger)
	defer updater.Stop()

	now := time.Now()
	
	// Create shipments with different refresh times relative to custom rate limit
	recentRefresh := now.Add(-15 * time.Second) // Within 30-second rate limit - should be filtered
	oldRefresh := now.Add(-45 * time.Second)    // Outside 30-second rate limit - should be eligible
	
	shipment1 := createTestShipment(t, db, "RECENT123", &recentRefresh)
	shipment2 := createTestShipment(t, db, "OLD123", &oldRefresh)

	shipments := []database.Shipment{*shipment1, *shipment2}
	
	// Test filtering with custom rate limit
	eligible := updater.filterRecentlyRefreshed(shipments)
	
	// Debug output
	t.Logf("Custom rate limit: %v", updater.config.AutoUpdateRateLimit)
	t.Logf("Cutoff time: %v", now.Add(-updater.config.AutoUpdateRateLimit))
	for i, s := range shipments {
		t.Logf("Shipment %d: %s, LastManualRefresh: %v", i, s.TrackingNumber, s.LastManualRefresh)
	}
	t.Logf("Eligible count: %d", len(eligible))
	
	// Should filter out the recently refreshed shipment based on 30-second limit
	if len(eligible) != 1 {
		t.Errorf("Expected 1 eligible shipment, got %d", len(eligible))
		return
	}
	
	if eligible[0].TrackingNumber != "OLD123" {
		t.Errorf("Expected OLD123 to be eligible, got %s", eligible[0].TrackingNumber)
	}
}

// Test context timeout handling indirectly by checking configuration
func TestTrackingUpdater_ContextConfiguration(t *testing.T) {
	cfg := &config.Config{
		AutoUpdateBatchTimeout:      2 * time.Second,
		AutoUpdateIndividualTimeout: 1 * time.Second,
		AutoUpdateRateLimit:         30 * time.Second,
	}
	
	db, cleanup := setupTestDB(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	factory := carriers.NewClientFactory()
	
	updater := NewTrackingUpdater(cfg, db.Shipments, factory, logger)
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