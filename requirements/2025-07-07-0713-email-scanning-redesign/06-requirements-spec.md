# Requirements Specification: Email Scanning Redesign

## Problem Statement

The current email scanning system uses Gmail search queries to find shipping-related emails, which limits discovery to emails matching specific patterns. This approach misses emails from new carriers, non-standard formats, or emails where tracking information appears in different contexts. The system also lacks the ability to review email chains and re-summarize tracking information without re-requesting data from Gmail.

## Solution Overview

Redesign the email scanning system to use time-based scanning instead of search patterns, scanning all emails within a configurable time window (default 30 days). Store full email bodies with shipment linking to enable email chain review and re-summarization capabilities accessible through the web interface.

## Functional Requirements

### FR1: Time-Based Email Scanning
- **FR1.1**: Replace search-based scanning with time-based scanning of all emails in the last N days (configurable, default 30 days)
- **FR1.2**: Scan retroactively on first implementation to populate historical data
- **FR1.3**: Use Gmail API to retrieve emails by date range rather than search queries
- **FR1.4**: Process all emails within the time window regardless of sender or content patterns

### FR2: Email Body Storage
- **FR2.1**: Store full email bodies (text content or HTML if only HTML available) in SQLite database
- **FR2.2**: Do not store email attachments
- **FR2.3**: Use compression for email body storage to minimize database size
- **FR2.4**: Email bodies must be displayable through both web UI and CLI

### FR3: Email-Shipment Linking
- **FR3.1**: Automatically link emails to shipments when tracking numbers are found
- **FR3.2**: Support manual linking of emails to shipments through web interface
- **FR3.3**: Support unlinking emails from shipments through web interface
- **FR3.4**: Maintain many-to-many relationship between emails and shipments

### FR4: Email Thread Conversation Tracking
- **FR4.1**: Use Gmail thread IDs to group related emails into conversations
- **FR4.2**: Store email thread data in dedicated database table
- **FR4.3**: Display emails as threaded conversations in web interface
- **FR4.4**: Support viewing complete email chains for shipments

### FR5: Web Interface Integration
- **FR5.1**: Add email chain viewer component to shipment detail pages
- **FR5.2**: Display email conversations with proper threading
- **FR5.3**: Show email body content with proper formatting
- **FR5.4**: Provide manual email linking controls in web interface

### FR6: Automatic Shipment Creation
- **FR6.1**: Continue automatic shipment creation when tracking numbers are found
- **FR6.2**: Maintain existing tracking number extraction and validation logic
- **FR6.3**: Link created shipments to source emails automatically

### FR7: Configuration Management
- **FR7.1**: Store new configuration options in existing Viper configuration system
- **FR7.2**: Support time period configuration (days to scan)
- **FR7.3**: Support email body storage enable/disable
- **FR7.4**: Support email retention period configuration

## Technical Requirements

### TR1: Database Schema Changes
- **TR1.1**: Modify `processed_emails` table in `email-state.db` to add email body storage fields
- **TR1.2**: Add `email_threads` table to store conversation data
- **TR1.3**: Add `email_shipments` table to link emails to shipments (many-to-many)
- **TR1.4**: Add database indexes for efficient time-based queries
- **TR1.5**: Implement database migration functions following existing patterns in `internal/database/db.go`

### TR2: Gmail API Enhancement
- **TR2.1**: Add `GetMessagesSince(timestamp)` method to `internal/email/gmail.go`
- **TR2.2**: Add `GetThreadMessages(threadId)` method for conversation retrieval
- **TR2.3**: Enhance `GetMessage()` method to capture full body content
- **TR2.4**: Implement time-based pagination for large email volumes
- **TR2.5**: Remove existing search-based methods and configuration

### TR3: Email Processing Updates
- **TR3.1**: Modify `internal/workers/email_processor.go` for time-based workflow
- **TR3.2**: Update `internal/email/state.go` for body storage and shipment linking
- **TR3.3**: Extend `internal/email/types.go` with new email scanning types
- **TR3.4**: Create `internal/database/emails.go` for email-specific database operations

### TR4: API Endpoints
- **TR4.1**: Add `GET /api/shipments/{id}/emails` endpoint
- **TR4.2**: Add `GET /api/emails/{id}/thread` endpoint
- **TR4.3**: Add `GET /api/emails/{id}/body` endpoint
- **TR4.4**: Add `POST /api/emails/{id}/link/{shipmentId}` endpoint
- **TR4.5**: Add `DELETE /api/emails/{id}/link/{shipmentId}` endpoint
- **TR4.6**: Create `internal/handlers/emails.go` for email-specific HTTP handlers
- **TR4.7**: Register new endpoints in `internal/server/router.go`

### TR5: Frontend Components
- **TR5.1**: Add email-related TypeScript interfaces to `web/src/types/api.ts`
- **TR5.2**: Add email API calls to `web/src/services/api.ts`
- **TR5.3**: Create `web/src/components/EmailChainViewer.tsx` component
- **TR5.4**: Integrate email chain viewer into `web/src/pages/ShipmentDetail.tsx`

### TR6: Configuration System
- **TR6.1**: Add new configuration options to `internal/config/viper_email.go`
- **TR6.2**: Support `EMAIL_SCAN_DAYS` configuration (default: 30)
- **TR6.3**: Support `EMAIL_BODY_STORAGE_ENABLED` configuration (default: true)
- **TR6.4**: Support `EMAIL_RETENTION_DAYS` configuration (default: 90)

## Implementation Hints

### Database Schema Pattern
Follow existing migration patterns in `internal/database/db.go`:
```go
func (db *DB) migrateEmailTables() error {
    // Add email body storage fields
    // Create email_threads table
    // Create email_shipments linking table
    // Add indexes for time-based queries
}
```

### Gmail API Time-Based Scanning
Replace search queries with time-based message listing:
```go
func (g *GmailClient) GetMessagesSince(since time.Time) ([]*gmail.Message, error) {
    // Use Gmail API to list messages after timestamp
    // Handle pagination for large result sets
    // Return all messages within time window
}
```

### Email Body Storage
Use SQLite with compression:
```sql
ALTER TABLE processed_emails ADD COLUMN body_text TEXT;
ALTER TABLE processed_emails ADD COLUMN body_html TEXT;
ALTER TABLE processed_emails ADD COLUMN body_compressed BLOB;
```

### Frontend Integration
Follow existing component patterns in `web/src/components/`:
```typescript
interface EmailChainViewerProps {
  shipmentId: number;
  emails: EmailMessage[];
  onLinkEmail: (emailId: string) => void;
}
```

## Acceptance Criteria

### AC1: Time-Based Scanning
- [ ] System scans all emails from the last N days (configurable)
- [ ] Search-based scanning is completely removed
- [ ] Retroactive scanning populates historical data on first run
- [ ] Email scanning respects Gmail API rate limits

### AC2: Email Body Storage
- [ ] Full email bodies are stored in SQLite database
- [ ] Text content is preferred over HTML when both available
- [ ] Attachments are not stored
- [ ] Email bodies are compressed to minimize storage size

### AC3: Email Chain Review
- [ ] Email chains are accessible through web interface
- [ ] Emails are grouped by Gmail thread ID
- [ ] Complete email conversations are displayable
- [ ] Email body content is properly formatted in UI

### AC4: Email-Shipment Linking
- [ ] Emails are automatically linked to shipments when tracking numbers match
- [ ] Manual linking is available through web interface
- [ ] Manual unlinking is available through web interface
- [ ] Many-to-many relationships are supported

### AC5: Configuration
- [ ] Time period is configurable through Viper configuration
- [ ] Email body storage can be disabled
- [ ] Email retention period is configurable
- [ ] Configuration follows existing patterns

### AC6: Performance
- [ ] Time-based scanning performs within acceptable limits
- [ ] Email body queries are efficient with proper indexing
- [ ] Email chain display loads without performance issues
- [ ] Database storage growth is manageable

## Assumptions

1. **Gmail API Access**: Existing Gmail API credentials and permissions will continue to work for time-based scanning
2. **Storage Capacity**: SQLite database can handle the increased storage requirements for email bodies
3. **Performance**: Time-based scanning will not significantly impact system performance
4. **Backwards Compatibility**: Existing shipment creation and tracking functionality will remain unchanged
5. **UI Framework**: Current React/TypeScript frontend architecture will accommodate new email components
6. **Configuration**: Existing Viper configuration system can be extended without breaking changes

## Related Features

Based on codebase analysis, this feature relates to:
- Email Processing (`internal/email/`, `internal/workers/email_processor.go`)
- Configuration Management (`internal/config/viper_email.go`)
- Database Layer (`internal/database/`)
- Web Interface (`web/src/`)
- API Endpoints (`internal/handlers/`)

## Success Metrics

- **Completeness**: All emails within configured time window are processed
- **Accuracy**: Tracking number extraction accuracy is maintained or improved
- **Performance**: Email scanning completes within acceptable time limits
- **Usability**: Email chain review is accessible and useful through web interface
- **Reliability**: System handles Gmail API rate limits and errors gracefully