# Detail Answers: UPS Automatic Tracking Implementation

## Q1: Should we add UPS_CLIENT_ID and UPS_CLIENT_SECRET to replace the existing UPS_API_KEY?
**Answer:** Yes, align the auth mechanisms

**Implications:**
- Replace UPS_API_KEY with UPS_CLIENT_ID and UPS_CLIENT_SECRET
- Update configuration loading to use the new environment variables
- Update factory pattern to use both credentials for UPS OAuth
- Ensure backward compatibility or provide migration guidance
- Align with OAuth 2.0 client credentials flow requirements

## Q2: Should UPS auto-updates be included in the existing TrackingUpdater.performUpdates() method?
**Answer:** Yes

**Implications:**
- Add UPS auto-update logic to the existing performUpdates() method
- UPS and USPS updates run in the same cycle on the same schedule
- Maintains unified approach and consistent behavior
- Single point of control for all carrier auto-updates
- Ensures both carriers benefit from the same scheduling and control mechanisms

## Q3: Should the failure threshold (currently hard-coded at 10) become a global setting or per-carrier?
**Answer:** Global setting

**Implications:**
- Add AUTO_UPDATE_FAILURE_THRESHOLD environment variable (default: 10)
- Single configuration applies to all carriers for consistency
- Update GetActiveForAutoUpdate() method to accept threshold parameter
- Simpler configuration management
- Consistent behavior across all carriers

## Q4: Should UPS auto-updates respect the existing 5-minute cache-based rate limiting?
**Answer:** Yes, this cache TTL should be user configurable also

**Implications:**
- UPS auto-updates use the same processShipmentsWithCache() method as USPS
- Add CACHE_TTL environment variable to make the 5-minute cache duration configurable
- Both manual and automatic refreshes use the same configurable cache TTL
- Maintains unified rate limiting approach across all carriers
- Allows users to adjust cache behavior based on their needs

## Q5: Should we update the GetActiveForAutoUpdate() method to accept a configurable failure threshold parameter?
**Answer:** Yes

**Implications:**
- Update GetActiveForAutoUpdate() method signature to accept failureThreshold parameter
- Remove hard-coded 10 failure limit from SQL query
- Pass configurable threshold from TrackingUpdater to database method
- Enables the global AUTO_UPDATE_FAILURE_THRESHOLD setting
- Maintains backward compatibility with existing functionality