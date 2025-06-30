# Context Findings

## 1. Refresh Handler Analysis (internal/handlers/shipments.go:265-453)

The RefreshShipment handler follows this flow:
1. Validates shipment existence and checks if already delivered
2. Enforces rate limiting (5 minutes between refreshes unless DISABLE_RATE_LIMIT=true)
3. Creates carrier client (API, headless, or scraping)
4. Fetches tracking data with 120-second timeout
5. Processes events and updates shipment status
6. Updates refresh tracking (timestamp and count)
7. Returns RefreshResponse with all events

**Key Response Structure:**
```go
type RefreshResponse struct {
    ShipmentID  int                      `json:"shipment_id"`
    UpdatedAt   time.Time                `json:"updated_at"`
    EventsAdded int                      `json:"events_added"`
    TotalEvents int                      `json:"total_events"`
    Events      []database.TrackingEvent `json:"events"`
}
```

## 2. Go Caching Best Practices Research

Popular libraries and patterns:
- **patrickmn/go-cache**: Simple in-memory cache with expiration
- **Persistence Options**: 
  - Serialize cache to file on shutdown/startup
  - Use embedded databases like Bitcask for persistent KV storage
- **Thread Safety**: Mutex-protected operations required
- **Memory Management**: Time-based expiry preferred over LRU for simplicity

## 3. Existing Persistence Patterns

**Current State:**
- All data persisted immediately to SQLite
- No caching layer exists
- Configuration loaded once from environment/.env files
- CLI has JSON config file at ~/.package-tracker.json

**Shutdown/Startup:**
- Graceful shutdown with signal handling
- Database connection cleanup via defer
- No state saving needed (all in SQLite)

**Serialization:**
- All models have JSON tags
- Standard library json.Marshal/Unmarshal used

## 4. Files That Need Modification

1. **internal/handlers/shipments.go**:
   - Add cache check before rate limit check
   - Store successful refresh results in cache
   - Serve from cache when fresh enough

2. **internal/server/server.go** (or new cache package):
   - Initialize cache on startup
   - Provide cache access to handlers
   - Handle cache persistence on shutdown

3. **internal/config/config.go**:
   - Add cache-related configuration options
   - Cache TTL, persistence path, etc.

4. **cmd/server/main.go**:
   - Initialize cache before server start
   - Load cache from disk on startup
   - Save cache to disk on shutdown

## 5. Technical Constraints and Considerations

1. **Concurrency**: Multiple goroutines will access cache (HTTP handlers)
2. **Memory**: No memory pressure handling needed (time-based expiry only)
3. **Persistence**: Must survive server restarts
4. **Key Design**: Use shipment ID as cache key
5. **Value Storage**: Store entire RefreshResponse
6. **TTL**: 5 minutes (matches rate limit)
7. **Integration**: Transparent to API consumers

## 6. Integration Points

1. **Rate Limiting Interaction**:
   - Cache serves data without triggering rate limit
   - Only actual refresh updates rate limit timestamp

2. **Event Deduplication**:
   - Cache stores response after database deduplication
   - No need to re-process events from cache

3. **Delivery Status**:
   - Delivered shipments should not be cached (409 response)
   - Or cache the 409 response itself

## 7. Similar Features

- **CLI Config Persistence**: Uses JSON file serialization
- **Database Models**: All have JSON tags for serialization
- **HTTP Handlers**: Standard pattern for request/response handling

## 8. Recommended Implementation Approach

1. Use **patrickmn/go-cache** for simplicity
2. Serialize cache to JSON file on shutdown
3. Load cache from JSON file on startup
4. Store RefreshResponse objects with 5-minute TTL
5. Check cache before rate limit enforcement
6. Update cache after successful refresh