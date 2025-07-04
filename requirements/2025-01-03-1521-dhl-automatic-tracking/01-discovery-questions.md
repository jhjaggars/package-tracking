# Discovery Questions - DHL Automatic Tracking Updates

Based on the codebase analysis, here are the key discovery questions to understand the requirements for implementing DHL automatic tracking updates:

## Q1: Should DHL automatic updates be enabled by default like UPS (true) or disabled by default like the initial implementation pattern?
**Default if unknown:** Yes (enabled by default, following the current UPS pattern where UPS_AUTO_UPDATE_ENABLED defaults to true)

## Q2: Will DHL automatic updates need a carrier-specific cutoff period different from the global 30-day default?
**Default if unknown:** No (use the global AUTO_UPDATE_CUTOFF_DAYS value of 30 days, consistent with other carriers)

## Q3: Should the system prioritize API usage over web scraping when DHL credentials are available?
**Default if unknown:** Yes (API is more reliable and efficient than scraping, following the factory pattern already in place)

## Q4: Will administrators need to monitor DHL-specific update metrics separately from other carriers?
**Default if unknown:** No (use existing unified logging and metrics, maintaining consistency with current implementation)

## Q5: Should DHL rate limits (250/day) trigger any special handling or warnings in the automatic update process?
**Default if unknown:** Yes (log warnings when approaching limits to help administrators manage API usage)