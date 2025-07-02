# Initial Request: Implement USPS automatic tracking updates

## Issue #8 Details

**Title:** Implement USPS automatic tracking updates
**State:** OPEN
**Author:** jhjaggars
**Labels:** enhancement
**Number:** 8

## Summary
Implement automatic tracking updates specifically for USPS shipments using the existing USPS API and web scraping clients.

## Acceptance Criteria
- [ ] Fetch active USPS shipments from database
- [ ] Use USPS client factory (API with web scraping fallback)
- [ ] Handle USPS-specific tracking number formats (20+ formats supported)
- [ ] Process USPS tracking events and status updates
- [ ] Handle USPS API rate limits (XML API restrictions)
- [ ] Update database with new tracking information
- [ ] Log USPS-specific update operations and errors
- [ ] Support USPS batch tracking (up to 10 tracking numbers)

## USPS-Specific Considerations
- **API**: XML-based API with user ID authentication
- **Rate Limits**: Conservative rate limiting for XML API
- **Batch Support**: Up to 10 tracking numbers per API call
- **Tracking Formats**: 20+ supported formats (Priority, Express, etc.)
- **Web Scraping**: Fallback for when API is unavailable
- **Status Mapping**: Map USPS statuses to standardized TrackingStatus

## Technical Implementation
- Extend existing USPS client in `internal/carriers/usps/`
- Use existing factory pattern for API/scraping selection
- Implement USPS-specific error handling
- Support for USPS tracking number validation
- Integration with background service scheduler

## Error Handling
- USPS XML API authentication failures
- USPS service availability issues
- Invalid tracking number responses
- Rate limit exceeded scenarios
- Network connectivity issues

## Testing
- Unit tests for USPS update logic
- Integration tests with mock USPS responses
- Error scenario testing
- Rate limit handling validation

## Dependencies
- Parent issue: #2 (automatic tracking updates)
- Related: #3 (configurable intervals)
- Related: #6 (retry logic)
- Related: #7 (smart fallback)

## Definition of Done
- [ ] USPS shipments update automatically on schedule
- [ ] API and web scraping fallback both working
- [ ] Comprehensive error handling and logging
- [ ] Unit and integration tests passing
- [ ] Documentation updated

Part of: Phase 3: Background Services
Relates to: #2