# Requirements Specification: USPS Automatic Tracking Updates

## Problem Statement
The package tracking system currently requires manual refresh actions to update shipment tracking information. This creates unnecessary work for users and can result in stale tracking data. We need an automated background service that periodically updates tracking information for active USPS shipments.

## Solution Overview
Implement a background service that runs periodically to automatically update tracking information for eligible USPS shipments. The service will leverage the existing USPS carrier implementation (API with web scraping fallback), respect rate limits, handle failures gracefully, and provide operational controls for administrators.

## Functional Requirements

### 1. Automatic Update Scheduling
- **FR1.1**: The system SHALL run automatic updates at configurable intervals (default: 1 hour via existing UPDATE_INTERVAL)
- **FR1.2**: The system SHALL only update shipments that are NOT marked as delivered
- **FR1.3**: The system SHALL only update shipments created within a configurable cutoff period (e.g., last 30 days)
- **FR1.4**: The system SHALL respect the 5-minute rate limit for shipments that were manually refreshed

### 2. USPS Batch Processing
- **FR2.1**: The system SHALL batch USPS tracking requests up to 10 tracking numbers per API call
- **FR2.2**: The system SHALL handle all 20+ USPS tracking number formats already supported
- **FR2.3**: The system SHALL use the existing USPS client factory (API with web scraping fallback)

### 3. Error Handling and Retry Logic
- **FR3.1**: The system SHALL retry failed batch requests by processing tracking numbers individually
- **FR3.2**: The system SHALL track consecutive failures per shipment (max 10 before marking as problematic)
- **FR3.3**: The system SHALL skip future auto-updates for shipments exceeding the retry limit
- **FR3.4**: The system SHALL handle USPS API authentication failures gracefully

### 4. Operational Controls
- **FR4.1**: The system SHALL provide pause/resume capability without server restart
- **FR4.2**: The pause state SHALL NOT persist across server restarts (ephemeral)
- **FR4.3**: The system SHALL expose admin endpoints for pause/resume control
- **FR4.4**: The system SHALL integrate with existing graceful shutdown mechanisms

### 5. Monitoring and Metrics
- **FR5.1**: The system SHALL expose Prometheus metrics for monitoring
- **FR5.2**: Metrics SHALL include carrier-specific labels (e.g., carrier="usps")
- **FR5.3**: Metrics SHALL track: attempts, successes, failures, duration, API calls
- **FR5.4**: The system SHALL log all update operations at appropriate levels

## Technical Requirements

### 1. Configuration Management
Add to `internal/config/config.go`:
```go
type Config struct {
    // ... existing fields
    AutoUpdateEnabled    bool          // AUTO_UPDATE_ENABLED (default: true)
    AutoUpdateCutoffDays int           // AUTO_UPDATE_CUTOFF_DAYS (default: 30)
    AutoUpdateBatchSize  int           // AUTO_UPDATE_BATCH_SIZE (default: 10, max: 10 for USPS)
    AutoUpdateMaxRetries int           // AUTO_UPDATE_MAX_RETRIES (default: 10)
}
```

### 2. Database Schema Updates
Create migration to add fields to `shipments` table:
```sql
ALTER TABLE shipments ADD COLUMN last_auto_refresh TIMESTAMP;
ALTER TABLE shipments ADD COLUMN auto_refresh_count INTEGER DEFAULT 0;
ALTER TABLE shipments ADD COLUMN auto_refresh_enabled BOOLEAN DEFAULT TRUE;
ALTER TABLE shipments ADD COLUMN auto_refresh_error TEXT;
ALTER TABLE shipments ADD COLUMN auto_refresh_fail_count INTEGER DEFAULT 0;
```

Add methods to `ShipmentStore`:
```go
GetActiveForAutoUpdate(carrier string, cutoffDate time.Time) ([]*Shipment, error)
UpdateAutoRefreshTracking(id int64, success bool, errorMsg string) error
ResetAutoRefreshFailCount(id int64) error
```

### 3. Background Service Implementation
Create `internal/workers/tracking_updater.go`:
- Follow cache manager's goroutine/ticker pattern
- Use atomic.Bool for pause/resume state
- Implement batch processing with individual retry fallback
- Integrate with Prometheus metrics

### 4. Admin API Endpoints
Add to router:
```
POST /api/admin/tracking-updater/pause
POST /api/admin/tracking-updater/resume
GET  /api/admin/tracking-updater/status
```

### 5. Prometheus Integration
Create `internal/metrics/metrics.go`:
- Add prometheus/client_golang dependency
- Define metrics: `tracking_updates_total`, `tracking_updates_duration_seconds`, etc.
- Add `/metrics` endpoint to server

## Implementation Hints

### 1. Service Structure
```go
type TrackingUpdater struct {
    ctx              context.Context
    cancel           context.CancelFunc
    config           *config.Config
    shipmentStore    *database.ShipmentStore
    carrierFactory   *carriers.Factory
    paused           atomic.Bool
    metrics          *metrics.Metrics
}
```

### 2. Update Logic Flow
1. Query active shipments: `WHERE is_delivered = false AND carrier = 'usps' AND created_at > ? AND (last_auto_refresh IS NULL OR last_auto_refresh < ?)`
2. Filter out recently manually refreshed: check `last_manual_refresh`
3. Filter out shipments exceeding retry limit: check `auto_refresh_fail_count`
4. Batch into groups of 10 for USPS API
5. Process each batch, retry individuals on failure
6. Update database with results

### 3. Integration Points
- Start service in `cmd/server/main.go` after cache manager
- Use existing signal handling for graceful shutdown
- Reuse tracking update logic from `RefreshShipment` handler
- Use existing USPS client from carrier factory

## Acceptance Criteria
- [x] Automatic updates run on schedule for active USPS shipments
- [x] Respects delivery status and cutoff date configuration
- [x] Handles batch processing with individual retry fallback
- [x] Provides pause/resume capability via admin API
- [x] Exposes Prometheus metrics with carrier labels
- [x] Integrates cleanly with existing codebase patterns
- [x] Includes comprehensive unit and integration tests
- [x] Updates documentation with new configuration options

## Assumptions
1. The existing USPS carrier implementation is stable and well-tested
2. SQLite performance is acceptable for batch queries and updates
3. Prometheus metrics endpoint is acceptable for monitoring (vs. other solutions)
4. 10 consecutive failures is a reasonable threshold for marking shipments as problematic
5. The existing 5-minute rate limit should apply to both manual and automatic refreshes

## Future Considerations
- This implementation sets the pattern for adding other carriers (UPS, FedEx, DHL)
- Consider implementing exponential backoff for retries in a future iteration
- May want to add bulk operations endpoints for managing auto-update settings
- Consider adding a dashboard or UI for monitoring update status