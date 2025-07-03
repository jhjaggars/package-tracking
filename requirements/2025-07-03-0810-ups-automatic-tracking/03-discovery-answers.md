# Discovery Answers: UPS Automatic Tracking Updates

## Q1: Should UPS automatic updates run on the same schedule as USPS updates?
**Answer:** Yes, the entire update cadence should be user configurable though

**Implications:**
- UPS updates will use the same scheduling system as USPS
- The UPDATE_INTERVAL environment variable should control all carriers
- User wants more granular control over update timing
- May need per-carrier scheduling configuration options

## Q2: Should UPS automatic updates respect the same 30-day cutoff as USPS?
**Answer:** Yes, the cutoff should also be user configurable but the 30 day limit is a good default

**Implications:**
- UPS updates will use the same AUTO_UPDATE_CUTOFF_DAYS setting as USPS
- 30 days remains the default cutoff for efficiency
- User wants configurability for the cutoff period
- Single setting applies to all carriers for consistency

## Q3: Should UPS automatic updates be enabled by default for all new UPS shipments?
**Answer:** Yes

**Implications:**
- UPS shipments will have auto_refresh_enabled=true by default
- Consistent behavior across all carriers
- Users can still disable auto-updates per shipment if needed
- Matches existing USPS behavior

## Q4: Should UPS automatic updates continue trying after consecutive failures?
**Answer:** Yes, the threshold should be user configurable

**Implications:**
- UPS updates will track consecutive failure counts like USPS
- Need new environment variable for failure threshold configuration
- Default threshold should be reasonable (3-5 failures)
- After threshold reached, auto-updates disabled for that shipment
- User can re-enable auto-updates manually if needed

## Q5: Should UPS automatic updates work without API credentials using scraping fallback?
**Answer:** Yes

**Implications:**
- UPS auto-updates will use the existing factory pattern (API preferred, scraping fallback)
- Works without UPS API credentials configured
- Scraping method will be slower but still functional
- Rate limiting considerations differ between API and scraping methods