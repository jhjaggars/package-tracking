# Email Tracking Workflow Requirements Specification

## Problem Statement
Users receive numerous shipment notification emails from various carriers and vendors. Currently, they must manually extract tracking numbers and add them to the package tracking system. This manual process is time-consuming and error-prone. An automated email processing workflow would eliminate this friction by continuously monitoring email accounts and automatically creating shipments when tracking numbers are detected.

## Solution Overview
Build a standalone daemon process that monitors Gmail accounts using Gmail's advanced search capabilities, extracts tracking numbers from shipment notification emails, and automatically creates shipments in the package tracking system via the REST API.

## Functional Requirements

### Gmail Integration
1. **Gmail-Specific Features**
   - Use Gmail API for enhanced search capabilities
   - Fallback to IMAP with Gmail-specific settings
   - OAuth2 authentication for secure access
   - Configuration via environment variables

2. **Gmail Search Syntax**
   - Use Gmail search operators to filter shipping emails:
     - `from:(noreply@ups.com OR shipment@amazon.com OR tracking@fedex.com)`
     - `subject:("tracking" OR "shipment" OR "package" OR "delivery")`
     - `has:attachment` (for shipping labels)
     - `after:2024/1/1` (configurable date range)
     - `is:unread` (optional, for new emails only)
   - Combine operators for precise filtering:
     - `{from:(noreply@ups.com) subject:tracking} OR {from:usps.com subject:"tracking number"}`
   - Support custom search queries via configuration

3. **Email Processing**
   - Monitor Gmail using search queries as a daemon process
   - Process emails from multiple carriers and vendors
   - Extract tracking numbers using carrier-specific patterns
   - DO NOT mark emails as read/processed
   - Store processing state to avoid duplicates
   - Use Gmail message IDs for deduplication

3. **Tracking Number Extraction**
   - Parse common carrier email formats (UPS, USPS, FedEx, DHL)
   - Validate tracking numbers using existing carrier validation logic
   - Extract additional metadata when available:
     - Carrier identification
     - Description/item name
     - Expected delivery date

4. **Shipment Creation**
   - Automatically create shipments via POST /api/shipments
   - Use extracted tracking number, carrier, and description
   - Handle duplicate tracking numbers gracefully (409 responses)
   - Implement retry logic for API unavailability

5. **Operational Features**
   - Dry-run mode for testing without creating shipments
   - Structured logging for all operations
   - Graceful shutdown on signals
   - Configuration validation on startup

## Technical Requirements

### Architecture
1. **New Command**: `cmd/email-tracker/main.go`
   - Follow pattern from `cmd/server/main.go`
   - Signal handling for graceful shutdown
   - Configuration loading and validation

2. **Email Processor Worker**: `internal/workers/email_processor.go`
   - Follow pattern from `internal/workers/tracking_updater.go`
   - Context-based lifecycle (Start/Stop/Pause/Resume)
   - Configurable check intervals
   - Structured logging with slog

3. **Email Package**: `internal/email/`
   - `client.go` - Email provider abstraction
   - `gmail.go` - Gmail API implementation
   - `gmail_search.go` - Gmail search query builder
   - `imap_gmail.go` - Gmail IMAP fallback
   - `types.go` - Common types and interfaces

4. **Parser Package**: `internal/parser/`
   - `extractor.go` - Main extraction logic
   - `patterns.go` - Carrier-specific patterns
   - `validator.go` - Tracking number validation
   - Reuse validation from `internal/carriers/`

5. **State Management**: `internal/email/state.go`
   - SQLite database for processed email tracking
   - Store email ID/hash to prevent reprocessing
   - Cleanup old entries periodically

### Configuration
Environment variables to add:
```
# Gmail Authentication
GMAIL_CLIENT_ID=your-client-id.apps.googleusercontent.com
GMAIL_CLIENT_SECRET=your-client-secret
GMAIL_REFRESH_TOKEN=stored-refresh-token
GMAIL_ACCESS_TOKEN=stored-access-token
GMAIL_TOKEN_FILE=./gmail-token.json

# Gmail IMAP Fallback
GMAIL_USERNAME=user@gmail.com
GMAIL_APP_PASSWORD=app-specific-password

# Search Configuration
GMAIL_SEARCH_QUERY=from:(noreply@ups.com OR tracking@fedex.com) subject:(tracking OR shipment)
GMAIL_SEARCH_AFTER_DAYS=30
GMAIL_SEARCH_UNREAD_ONLY=false
GMAIL_MAX_RESULTS=100

# Processing Configuration
EMAIL_CHECK_INTERVAL=5m
EMAIL_STATE_DB_PATH=./email-state.db
EMAIL_DRY_RUN=false

# API Configuration
EMAIL_API_URL=http://localhost:8080
EMAIL_API_RETRY_COUNT=3
EMAIL_API_RETRY_DELAY=30s
```

### API Integration
- Use existing REST API endpoint: `POST /api/shipments`
- No authentication required (current API design)
- Handle responses:
  - 201 Created - Success
  - 409 Conflict - Duplicate (not an error)
  - 400 Bad Request - Invalid data
  - 500+ Server Error - Retry

### Error Handling
1. **Email Connection Errors**
   - Log and retry with exponential backoff
   - Continue running, don't crash

2. **Parsing Errors**
   - Log unparseable emails for analysis
   - Skip and continue processing

3. **API Errors**
   - Retry transient errors (500, 502, 503)
   - Queue failed requests for retry
   - Log permanent failures

### Implementation Hints

1. **Follow Existing Patterns**
   - Worker lifecycle from `tracking_updater.go`
   - Configuration from `config.go`
   - HTTP client patterns from `internal/cli/client.go`

2. **Reuse Components**
   - Carrier validation from `internal/carriers/`
   - Logging setup from main server
   - Signal handling from `internal/server/signals.go`

3. **Database Schema** for state tracking:
   ```sql
   CREATE TABLE processed_emails (
     id INTEGER PRIMARY KEY,
     gmail_message_id TEXT UNIQUE NOT NULL,
     gmail_thread_id TEXT,
     processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
     tracking_numbers TEXT,
     status TEXT,
     sender TEXT,
     subject TEXT
   );
   
   CREATE INDEX idx_gmail_message_id ON processed_emails(gmail_message_id);
   CREATE INDEX idx_processed_at ON processed_emails(processed_at);
   ```

## Acceptance Criteria

1. **Basic Functionality**
   - [ ] Connects to configured email provider
   - [ ] Extracts tracking numbers from carrier emails
   - [ ] Creates shipments via API
   - [ ] Avoids duplicate processing

2. **Reliability**
   - [ ] Handles API downtime with retries
   - [ ] Continues operating on parsing errors
   - [ ] Graceful shutdown on signals
   - [ ] Validates configuration on startup

3. **Observability**
   - [ ] Structured logging of all operations
   - [ ] Dry-run mode for testing
   - [ ] Clear error messages

4. **Performance**
   - [ ] Processes emails within 1 minute of receipt
   - [ ] Minimal memory footprint
   - [ ] Efficient state storage

## Assumptions
- The main package tracking server will handle notifications (not the email processor)
- Standard carrier email formats will be supported initially
- The API will remain unauthenticated (current design)
- Email processor will run on same network as API server
- SQLite is acceptable for state management
- Gmail API quotas are sufficient for polling frequency
- OAuth2 tokens will be pre-configured or obtained via initial setup flow

---

## Gmail-Specific Implementation Notes

### Search Query Examples
1. **Basic carrier search**:
   ```
   from:(noreply@ups.com OR shipment@amazon.com OR tracking@fedex.com OR inform@dhl.com OR usps.com)
   ```

2. **Enhanced with subject filtering**:
   ```
   (from:ups.com subject:"tracking number") OR 
   (from:fedex.com subject:"shipment") OR
   (from:amazon.com subject:"shipped")
   ```

3. **Date-based filtering**:
   ```
   after:2024/12/1 before:2024/12/31
   ```

4. **Attachment filtering for labels**:
   ```
   has:attachment filename:pdf subject:"shipping label"
   ```

### Gmail API Benefits
- Batch message retrieval
- Partial message fetch (headers only for initial scan)
- Thread grouping
- Label support for processed emails
- Watch/push notifications (future enhancement)

---

*Requirements specification updated on: 2025-07-02 20:10*