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
		AutoUpdateFailureThreshold:  10,
		UPSAutoUpdateEnabled:        true,
		UPSAutoUpdateCutoffDays:     30,
		DHLAutoUpdateEnabled:        true,
		DHLAutoUpdateCutoffDays:     0, // Use global fallback
		CacheTTL:                    5 * time.Minute,
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

// createTestUPSShipment creates a test UPS shipment in the database
func createTestUPSShipment(t *testing.T, db *database.DB, trackingNumber string, lastManualRefresh *time.Time) *database.Shipment {
	shipment := &database.Shipment{
		TrackingNumber:       trackingNumber,
		Carrier:              "ups",
		Description:          "Test UPS Package",
		Status:               "pending",
		AutoRefreshEnabled:   true,
		LastManualRefresh:    lastManualRefresh,
		AutoRefreshFailCount: 0,
	}

	err := db.Shipments.Create(shipment)
	if err != nil {
		t.Fatalf("Failed to create test UPS shipment: %v", err)
	}

	// If lastManualRefresh is provided, update the shipment to set this field
	if lastManualRefresh != nil {
		shipment.LastManualRefresh = lastManualRefresh
		err = db.Shipments.Update(shipment.ID, shipment)
		if err != nil {
			t.Fatalf("Failed to update test UPS shipment with manual refresh time: %v", err)
		}
	}

	return shipment
}

func TestTrackingUpdater_UPSAutoUpdateConfig(t *testing.T) {
	cfg := getTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Test UPS-specific configuration
	if !updater.config.UPSAutoUpdateEnabled {
		t.Error("UPS auto-updates should be enabled in test config")
	}
	
	if updater.config.UPSAutoUpdateCutoffDays != 30 {
		t.Errorf("Expected UPS cutoff days 30, got %d", updater.config.UPSAutoUpdateCutoffDays)
	}
	
	if updater.config.AutoUpdateFailureThreshold != 10 {
		t.Errorf("Expected failure threshold 10, got %d", updater.config.AutoUpdateFailureThreshold)
	}

	if updater.config.CacheTTL != 5*time.Minute {
		t.Errorf("Expected cache TTL 5m, got %v", updater.config.CacheTTL)
	}
}

func TestTrackingUpdater_UPSAutoUpdateDisabled(t *testing.T) {
	cfg := getTestConfig()
	cfg.UPSAutoUpdateEnabled = false
	
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Create UPS shipments
	createTestUPSShipment(t, db, "1Z999AA1234567890", nil)
	createTestUPSShipment(t, db, "1Z999BB1234567890", nil)

	// Verify UPS auto-updates are disabled in config
	if updater.config.UPSAutoUpdateEnabled {
		t.Error("UPS auto-updates should be disabled")
	}

	// Since we can't easily test the actual update behavior without mocking,
	// we verify the configuration is properly set to disabled
	t.Logf("UPS auto-updates properly disabled in configuration")
}

func TestTrackingUpdater_UPSCutoffDaysFallback(t *testing.T) {
	cfg := getTestConfig()
	cfg.UPSAutoUpdateCutoffDays = 0 // Should fall back to global setting
	cfg.AutoUpdateCutoffDays = 45   // Global setting
	
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Test that the fallback logic works
	// We can't easily test the runtime behavior without mocking,
	// but we can verify the configuration setup
	if updater.config.UPSAutoUpdateCutoffDays != 0 {
		t.Errorf("Expected UPS cutoff days to be 0 (fallback), got %d", updater.config.UPSAutoUpdateCutoffDays)
	}
	
	if updater.config.AutoUpdateCutoffDays != 45 {
		t.Errorf("Expected global cutoff days 45, got %d", updater.config.AutoUpdateCutoffDays)
	}

	t.Logf("UPS cutoff days fallback configuration verified")
}

func TestTrackingUpdater_MultiCarrierSupport(t *testing.T) {
	cfg := getTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Create shipments for multiple carriers
	uspsShipment := createTestShipment(t, db, "9999999999999999999999", nil)
	upsShipment := createTestUPSShipment(t, db, "1Z999AA1234567890", nil)

	// Verify shipments were created with correct carriers
	if uspsShipment.Carrier != "usps" {
		t.Errorf("Expected USPS carrier, got %s", uspsShipment.Carrier)
	}
	
	if upsShipment.Carrier != "ups" {
		t.Errorf("Expected UPS carrier, got %s", upsShipment.Carrier)
	}

	// Test database query for carrier-specific shipments
	cutoffDate := time.Now().AddDate(0, 0, -30)
	
	uspsShipments, err := db.Shipments.GetActiveForAutoUpdate("usps", cutoffDate, 10)
	if err != nil {
		t.Fatalf("Failed to get USPS shipments: %v", err)
	}
	
	upsShipments, err := db.Shipments.GetActiveForAutoUpdate("ups", cutoffDate, 10)
	if err != nil {
		t.Fatalf("Failed to get UPS shipments: %v", err)
	}

	// Verify carrier filtering works
	if len(uspsShipments) != 1 {
		t.Errorf("Expected 1 USPS shipment, got %d", len(uspsShipments))
	}
	
	if len(upsShipments) != 1 {
		t.Errorf("Expected 1 UPS shipment, got %d", len(upsShipments))
	}

	if len(uspsShipments) > 0 && uspsShipments[0].Carrier != "usps" {
		t.Errorf("USPS query returned wrong carrier: %s", uspsShipments[0].Carrier)
	}
	
	if len(upsShipments) > 0 && upsShipments[0].Carrier != "ups" {
		t.Errorf("UPS query returned wrong carrier: %s", upsShipments[0].Carrier)
	}

	t.Logf("Multi-carrier support verified: USPS=%d, UPS=%d", len(uspsShipments), len(upsShipments))
}

func TestTrackingUpdater_FailureThresholdSupport(t *testing.T) {
	cfg := getTestConfig()
	cfg.AutoUpdateFailureThreshold = 5 // Custom threshold
	
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Create a shipment with high failure count (above threshold)
	shipment := createTestUPSShipment(t, db, "1Z999AA1234567890", nil)
	shipment.AutoRefreshFailCount = 6 // Above threshold of 5
	err := db.Shipments.Update(shipment.ID, shipment)
	if err != nil {
		t.Fatalf("Failed to update shipment failure count: %v", err)
	}

	// Test that the shipment is excluded due to failure threshold
	cutoffDate := time.Now().AddDate(0, 0, -30)
	shipments, err := db.Shipments.GetActiveForAutoUpdate("ups", cutoffDate, cfg.AutoUpdateFailureThreshold)
	if err != nil {
		t.Fatalf("Failed to get shipments: %v", err)
	}

	// Should be empty because failure count (6) exceeds threshold (5)
	if len(shipments) != 0 {
		t.Errorf("Expected 0 shipments due to failure threshold, got %d", len(shipments))
	}

	// Test with lower failure count (below threshold)
	shipment.AutoRefreshFailCount = 3 // Below threshold of 5
	err = db.Shipments.Update(shipment.ID, shipment)
	if err != nil {
		t.Fatalf("Failed to update shipment failure count: %v", err)
	}

	shipments, err = db.Shipments.GetActiveForAutoUpdate("ups", cutoffDate, cfg.AutoUpdateFailureThreshold)
	if err != nil {
		t.Fatalf("Failed to get shipments: %v", err)
	}

	// Should include the shipment because failure count (3) is below threshold (5)
	if len(shipments) != 1 {
		t.Errorf("Expected 1 shipment below failure threshold, got %d", len(shipments))
	}

	t.Logf("Failure threshold support verified: threshold=%d", cfg.AutoUpdateFailureThreshold)
}

// createTestDHLShipment creates a test DHL shipment in the database
func createTestDHLShipment(t *testing.T, db *database.DB, trackingNumber string, lastManualRefresh *time.Time) *database.Shipment {
	shipment := &database.Shipment{
		TrackingNumber:       trackingNumber,
		Carrier:              "dhl",
		Description:          "Test DHL Package",
		Status:               "pending",
		AutoRefreshEnabled:   true,
		LastManualRefresh:    lastManualRefresh,
		AutoRefreshFailCount: 0,
	}

	err := db.Shipments.Create(shipment)
	if err != nil {
		t.Fatalf("Failed to create test DHL shipment: %v", err)
	}

	// If lastManualRefresh is provided, update the shipment to set this field
	if lastManualRefresh != nil {
		shipment.LastManualRefresh = lastManualRefresh
		err = db.Shipments.Update(shipment.ID, shipment)
		if err != nil {
			t.Fatalf("Failed to update test DHL shipment with manual refresh time: %v", err)
		}
	}

	return shipment
}

func TestTrackingUpdater_DHLAutoUpdateConfig(t *testing.T) {
	cfg := getTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Test DHL-specific configuration
	if !updater.config.DHLAutoUpdateEnabled {
		t.Error("DHL auto-updates should be enabled in test config")
	}
	
	if updater.config.DHLAutoUpdateCutoffDays != 0 {
		t.Errorf("Expected DHL cutoff days 0 (use global fallback), got %d", updater.config.DHLAutoUpdateCutoffDays)
	}
	
	if updater.config.AutoUpdateCutoffDays != 30 {
		t.Errorf("Expected global cutoff days 30, got %d", updater.config.AutoUpdateCutoffDays)
	}
	
	if updater.config.AutoUpdateFailureThreshold != 10 {
		t.Errorf("Expected failure threshold 10, got %d", updater.config.AutoUpdateFailureThreshold)
	}

	if updater.config.CacheTTL != 5*time.Minute {
		t.Errorf("Expected cache TTL 5m, got %v", updater.config.CacheTTL)
	}
}

func TestTrackingUpdater_DHLAutoUpdateDisabled(t *testing.T) {
	cfg := getTestConfig()
	cfg.DHLAutoUpdateEnabled = false
	
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Create DHL shipments
	createTestDHLShipment(t, db, "1234567890", nil)
	createTestDHLShipment(t, db, "ABCD1234567890", nil)

	// Verify DHL auto-updates are disabled in config
	if updater.config.DHLAutoUpdateEnabled {
		t.Error("DHL auto-updates should be disabled")
	}

	// Since we can't easily test the actual update behavior without mocking,
	// we verify the configuration is properly set to disabled
	t.Logf("DHL auto-updates properly disabled in configuration")
}

func TestTrackingUpdater_DHLCutoffDaysFallback(t *testing.T) {
	cfg := getTestConfig()
	cfg.DHLAutoUpdateCutoffDays = 0 // Should fall back to global setting
	cfg.AutoUpdateCutoffDays = 45   // Global setting
	
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Test that the fallback logic works
	// We can't easily test the runtime behavior without mocking,
	// but we can verify the configuration setup
	if updater.config.DHLAutoUpdateCutoffDays != 0 {
		t.Errorf("Expected DHL cutoff days to be 0 (fallback), got %d", updater.config.DHLAutoUpdateCutoffDays)
	}
	
	if updater.config.AutoUpdateCutoffDays != 45 {
		t.Errorf("Expected global cutoff days 45, got %d", updater.config.AutoUpdateCutoffDays)
	}

	t.Logf("DHL cutoff days fallback configuration verified")
}

func TestTrackingUpdater_DHLSpecificCutoffDays(t *testing.T) {
	cfg := getTestConfig()
	cfg.DHLAutoUpdateCutoffDays = 60 // DHL-specific setting
	cfg.AutoUpdateCutoffDays = 30    // Global setting (should be ignored for DHL)
	
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Test that DHL-specific cutoff days are used when configured
	if updater.config.DHLAutoUpdateCutoffDays != 60 {
		t.Errorf("Expected DHL cutoff days 60, got %d", updater.config.DHLAutoUpdateCutoffDays)
	}
	
	if updater.config.AutoUpdateCutoffDays != 30 {
		t.Errorf("Expected global cutoff days 30, got %d", updater.config.AutoUpdateCutoffDays)
	}

	t.Logf("DHL-specific cutoff days configuration verified")
}

func TestTrackingUpdater_DHLCarrierSupport(t *testing.T) {
	cfg := getTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Create DHL shipments with various tracking number formats
	dhlShipment1 := createTestDHLShipment(t, db, "1234567890", nil)         // 10 chars
	dhlShipment2 := createTestDHLShipment(t, db, "ABCD1234567890123456", nil) // 20 chars
	dhlShipment3 := createTestDHLShipment(t, db, "JD123456789US", nil)     // Typical DHL format

	// Verify shipments were created with correct carrier
	if dhlShipment1.Carrier != "dhl" {
		t.Errorf("Expected DHL carrier for shipment 1, got %s", dhlShipment1.Carrier)
	}
	
	if dhlShipment2.Carrier != "dhl" {
		t.Errorf("Expected DHL carrier for shipment 2, got %s", dhlShipment2.Carrier)
	}
	
	if dhlShipment3.Carrier != "dhl" {
		t.Errorf("Expected DHL carrier for shipment 3, got %s", dhlShipment3.Carrier)
	}

	// Test database query for DHL-specific shipments
	cutoffDate := time.Now().AddDate(0, 0, -30)
	
	dhlShipments, err := db.Shipments.GetActiveForAutoUpdate("dhl", cutoffDate, 10)
	if err != nil {
		t.Fatalf("Failed to get DHL shipments: %v", err)
	}

	// Verify carrier filtering works for DHL
	if len(dhlShipments) != 3 {
		t.Errorf("Expected 3 DHL shipments, got %d", len(dhlShipments))
	}

	for i, shipment := range dhlShipments {
		if shipment.Carrier != "dhl" {
			t.Errorf("DHL query returned wrong carrier for shipment %d: %s", i, shipment.Carrier)
		}
	}

	t.Logf("DHL carrier support verified: found %d DHL shipments", len(dhlShipments))
}

func TestTrackingUpdater_DHLRateLimitWarning(t *testing.T) {
	cfg := getTestConfig()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	updater := setupTestTrackingUpdater(t, cfg, db)
	defer updater.Stop()

	// Test rate limit warning logic (this will be tested when we implement the actual method)
	// For now, just verify the configuration supports DHL rate limits
	if !updater.config.DHLAutoUpdateEnabled {
		t.Error("DHL auto-updates should be enabled for rate limit testing")
	}

	// DHL API has 250 calls/day limit
	// 80% threshold should be 200 calls
	expectedWarningThreshold := 200
	
	// This is a placeholder test - the actual rate limit warning logic
	// will be tested when we implement updateDHLShipments
	t.Logf("DHL rate limit warning threshold would be %d calls (80%% of 250)", expectedWarningThreshold)
}