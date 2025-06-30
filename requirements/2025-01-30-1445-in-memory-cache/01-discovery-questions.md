# Discovery Questions

Based on the codebase analysis, here are the key questions to understand the caching requirements:

## Q1: Should the cache be shared across all concurrent requests to the server?
**Default if unknown:** Yes (a global cache maximizes efficiency and prevents redundant API calls across different users/sessions)

## Q2: Should the cache persist across server restarts?
**Default if unknown:** No (in-memory cache typically implies ephemeral storage that clears on restart)

## Q3: Should the cache respect the existing rate limit configuration (DISABLE_RATE_LIMIT)?
**Default if unknown:** Yes (cache should complement existing rate limiting, not replace it)

## Q4: Should cache entries be evicted based on memory pressure or only time-based expiry?
**Default if unknown:** No (time-based expiry only for simplicity and predictable behavior)

## Q5: Should the cache store the full response including all tracking events or just the latest status?
**Default if unknown:** Yes (caching full response maintains API compatibility and reduces database queries)