# Discovery Questions for USPS Automatic Tracking Updates

## Q1: Should the automatic updates run for all USPS shipments regardless of their age?
**Default if unknown:** No (older delivered packages don't need updates, focus on active shipments)

## Q2: Should automatic updates continue for packages marked as delivered?
**Default if unknown:** No (delivered packages are complete, no need to waste API calls)

## Q3: Will administrators need the ability to pause/resume automatic updates without restarting the server?
**Default if unknown:** Yes (operational flexibility is important for production systems)

## Q4: Should failed update attempts be retried automatically on the next cycle?
**Default if unknown:** Yes (transient failures shouldn't permanently stop tracking)

## Q5: Do we need to track metrics about automatic update performance (success rate, API calls, etc.)?
**Default if unknown:** Yes (observability helps troubleshoot issues and optimize performance)