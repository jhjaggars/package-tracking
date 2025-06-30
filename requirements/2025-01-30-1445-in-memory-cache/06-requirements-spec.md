# Requirements Specification: In-Memory Cache for Refresh Functionality

## Problem Statement
The package tracking system currently enforces a 5-minute rate limit between refresh requests, returning HTTP 429 errors when users attempt to refresh too frequently. This creates a poor user experience when users legitimately want to check status but haven't waited long enough. Additionally, the system performs expensive carrier API calls or web scraping operations even when recent data might be sufficient.

## Solution Overview
Implement an in-memory cache with database persistence that stores refresh responses for 5 minutes. When users request a refresh, the system will:
1. Check if cached data exists and is less than 5 minutes old
2. If yes, serve from cache without rate limit errors
3. If no, perform actual refresh and cache the result
4. Persist cache to SQLite for survival across server restarts

## Functional Requirements

### FR1: Cache Storage
- Store complete RefreshResponse objects in memory with 5-minute TTL
- Use shipment ID as the cache key
- Include all fields: shipment_id, updated_at, events_added, total_events, events array

### FR2: Cache Checking
- Check cache BEFORE rate limit validation in RefreshShipment handler
- If cache hit and data < 5 minutes old, return cached response with HTTP 200
- If cache miss or stale, proceed with normal refresh flow

### FR3: Cache Updates
- After successful refresh (no carrier errors), store response in cache
- Do NOT cache failed refresh attempts (carrier errors, timeouts, etc.)
- Cache entry expires exactly 5 minutes after creation

### FR4: Cache Invalidation
- Clear cache entry when shipment is updated via PUT /api/shipments/{id}
- No cache for delivered shipments (they return 409 anyway)
- Manual cache management not exposed via API

### FR5: Persistence
- Store cache in SQLite database (same database as shipments)
- Create new table: refresh_cache
- Automatically persist on write (no shutdown hooks needed)
- Load cache entries on server startup

### FR6: Configuration
- Add DISABLE_CACHE environment variable (default: false)
- When true, bypass all caching logic
- Document in README.md and configuration files

## Technical Requirements

### TR1: Database Schema
Create new table in existing SQLite database:
```sql
CREATE TABLE refresh_cache (
    shipment_id INTEGER PRIMARY KEY,
    response_data TEXT NOT NULL,
    cached_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    FOREIGN KEY (shipment_id) REFERENCES shipments(id) ON DELETE CASCADE
);
```

### TR2: Implementation Components
1. **internal/database/cache.go**: New file with RefreshCache model and store
2. **internal/handlers/shipments.go**: Modify RefreshShipment handler
3. **internal/config/config.go**: Add DisableCache configuration
4. **internal/database/db.go**: Add cache table migration
5. **cmd/server/main.go**: Initialize cache store

### TR3: Cache Store Interface
```go
type RefreshCacheStore interface {
    Get(shipmentID int) (*RefreshResponse, error)
    Set(shipmentID int, response *RefreshResponse, ttl time.Duration) error
    Delete(shipmentID int) error
    DeleteExpired() error
    LoadAll() (map[int]*CachedResponse, error)
}
```

### TR4: In-Memory Structure
- Use sync.Map or mutex-protected map[int]*CachedResponse
- CachedResponse includes RefreshResponse + expiry time
- Background goroutine for periodic expired entry cleanup (every minute)

### TR5: Startup/Shutdown Behavior
- On startup: Load non-expired entries from database into memory
- During runtime: Write-through cache (memory + database)
- On shutdown: No special handling needed (database already has data)

## Implementation Hints

### Handler Modification Pattern
In RefreshShipment handler (internal/handlers/shipments.go:266+):
1. After shipment validation, before rate limit check
2. Check if caching is enabled (!config.DisableCache)
3. Try to get from cache
4. If found and not expired, return cached response
5. Otherwise continue with existing flow
6. After successful refresh, store in cache

### Database Migration
Add to migrate() function in internal/database/db.go:
- Create refresh_cache table if not exists
- Add index on expires_at for efficient cleanup queries

### Testing Considerations
- Test cache hit/miss scenarios
- Test expiration behavior
- Test persistence across restarts
- Test cache invalidation on shipment update
- Test with DISABLE_CACHE=true

## Acceptance Criteria

1. **Users can refresh without rate limit errors if last refresh was < 5 minutes ago**
   - GET /api/shipments/{id}/refresh returns cached data instead of 429 error
   - Response indicates data is from cache (optional)

2. **Cache survives server restarts**
   - Cached data persists in SQLite
   - Non-expired entries reload on startup

3. **Cache maintains consistency**
   - Updated shipments clear their cache
   - Failed refreshes don't pollute cache
   - Expired entries are cleaned up

4. **Performance improvement**
   - Cached responses return in < 10ms
   - No carrier API calls for cached data

5. **Operational control**
   - DISABLE_CACHE=true bypasses all caching
   - No memory leaks from expired entries

## Assumptions
- 5-minute TTL is sufficient for user needs
- SQLite performance is adequate for cache operations
- Memory usage is acceptable (limited by 5-minute expiry)
- No need for cache statistics or monitoring endpoints
- No need for manual cache clear functionality