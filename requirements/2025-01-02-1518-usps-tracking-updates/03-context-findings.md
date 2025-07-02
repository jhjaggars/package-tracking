# Context Findings for USPS Automatic Tracking Updates

## Files That Need Modification

### 1. Configuration (`internal/config/config.go`)
- Add new config fields:
  - `AutoUpdateEnabled` (bool)
  - `AutoUpdateCutoffDays` (int) - How many days back to update
  - `AutoUpdateBatchSize` (int) - Max 10 for USPS
- Use existing `getEnvDurationOrDefault()` pattern for durations
- Environment variables: `AUTO_UPDATE_ENABLED`, `AUTO_UPDATE_CUTOFF_DAYS`, etc.

### 2. Database Schema (`internal/database/`)
- Need new fields in shipments table:
  - `last_auto_refresh` (timestamp)
  - `auto_refresh_count` (int)
  - `auto_refresh_enabled` (bool per shipment)
  - `last_auto_refresh_error` (text)
- Create migration file in future `migrations/` directory
- Add methods to ShipmentStore:
  - `GetActiveForAutoUpdate(carrier, cutoffDate)`
  - `UpdateAutoRefreshTracking(id, error)`

### 3. New Background Service (`internal/workers/tracking_updater.go`)
- Create new package following cache manager pattern
- Use context-based cancellation for graceful shutdown
- Implement pause/resume with atomic.Bool
- Use time.Ticker with UpdateInterval from config

### 4. Server Integration (`cmd/server/main.go`)
- Start tracking updater service alongside cache manager
- Pass dependencies: config, database, carrier factory
- Ensure graceful shutdown coordination

### 5. Prometheus Metrics (New)
- Add prometheus/client_golang dependency
- Create metrics registry in new `internal/metrics/` package
- Track: update attempts, successes, failures, duration, API calls
- Expose /metrics endpoint

## Patterns to Follow

### Background Service Pattern (from cache/manager.go:157-170)
```go
type TrackingUpdater struct {
    ctx    context.Context
    cancel context.CancelFunc
    paused atomic.Bool
    // ... dependencies
}

func (u *TrackingUpdater) Start() {
    go u.updateLoop()
}

func (u *TrackingUpdater) updateLoop() {
    ticker := time.NewTicker(u.config.UpdateInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-u.ctx.Done():
            return
        case <-ticker.C:
            if !u.paused.Load() {
                u.performUpdates()
            }
        }
    }
}
```

### Refresh Logic Pattern (from handlers/shipments.go:302-456)
- Skip delivered packages
- Check cutoff date (new logic)
- Batch USPS requests (up to 10)
- Handle rate limits and retries
- Update database with results
- Log errors appropriately

### Configuration Pattern (from config/config.go:157-170)
```go
AutoUpdateEnabled:    getEnvBoolOrDefault("AUTO_UPDATE_ENABLED", true),
AutoUpdateCutoffDays: getEnvIntOrDefault("AUTO_UPDATE_CUTOFF_DAYS", 30),
```

### Retry Pattern (from fedex_api.go:695-711)
```go
func isRetryableError(err error) bool {
    // Check for network errors, rate limits, temporary failures
    // Return true if should retry
}
```

## Similar Features Analyzed

### 1. Cache Manager Cleanup
- Runs every minute to clean expired entries
- Simple ticker-based loop with context cancellation
- No pause/resume, but good base pattern

### 2. Manual Refresh Endpoint
- Has rate limiting (5-minute cooldown)
- Caches responses for efficiency
- Updates tracking counters
- Good error handling patterns

### 3. Browser Pool Management
- Has stats collection (could inspire metrics)
- Manages resource lifecycle
- No direct parallel, but shows resource management

## Technical Constraints

### USPS API Limitations
- XML-based API (not REST)
- Batch limit: 10 tracking numbers per request
- Conservative rate limiting needed
- Requires USPS_API_KEY configuration

### Database Considerations
- SQLite doesn't support concurrent writes well
- Need to batch database updates efficiently
- Consider transaction boundaries for batch updates

### Performance Considerations
- USPS API can be slow (2-3 seconds per batch)
- Need to handle long-running updates gracefully
- Don't block server shutdown

## Integration Points

### 1. Carrier Factory (`internal/carriers/factory.go`)
- Use existing factory to create USPS client
- Handles API key configuration automatically

### 2. Database Stores (`internal/database/stores.go`)
- Extend ShipmentStore interface
- Reuse existing transaction patterns

### 3. Server Lifecycle (`cmd/server/main.go`)
- Add to existing graceful shutdown flow
- Start after database initialization

### 4. Logging
- Use existing slog patterns
- Log at appropriate levels (info for success, error for failures)

## Missing Components

### 1. Prometheus Integration
- No existing metrics infrastructure
- Need to add prometheus/client_golang dependency
- Create metrics middleware for HTTP handlers
- Add background service metrics

### 2. Admin API Endpoints
- Need endpoints for pause/resume control
- Status endpoint to check updater health
- Metrics endpoint (/metrics)

### 3. Database Migrations
- No migration system currently exists
- Need to add migration support first
- Then create migration for new fields