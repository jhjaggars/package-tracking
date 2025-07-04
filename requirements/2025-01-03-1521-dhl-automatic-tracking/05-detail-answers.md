# Detail Answers - DHL Automatic Tracking Updates

## Q6: When the DHL API rate limit reaches 80% usage (200 of 250 calls), should the warning be logged every time a subsequent API call is made, or only once when the threshold is first crossed?
**Answer:** No (log only once when threshold is crossed)

## Q7: If DHL API credentials are not configured but automatic updates are enabled, should the system automatically fall back to web scraping for automatic updates?
**Answer:** Yes (automatic fallback to scraping)

## Q8: Should DHL automatic updates respect the same pause/resume admin controls that affect all carrier updates, or need separate DHL-specific controls?
**Answer:** No (use unified pause/resume controls)

## Q9: When both API and scraping fail for a DHL shipment during automatic updates, should the failure count increment by 1 regardless of the failure type?
**Answer:** Yes (treat all failures equally)

## Q10: Should the rate limit warning include actionable information like the estimated time until the limit resets?
**Answer:** Yes (include reset time information)