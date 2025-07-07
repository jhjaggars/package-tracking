# Discovery Questions

## Q1: Should the email scanning validation use the same caching system as refresh?
**Default if unknown:** Yes (maintains consistency and reduces carrier API load)

**Rationale:** The existing refresh system has intelligent two-tier caching (memory + SQLite) with 5-minute TTL. Using the same cache would prevent duplicate API calls when scanning emails containing tracking numbers that were recently refreshed.

## Q2: Should tracking numbers that fail validation be logged for debugging purposes?
**Default if unknown:** Yes (helps improve pattern matching and identify carrier API issues)

**Rationale:** Since we're rejecting potentially valid tracking numbers, logging failed validations would help identify false negatives and improve the system over time.

## Q3: Should the validation process respect the same rate limiting as refresh?
**Default if unknown:** Yes (prevents overwhelming carrier APIs during bulk email processing)

**Rationale:** The current system has rate limiting to prevent abuse. Email scanning could process many emails at once, so rate limiting would protect carrier API quotas.

## Q4: Should failed validation attempts count toward the auto-update failure threshold?
**Default if unknown:** No (email scanning validation is different from ongoing tracking updates)

**Rationale:** The system tracks consecutive failures for shipments and disables auto-updates after a threshold. Email validation failures are about initial verification, not ongoing tracking issues.

## Q5: Should the system provide different validation behavior for different carriers?
**Default if unknown:** Yes (some carriers have more reliable APIs than others)

**Rationale:** Based on the codebase, FedEx prefers API calls while others use headless/scraping. Different carriers may need different validation strategies or timeouts.