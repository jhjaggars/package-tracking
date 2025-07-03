# Codebase Analysis: UPS Automatic Tracking Implementation

## Current UPS Implementation Status

### 1. UPS API Client Implementation (`/internal/carriers/ups.go`)

**Status: Fully Implemented and Production-Ready**

**Key Features:**
- **OAuth 2.0 Authentication**: Complete implementation with client credentials flow
- **Rate Limiting**: Built-in rate limit handling with retry logic
- **Token Management**: Automatic token refresh and expiry handling
- **Error Handling**: Comprehensive error handling with specific UPS API error codes
- **Multiple Tracking Numbers**: Supports individual tracking (UPS limitation - one per request)

**API Integration Details:**
- **Base URLs**: 
  - Production: `https://onlinetools.ups.com`
  - Sandbox: `https://wwwcie.ups.com`
- **Endpoints Used**:
  - OAuth: `/security/v1/oauth/token`
  - Tracking: `/track/v1/details/{trackingNumber}`
- **Authentication**: OAuth 2.0 with Basic Auth for token requests
- **Rate Limiting**: 100 requests per hour (configurable)

**Configuration Requirements:**
```go
// Environment variables needed
UPS_CLIENT_ID="your_client_id"
UPS_CLIENT_SECRET="your_client_secret"
UPS_SANDBOX=false  // Optional, defaults to production
```

### 2. UPS Scraping Fallback (`/internal/carriers/ups_scraping.go`)

**Status: Implemented with Basic Patterns**

**Features:**
- Web scraping fallback when API credentials unavailable
- Multiple HTML parsing patterns for robustness
- Tracking number validation consistent with API client
- Error detection for "not found" scenarios

### 3. UPS Test Coverage (`/internal/carriers/ups_test.go`)

**Status: Comprehensive Test Suite**

**Test Coverage:**
- OAuth authentication (success/failure scenarios)
- Token expiry and refresh
- Rate limiting handling
- Multiple package tracking
- HTTP error scenarios
- Tracking number validation

## Background Service Architecture

### 1. TrackingUpdater Service (`/internal/workers/tracking_updater.go`)

**Status: Production-Ready with Advanced Features**

**Key Capabilities:**
- **Automatic Updates**: Configurable interval-based updates (default: 1 hour)
- **Cache Integration**: Unified cache-based rate limiting
- **Graceful Control**: Start/stop/pause/resume functionality
- **Error Handling**: Retry logic with failure counting
- **Rate Limiting**: Respects both API limits and manual refresh intervals
- **Batch Processing**: Optimized for carrier API limits

**Current Implementation Scope:**
- **USPS Only**: Currently implements auto-updates for USPS shipments only
- **UPS Ready**: Architecture designed to easily extend to UPS

**Configuration Options:**
```go
// Environment variables for auto-update control
AUTO_UPDATE_ENABLED=true           // Enable/disable auto-updates
AUTO_UPDATE_CUTOFF_DAYS=30         // Only update shipments newer than X days
AUTO_UPDATE_BATCH_SIZE=10          // Batch size for API calls
AUTO_UPDATE_BATCH_TIMEOUT=60s      // Timeout for batch operations
AUTO_UPDATE_INDIVIDUAL_TIMEOUT=30s // Timeout for individual calls
UPDATE_INTERVAL=1h                 // How often to run updates
```

### 2. Cache-Based Rate Limiting (`/internal/cache/manager.go`, `/internal/ratelimit/ratelimit.go`)

**Status: Advanced Implementation**

**Features:**
- **Unified Rate Limiting**: 5-minute cooldown for both manual and auto-refresh
- **Cache-Aware Logic**: Uses cached responses to avoid unnecessary API calls
- **Persistent Cache**: SQLite-backed cache survives server restarts
- **Memory + Database**: Two-tier caching for performance

**Rate Limiting Strategy:**
- Manual refresh: 5-minute cooldown between requests
- Auto-refresh: Uses same logic but without user-facing restrictions
- Cache hits: Don't count against rate limits
- Forced refresh: Bypasses rate limits (manual override)

## Database Schema Analysis

### 1. Shipments Table (`/internal/database/db.go`)

**Auto-Refresh Fields:**
```sql
last_auto_refresh DATETIME           -- Last auto-refresh timestamp
auto_refresh_count INTEGER DEFAULT 0 -- Number of auto-refreshes performed
auto_refresh_enabled BOOLEAN DEFAULT TRUE -- Can be disabled per shipment
auto_refresh_error TEXT              -- Last error message
auto_refresh_fail_count INTEGER DEFAULT 0 -- Consecutive failure count
```

**Tracking Fields:**
```sql
last_manual_refresh DATETIME        -- Last manual refresh timestamp
manual_refresh_count INTEGER DEFAULT 0 -- Number of manual refreshes
```

**Indexes for Performance:**
```sql
-- Optimized for auto-update queries
idx_shipments_auto_update ON shipments(carrier, is_delivered, auto_refresh_enabled, auto_refresh_fail_count, created_at)

-- General performance indexes
idx_shipments_carrier_delivered ON shipments(carrier, is_delivered)
```

### 2. Refresh Cache Table

**Schema:**
```sql
CREATE TABLE refresh_cache (
    shipment_id INTEGER PRIMARY KEY,
    response_data TEXT NOT NULL,        -- JSON serialized response
    cached_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,       -- TTL-based expiry
    FOREIGN KEY (shipment_id) REFERENCES shipments(id) ON DELETE CASCADE
);
```

## Authentication/OAuth Patterns

### 1. UPS OAuth Implementation

**Pattern Used:**
- **Client Credentials Flow**: Standard OAuth 2.0 for server-to-server
- **Token Storage**: In-memory with automatic refresh
- **Error Handling**: Graceful retry on 401 responses
- **Base64 Encoding**: For Basic Auth in token requests

**Code Pattern:**
```go
// OAuth request with Basic Auth
auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
req.Header.Set("Authorization", "Basic "+auth)

// API requests with Bearer token
req.Header.Set("Authorization", "Bearer "+accessToken)
```

### 2. Token Management Strategy

**Features:**
- **Lazy Authentication**: Only authenticate when needed
- **Automatic Refresh**: On 401 responses during API calls
- **Expiry Tracking**: Proactive renewal (though currently not implemented)
- **Thread Safety**: Safe for concurrent access

## Scheduling and Job Runner Infrastructure

### 1. Current Implementation

**Architecture:**
- **Goroutine-Based**: Simple goroutine with ticker for scheduling
- **Context-Based Cancellation**: Graceful shutdown using context
- **Configurable Intervals**: Environment variable controlled
- **Atomic Operations**: Thread-safe pause/resume functionality

**Control Flow:**
```go
// Main update loop
func (u *TrackingUpdater) updateLoop() {
    ticker := time.NewTicker(u.config.UpdateInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-u.ctx.Done():
            return // Graceful shutdown
        case <-ticker.C:
            u.performUpdates() // Execute updates
        }
    }
}
```

### 2. Admin Control API

**Endpoints:**
- `GET /api/admin/tracking-updater/status` - Get updater status
- `POST /api/admin/tracking-updater/pause` - Pause updates
- `POST /api/admin/tracking-updater/resume` - Resume updates

## Integration Patterns for Other Carriers

### 1. Factory Pattern (`/internal/carriers/factory.go`)

**Current Carrier Support:**
- **USPS**: API + Scraping + Headless
- **UPS**: API + Scraping (headless not implemented)
- **FedEx**: API + Scraping + Headless
- **DHL**: API + Scraping

**Configuration Pattern:**
```go
// Example UPS configuration
upsConfig := &carriers.CarrierConfig{
    ClientID:      "ups_client_id",
    ClientSecret:  "ups_client_secret",
    PreferredType: carriers.ClientTypeAPI,
    UseSandbox:    false,
}
carrierFactory.SetCarrierConfig("ups", upsConfig)
```

### 2. Fallback Strategy

**Priority Order:**
1. **API Client** (if credentials configured)
2. **Headless Browser** (if required for carrier)
3. **Scraping Client** (fallback)

## Key Findings and Recommendations

### 1. UPS Implementation Readiness

**Production Ready:**
- ✅ Complete OAuth 2.0 implementation
- ✅ Comprehensive error handling
- ✅ Rate limiting and retry logic
- ✅ Extensive test coverage
- ✅ Factory integration complete

### 2. Background Service Extensibility

**Easy UPS Integration:**
- The `TrackingUpdater` is designed for multi-carrier support
- Only requires adding UPS-specific logic to `updateUPSShipments()` method
- Cache and rate limiting already unified
- Database schema supports all carriers

### 3. Current Limitations

**USPS-Only Auto-Updates:**
- Background service currently only processes USPS shipments
- UPS auto-update capability exists but not activated

**Recommended Next Steps:**
1. Add UPS credentials to environment configuration in `main.go`
2. Implement `updateUPSShipments()` method in `TrackingUpdater`
3. Add UPS-specific configuration validation
4. Update admin controls to support per-carrier management

The existing architecture provides a solid foundation for implementing UPS automatic tracking updates with minimal changes required.