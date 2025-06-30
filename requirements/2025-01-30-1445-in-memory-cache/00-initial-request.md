# Initial Request

**Date:** 2025-01-30 14:45
**Request:** Introduce an in-memory cache for the refresh functionality. When a user requests a refresh, if the cache for that shipment is older than 5 minutes perform the refresh action, otherwise serve the results from the cache.

## Summary
The user wants to implement caching for the refresh functionality to improve performance and reduce unnecessary API calls or scraping operations. The cache should:
- Store refresh results in memory
- Check cache age before performing refresh
- Use cached results if less than 5 minutes old
- Perform actual refresh if cache is stale or missing