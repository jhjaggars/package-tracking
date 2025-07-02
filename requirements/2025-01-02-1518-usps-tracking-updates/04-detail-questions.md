# Expert Detail Questions for USPS Automatic Tracking Updates

## Q6: Should the auto-updater skip packages that have been manually refreshed within the last 5 minutes (matching the manual refresh rate limit)?
**Default if unknown:** Yes (avoid duplicate API calls and respect the existing rate limiting logic)

## Q7: When a USPS API call fails for a batch of 10 tracking numbers, should we retry individual tracking numbers separately to isolate the problematic one?
**Default if unknown:** Yes (one bad tracking number shouldn't block updates for 9 good ones)

## Q8: Should the pause/resume state persist across server restarts (stored in database or config file)?
**Default if unknown:** No (ephemeral pause is simpler and safer - updates resume on restart by default)

## Q9: Should auto-update errors count toward a maximum retry limit per shipment to prevent infinite retry loops?
**Default if unknown:** Yes (after 10 consecutive failures, mark shipment as problematic and skip future auto-updates)

## Q10: Should the Prometheus metrics include per-carrier labels to track USPS performance separately from future carrier implementations?
**Default if unknown:** Yes (carrier-specific metrics enable better monitoring and debugging)