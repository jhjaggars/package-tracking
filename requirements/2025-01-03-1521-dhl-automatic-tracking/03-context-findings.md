# Context Findings - DHL Automatic Tracking Updates

## Files That Need Modification

1. **internal/config/config.go**
   - Add `DHLAutoUpdateEnabled` bool field (line ~69, after UPS fields)
   - Add `DHLAutoUpdateCutoffDays` int field
   - Add environment variable loading in `loadFromEnv()` method
   - Add validation in `Validate()` method

2. **internal/workers/tracking_updater.go**
   - Add `updateDHLShipments()` method following UPS pattern (after line 195)
   - Modify `performUpdates()` to call DHL updates (line ~143)
   - Enhance `performAPICallAndCache()` to log DHL rate limit warnings (line ~299)

## Patterns to Follow

### Configuration Pattern (from UPS implementation)
```go
// In Config struct
UPSAutoUpdateEnabled     bool
UPSAutoUpdateCutoffDays  int

// In loadFromEnv()
config.UPSAutoUpdateEnabled = getBoolEnv("UPS_AUTO_UPDATE_ENABLED", true)
config.UPSAutoUpdateCutoffDays = getIntEnv("UPS_AUTO_UPDATE_CUTOFF_DAYS", 0)

// In Validate()
if c.UPSAutoUpdateCutoffDays < 0 {
    return fmt.Errorf("UPS_AUTO_UPDATE_CUTOFF_DAYS cannot be negative")
}
```

### Update Method Pattern (from updateUPSShipments)
```go
func (u *TrackingUpdater) updateDHLShipments() {
    cutoffDays := u.config.DHLAutoUpdateCutoffDays
    if cutoffDays == 0 {
        cutoffDays = u.config.AutoUpdateCutoffDays
    }
    
    cutoffDate := time.Now().UTC().AddDate(0, 0, -cutoffDays)
    shipments, err := u.shipmentStore.GetActiveForAutoUpdate("dhl", cutoffDate)
    // ... rest of implementation
}
```

## Technical Constraints

### DHL Rate Limiting
- **API Limit**: 250 calls per day (very low)
- **Reset Time**: Likely midnight UTC
- **Current Implementation**: Rate limit tracked in `RateLimitInfo` struct
- **Headers Parsed**: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, etc.

### Rate Limit Warning Implementation
Since carrier clients don't have logger access, implement warnings in the tracking updater:

```go
// In performAPICallAndCache() after successful API call
if resp.RateLimit != nil && shipment.Carrier == "dhl" {
    percentUsed := float64(resp.RateLimit.Limit - resp.RateLimit.Remaining) / float64(resp.RateLimit.Limit)
    if percentUsed >= 0.8 {
        u.logger.Warn("DHL API rate limit approaching",
            "limit", resp.RateLimit.Limit,
            "remaining", resp.RateLimit.Remaining,
            "percent_used", fmt.Sprintf("%.1f%%", percentUsed*100))
    }
}
```

## Integration Points

### In performUpdates() method
```go
// After UPS updates (line ~143)
if u.config.DHLAutoUpdateEnabled {
    u.logger.Info("Starting DHL automatic updates")
    u.updateDHLShipments()
}
```

### Database Query
Uses existing `GetActiveForAutoUpdate()` method:
- Filters by carrier = "dhl"
- Excludes delivered shipments
- Excludes shipments with too many failures
- Only includes shipments created after cutoff date

## Similar Features Analyzed

### UPS Auto-Updates (most similar)
- Single tracking per API call (like DHL)
- Uses carrier-specific cutoff days with global fallback
- Processes through unified `processShipmentsWithCache()` method
- Enabled by default

### USPS Auto-Updates
- Batch processing (up to 10 per call) - different from DHL
- Uses global cutoff days only
- Same cache and rate limiting infrastructure

## Cache Considerations

- **Cache TTL**: 5 minutes (configurable via CACHE_TTL)
- **Importance**: Critical for DHL due to 250/day limit
- **Shared Cache**: Same cache used for manual and auto updates
- **Database Persistence**: Cache survives server restarts

## Logging Conventions

```go
// Starting updates
u.logger.Info("Starting DHL automatic updates")

// Found shipments
u.logger.Info("Found DHL shipments for automatic update", "count", len(shipments))

// Processing individual shipment
u.logger.Debug("Processing DHL shipment", "tracking_number", shipment.TrackingNumber)

// Rate limit warning (new)
u.logger.Warn("DHL API rate limit approaching", ...)

// Completion
u.logger.Info("DHL automatic updates completed",
    "processed", processed,
    "errors", errors,
    "elapsed", elapsed)
```

## Error Handling

- Follows existing pattern: log errors but continue processing
- Increment failure count on errors
- Let existing failure threshold logic disable problematic shipments
- Rate limit errors already handled by carrier client

## No Changes Required

- Database schema (fully generic)
- Carrier client factory (already handles DHL)
- Cache implementation (shared infrastructure)
- Admin API (uses generic pause/resume)
- Metrics/monitoring (unified approach)