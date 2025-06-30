# Expert Detail Questions

Now that I understand the codebase architecture, here are specific implementation questions:

## Q6: Should the cache persistence file be stored in the same directory as the SQLite database (configurable via DB_PATH)?
**Default if unknown:** Yes (keeps all persistent data together and respects user's data directory preference)

## Q7: Should the cache automatically save to disk periodically during runtime or only on shutdown?
**Default if unknown:** No (only on shutdown to minimize I/O and complexity, matching typical cache patterns)

## Q8: Should failed refresh attempts (errors from carriers) also be cached to prevent repeated failing requests?
**Default if unknown:** No (allow retries in case the error was transient)

## Q9: Should the cache be cleared when a shipment is manually updated via PUT /api/shipments/{id}?
**Default if unknown:** Yes (ensures cache consistency when shipment data changes)

## Q10: Should we add a new environment variable to disable caching entirely (e.g., DISABLE_CACHE=true)?
**Default if unknown:** Yes (provides operational flexibility and easier debugging)