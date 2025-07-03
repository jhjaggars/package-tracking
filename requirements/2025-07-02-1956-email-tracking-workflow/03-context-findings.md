# Context Findings

## API Integration Details

### Shipment Creation Endpoint
- **Endpoint**: `POST /api/shipments`
- **Handler**: `internal/handlers/shipments.go:CreateShipment`
- **Required payload**:
  ```json
  {
    "tracking_number": "string",
    "carrier": "ups|usps|fedex|dhl",
    "description": "string"
  }
  ```
- **Validation**:
  - Duplicate tracking numbers return 409 Conflict
  - Carrier must be one of the supported values
  - All three fields are required
- **No authentication** currently implemented on API

## Existing Patterns to Follow

### Background Worker Pattern (from TrackingUpdater)
- Location: `internal/workers/tracking_updater.go`
- Key features:
  - Context-based lifecycle management
  - Start/Stop/Pause/Resume methods
  - Graceful shutdown support
  - Structured logging with slog
  - Configuration injection
  - Rate limiting and error handling

### Configuration Pattern
- Location: `internal/config/config.go`
- Features:
  - Environment variable loading
  - `.env` file support
  - Validation of config values
  - Pattern for external service credentials

### Daemon/Service Pattern
- Signal handling: `internal/server/signals.go`
- Graceful shutdown with timeout
- Background workers started in main.go with deferred Stop()

## Carrier Integration

### Tracking Number Validation
- Each carrier implements `ValidateTrackingNumber` method
- Uses regex patterns for format validation
- Examples in `internal/carriers/usps.go`, `ups.go`, etc.

### Supported Carriers
- USPS (with comprehensive tracking patterns)
- UPS
- FedEx (API and scraping support)
- DHL

## Architecture Considerations

### No Existing Email/MCP Code
- This would be a completely new feature
- No existing email parsing or MCP integration to build upon

### Security Considerations
- API currently has no authentication
- CORS allows all origins
- May need to add authentication if email processor is external

### Database Integration
- Shipments have unique constraint on tracking_number
- Auto-refresh fields available but not needed for email workflow
- TrackingEvent model for storing event history

## Recommended Architecture

1. **New Command Structure**:
   - Create `cmd/email-tracker/main.go`
   - Follow similar structure to `cmd/server/main.go`

2. **Worker Pattern**:
   - Create `internal/workers/email_processor.go`
   - Implement similar to `tracking_updater.go`

3. **Email Integration**:
   - New package `internal/email/` for email provider integration
   - Support for IMAP/POP3 protocols
   - MCP server integration option

4. **Tracking Parser**:
   - New package `internal/parser/` for extracting tracking numbers
   - Leverage existing carrier validation methods
   - Support multiple email formats

5. **API Client**:
   - Reuse or extend `internal/cli/client.go` patterns
   - HTTP client for calling shipment creation endpoint

## Files Analyzed
- `internal/handlers/shipments.go`
- `internal/workers/tracking_updater.go`
- `internal/config/config.go`
- `internal/server/signals.go`
- `internal/carriers/*.go`
- `internal/database/models.go`
- `cmd/server/main.go`
- `cmd/cli/main.go`

---

*Context gathered on: 2025-07-02 19:58*