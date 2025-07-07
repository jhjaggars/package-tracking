# Requirements Specification: Email Tracking Validation

## Problem Statement

Currently, the email scanning system extracts tracking numbers from emails and creates shipments without validating that the tracking numbers are legitimate. This can result in invalid tracking numbers being added to the system, creating shipments that will never have tracking data and wasting carrier API quota on failed refresh attempts.

## Solution Overview

Integrate the existing refresh functionality into the email scanning process to validate tracking numbers before creating shipments. This will ensure only legitimate tracking numbers are added to the system and populate initial tracking events when shipments are created.

## Functional Requirements

### FR1: Tracking Number Validation During Email Processing
- **Location**: `internal/workers/email_processor_time.go:319` (before `createShipment()`)
- **Behavior**: Before creating a shipment, validate the tracking number by performing a refresh operation
- **Success**: If validation succeeds, create the shipment with initial tracking events populated
- **Failure**: If validation fails, reject the tracking number and log the failure with email content

### FR2: Cache Integration
- **System**: Use existing `refresh_cache` table and `internal/cache/manager.go`
- **Behavior**: Validation operations should check and populate the same cache as refresh operations
- **TTL**: Use same 5-minute cache TTL as refresh system
- **Keys**: Cache by tracking number (not shipment ID since shipment doesn't exist yet)

### FR3: Rate Limiting Integration
- **System**: Use existing `internal/ratelimit/ratelimit.go` rate limiting
- **Behavior**: Validation operations should respect the same 5-minute rate limits as refresh
- **Bypass**: No bypassing for batch processing - rate limiting applies to all validation attempts
- **Tracking**: Use same refresh tracking fields in database

### FR4: Carrier Client Integration
- **System**: Use existing `internal/carriers/factory.go` factory pattern
- **Behavior**: Validation should use same carrier client selection (API → Headless → Scraping)
- **Preferences**: Respect existing carrier preferences (FedEx API, others headless/scraping)

### FR5: Error Logging and Debugging
- **Requirement**: Log all validation failures with detailed information
- **Content**: Include tracking number, carrier, email subject, email body, and error details
- **Purpose**: Enable debugging of false negatives and pattern matching improvements
- **Level**: Use WARN level logging for failed validations

## Technical Requirements

### TR1: Integration Point
```go
// File: internal/workers/email_processor_time.go
// Location: Line 319, before p.apiClient.CreateShipment(tracking)

validationResult, err := p.validateTracking(ctx, tracking.TrackingNumber, tracking.Carrier)
if err != nil || !validationResult.IsValid {
    p.logger.WarnContext(ctx, "Tracking validation failed", 
        "tracking", tracking.TrackingNumber,
        "carrier", tracking.Carrier,
        "email_subject", emailSubject,
        "email_body", emailBody,
        "error", err)
    continue // Skip this tracking number
}
```

### TR2: Validation Service Interface
```go
// File: internal/workers/email_processor_time.go (add to struct)
type TimeBasedEmailProcessor struct {
    // ... existing fields
    refreshHandler *handlers.ShipmentHandler // Reuse existing refresh logic
}

// Method to perform validation
func (p *TimeBasedEmailProcessor) validateTracking(ctx context.Context, trackingNumber, carrier string) (*ValidationResult, error) {
    // Create temporary shipment-like structure for refresh operation
    // Use existing refresh handler logic
    // Return validation result
}
```

### TR3: Cache Key Strategy
- **Current refresh cache**: Uses shipment ID as key
- **Validation cache**: Use tracking number as key (format: `validation:{trackingNumber}`)
- **Shared storage**: Same `refresh_cache` table, different key prefix
- **Data structure**: Store validation result in same JSON format as refresh response

### TR4: Rate Limiting Integration
```go
// File: internal/ratelimit/ratelimit.go
// Extend existing CheckRefreshRateLimit function or create validation-specific variant
func CheckValidationRateLimit(cfg Config, trackingNumber string) RateLimitResult {
    // Use same 5-minute rate limiting logic
    // Track validation attempts in refresh_tracking table
}
```

### TR5: Database Schema (No Changes Required)
- **Existing tables**: Use existing `refresh_cache` and `refresh_tracking` tables
- **Cache storage**: Store validation results in `refresh_cache` with tracking number keys
- **Rate limiting**: Use existing `refresh_tracking` table with tracking number as identifier

## Implementation Hints

### 1. Reuse Existing Refresh Logic
The refresh handler at `internal/handlers/shipments.go:319-573` contains all the logic needed:
- Cache checking (line 371)
- Rate limiting (line 397)
- Client factory usage (line 404)
- Carrier API calls (line 449)
- Cache storage (line 561)

### 2. Minimal Code Changes
Create a validation wrapper that calls the existing refresh logic:
```go
func (p *TimeBasedEmailProcessor) validateTracking(ctx context.Context, trackingNumber, carrier string) (*ValidationResult, error) {
    // Create minimal shipment structure
    tempShipment := &database.Shipment{
        TrackingNumber: trackingNumber,
        Carrier:        carrier,
    }
    
    // Use existing refresh handler logic
    result, err := p.refreshHandler.performRefresh(ctx, tempShipment)
    if err != nil {
        return &ValidationResult{IsValid: false}, err
    }
    
    return &ValidationResult{
        IsValid: true,
        TrackingEvents: result.TrackingEvents,
    }, nil
}
```

### 3. Error Handling Patterns
Follow existing error handling patterns in the codebase:
- Use structured logging with context
- Distinguish between retryable and non-retryable errors
- Log sensitive information appropriately (email content may contain PII)

## Acceptance Criteria

### AC1: Validation Integration
- [ ] Email processing validates tracking numbers before creating shipments
- [ ] Only validated tracking numbers create shipments in the system
- [ ] Initial tracking events are populated when shipments are created

### AC2: Cache Behavior
- [ ] Validation results are cached for 5 minutes
- [ ] Subsequent validation attempts for same tracking number use cache
- [ ] Cache persists across server restarts

### AC3: Rate Limiting
- [ ] Validation respects 5-minute rate limits per tracking number
- [ ] Rate limiting applies to all processing modes (including dry-run)
- [ ] Rate limiting can be disabled via existing `DISABLE_RATE_LIMIT` config

### AC4: Error Handling
- [ ] Failed validations are logged with tracking number, carrier, and email content
- [ ] Failed validations don't create shipments
- [ ] System continues processing other tracking numbers after validation failures

### AC5: Carrier Integration
- [ ] Validation uses same carrier client selection as refresh
- [ ] FedEx validation uses API when available
- [ ] Other carriers use headless/scraping as configured

## Assumptions

1. **Email content logging**: Assuming email content doesn't contain excessive PII that would prevent logging
2. **Performance impact**: Assuming the validation delay is acceptable for email processing workflow
3. **Carrier API limits**: Assuming validation calls won't exceed carrier API quotas
4. **Backward compatibility**: Existing refresh functionality remains unchanged
5. **Configuration**: Existing refresh configuration (cache TTL, rate limits) applies to validation