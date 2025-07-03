# Detail Questions: UPS Automatic Tracking Implementation

## Q1: Should we add UPS_CLIENT_ID and UPS_CLIENT_SECRET to replace the existing UPS_API_KEY?
**Default if unknown:** Yes (the current UPS_API_KEY is incomplete for OAuth authentication)

The current UPS implementation expects both ClientID and ClientSecret for OAuth 2.0, but the config only has UPS_API_KEY. We need both credentials for the UPS API to work properly.

## Q2: Should UPS auto-updates be included in the existing TrackingUpdater.performUpdates() method?
**Default if unknown:** Yes (maintains single update cycle and consistent scheduling)

The current USPS auto-updates run in performUpdates(). Adding UPS to the same method ensures both carriers update together on the same schedule, consistent with the unified approach.

## Q3: Should the failure threshold (currently hard-coded at 10) become a global setting or per-carrier?
**Default if unknown:** Global setting (AUTO_UPDATE_FAILURE_THRESHOLD applies to all carriers)

The current system has a hard-coded 10 failure threshold. A global setting would be simpler to configure and maintain consistency across carriers.

## Q4: Should UPS auto-updates respect the existing 5-minute cache-based rate limiting?
**Default if unknown:** Yes (uses the same processShipmentsWithCache() method as USPS)

The current USPS auto-updates use unified cache-based rate limiting through processShipmentsWithCache(). UPS should use the same pattern for consistency.

## Q5: Should we update the GetActiveForAutoUpdate() method to accept a configurable failure threshold parameter?
**Default if unknown:** Yes (enables the configurable failure threshold from Q3)

The current method has a hard-coded 10 failure threshold in the SQL query. Making it a parameter allows for the configurable threshold setting.