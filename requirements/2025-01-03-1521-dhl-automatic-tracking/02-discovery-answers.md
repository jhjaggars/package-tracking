# Discovery Answers - DHL Automatic Tracking Updates

## Q1: Should DHL automatic updates be enabled by default like UPS (true) or disabled by default like the initial implementation pattern?
**Answer:** Yes (enabled by default)

## Q2: Will DHL automatic updates need a carrier-specific cutoff period different from the global 30-day default?
**Answer:** No (use global default)

## Q3: Should the system prioritize API usage over web scraping when DHL credentials are available?
**Answer:** Yes (prioritize API)

## Q4: Will administrators need to monitor DHL-specific update metrics separately from other carriers?
**Answer:** No (use unified monitoring)

## Q5: Should DHL rate limits (250/day) trigger any special handling or warnings in the automatic update process?
**Answer:** Yes (log warnings)