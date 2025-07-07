# Detail Questions

## Q1: Should validation use the same 5-minute cache TTL as refresh, or a longer period since tracking data is more stable?
**Default if unknown:** Yes, use the same 5-minute TTL (maintains consistency with refresh system)

**Rationale:** The refresh system uses 5-minute TTL in `/internal/cache/manager.go`. Since validation is checking if a tracking number is legitimate (which doesn't change), we could use a longer TTL, but consistency with the refresh system is more important.

## Q2: Should validation failures be stored in the same `refresh_cache` table or a separate `validation_cache` table?
**Default if unknown:** Separate `validation_cache` table (different data structure and lifecycle)

**Rationale:** Validation results have different data (tracking number + validity) vs refresh results (shipment ID + tracking events). The cache manager in `/internal/cache/manager.go` could handle both, but separate tables allow for different cleanup strategies.

## Q3: Should the validation service integrate with the existing `carriers.Factory` at line 57 in `internal/carriers/factory.go`?
**Default if unknown:** Yes (maintains carrier preference logic: API → Headless → Scraping)

**Rationale:** The factory already handles carrier-specific client selection. Validation should respect the same preferences (FedEx API, others headless/scraping) to maintain consistency.

## Q4: Should validation bypass rate limiting when processing emails in batch mode (like the `--dry-run` flag)?
**Default if unknown:** No (rate limiting protects carrier APIs regardless of processing mode)

**Rationale:** The rate limiter in `/internal/ratelimit/ratelimit.go` protects against API abuse. Even in dry-run mode, we'd be making real carrier API calls for validation, so rate limiting should still apply.

## Q5: Should failed validation attempts count against the same tracking fields as refresh (like `last_manual_refresh` in the database)?
**Default if unknown:** No (use separate validation tracking fields)

**Rationale:** Validation is a different operation than refresh. The database already tracks refresh attempts, but validation attempts should have their own tracking to avoid interfering with existing refresh logic.