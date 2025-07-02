package workers

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"package-tracking/internal/carriers"
	"package-tracking/internal/config"
	"package-tracking/internal/database"
)

// TrackingUpdater handles automatic background updates of shipment tracking information
type TrackingUpdater struct {
	ctx            context.Context
	cancel         context.CancelFunc
	config         *config.Config
	shipmentStore  *database.ShipmentStore
	carrierFactory *carriers.ClientFactory
	paused         atomic.Bool
	logger         *slog.Logger
}

// NewTrackingUpdater creates a new tracking updater service
func NewTrackingUpdater(cfg *config.Config, shipmentStore *database.ShipmentStore, carrierFactory *carriers.ClientFactory, logger *slog.Logger) *TrackingUpdater {
	ctx, cancel := context.WithCancel(context.Background())
	return &TrackingUpdater{
		ctx:            ctx,
		cancel:         cancel,
		config:         cfg,
		shipmentStore:  shipmentStore,
		carrierFactory: carrierFactory,
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

	// Currently only implementing USPS as specified in the requirements
	u.updateUSPSShipments()

	duration := time.Since(startTime)
	u.logger.Info("Completed automatic tracking updates", "duration", duration)
}

// updateUSPSShipments updates all eligible USPS shipments
func (u *TrackingUpdater) updateUSPSShipments() {
	cutoffDate := time.Now().AddDate(0, 0, -u.config.AutoUpdateCutoffDays)
	
	u.logger.Debug("Fetching USPS shipments for auto-update",
		"cutoff_date", cutoffDate,
		"cutoff_days", u.config.AutoUpdateCutoffDays)

	shipments, err := u.shipmentStore.GetActiveForAutoUpdate("usps", cutoffDate)
	if err != nil {
		u.logger.Error("Failed to fetch USPS shipments for auto-update", "error", err)
		return
	}

	if len(shipments) == 0 {
		u.logger.Debug("No USPS shipments found for auto-update")
		return
	}

	u.logger.Info("Found USPS shipments for auto-update", "count", len(shipments))

	// Filter out recently manually refreshed shipments (respect rate limit)
	eligibleShipments := u.filterRecentlyRefreshed(shipments)
	
	if len(eligibleShipments) == 0 {
		u.logger.Debug("No eligible USPS shipments after rate limit filtering")
		return
	}

	u.logger.Info("Processing eligible USPS shipments", "count", len(eligibleShipments))

	// Create USPS carrier client with headless browser support
	uspsClient, _, err := u.carrierFactory.CreateClient("usps")
	if err != nil {
		u.logger.Error("Failed to create USPS carrier client", "error", err)
		return
	}

	// Process shipments in batches
	u.processBatches(eligibleShipments, uspsClient)
}

// filterRecentlyRefreshed removes shipments that were manually refreshed within the rate limit window
func (u *TrackingUpdater) filterRecentlyRefreshed(shipments []database.Shipment) []database.Shipment {
	rateLimit := u.config.AutoUpdateRateLimit
	cutoff := time.Now().Add(-rateLimit)
	
	var eligible []database.Shipment
	for _, shipment := range shipments {
		// Skip if manually refreshed recently
		if shipment.LastManualRefresh != nil && shipment.LastManualRefresh.After(cutoff) {
			u.logger.Debug("Skipping recently manually refreshed shipment",
				"shipment_id", shipment.ID,
				"last_manual_refresh", shipment.LastManualRefresh)
			continue
		}
		
		eligible = append(eligible, shipment)
	}
	
	return eligible
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