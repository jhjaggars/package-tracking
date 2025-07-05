# Amazon Shipments Requirements Specification

## Problem Statement

Users want to track Amazon shipments through the existing package tracking system. Amazon presents unique challenges as they:
- Use order numbers (###-#######-#######) instead of traditional tracking numbers
- Delegate deliveries to multiple carriers (UPS, USPS, FedEx, DHL, Amazon Logistics)
- Don't provide public APIs for tracking
- Have sophisticated anti-bot measures

## Solution Overview

Implement Amazon as a separate carrier type that:
- Accepts Amazon order numbers as tracking identifiers
- Parses Amazon emails to extract carrier delegation information
- Updates shipment records with delegation fields rather than creating separate shipments
- Uses existing carrier integrations when Amazon delegates to third-party carriers
- Relies on email parsing for Amazon Logistics (AMZL) tracking

## Functional Requirements

### FR1: Amazon Order Number Support
- **Requirement**: System must accept Amazon order numbers in format ###-#######-#######
- **Implementation**: Add `amazon_order_number` field to shipments table
- **Validation**: Implement validation in `AmazonClient.ValidateTrackingNumber()`
- **API**: Extend existing shipment creation endpoints to accept Amazon order numbers

### FR2: Carrier Delegation Handling
- **Requirement**: When Amazon delegates to UPS/FedEx/USPS/DHL, update original shipment with delegation info
- **Implementation**: Add `delegated_carrier` and `delegated_tracking_number` fields
- **Behavior**: Single shipment record shows "Amazon" as carrier but uses delegated carrier for tracking
- **API**: Tracking calls should transparently delegate to appropriate carrier client

### FR3: Amazon Logistics Support
- **Requirement**: Support Amazon's own delivery network (AMZL) tracking
- **Implementation**: Handle TBA############ tracking numbers
- **Limitation**: Email parsing only (no direct Amazon scraping)
- **Tracking**: Extract AMZL updates from Amazon shipping emails

### FR4: Email Processing Integration
- **Requirement**: Automatically detect and process Amazon shipping emails
- **Implementation**: Extend existing email parsing patterns
- **Domains**: amazon.com, shipment-tracking.amazon.com, marketplace.amazon.com
- **Patterns**: Order numbers, AMZL tracking, delegated carrier information

### FR5: User Interface Integration
- **Requirement**: Amazon appears as a carrier option in CLI and API
- **Implementation**: Add "amazon" to supported carriers list
- **Display**: Show Amazon as carrier with order number and/or delegated tracking info
- **Validation**: Prevent creation of Amazon shipments without order number or tracking number

## Technical Requirements

### TR1: Database Schema Changes
**File**: `internal/database/db.go`
```sql
-- Add to shipments table
ALTER TABLE shipments ADD COLUMN amazon_order_number TEXT;
ALTER TABLE shipments ADD COLUMN delegated_carrier TEXT;
ALTER TABLE shipments ADD COLUMN delegated_tracking_number TEXT;
ALTER TABLE shipments ADD COLUMN is_amazon_logistics BOOLEAN DEFAULT FALSE;

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_shipments_amazon_order ON shipments(amazon_order_number);
CREATE INDEX IF NOT EXISTS idx_shipments_delegated_tracking ON shipments(delegated_carrier, delegated_tracking_number);
```

### TR2: Database Model Updates
**File**: `internal/database/models.go`
```go
// Add to Shipment struct
AmazonOrderNumber       *string `json:"amazon_order_number,omitempty"`
DelegatedCarrier        *string `json:"delegated_carrier,omitempty"`
DelegatedTrackingNumber *string `json:"delegated_tracking_number,omitempty"`
IsAmazonLogistics       bool    `json:"is_amazon_logistics"`
```

### TR3: Carrier Client Implementation
**Files**: 
- `internal/carriers/amazon.go` - New Amazon client
- `internal/carriers/factory.go` - Add Amazon support

**Implementation Pattern**: Follow existing carrier client interface with delegation logic

### TR4: Email Pattern Updates
**File**: `internal/parser/patterns.go`
- Add Amazon order number patterns
- Add Amazon Logistics tracking patterns  
- Add delegated carrier extraction patterns
- Add Amazon domain hints

### TR5: API Validation Updates
**File**: `internal/handlers/shipments.go`
- Add "amazon" to valid carriers list
- Implement Amazon-specific validation in `validateShipment()`
- Ensure either order number or tracking number is provided

### TR6: Configuration Support
**File**: `internal/config/config.go`
- Add `AMAZON_SCRAPING_ENABLED` environment variable
- Add placeholder for future Amazon API credentials

## Implementation Hints

### Database Migration
Add new migration function `migrateAmazonFields()` to `internal/database/db.go` that gets called from the existing `migrate()` function.

### Carrier Factory Pattern
Follow existing pattern in `internal/carriers/factory.go`:
- Add Amazon to `GetAvailableCarriers()`
- Add Amazon case to `createScrapingClient()` 
- Amazon won't have API client initially

### Email Processing Extension
Extend existing patterns in `internal/parser/patterns.go`:
- Add Amazon to `initPatterns()` function
- Follow existing pattern structure for carrier-specific patterns
- Use confidence scoring for Amazon pattern matches

### Validation Logic
In `internal/handlers/shipments.go`, extend `validateShipment()`:
- Add Amazon-specific validation branch
- Validate order number format if provided
- Ensure at least one identifier (order number or tracking number) exists

## Acceptance Criteria

### AC1: Amazon Shipment Creation
- ✅ User can create Amazon shipment with order number via CLI: `./bin/package-tracker add --tracking "113-1234567-1234567" --carrier "amazon"`
- ✅ User can create Amazon shipment with AMZL tracking via CLI: `./bin/package-tracker add --tracking "TBA123456789012" --carrier "amazon"`
- ✅ API accepts Amazon shipments with appropriate validation
- ✅ Database stores Amazon order numbers and delegation information

### AC2: Email Processing
- ✅ Email tracker automatically detects Amazon shipping emails
- ✅ System extracts Amazon order numbers from emails
- ✅ System extracts delegated carrier information when present
- ✅ System creates Amazon shipments automatically from emails
- ✅ System updates shipments with delegation info when carriers are assigned

### AC3: Tracking Functionality
- ✅ Amazon shipments with delegated carriers show tracking from actual carrier
- ✅ Amazon Logistics shipments show available tracking info from emails
- ✅ Refresh functionality works for delegated carriers
- ✅ System handles both order number and tracking number lookups

### AC4: User Interface
- ✅ CLI shows Amazon as available carrier option
- ✅ Amazon shipments display appropriately in list/table formats
- ✅ Users can search/filter Amazon shipments
- ✅ System shows delegation information when available

### AC5: Error Handling
- ✅ Invalid Amazon order numbers are rejected with clear error messages
- ✅ Amazon shipments without identifiers are rejected
- ✅ Email parsing errors are handled gracefully
- ✅ Delegation failures don't break shipment creation

## Assumptions

### A1: Email Access
- Users will have Gmail integration configured for email processing
- Amazon shipping emails will be accessible via existing Gmail search patterns
- Email parsing will be sufficient for most Amazon tracking needs

### A2: Amazon API Limitations
- Amazon will not provide public tracking APIs in the near term
- Direct scraping of Amazon is not feasible due to anti-bot measures
- Email parsing provides sufficient tracking information for user needs

### A3: Carrier Delegation
- Amazon will continue using existing carriers (UPS, USPS, FedEx, DHL) for delegation
- Delegated tracking numbers will be standard format for respective carriers
- Amazon emails will contain sufficient information to identify delegated carriers

### A4: Database Performance
- Additional Amazon fields will not significantly impact database performance
- New indexes will provide adequate query performance
- Single shipment approach will not cause data consistency issues

### A5: User Experience
- Users prefer single shipment records over separate Amazon/carrier records
- Amazon order numbers are acceptable as primary identifiers
- Automatic email detection is preferable to manual configuration

## Dependencies

### D1: Database Migration
- Migration must be backward compatible
- Existing shipments must not be affected
- New fields must have appropriate defaults

### D2: Email Processing
- Depends on existing Gmail integration
- Requires existing email parsing infrastructure
- May need LLM enhancement for complex Amazon emails

### D3: Carrier Integration
- Depends on existing carrier client interfaces
- Requires existing carrier factory patterns
- May need updates to existing carrier implementations

### D4: Configuration System
- Depends on existing Viper configuration system
- Requires environment variable support
- Must maintain backward compatibility

## Future Enhancements

### F1: Amazon API Integration
- When Amazon provides public APIs, add API client support
- Maintain email parsing as fallback mechanism
- Add configuration for Amazon API credentials

### F2: Amazon Logistics Enhancement
- Implement direct AMZL tracking when possible
- Add Amazon delivery estimation features
- Enhance Amazon-specific tracking events

### F3: Order Management Features
- Add support for Amazon order history
- Implement multiple package tracking per order
- Add Amazon-specific delivery preferences

### F4: Advanced Email Processing
- Add support for Amazon business accounts
- Implement international Amazon domain support
- Add enhanced product information extraction