# Requirements Specification: UPS Automatic Tracking Updates

## Problem Statement

The package tracking system currently supports automatic tracking updates for USPS shipments only. Users need UPS automatic tracking updates to be implemented using the existing UPS API and web scraping clients, with full configurability for update intervals, failure thresholds, and cache behavior.

## Solution Overview

Extend the existing TrackingUpdater background service to support UPS automatic tracking updates. This will leverage the existing UPS API/scraping infrastructure, unified cache-based rate limiting, and configurable scheduling system while adding UPS-specific configuration options.

## Functional Requirements

### FR1: UPS Automatic Updates
- **Requirement**: Implement automatic tracking updates for UPS shipments
- **Behavior**: UPS shipments are automatically refreshed on the same schedule as USPS shipments
- **Implementation**: Add UPS auto-update logic to existing TrackingUpdater.performUpdates() method

### FR2: Configuration Management
- **Requirement**: All update parameters must be user-configurable
- **Behavior**: Users can configure update intervals, cutoff periods, failure thresholds, and cache TTL
- **Implementation**: Add new environment variables with sensible defaults

### FR3: Unified Scheduling
- **Requirement**: UPS and USPS updates run on the same schedule
- **Behavior**: Both carriers update together in the same cycle
- **Implementation**: Single performUpdates() method handles both carriers

### FR4: Failure Handling
- **Requirement**: Configurable failure threshold with automatic disable
- **Behavior**: After N consecutive failures, auto-updates are disabled for that shipment
- **Implementation**: Global failure threshold setting applies to all carriers

### FR5: Cache Integration
- **Requirement**: UPS auto-updates use the same cache-based rate limiting as USPS
- **Behavior**: 5-minute cache TTL (configurable) applies to both manual and automatic refreshes
- **Implementation**: Use existing processShipmentsWithCache() method

### FR6: Authentication Alignment
- **Requirement**: UPS authentication must use proper OAuth 2.0 credentials
- **Behavior**: Replace single UPS_API_KEY with UPS_CLIENT_ID and UPS_CLIENT_SECRET
- **Implementation**: Update configuration and factory pattern

### FR7: API/Scraping Fallback
- **Requirement**: Auto-updates work with both UPS API and scraping methods
- **Behavior**: Automatically selects API when credentials available, falls back to scraping
- **Implementation**: Use existing UPS factory pattern

## Technical Requirements

### TR1: Configuration Updates
**File**: `internal/config/config.go`
```go
// Add to Config struct:
UPSClientID                string
UPSClientSecret            string
AutoUpdateFailureThreshold int
CacheTTL                   time.Duration

// Add to Load() function:
UPSClientID:                os.Getenv("UPS_CLIENT_ID"),
UPSClientSecret:            os.Getenv("UPS_CLIENT_SECRET"),
AutoUpdateFailureThreshold: getEnvIntOrDefault("AUTO_UPDATE_FAILURE_THRESHOLD", 10),
CacheTTL:                   getEnvDurationOrDefault("CACHE_TTL", 5*time.Minute),
```

### TR2: Server Configuration
**File**: `cmd/server/main.go`
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

### TR3: TrackingUpdater Extension
**File**: `internal/workers/tracking_updater.go`
```go
// Add to performUpdates():
if u.config.UPSAutoUpdateEnabled {
    u.updateUPSShipments()
}

// Add new method:
func (u *TrackingUpdater) updateUPSShipments() {
    cutoffDate := time.Now().AddDate(0, 0, -u.config.AutoUpdateCutoffDays)
    
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
    u.processShipmentsWithCache(shipments)
}
```

### TR4: Database Updates
**File**: `internal/database/models.go`
```go
// Update GetActiveForAutoUpdate method signature:
func (s *ShipmentStore) GetActiveForAutoUpdate(carrier string, cutoffDate time.Time, failureThreshold int) ([]Shipment, error) {
    query := `SELECT id, tracking_number, carrier, description, status, is_delivered, 
              created_at, updated_at, last_manual_refresh, manual_refresh_count,
              last_auto_refresh, auto_refresh_count, auto_refresh_enabled, 
              auto_refresh_error, auto_refresh_fail_count
              FROM shipments 
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

### TR5: Cache Configuration
**File**: `internal/cache/manager.go`
```go
// Update to use configurable TTL:
func (cm *CacheManager) SetCacheTTL(ttl time.Duration) {
    cm.ttl = ttl
}
```

## Environment Variables

### New Environment Variables
```bash
# UPS Authentication (replaces UPS_API_KEY)
UPS_CLIENT_ID=""              # UPS OAuth Client ID
UPS_CLIENT_SECRET=""          # UPS OAuth Client Secret

# Failure Threshold Configuration
AUTO_UPDATE_FAILURE_THRESHOLD=10  # Number of consecutive failures before disabling

# Cache Configuration
CACHE_TTL=5m                  # Cache TTL duration (5 minutes default)

# Per-Carrier Auto-Update Control (optional enhancement)
UPS_AUTO_UPDATE_ENABLED=true     # Enable/disable UPS auto-updates
UPS_AUTO_UPDATE_CUTOFF_DAYS=30   # Cutoff days for UPS shipments
```

### Existing Environment Variables (unchanged)
```bash
AUTO_UPDATE_ENABLED=true
AUTO_UPDATE_CUTOFF_DAYS=30
AUTO_UPDATE_BATCH_SIZE=10
AUTO_UPDATE_BATCH_TIMEOUT=60s
AUTO_UPDATE_INDIVIDUAL_TIMEOUT=30s
UPDATE_INTERVAL=1h
```

## Implementation Patterns

### Pattern 1: Configuration Loading
Follow existing pattern in `internal/config/config.go`:
- Use `os.Getenv()` for string values
- Use `getEnvBoolOrDefault()` for boolean values
- Use `getEnvIntOrDefault()` for integer values
- Use `getEnvDurationOrDefault()` for duration values

### Pattern 2: Database Operations
Follow existing pattern in `internal/database/models.go`:
- Use prepared statements for SQL queries
- Handle errors appropriately
- Use transactions for atomic operations
- Follow existing naming conventions

### Pattern 3: Logging
Follow existing pattern in `internal/workers/tracking_updater.go`:
- Use structured logging with context
- Log at appropriate levels (Debug, Info, Error)
- Include relevant metadata in log messages

### Pattern 4: Error Handling
Follow existing pattern throughout codebase:
- Return errors rather than panicking
- Wrap errors with context when appropriate
- Log errors before returning them
- Use consistent error messages

## Acceptance Criteria

### AC1: Configuration
- [ ] UPS_CLIENT_ID and UPS_CLIENT_SECRET environment variables are supported
- [ ] AUTO_UPDATE_FAILURE_THRESHOLD is configurable (default: 10)
- [ ] CACHE_TTL is configurable (default: 5 minutes)
- [ ] UPS_API_KEY environment variable is deprecated in favor of new credentials

### AC2: Functionality
- [ ] UPS shipments are automatically updated on the same schedule as USPS
- [ ] UPS auto-updates work with both API and scraping methods
- [ ] Failure threshold is respected and configurable
- [ ] Cache-based rate limiting applies to UPS auto-updates
- [ ] Auto-updates are disabled after reaching failure threshold

### AC3: Integration
- [ ] UPS auto-updates use existing processShipmentsWithCache() method
- [ ] Database schema supports all UPS auto-update fields
- [ ] Admin API endpoints show UPS auto-update status
- [ ] Logging includes UPS-specific auto-update operations

### AC4: Backward Compatibility
- [ ] Existing USPS auto-update functionality remains unchanged
- [ ] Migration path from UPS_API_KEY to new credentials is documented
- [ ] Default values maintain existing behavior
- [ ] No breaking changes to existing API endpoints

## Assumptions

1. **OAuth Migration**: Users will migrate from UPS_API_KEY to UPS_CLIENT_ID/UPS_CLIENT_SECRET
2. **Unified Scheduling**: Users prefer single update schedule for all carriers
3. **Global Settings**: Users prefer global failure threshold over per-carrier settings
4. **Cache Behavior**: Users want the same cache TTL for all carriers and operations
5. **Default Values**: Current defaults (30 days cutoff, 10 failure threshold) are appropriate

## Testing Requirements

### Unit Tests
- [ ] Test UPS auto-update method implementation
- [ ] Test configuration loading for new environment variables
- [ ] Test failure threshold handling
- [ ] Test cache TTL configuration

### Integration Tests
- [ ] Test UPS auto-updates with API credentials
- [ ] Test UPS auto-updates with scraping fallback
- [ ] Test failure threshold behavior
- [ ] Test cache integration

### Error Scenario Tests
- [ ] Test OAuth authentication failures
- [ ] Test API service unavailability
- [ ] Test consecutive failure handling
- [ ] Test invalid tracking numbers

## Files to Modify

1. `internal/config/config.go` - Add UPS and threshold configuration
2. `cmd/server/main.go` - Configure UPS client factory
3. `internal/workers/tracking_updater.go` - Add UPS auto-update method
4. `internal/database/models.go` - Update GetActiveForAutoUpdate method
5. `internal/cache/manager.go` - Add configurable cache TTL
6. `internal/carriers/factory.go` - Update UPS credential handling

## Definition of Done

- [ ] All acceptance criteria are met
- [ ] Unit and integration tests are passing
- [ ] UPS shipments update automatically on schedule
- [ ] Configuration is fully user-configurable
- [ ] Error handling and logging are comprehensive
- [ ] Documentation is updated
- [ ] Backward compatibility is maintained