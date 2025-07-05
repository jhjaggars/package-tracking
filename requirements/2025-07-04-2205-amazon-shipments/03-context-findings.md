# Context Findings

## Database Schema Analysis

### Current Schema Structure
The system uses SQLite with a well-structured schema in `internal/database/db.go`. The `shipments` table has comprehensive tracking fields including auto-refresh capabilities and failure tracking.

### Required Schema Changes
**New fields needed in shipments table:**
- `amazon_order_number TEXT` - Store Amazon order numbers (###-#######-#######)
- `delegated_carrier TEXT` - Store the actual carrier when Amazon delegates
- `delegated_tracking_number TEXT` - Store the carrier's tracking number
- `is_amazon_logistics BOOLEAN DEFAULT FALSE` - Flag for Amazon's own delivery network

**Database migration required:** Add `migrateAmazonFields()` function to `internal/database/db.go`

## Carrier Implementation Pattern Analysis

### Current Carrier Architecture
The system uses a clean interface-based design with factory pattern:
- `internal/carriers/types.go` - Defines `Client` interface
- `internal/carriers/factory.go` - Creates appropriate clients
- Multiple client types: API, Headless, Scraping with automatic fallback

### Amazon Integration Requirements
**Key files to modify:**
1. `internal/carriers/factory.go` - Add Amazon to supported carriers
2. New `internal/carriers/amazon.go` - Amazon client implementation
3. New `internal/carriers/amazon_scraping.go` - Amazon scraping fallback

**Unique Amazon challenges:**
- No public API available
- Requires session management for order tracking
- Must handle both AMZL and delegated carrier tracking

## Email Parsing System Analysis

### Current Email Processing
The system has sophisticated email parsing in `internal/parser/`:
- Pattern-based extraction with confidence scoring
- LLM enhancement support (OpenAI, Anthropic, Ollama)
- Carrier-specific pattern sets
- Context-aware matching (labels, tables, formatted text)

### Amazon Email Patterns Needed
**New patterns for `internal/parser/patterns.go`:**
- Amazon order number pattern: `\d{3}-\d{7}-\d{7}`
- Amazon Logistics tracking: `TBA\d{12}`
- Delegated carrier extraction from Amazon emails
- Amazon domain identification for email hints

## Configuration System Analysis

### Current Configuration
The system uses both:
- Modern Viper-based configuration (`internal/config/viper_server.go`)
- Legacy environment variables for backward compatibility
- Support for YAML, TOML, JSON, .env files

### Amazon Configuration Needs
**Environment variables to add:**
- `AMAZON_SCRAPING_ENABLED` - Enable/disable Amazon scraping
- `AMAZON_API_KEY` - For future API access
- `AMAZON_REFRESH_TOKEN` - For future authentication

## API Endpoints Analysis

### Current REST API Structure
Well-designed REST API in `internal/handlers/shipments.go`:
- Standard CRUD operations
- Validation with carrier-specific rules
- Error handling with proper HTTP status codes
- Refresh endpoint with caching

### Amazon API Modifications Required
**Existing endpoints to modify:**
- Update `validateShipment()` to handle Amazon order numbers
- Add Amazon to valid carriers list
- Extend validation for Amazon-specific fields

**New endpoints potentially needed:**
- `GET /api/shipments/amazon/{orderNumber}` - Lookup by Amazon order
- `POST /api/shipments/amazon-delegation` - Handle carrier delegation

## Similar Features Analysis

### Carrier Delegation Pattern
The system doesn't currently have delegation patterns, but the architecture supports it well:
- Could extend the `Shipment` model with delegation fields
- Factory pattern allows for hybrid Amazon/carrier clients
- Email parsing already supports multiple carrier identification

### Email-First Strategy
The existing email processing daemon (`cmd/email-tracker/main.go`) provides excellent foundation:
- Gmail OAuth2 integration
- Configurable search queries
- Duplicate detection
- Automatic shipment creation

## Technical Constraints

### Amazon-Specific Challenges
1. **No Public API**: Must rely on scraping or unofficial methods
2. **Authentication Required**: Order tracking requires login
3. **Anti-bot Measures**: Sophisticated detection systems
4. **Rate Limiting**: Must be careful not to trigger blocks
5. **Legal Considerations**: Scraping terms of service

### Recommended Implementation Strategy
1. **Email-First Approach**: Focus on parsing Amazon emails rather than direct scraping
2. **Delegation Pattern**: Extract carrier tracking numbers and delegate to existing implementations
3. **Amazon Logistics Support**: Handle AMZL tracking separately from third-party carriers
4. **Gradual Rollout**: Start with email parsing, add direct tracking later

## Integration Points Identified

### Key Files Requiring Changes
1. `internal/database/models.go` - Add Amazon fields to Shipment struct
2. `internal/database/db.go` - Add migration for Amazon fields
3. `internal/parser/patterns.go` - Add Amazon email patterns
4. `internal/carriers/factory.go` - Add Amazon carrier support
5. `internal/handlers/shipments.go` - Update validation for Amazon

### New Files Required
1. `internal/carriers/amazon.go` - Amazon client implementation
2. `internal/carriers/amazon_scraping.go` - Amazon scraping client
3. `internal/amazon/delegation.go` - Handle delegation logic
4. Test files for Amazon-specific functionality

## Performance Considerations

### Email Processing Impact
- Amazon emails may be more complex to parse
- LLM enhancement could be valuable for Amazon order extraction
- May need specific rate limiting for Amazon operations

### Database Performance
- New indexes needed for Amazon order number lookups
- Consider impact of delegation relationships on query performance

## Security Considerations

### Amazon Scraping Risks
- Must avoid triggering Amazon's anti-bot measures
- Should implement respectful rate limiting
- Consider user agent rotation and proxy support

### Data Privacy
- Amazon order numbers may contain sensitive information
- Consider encryption for stored Amazon credentials (future)
- Ensure proper handling of Amazon email content