# Context Findings

## Email Validation Integration Points

### 1. **Primary Integration Point: Email Processing Flow**

**File:** `internal/workers/email_processor_time.go`
- **Line 319**: `createShipment()` method where validation should be inserted
- **Line 346**: `p.apiClient.CreateShipment(tracking)` call should be preceded by validation
- **Current flow**: Email → Extract tracking → Create shipment
- **New flow**: Email → Extract tracking → **Validate tracking** → Create shipment

### 2. **Refresh System Integration Points**

**File:** `internal/handlers/shipments.go` (lines 319-573)
- **Line 371**: Cache check logic - validation should reuse this cache system
- **Line 397**: Rate limiting check - validation should respect same limits
- **Line 404**: Client creation via factory - validation should use same client selection
- **Line 449**: `client.Track(ctx, req)` - this is the exact call validation needs to make
- **Line 561**: Cache storage - validation results should be cached here

### 3. **Cache System Reuse**

**File:** `internal/cache/manager.go`
- **Existing cache structure**: Uses `sync.Map` + SQLite persistence
- **5-minute TTL**: Same TTL should apply to validation cache
- **Cache keys**: Currently use shipment ID, validation needs tracking number keys
- **Auto-cleanup**: Every minute cleanup process can handle validation cache too

### 4. **Rate Limiting Integration**

**File:** `internal/ratelimit/ratelimit.go`
- **Line 21**: `CheckRefreshRateLimit()` function should be extended for validation
- **5-minute limit**: Same limit should apply to validation attempts
- **Bypass logic**: Validation should respect `DISABLE_RATE_LIMIT` config
- **Database tracking**: Uses `lastManualRefresh` field, validation needs similar tracking

### 5. **Carrier Client Factory**

**File:** `internal/carriers/factory.go`
- **Line 57**: `CreateClient()` method - validation should use same client selection logic
- **API preference**: FedEx prefers API, others use headless/scraping
- **Fallback chain**: API → Headless → Scraping should apply to validation too

### 6. **API Client Integration**

**File:** `internal/api/client.go`
- **Line 99**: `CreateShipment()` method needs validation before creation
- **Line 182**: HTTP status handling - validation failures should be handled similarly
- **Retry logic**: Should validation failures trigger retries? (Probably not)

## Technical Implementation Strategy

### 1. **New Validation Service**
```go
// Location: internal/validation/service.go
type TrackingValidationService struct {
    cache   *cache.Manager
    factory *carriers.Factory
    limiter *ratelimit.RateLimiter
}

func (s *TrackingValidationService) ValidateTracking(ctx context.Context, tracking string, carrier string) (*ValidationResult, error)
```

### 2. **Integration Points**

**Primary Integration:** `email_processor_time.go:319`
```go
// Before calling p.apiClient.CreateShipment(tracking)
validationResult, err := p.validator.ValidateTracking(ctx, tracking.TrackingNumber, tracking.Carrier)
if err != nil || !validationResult.IsValid {
    // Log failure with email content
    p.logger.WarnContext(ctx, "Tracking validation failed", 
        "tracking", tracking.TrackingNumber,
        "carrier", tracking.Carrier,
        "email_subject", emailSubject,
        "email_body", emailBody,
        "error", err)
    return nil // Skip this tracking number
}
```

**Cache Integration:** Extend existing cache manager
```go
// Add validation-specific cache methods
func (m *Manager) GetValidation(trackingNumber string) (*ValidationResult, bool)
func (m *Manager) SetValidation(trackingNumber string, result *ValidationResult)
```

**Rate Limiting Integration:** Extend existing rate limiter
```go
// Add validation-specific rate limiting
func CheckValidationRateLimit(cfg Config, trackingNumber string) RateLimitResult
```

### 3. **Database Schema Extensions**

**New table for validation cache:**
```sql
CREATE TABLE validation_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tracking_number TEXT UNIQUE NOT NULL,
    carrier TEXT NOT NULL,
    is_valid BOOLEAN NOT NULL,
    validation_data TEXT, -- JSON of tracking events
    cached_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);
```

**Extend existing tables:**
```sql
-- Add validation tracking to refresh_tracking
ALTER TABLE refresh_tracking ADD COLUMN last_validation_attempt TIMESTAMP;
ALTER TABLE refresh_tracking ADD COLUMN validation_attempt_count INTEGER DEFAULT 0;
```

## Specific Files That Need Modification

1. **`internal/workers/email_processor_time.go`** - Add validation before shipment creation
2. **`internal/cache/manager.go`** - Extend cache for validation results
3. **`internal/ratelimit/ratelimit.go`** - Add validation rate limiting
4. **`internal/handlers/shipments.go`** - Ensure validation cache invalidation
5. **`internal/database/models.go`** - Add validation cache table
6. **`internal/database/migrations/`** - Add migration for validation cache table

## Similar Features Analyzed

The refresh system provides an excellent blueprint:
- **Caching strategy**: 5-minute TTL with persistence
- **Rate limiting**: 5-minute intervals with bypass options
- **Carrier integration**: Factory pattern with fallback chain
- **Error handling**: Comprehensive logging and retry logic
- **Database integration**: State tracking and cleanup

The validation system should mirror this architecture exactly, just with different cache keys (tracking numbers instead of shipment IDs) and different database tables.