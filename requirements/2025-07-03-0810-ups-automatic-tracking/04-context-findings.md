# Context Findings: UPS Automatic Tracking Implementation

## Current Implementation Analysis

Based on examination of the codebase, here's a comprehensive analysis of the current implementation patterns:

### 1. Configuration System Patterns

**Current Environment Variables:**
- `AUTO_UPDATE_ENABLED` (bool, default: true)
- `AUTO_UPDATE_CUTOFF_DAYS` (int, default: 30)
- `AUTO_UPDATE_BATCH_SIZE` (int, default: 10)
- `AUTO_UPDATE_MAX_RETRIES` (int, default: 10)
- `AUTO_UPDATE_BATCH_TIMEOUT` (duration, default: 60s)
- `AUTO_UPDATE_INDIVIDUAL_TIMEOUT` (duration, default: 30s)
- `UPS_API_KEY` (string, optional)

**Configuration Helper Functions:**
- `getEnvBoolOrDefault(key, defaultValue)`
- `getEnvIntOrDefault(key, defaultValue)`
- `getEnvDurationOrDefault(key, defaultValue)`

### 2. TrackingUpdater Implementation

**Current USPS Auto-Update Flow:**
1. `updateUSPSShipments()` - Gets eligible shipments using cutoff date
2. `processShipmentsWithCache()` - Handles cache-aware rate limiting
3. `performAPICallAndCache()` - Makes API calls and caches results
4. Uses unified cache-based rate limiting (5-minute intervals)

**Key Methods:**
- `GetActiveForAutoUpdate(carrier, cutoffDate)` - Database query with filters
- `UpdateShipmentWithAutoRefresh()` - Atomic update with success/failure tracking
- `handleUpdateError()` - Records failures and increments fail count

### 3. Failure Tracking Mechanism

**Database Schema (shipments table):**
- `auto_refresh_enabled` (bool, default: true)
- `auto_refresh_fail_count` (int, default: 0)
- `auto_refresh_error` (text, nullable)
- `last_auto_refresh` (datetime, nullable)
- `auto_refresh_count` (int, default: 0)

**Failure Threshold:** Hard-coded at 10 failures in `GetActiveForAutoUpdate()`
- Shipments with `auto_refresh_fail_count >= 10` are excluded from auto-updates

### 4. UPS Factory Pattern

**Factory Selection Logic:**
1. Checks for API credentials (`UPS_API_KEY` as ClientID, need ClientSecret)
2. Falls back to scraping if no API credentials
3. No UPS headless client implementation

**Current Issue:** UPS API client expects both `ClientID` and `ClientSecret`, but config only has `UPS_API_KEY`

### 5. Missing Environment Variables for UPS Extension

**UPS-Specific Configuration:**
- `UPS_CLIENT_ID` (string, optional)
- `UPS_CLIENT_SECRET` (string, optional)
- `UPS_AUTO_UPDATE_ENABLED` (bool, default: true)
- `UPS_AUTO_UPDATE_CUTOFF_DAYS` (int, default: 30)

**General Failure Threshold Configuration:**
- `AUTO_UPDATE_FAILURE_THRESHOLD` (int, default: 10)

**Per-Carrier Scheduling:**
- `AUTO_UPDATE_INTERVAL_USPS` (duration, default: uses main UPDATE_INTERVAL)
- `AUTO_UPDATE_INTERVAL_UPS` (duration, default: uses main UPDATE_INTERVAL)

## Required Changes for UPS Auto-Updates

### 1. **Configuration Updates** (`internal/config/config.go`):
```go
// Add to Config struct:
UPSClientID     string
UPSClientSecret string
AutoUpdateFailureThreshold int

// Per-carrier auto-update settings
USPSAutoUpdateEnabled    bool
USPSAutoUpdateCutoffDays int
UPSAutoUpdateEnabled     bool
UPSAutoUpdateCutoffDays  int

// Add to Load() function:
UPSClientID:     os.Getenv("UPS_CLIENT_ID"),
UPSClientSecret: os.Getenv("UPS_CLIENT_SECRET"),
AutoUpdateFailureThreshold: getEnvIntOrDefault("AUTO_UPDATE_FAILURE_THRESHOLD", 10),
USPSAutoUpdateEnabled:    getEnvBoolOrDefault("USPS_AUTO_UPDATE_ENABLED", true),
USPSAutoUpdateCutoffDays: getEnvIntOrDefault("USPS_AUTO_UPDATE_CUTOFF_DAYS", 30),
UPSAutoUpdateEnabled:     getEnvBoolOrDefault("UPS_AUTO_UPDATE_ENABLED", true),
UPSAutoUpdateCutoffDays:  getEnvIntOrDefault("UPS_AUTO_UPDATE_CUTOFF_DAYS", 30),
```

### 2. **Server Configuration** (`cmd/server/main.go`):
```go
// Add UPS configuration:
if cfg.UPSClientID != "" && cfg.UPSClientSecret != "" {
    upsConfig := &carriers.CarrierConfig{
        ClientID:      cfg.UPSClientID,
        ClientSecret:  cfg.UPSClientSecret,
        PreferredType: carriers.ClientTypeAPI,
    }
    carrierFactory.SetCarrierConfig("ups", upsConfig)
    log.Printf("UPS API credentials configured")
}
```

### 3. **TrackingUpdater Extension** (`internal/workers/tracking_updater.go`):
```go
// Add to performUpdates():
if u.config.UPSAutoUpdateEnabled {
    u.updateUPSShipments()
}

// Add new method:
func (u *TrackingUpdater) updateUPSShipments() {
    cutoffDate := time.Now().AddDate(0, 0, -u.config.UPSAutoUpdateCutoffDays)
    
    shipments, err := u.shipmentStore.GetActiveForAutoUpdate("ups", cutoffDate)
    if err != nil {
        u.logger.Error("Failed to fetch UPS shipments for auto-update", "error", err)
        return
    }
    
    if len(shipments) == 0 {
        u.logger.Debug("No UPS shipments found for auto-update")
        return
    }
    
    u.logger.Info("Found UPS shipments for auto-update", "count", len(shipments))
    u.processShipmentsWithCache(shipments)
}
```

### 4. **Database Updates** (`internal/database/models.go`):
```go
// Update GetActiveForAutoUpdate to use configurable threshold:
func (s *ShipmentStore) GetActiveForAutoUpdate(carrier string, cutoffDate time.Time, failureThreshold int) ([]Shipment, error) {
    query := `SELECT ... FROM shipments 
              WHERE is_delivered = false 
              AND carrier = ? 
              AND created_at > ?
              AND auto_refresh_enabled = true
              AND auto_refresh_fail_count < ?
              ORDER BY created_at DESC`
    
    rows, err := s.db.Query(query, carrier, cutoffDate, failureThreshold)
    // ... rest of implementation
}
```

### 5. **Factory Pattern Fix** (`internal/carriers/factory.go`):
```go
// Update UPS case in createAPIClient:
case "ups":
    if config.ClientID == "" || config.ClientSecret == "" {
        return nil, fmt.Errorf("UPS Client ID/Secret not configured")
    }
    return NewUPSClient(config.ClientID, config.ClientSecret, config.UseSandbox), nil
```

## Key Implementation Details

1. **Unified Rate Limiting**: The system uses cache-based rate limiting with a 5-minute interval that applies to both manual and auto-refresh operations.

2. **Failure Tracking**: The current system tracks consecutive failures and stops auto-updates after 10 failures. This should be made configurable.

3. **Carrier-Specific Settings**: Each carrier should have its own auto-update settings (enabled/disabled, cutoff days) while sharing the common infrastructure.

4. **API vs Scraping**: The factory pattern automatically selects API when credentials are available, falling back to scraping. UPS requires both ClientID and ClientSecret.

5. **Atomic Updates**: The system uses transactions to atomically update shipment data and auto-refresh tracking to prevent race conditions.

6. **Cache Integration**: Auto-updates populate the same cache used by manual refreshes, ensuring consistent performance.

## Files That Need Modification

1. `internal/config/config.go` - Add UPS-specific configuration fields
2. `cmd/server/main.go` - Configure UPS client factory
3. `internal/workers/tracking_updater.go` - Add UPS auto-update method
4. `internal/database/models.go` - Update failure threshold handling
5. `internal/carriers/factory.go` - Fix UPS API credential configuration

This analysis shows that the codebase has a solid foundation for extending to UPS auto-updates. The main work involves:
- Adding UPS-specific configuration options
- Implementing `updateUPSShipments()` method
- Making failure thresholds configurable
- Fixing the UPS API credential configuration
- Adding per-carrier auto-update settings