package workers

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"package-tracking/internal/cache"
	"package-tracking/internal/carriers"
	"package-tracking/internal/config"
	"package-tracking/internal/database"
	"package-tracking/internal/ratelimit"
)

// TrackingUpdater handles automatic background updates of shipment tracking information
type TrackingUpdater struct {
	ctx            context.Context
	cancel         context.CancelFunc
	config         *config.Config
	shipmentStore  *database.ShipmentStore
	carrierFactory *carriers.ClientFactory
	cache          *cache.Manager
	paused         atomic.Bool
	logger         *slog.Logger
}

// NewTrackingUpdater creates a new tracking updater service
func NewTrackingUpdater(cfg *config.Config, shipmentStore *database.ShipmentStore, carrierFactory *carriers.ClientFactory, cacheManager *cache.Manager, logger *slog.Logger) *TrackingUpdater {
	ctx, cancel := context.WithCancel(context.Background())
	return &TrackingUpdater{
		ctx:            ctx,
		cancel:         cancel,
		config:         cfg,
		shipmentStore:  shipmentStore,
		carrierFactory: carrierFactory,
		cache:          cacheManager,
		logger:         logger,
	}
}

// Start begins the background update process
func (u *TrackingUpdater) Start() {
	if !u.config.AutoUpdateEnabled {
		u.logger.Info("Auto-update is disabled, skipping background updates")
		return
	}

	u.logger.Info("Starting tracking updater service", 
		"interval", u.config.UpdateInterval,
		"cutoff_days", u.config.AutoUpdateCutoffDays,
		"batch_size", u.config.AutoUpdateBatchSize)
	
	go u.updateLoop()
}

// Stop gracefully stops the background update process
func (u *TrackingUpdater) Stop() {
	u.logger.Info("Stopping tracking updater service")
	u.cancel()
}

// Pause temporarily pauses automatic updates
func (u *TrackingUpdater) Pause() {
	u.paused.Store(true)
	u.logger.Info("Tracking updater paused")
}

// Resume resumes automatic updates
func (u *TrackingUpdater) Resume() {
	u.paused.Store(false)
	u.logger.Info("Tracking updater resumed")
}

// IsPaused returns true if the updater is currently paused
func (u *TrackingUpdater) IsPaused() bool {
	return u.paused.Load()
}

// IsRunning returns true if the updater is currently running
func (u *TrackingUpdater) IsRunning() bool {
	select {
	case <-u.ctx.Done():
		return false
	default:
		return true
	}
}

// updateLoop is the main background loop that performs periodic updates
func (u *TrackingUpdater) updateLoop() {
	ticker := time.NewTicker(u.config.UpdateInterval)
	defer ticker.Stop()

	// Perform initial update after a short delay
	initialDelay := time.NewTimer(30 * time.Second)
	defer initialDelay.Stop()

	for {
		select {
		case <-u.ctx.Done():
			u.logger.Info("Tracking updater stopped")
			return

		case <-initialDelay.C:
			// Perform first update
			u.performUpdates()

		case <-ticker.C:
			// Perform periodic updates
			u.performUpdates()
		}
	}
}

// performUpdates executes the update logic for all supported carriers
func (u *TrackingUpdater) performUpdates() {
	if u.paused.Load() {
		u.logger.Debug("Updates paused, skipping update cycle")
		return
	}

	u.logger.Info("Starting automatic tracking updates")
	startTime := time.Now()

	// Update USPS shipments
	u.updateUSPSShipments()
	
	// Update UPS shipments if enabled
	if u.config.UPSAutoUpdateEnabled {
		u.updateUPSShipments()
	}
	
	// Update DHL shipments if enabled
	if u.config.DHLAutoUpdateEnabled {
		u.updateDHLShipments()
	}

	duration := time.Since(startTime)
	u.logger.Info("Completed automatic tracking updates", "duration", duration)
}

// updateUSPSShipments updates all eligible USPS shipments
func (u *TrackingUpdater) updateUSPSShipments() {
	cutoffDate := time.Now().AddDate(0, 0, -u.config.AutoUpdateCutoffDays)
	
	u.logger.Debug("Fetching USPS shipments for auto-update",
		"cutoff_date", cutoffDate,
		"cutoff_days", u.config.AutoUpdateCutoffDays)

	shipments, err := u.shipmentStore.GetActiveForAutoUpdate("usps", cutoffDate, u.config.AutoUpdateFailureThreshold)
	if err != nil {
		u.logger.Error("Failed to fetch USPS shipments for auto-update", "error", err)
		return
	}

	if len(shipments) == 0 {
		u.logger.Debug("No USPS shipments found for auto-update")
		return
	}

	u.logger.Info("Found USPS shipments for auto-update", "count", len(shipments))

	u.logger.Info("Processing USPS shipments with cache-aware rate limiting", "count", len(shipments))

	// Process shipments with unified cache-based rate limiting
	u.processShipmentsWithCache(shipments)
}

// updateUPSShipments updates all eligible UPS shipments
func (u *TrackingUpdater) updateUPSShipments() {
	// Use UPS-specific cutoff days if configured, otherwise use global setting
	cutoffDays := u.config.UPSAutoUpdateCutoffDays
	if cutoffDays == 0 {
		cutoffDays = u.config.AutoUpdateCutoffDays
	}
	
	cutoffDate := time.Now().AddDate(0, 0, -cutoffDays)
	
	u.logger.Debug("Fetching UPS shipments for auto-update",
		"cutoff_date", cutoffDate,
		"cutoff_days", cutoffDays)

	shipments, err := u.shipmentStore.GetActiveForAutoUpdate("ups", cutoffDate, u.config.AutoUpdateFailureThreshold)
	if err != nil {
		u.logger.Error("Failed to fetch UPS shipments for auto-update", "error", err)
		return
	}

	if len(shipments) == 0 {
		u.logger.Debug("No UPS shipments found for auto-update")
		return
	}

	u.logger.Info("Found UPS shipments for auto-update", "count", len(shipments))

	// Process shipments with unified cache-based rate limiting
	u.processShipmentsWithCache(shipments)
}

// updateDHLShipments updates all eligible DHL shipments
func (u *TrackingUpdater) updateDHLShipments() {
	// Use DHL-specific cutoff days if configured, otherwise use global setting
	cutoffDays := u.config.DHLAutoUpdateCutoffDays
	if cutoffDays == 0 {
		cutoffDays = u.config.AutoUpdateCutoffDays
	}
	
	cutoffDate := time.Now().AddDate(0, 0, -cutoffDays)
	
	u.logger.Debug("Fetching DHL shipments for auto-update",
		"cutoff_date", cutoffDate,
		"cutoff_days", cutoffDays)

	shipments, err := u.shipmentStore.GetActiveForAutoUpdate("dhl", cutoffDate, u.config.AutoUpdateFailureThreshold)
	if err != nil {
		u.logger.Error("Failed to fetch DHL shipments for auto-update", "error", err)
		return
	}

	if len(shipments) == 0 {
		u.logger.Debug("No DHL shipments found for auto-update")
		return
	}

	u.logger.Info("Found DHL shipments for auto-update", "count", len(shipments))

	// Check for rate limit warning (80% of 250 daily limit = 200 calls)
	u.checkDHLRateLimitWarning(shipments)

	// Process shipments with unified cache-based rate limiting
	u.processShipmentsWithCache(shipments)
}

// processShipmentsWithCache processes shipments with cache-aware rate limiting
// This replaces the old filterRecentlyRefreshed approach with unified cache-based logic
func (u *TrackingUpdater) processShipmentsWithCache(shipments []database.Shipment) {
	apiCallCount := 0
	
	for i, shipment := range shipments {
		if u.ctx.Err() != nil {
			return // Service is stopping
		}

		u.logger.Debug("Processing shipment",
			"shipment_id", shipment.ID,
			"tracking_number", shipment.TrackingNumber,
			"progress", fmt.Sprintf("%d/%d", i+1, len(shipments)))

		// Check cache first (same as manual refresh)
		if cachedResponse, err := u.cache.Get(shipment.ID); err == nil && cachedResponse != nil {
			u.logger.Debug("Using cached data for auto-update",
				"shipment_id", shipment.ID,
				"cache_age", time.Since(cachedResponse.UpdatedAt))
			u.processCachedResponse(&shipment, cachedResponse)
			continue
		}

		// Check rate limiting using unified logic (no force refresh for auto-updates)
		rateLimitResult := ratelimit.CheckRefreshRateLimit(u.config, shipment.LastManualRefresh, false)
		if rateLimitResult.ShouldBlock {
			u.logger.Debug("Skipping shipment due to rate limiting",
				"shipment_id", shipment.ID,
				"last_manual_refresh", shipment.LastManualRefresh,
				"remaining_time", rateLimitResult.RemainingTime,
				"reason", rateLimitResult.Reason)
			continue
		}

		// Proceed with API call and cache the result
		u.performAPICallAndCache(&shipment)
		apiCallCount++

		// Add delay between API calls to be respectful to the carrier API
		// Only delay if there are more shipments to process
		if i < len(shipments)-1 {
			select {
			case <-u.ctx.Done():
				return
			case <-time.After(1 * time.Second):
				// Continue
			}
		}
	}

	u.logger.Info("Completed shipment processing",
		"total_shipments", len(shipments),
		"api_calls_made", apiCallCount,
		"cache_hits", len(shipments)-apiCallCount)
}

// processCachedResponse processes a shipment using cached data
func (u *TrackingUpdater) processCachedResponse(shipment *database.Shipment, cachedResponse *database.RefreshResponse) {
	// Update shipment's auto-refresh timestamp to indicate it was processed
	// but don't increment counts since this is using cached data
	err := u.shipmentStore.UpdateAutoRefreshTracking(int64(shipment.ID), true, "")
	if err != nil {
		u.logger.Error("Failed to update auto-refresh tracking for cached response",
			"shipment_id", shipment.ID,
			"error", err)
	} else {
		u.logger.Info("Processed shipment using cached data",
			"shipment_id", shipment.ID,
			"tracking_number", shipment.TrackingNumber,
			"cached_events", len(cachedResponse.Events))
	}
}

// performAPICallAndCache makes an API call and caches the result
func (u *TrackingUpdater) performAPICallAndCache(shipment *database.Shipment) {
	// Create carrier client based on shipment carrier
	client, _, err := u.carrierFactory.CreateClient(shipment.Carrier)
	if err != nil {
		u.logger.Error("Failed to create carrier client", 
			"carrier", shipment.Carrier,
			"error", err)
		u.handleUpdateError(shipment, err)
		return
	}

	// Create tracking request with configurable timeout
	ctx, cancel := context.WithTimeout(u.ctx, u.config.AutoUpdateIndividualTimeout)
	defer cancel()

	req := &carriers.TrackingRequest{
		TrackingNumbers: []string{shipment.TrackingNumber},
		Carrier:         shipment.Carrier,
	}

	// Make API call
	resp, err := client.Track(ctx, req)
	if err != nil {
		u.handleUpdateError(shipment, err)
		return
	}

	// Process the first result if available
	if len(resp.Results) > 0 {
		trackingInfo := &resp.Results[0]
		
		// Update shipment data
		originalStatus := shipment.Status
		if trackingInfo.Status != "" && string(trackingInfo.Status) != shipment.Status {
			shipment.Status = string(trackingInfo.Status)
			shipment.IsDelivered = (trackingInfo.Status == carriers.StatusDelivered)
		}

		// Update expected delivery if provided
		if trackingInfo.EstimatedDelivery != nil {
			shipment.ExpectedDelivery = trackingInfo.EstimatedDelivery
		}
		if trackingInfo.ActualDelivery != nil && shipment.IsDelivered {
			shipment.ExpectedDelivery = trackingInfo.ActualDelivery
		}

		// Atomically update shipment and auto-refresh tracking
		err = u.shipmentStore.UpdateShipmentWithAutoRefresh(shipment.ID, shipment, true, "")
		if err != nil {
			u.logger.Error("Failed to update shipment with auto-refresh tracking",
				"shipment_id", shipment.ID,
				"error", err)
			u.handleUpdateError(shipment, err)
			return
		}

		// Cache the response for future manual refreshes
		refreshResponse := &database.RefreshResponse{
			ShipmentID:      shipment.ID,
			UpdatedAt:       time.Now(),
			EventsAdded:     len(trackingInfo.Events),
			TotalEvents:     len(trackingInfo.Events),
			Events:          u.convertToTrackingEvents(trackingInfo.Events),
		}

		// Populate cache (same as manual refresh)
		err = u.cache.Set(shipment.ID, refreshResponse)
		if err != nil {
			u.logger.Warn("Failed to cache auto-refresh response",
				"shipment_id", shipment.ID,
				"error", err)
			// Don't fail the update just because caching failed
		}

		u.logger.Info("Successfully updated and cached shipment",
			"shipment_id", shipment.ID,
			"tracking_number", shipment.TrackingNumber,
			"carrier", shipment.Carrier,
			"status_change", fmt.Sprintf("%s -> %s", originalStatus, shipment.Status),
			"events", len(trackingInfo.Events))
	} else {
		u.logger.Warn("No tracking results for shipment",
			"shipment_id", shipment.ID,
			"tracking_number", shipment.TrackingNumber,
			"carrier", shipment.Carrier)
	}
}

// convertToTrackingEvents converts carrier events to database tracking events
func (u *TrackingUpdater) convertToTrackingEvents(events []carriers.TrackingEvent) []database.TrackingEvent {
	dbEvents := make([]database.TrackingEvent, len(events))
	for i, event := range events {
		dbEvents[i] = database.TrackingEvent{
			Timestamp:   event.Timestamp,
			Location:    event.Location,
			Status:      string(event.Status),
			Description: event.Description,
		}
	}
	return dbEvents
}

// processBatches processes shipments in batches according to USPS API limits
func (u *TrackingUpdater) processBatches(shipments []database.Shipment, uspsClient carriers.Client) {
	batchSize := u.config.AutoUpdateBatchSize
	if batchSize > 10 {
		batchSize = 10 // USPS API limit
	}

	for i := 0; i < len(shipments); i += batchSize {
		// Check if we should stop
		if u.ctx.Err() != nil {
			return
		}

		end := i + batchSize
		if end > len(shipments) {
			end = len(shipments)
		}

		batch := shipments[i:end]
		u.logger.Debug("Processing shipment batch",
			"batch_start", i,
			"batch_end", end,
			"batch_size", len(batch))

		u.processBatch(batch, uspsClient)

		// Add small delay between batches to be respectful to the API
		if end < len(shipments) {
			select {
			case <-u.ctx.Done():
				return
			case <-time.After(2 * time.Second):
				// Continue
			}
		}
	}
}

// processBatch processes a single batch of shipments
func (u *TrackingUpdater) processBatch(batch []database.Shipment, uspsClient carriers.Client) {
	trackingNumbers := make([]string, len(batch))
	shipmentMap := make(map[string]*database.Shipment)

	for i, shipment := range batch {
		trackingNumbers[i] = shipment.TrackingNumber
		shipmentCopy := shipment // Create a copy to avoid pointer issues
		shipmentMap[shipment.TrackingNumber] = &shipmentCopy
	}

	u.logger.Debug("Calling USPS carrier for batch update", "tracking_numbers", trackingNumbers)

	// Create tracking request with configurable timeout
	ctx, cancel := context.WithTimeout(u.ctx, u.config.AutoUpdateBatchTimeout)
	defer cancel()

	req := &carriers.TrackingRequest{
		TrackingNumbers: trackingNumbers,
		Carrier:         "usps",
	}

	// Try batch update first
	resp, err := uspsClient.Track(ctx, req)
	if err != nil {
		u.logger.Warn("Batch update failed, trying individual updates", "error", err)
		// Fall back to individual updates as specified in requirements
		u.processIndividually(batch, uspsClient)
		return
	}

	// Process batch responses
	for _, result := range resp.Results {
		shipment := shipmentMap[result.TrackingNumber]
		if shipment == nil {
			continue
		}

		u.processTrackingInfo(shipment, &result)
	}
}

// processIndividually processes shipments one by one when batch processing fails
func (u *TrackingUpdater) processIndividually(shipments []database.Shipment, uspsClient carriers.Client) {
	for _, shipment := range shipments {
		if u.ctx.Err() != nil {
			return
		}

		u.logger.Debug("Processing individual shipment", "shipment_id", shipment.ID, "tracking_number", shipment.TrackingNumber)

		// Create individual tracking request with configurable timeout
		ctx, cancel := context.WithTimeout(u.ctx, u.config.AutoUpdateIndividualTimeout)
		req := &carriers.TrackingRequest{
			TrackingNumbers: []string{shipment.TrackingNumber},
			Carrier:         "usps",
		}

		resp, err := uspsClient.Track(ctx, req)
		cancel() // Cancel immediately after use

		if err != nil {
			u.handleUpdateError(&shipment, err)
			continue
		}

		// Process the first result if available
		if len(resp.Results) > 0 {
			u.processTrackingInfo(&shipment, &resp.Results[0])
		} else {
			u.logger.Warn("No tracking results for shipment", 
				"shipment_id", shipment.ID,
				"tracking_number", shipment.TrackingNumber)
		}

		// Small delay between individual requests
		select {
		case <-u.ctx.Done():
			return
		case <-time.After(1 * time.Second):
			// Continue
		}
	}
}

// processTrackingInfo processes a successful tracking response
func (u *TrackingUpdater) processTrackingInfo(shipment *database.Shipment, info *carriers.TrackingInfo) {
	u.logger.Debug("Processing tracking response",
		"shipment_id", shipment.ID,
		"status", info.Status,
		"events_count", len(info.Events))

	// Update shipment status
	if info.Status != "" && string(info.Status) != shipment.Status {
		shipment.Status = string(info.Status)
		shipment.IsDelivered = (info.Status == carriers.StatusDelivered)
	}

	// Update expected delivery if provided
	if info.EstimatedDelivery != nil {
		shipment.ExpectedDelivery = info.EstimatedDelivery
	}
	if info.ActualDelivery != nil && shipment.IsDelivered {
		shipment.ExpectedDelivery = info.ActualDelivery
	}

	// Atomically update shipment and auto-refresh tracking
	err := u.shipmentStore.UpdateShipmentWithAutoRefresh(shipment.ID, shipment, true, "")
	if err != nil {
		u.logger.Error("Failed to update shipment with auto-refresh tracking",
			"shipment_id", shipment.ID,
			"error", err)
		u.handleUpdateError(shipment, err)
		return
	}

	u.logger.Info("Successfully updated shipment",
		"shipment_id", shipment.ID,
		"tracking_number", shipment.TrackingNumber,
		"status", info.Status)

	// TODO: Add tracking events to database
	// This would require extending the TrackingEventStore to handle auto-updates
	// For now, we just update the shipment status
}

// handleUpdateError records a failed update attempt
func (u *TrackingUpdater) handleUpdateError(shipment *database.Shipment, err error) {
	errorMsg := err.Error()
	if len(errorMsg) > 500 {
		errorMsg = errorMsg[:500] // Truncate very long error messages
	}

	dbErr := u.shipmentStore.UpdateAutoRefreshTracking(int64(shipment.ID), false, errorMsg)
	if dbErr != nil {
		u.logger.Error("Failed to record auto-refresh error",
			"shipment_id", shipment.ID,
			"original_error", err,
			"db_error", dbErr)
	}

	u.logger.Warn("Auto-update failed for shipment",
		"shipment_id", shipment.ID,
		"tracking_number", shipment.TrackingNumber,
		"error", err)
}

// checkDHLRateLimitWarning checks DHL API rate limits and logs warnings when approaching limits
func (u *TrackingUpdater) checkDHLRateLimitWarning(shipments []database.Shipment) {
	// Get DHL client to check rate limits
	client, _, err := u.carrierFactory.CreateClient("dhl")
	if err != nil {
		// If we can't create a DHL client, we're probably using scraping fallback
		u.logger.Debug("Could not create DHL API client for rate limit check", "error", err)
		return
	}

	// Get rate limit information
	rateLimit := client.GetRateLimit()
	if rateLimit == nil {
		u.logger.Debug("No rate limit information available for DHL")
		return
	}

	// Calculate usage percentage
	limit := rateLimit.Limit
	remaining := rateLimit.Remaining
	if limit <= 0 {
		return // Invalid limit
	}

	used := limit - remaining
	usagePercent := float64(used) / float64(limit) * 100

	// Log warning if usage is at or above 80%
	if usagePercent >= 80.0 {
		u.logger.Warn("DHL API rate limit approaching",
			"usage_percent", fmt.Sprintf("%.1f%%", usagePercent),
			"used", used,
			"limit", limit,
			"remaining", remaining,
			"reset_time", rateLimit.ResetTime,
			"pending_shipments", len(shipments))
		
		// If we're very close to the limit, log additional warning
		if remaining < len(shipments) {
			u.logger.Warn("DHL API calls remaining is less than pending shipments",
				"remaining_calls", remaining,
				"pending_shipments", len(shipments),
				"message", "Some shipments may not be updated due to rate limiting")
		}
	}
}