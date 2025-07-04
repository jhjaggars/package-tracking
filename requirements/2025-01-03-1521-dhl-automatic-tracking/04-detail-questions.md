# Expert Detail Questions - DHL Automatic Tracking Updates

Now that I understand the codebase architecture and patterns, here are detailed questions about expected system behavior:

## Q6: When the DHL API rate limit reaches 80% usage (200 of 250 calls), should the warning be logged every time a subsequent API call is made, or only once when the threshold is first crossed?
**Default if unknown:** No (log only once when threshold is first crossed to avoid log spam, reset when rate limit resets)

## Q7: If DHL API credentials are not configured but automatic updates are enabled, should the system automatically fall back to web scraping for automatic updates?
**Default if unknown:** Yes (follow the existing carrier client factory pattern which already handles this fallback gracefully)

## Q8: Should DHL automatic updates respect the same pause/resume admin controls that affect all carrier updates, or need separate DHL-specific controls?
**Default if unknown:** No (use unified pause/resume controls - the current implementation pauses all carriers together)

## Q9: When both API and scraping fail for a DHL shipment during automatic updates, should the failure count increment by 1 regardless of the failure type?
**Default if unknown:** Yes (treat all failures equally - the existing pattern doesn't distinguish between failure types)

## Q10: Should the rate limit warning include actionable information like the estimated time until the limit resets?
**Default if unknown:** Yes (include reset time if available from the API response to help administrators plan their usage)