# Detail Answers

## Q6: Should the cache persistence file be stored in the same directory as the SQLite database (configurable via DB_PATH)?
**Answer:** Yes, and maybe even store it in the database directly
**Implications:** We'll implement cache persistence using SQLite tables rather than a separate file. This provides atomic operations, built-in concurrency handling, and leverages existing database infrastructure.

## Q7: Should the cache automatically save to disk periodically during runtime or only on shutdown?
**Answer:** It isn't necessary, if it is stored in the database this happens automatically
**Implications:** With SQLite-based caching, persistence is automatic on every write. No need for periodic saves or shutdown hooks.

## Q8: Should failed refresh attempts (errors from carriers) also be cached to prevent repeated failing requests?
**Answer:** No
**Implications:** Only successful refresh responses will be cached. Failed requests can be retried immediately.

## Q9: Should the cache be cleared when a shipment is manually updated via PUT /api/shipments/{id}?
**Answer:** Yes
**Implications:** The UpdateShipment handler must invalidate any cached refresh data for the modified shipment.

## Q10: Should we add a new environment variable to disable caching entirely (e.g., DISABLE_CACHE=true)?
**Answer:** Yes
**Implications:** Add DISABLE_CACHE configuration option that bypasses all cache operations when set to true.