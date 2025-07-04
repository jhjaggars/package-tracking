# Requirements Specification - Email Tracker Enhancement

## Problem Statement

The current email-tracker uses Gmail search with specific carrier and subject filters, which may miss relevant emails containing tracking numbers. Additionally, it only extracts tracking numbers but doesn't capture meaningful product descriptions or merchant information, resulting in generic shipment descriptions like "Package from sender@domain.com" rather than useful information like "Apple iPhone 15 Pro from Amazon".

## Solution Overview

Enhance the email-tracker to:
1. Scan the last 30 days of unread emails using broader search criteria
2. Use LLM-enhanced extraction to capture meaningful product descriptions and merchant information
3. Embed merchant information within the description field
4. Maintain tracking number extraction as the primary priority with confidence-based fallback

## Functional Requirements

### F1: Enhanced Email Scanning
- **F1.1**: Scan last 30 days of unread emails without restrictive sender/subject filtering
- **F1.2**: Process both shipping notifications and order confirmation emails
- **F1.3**: Maintain existing state management to avoid duplicate processing

### F2: LLM-Enhanced Extraction
- **F2.1**: Use few-shot prompting with real email examples to improve accuracy
- **F2.2**: Extract tracking numbers (existing functionality - highest priority)
- **F2.3**: Extract meaningful product descriptions from email content
- **F2.4**: Extract merchant/retailer information from email content
- **F2.5**: Combine merchant and description into a single formatted description field

### F3: Confidence-Based Processing
- **F3.1**: Implement confidence scoring for LLM extractions
- **F3.2**: Fall back to current regex-only approach when LLM confidence is low
- **F3.3**: Prioritize tracking number extraction - prefer tracking number with basic description over no tracking number

### F4: Data Integration
- **F4.1**: Embed merchant information within the description field (not separate field)
- **F4.2**: Format descriptions as "Product description from Merchant" when both available
- **F4.3**: Maintain backward compatibility with existing shipment API structure

## Technical Requirements

### T1: Gmail Search Modification
- **File**: `internal/email/gmail.go`
- **Function**: `BuildSearchQuery()` (lines 246-292)
- **Change**: Broaden search criteria to avoid filtering by sender/subject
- **Implementation**: Use date-based unread email queries without carrier-specific filters

### T2: LLM Prompt Enhancement
- **File**: `internal/parser/llm.go`
- **Function**: `buildPrompt()` (lines 121-145)
- **Change**: Extend prompt to include merchant and description extraction
- **Implementation**: 
  - Add few-shot examples with real email samples
  - Update JSON response schema to include description and merchant fields
  - Modify parsing logic to handle enhanced response format

### T3: Data Structure Updates
- **File**: `internal/email/types.go`
- **Structure**: `TrackingInfo` (lines 54-65)
- **Change**: Add `Merchant` field for internal processing
- **Implementation**: Add `Merchant string` field after existing `Description` field

### T4: Result Processing Enhancement
- **File**: `internal/parser/extractor.go`
- **Function**: `mergeResults()` (lines 544-579)
- **Change**: Update merging logic to combine merchant and description
- **Implementation**: Format final description as "Product from Merchant" when both available

### T5: API Client Updates
- **File**: `internal/api/client.go`
- **Function**: `CreateShipment()` (lines 99-149)
- **Change**: Enhanced description formatting with merchant information
- **Implementation**: Update fallback description logic to include merchant data

## Implementation Hints and Patterns

### LLM Prompt Structure
```json
{
  "tracking_numbers": [
    {
      "number": "1Z999AA1234567890",
      "carrier": "ups",
      "confidence": 0.95,
      "description": "Apple iPhone 15 Pro 256GB Space Black",
      "merchant": "Amazon"
    }
  ]
}
```

### Few-Shot Examples Pattern
Include 2-3 anonymized email samples in the prompt showing:
- Shipping notification with tracking number
- Order confirmation with product details
- Expected JSON extraction format

### Confidence Fallback Logic
- LLM confidence < 0.7: Use current regex-only approach
- LLM confidence >= 0.7: Use enhanced extraction
- Always prioritize tracking number extraction over description quality

### Search Query Pattern
```go
// Current: carrier-specific filtering
query := `from:(ups.com OR fedex.com) subject:(tracking OR shipment)`

// Enhanced: broader date-based scanning
query := `after:2024/12/05 is:unread`
```

## Acceptance Criteria

### AC1: Email Scanning
- [ ] System processes unread emails from last 30 days
- [ ] No emails are missed due to sender/subject filtering
- [ ] State management prevents duplicate processing
- [ ] Processing maintains existing performance characteristics

### AC2: Enhanced Extraction
- [ ] LLM extracts meaningful product descriptions when available
- [ ] Merchant information is captured from emails
- [ ] Tracking numbers are extracted with same or better accuracy
- [ ] Few-shot prompting improves extraction quality

### AC3: Data Quality
- [ ] Descriptions include merchant information when available
- [ ] Format: "Product description from Merchant" or fallback to current behavior
- [ ] Confidence-based fallback prevents degradation in tracking number extraction
- [ ] No reduction in overall shipment creation success rate

### AC4: System Reliability
- [ ] Existing error handling and retry logic remains functional
- [ ] LLM failures don't prevent tracking number extraction
- [ ] Processing statistics and logging include enhanced extraction metrics
- [ ] System gracefully handles LLM service unavailability

## Assumptions

### A1: LLM Service Availability
- Existing LLM configuration (Ollama/OpenAI/Anthropic) is properly configured
- LLM service has sufficient capacity for enhanced prompts
- Network connectivity allows for increased LLM API calls

### A2: Email Content Quality
- Most relevant emails contain sufficient information for description extraction
- Email HTML/text content is accessible and parseable
- Product information is typically present in email subject or body

### A3: Performance Impact
- Increased LLM processing time is acceptable for enhanced data quality
- Gmail API rate limits can accommodate broader search queries
- Storage requirements for enhanced descriptions are manageable

### A4: Configuration Compatibility
- Existing email-tracker configuration remains valid
- No breaking changes to environment variables or CLI flags
- Current .env file configurations continue to work

## Dependencies

- Existing LLM integration infrastructure
- Gmail API OAuth2 credentials and permissions
- SQLite database for state management
- Main package-tracking API service availability

## Testing Strategy

### Unit Tests
- Enhanced LLM prompt parsing with new JSON schema
- Confidence-based fallback logic
- Description formatting with merchant information
- Broadened Gmail search query generation

### Integration Tests
- End-to-end email processing with real email samples
- LLM service integration with enhanced prompts
- State management with new data structures
- API client shipment creation with enhanced descriptions

### Performance Tests
- Processing time impact of enhanced LLM prompts
- Gmail API rate limit compliance with broader searches
- Memory usage with enhanced data structures