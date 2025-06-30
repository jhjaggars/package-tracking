# Discovery Answers

## Q1: Should the cache be shared across all concurrent requests to the server?
**Answer:** Yes
**Implications:** We'll implement a global cache that's accessible across all HTTP handlers and goroutines.

## Q2: Should the cache persist across server restarts?
**Answer:** Yes
**Implications:** This changes the requirement from pure in-memory to persistent cache. We'll need to implement cache serialization/deserialization on startup/shutdown.

## Q3: Should the cache respect the existing rate limit configuration (DISABLE_RATE_LIMIT)?
**Answer:** Yes, but the endpoint should not return a rate limit response, rather it should just serve from cache if it has not been long enough to perform the refresh
**Implications:** The cache serves as a transparent layer - if data is fresh enough (< 5 minutes), serve from cache without rate limit errors. Only perform actual refresh if cache is stale.

## Q4: Should cache entries be evicted based on memory pressure or only time-based expiry?
**Answer:** No
**Implications:** Simple time-based expiry only. No need for LRU or memory-based eviction strategies.

## Q5: Should the cache store the full response including all tracking events or just the latest status?
**Answer:** Yes
**Implications:** Cache the complete refresh response to maintain API compatibility and avoid additional database queries.