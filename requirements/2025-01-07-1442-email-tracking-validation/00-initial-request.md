# Initial Request

**Date:** 2025-01-07 14:42
**User Request:** when scanning email each tracking number should be checked the same way refresh works so that initial tracking events are populated when the event is added.  Additionally, if the tracking number fails to refresh then the event should be rejected since it probably isn't a real tracking number.

## Context
The user wants to enhance the email scanning functionality to validate tracking numbers by attempting to refresh them during the scanning process. This would:
1. Populate initial tracking events when a shipment is created from email
2. Reject invalid tracking numbers that fail to refresh
3. Ensure only legitimate tracking numbers are added to the system

## Technology Stack
- Go backend with SQLite database
- Gmail API integration for email processing
- Multiple carrier support (UPS, USPS, FedEx, DHL)
- Existing refresh functionality with caching