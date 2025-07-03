# Discovery Questions: UPS Automatic Tracking Updates

## Q1: Should UPS automatic updates run on the same schedule as USPS updates?
**Default if unknown:** Yes (maintains consistency with existing auto-update behavior)

The system currently runs USPS auto-updates every hour by default. UPS automatic updates could follow the same pattern for consistency, or have carrier-specific intervals.

## Q2: Should UPS automatic updates respect the same 30-day cutoff as USPS?
**Default if unknown:** Yes (prevents updating very old shipments that are likely delivered)

The current system only auto-updates shipments created within the last 30 days to avoid unnecessary API calls for old packages. This same logic should apply to UPS for efficiency.

## Q3: Should UPS automatic updates be enabled by default for all new UPS shipments?
**Default if unknown:** Yes (matches existing behavior for USPS shipments)

Currently, USPS shipments have auto-refresh enabled by default. UPS shipments should follow the same pattern unless there's a specific reason to disable it.

## Q4: Should UPS automatic updates continue trying after consecutive failures?
**Default if unknown:** Yes, with a reasonable failure threshold (matches USPS behavior)

The system tracks consecutive failure counts and could disable auto-updates after repeated failures to avoid hitting rate limits. A threshold of 3-5 consecutive failures would be reasonable.

## Q5: Should UPS automatic updates work without API credentials using scraping fallback?
**Default if unknown:** Yes (provides functionality even without API access)

The UPS implementation supports both API and scraping. Auto-updates should work with either method, though API is preferred for reliability and rate limit management.