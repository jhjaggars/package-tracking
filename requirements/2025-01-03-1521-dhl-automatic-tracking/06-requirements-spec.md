# Requirements Specification - DHL Automatic Tracking Updates

## Problem Statement
The package tracking system currently supports automatic tracking updates for USPS and UPS shipments but lacks support for DHL shipments. DHL is a major carrier with existing API and web scraping clients already implemented. Adding automatic updates for DHL will provide users with timely tracking information without manual refresh actions.

## Solution Overview
Implement automatic tracking updates for DHL shipments following the established patterns used for UPS and USPS. The solution will integrate seamlessly with the existing tracking updater service, leveraging the current cache system, rate limiting, and carrier client factory pattern.

## Functional Requirements

### 1. Automatic Update Scheduling
- DHL shipments shall be automatically updated on the same schedule as other carriers
- Updates shall run every hour by default (configurable via UPDATE_INTERVAL)
- DHL updates shall be processed after USPS and UPS updates in the sequence

### 2. Configuration Management
- DHL automatic updates shall be enabled by default
- System shall support DHL-specific cutoff days configuration with fallback to global setting
- Configuration shall be managed through environment variables:
  - `DHL_AUTO_UPDATE_ENABLED` (default: true)
  - `DHL_AUTO_UPDATE_CUTOFF_DAYS` (default: 0, falls back to global AUTO_UPDATE_CUTOFF_DAYS)

### 3. Rate Limit Management
- System shall track DHL API rate limits (250 calls/day)
- When API usage reaches 80% (200 calls), system shall log a warning ONCE per rate limit period
- Warning shall include:
  - Current usage (limit and remaining)
  - Percentage used
  - Reset time (if available from API response)
- Warning shall reset when rate limit resets

### 4. Carrier Client Selection
- System shall use DHL API client when credentials are configured
- System shall automatically fall back to web scraping when API credentials are unavailable
- Both API and scraping failures shall increment the failure count equally

### 5. Admin Controls
- DHL updates shall respect the unified pause/resume controls
- No separate DHL-specific pause/resume controls needed
- Admin API endpoints remain unchanged

### 6. Database Integration
- System shall query active DHL shipments using existing `GetActiveForAutoUpdate()` method
- Only non-delivered shipments created within cutoff period shall be processed
- Shipments exceeding failure threshold shall be excluded

## Technical Requirements

### 1. Configuration Updates (`internal/config/config.go`)
```go
// Add to Config struct
DHLAutoUpdateEnabled     bool
DHLAutoUpdateCutoffDays  int

// Add to loadFromEnv()
config.DHLAutoUpdateEnabled = getBoolEnv("DHL_AUTO_UPDATE_ENABLED", true)
config.DHLAutoUpdateCutoffDays = getIntEnv("DHL_AUTO_UPDATE_CUTOFF_DAYS", 0)

// Add to Validate()
if c.DHLAutoUpdateCutoffDays < 0 {
    return fmt.Errorf("DHL_AUTO_UPDATE_CUTOFF_DAYS cannot be negative")
}
```

### 2. Tracking Updater Implementation (`internal/workers/tracking_updater.go`)

#### Add updateDHLShipments() method:
```go
func (u *TrackingUpdater) updateDHLShipments() {
    // Use DHL-specific cutoff days if configured, otherwise use global setting
    cutoffDays := u.config.DHLAutoUpdateCutoffDays
    if cutoffDays == 0 {
        cutoffDays = u.config.AutoUpdateCutoffDays
    }
    
    cutoffDate := time.Now().UTC().AddDate(0, 0, -cutoffDays)
    
    u.logger.Info("Fetching active DHL shipments for automatic update",
        "cutoff_days", cutoffDays,
        "cutoff_date", cutoffDate.Format("2006-01-02"))
    
    shipments, err := u.shipmentStore.GetActiveForAutoUpdate("dhl", cutoffDate)
    if err != nil {
        u.logger.Error("Failed to fetch DHL shipments", "error", err)
        return
    }
    
    u.logger.Info("Found DHL shipments for automatic update", "count", len(shipments))
    
    processed, errors := u.processShipmentsWithCache(shipments, "dhl automatic update")
    
    elapsed := time.Since(start)
    u.logger.Info("DHL automatic updates completed",
        "processed", processed,
        "errors", errors,
        "elapsed", elapsed)
}
```

#### Update performUpdates() method:
```go
// Add after UPS updates
if u.config.DHLAutoUpdateEnabled {
    u.logger.Info("Starting DHL automatic updates")
    u.updateDHLShipments()
}
```

#### Add rate limit warning in performAPICallAndCache():
```go
// After successful API call and cache update
if resp.RateLimit != nil && shipment.Carrier == "dhl" {
    u.checkDHLRateLimit(resp.RateLimit)
}
```

#### Add helper method for rate limit checking:
```go
// Track if we've already logged the warning this period
var dhlRateLimitWarned bool
var dhlRateLimitResetTime time.Time

func (u *TrackingUpdater) checkDHLRateLimit(rateLimit *carriers.RateLimitInfo) {
    if rateLimit.Limit == 0 {
        return
    }
    
    // Reset warning flag if rate limit has reset
    if !dhlRateLimitResetTime.IsZero() && time.Now().After(dhlRateLimitResetTime) {
        dhlRateLimitWarned = false
    }
    
    percentUsed := float64(rateLimit.Limit - rateLimit.Remaining) / float64(rateLimit.Limit)
    
    if percentUsed >= 0.8 && !dhlRateLimitWarned {
        fields := []any{
            "limit", rateLimit.Limit,
            "remaining", rateLimit.Remaining,
            "percent_used", fmt.Sprintf("%.1f%%", percentUsed*100),
        }
        
        if !rateLimit.ResetTime.IsZero() {
            fields = append(fields, "reset_time", rateLimit.ResetTime.Format(time.RFC3339))
            fields = append(fields, "time_until_reset", time.Until(rateLimit.ResetTime).Round(time.Minute))
            dhlRateLimitResetTime = rateLimit.ResetTime
        }
        
        u.logger.Warn("DHL API rate limit approaching", fields...)
        dhlRateLimitWarned = true
    }
}
```

### 3. No Changes Required
- Database schema (supports all carriers generically)
- DHL carrier client (already implemented with factory pattern)
- Cache system (shared infrastructure)
- Admin API (unified controls)
- Existing tests (will be extended for DHL)

## Implementation Hints

### Following Existing Patterns
1. Copy the UPS update implementation pattern exactly
2. Use the same logging levels and message formats
3. Leverage `processShipmentsWithCache()` for unified processing
4. Let the carrier factory handle API vs scraping selection

### Rate Limit Warning Implementation
1. Store warning state at the tracking updater level
2. Reset warning flag when rate limit period resets
3. Include actionable information in warning logs
4. Use structured logging fields for easy parsing

### Testing Approach
1. Mock DHL carrier client in unit tests
2. Test rate limit warning at exactly 80% threshold
3. Test configuration validation for negative cutoff days
4. Test fallback from API to scraping
5. Verify cache integration works correctly

## Acceptance Criteria

1. ✅ DHL shipments update automatically on schedule
2. ✅ Configuration via environment variables works correctly
3. ✅ Rate limit warnings logged at 80% threshold (once per period)
4. ✅ Automatic fallback to web scraping when API unavailable
5. ✅ Unified pause/resume controls affect DHL updates
6. ✅ Failure counts increment correctly for both API and scraping
7. ✅ Cache system reduces API calls for DHL
8. ✅ Comprehensive logging at appropriate levels
9. ✅ All tests pass including new DHL tests
10. ✅ Documentation updated with DHL configuration

## Assumptions

1. **Rate Limit Reset**: DHL API rate limit resets at midnight UTC (to be confirmed during implementation)
2. **Rate Limit Headers**: DHL API provides standard rate limit headers (already implemented in client)
3. **No Batch Support**: DHL API processes one tracking number at a time (confirmed)
4. **Cache Sharing**: DHL updates share the same cache as manual refreshes (desired behavior)
5. **Failure Threshold**: Same failure threshold applies to all carriers (10 by default)