# Detail Answers for USPS Automatic Tracking Updates

## Q6: Should the auto-updater skip packages that have been manually refreshed within the last 5 minutes (matching the manual refresh rate limit)?
**Answer:** Yes, respect the rate limit rules

## Q7: When a USPS API call fails for a batch of 10 tracking numbers, should we retry individual tracking numbers separately to isolate the problematic one?
**Answer:** Yes

## Q8: Should the pause/resume state persist across server restarts (stored in database or config file)?
**Answer:** No

## Q9: Should auto-update errors count toward a maximum retry limit per shipment to prevent infinite retry loops?
**Answer:** Yes

## Q10: Should the Prometheus metrics include per-carrier labels to track USPS performance separately from future carrier implementations?
**Answer:** Yes