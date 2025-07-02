# Discovery Answers for USPS Automatic Tracking Updates

## Q1: Should the automatic updates run for all USPS shipments regardless of their age?
**Answer:** No, we should add a configuration option for how far back updates will be performed (for example, oldest shipped date)

## Q2: Should automatic updates continue for packages marked as delivered?
**Answer:** No

## Q3: Will administrators need the ability to pause/resume automatic updates without restarting the server?
**Answer:** Yes

## Q4: Should failed update attempts be retried automatically on the next cycle?
**Answer:** Yes

## Q5: Do we need to track metrics about automatic update performance (success rate, API calls, etc.)?
**Answer:** Yes, make a side note to instrument the api server project with prometheus metric exposition

## Additional Notes
- Need to add Prometheus metrics exposition to the entire API server project (not just USPS updates)