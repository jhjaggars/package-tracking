# Context Findings

## Database Schema Analysis

### Current Schema
- **Main Database** (`database.db`): shipments, tracking_events, carriers, refresh_cache
- **Email State Database** (`email-state.db`): processed_emails table with basic metadata only

### Required Schema Changes
**Main Database:**
- **NEW TABLE: `email_shipments`** - Links emails to shipments (many-to-many)
- **NEW TABLE: `email_threads`** - Stores email thread/conversation data
- **MODIFY: `shipments` table** - Add email-related fields for source tracking

**Email State Database:**
- **MODIFY: `processed_emails` table** - Add fields for full email body storage, internal_timestamp, scan_method
- **NEW TABLE: `email_bodies`** - Store full email content (plain text, HTML)
- **NEW INDEXES** - For time-based scanning queries

## Gmail API Implementation Changes

### Current Implementation
- Search-based system using `Search()` method in `internal/email/gmail.go`
- Query filtering with shipping email patterns
- Limited to known shipping email senders

### Required Changes
- **NEW METHOD: `GetMessagesSince(timestamp)`** - Scan all emails after specific time
- **NEW METHOD: `GetThreadMessages(threadId)`** - Retrieve full conversation threads
- **MODIFY: `GetMessage()`** - Enhanced to capture full body content
- **NEW: Time-based pagination** - Handle large volumes of recent emails

## Web Interface Integration Points

### Current Structure
- React frontend with TypeScript
- API types in `/web/src/types/api.ts`
- Components in `/web/src/components/` and `/web/src/pages/`

### Required Additions
- **NEW API Types**: EmailThread, EmailMessage, EmailShipmentLink interfaces
- **NEW Component**: EmailChainViewer for displaying email conversations
- **MODIFY: ShipmentDetail.tsx** - Add email chain section
- **NEW Endpoints**: `/api/shipments/{id}/emails`, `/api/emails/{id}/thread`

## Configuration System Changes

### Current Config
- Search-based configuration with query patterns
- Processing intervals and limits in `internal/config/email_config.go`

### Required New Options
- `SCAN_MODE`: "search" vs "time_based" vs "hybrid"
- `TIME_BASED_SCAN_INTERVAL`: How often to scan for new emails
- `EMAIL_BODY_STORAGE`: Enable/disable full body storage
- `EMAIL_RETENTION_DAYS`: How long to keep email bodies
- `EMAIL_CHAIN_DISPLAY`: Enable/disable email chain UI features

## Specific Files Requiring Modification

### Database Layer
- `internal/database/models.go`: Add EmailThread, EmailBody, EmailShipmentLink structs
- `internal/database/db.go`: Add new migration functions
- **NEW FILE: `internal/database/emails.go`**: Email-specific database operations

### Email Processing Layer
- `internal/email/gmail.go`: Add time-based scanning methods
- `internal/email/types.go`: Add new email scanning types
- `internal/email/state.go`: Extend for body storage and shipment linking
- `internal/workers/email_processor.go`: Add time-based processing workflow

### API Layer
- `internal/handlers/shipments.go`: Add email-related endpoints
- **NEW FILE: `internal/handlers/emails.go`**: Email-specific HTTP handlers
- `internal/server/router.go`: Register new email endpoints

### Frontend Layer
- `web/src/types/api.ts`: Add email-related TypeScript interfaces
- `web/src/services/api.ts`: Add email API calls
- **NEW FILE: `web/src/components/EmailChainViewer.tsx`**: Email conversation display
- `web/src/pages/ShipmentDetail.tsx`: Integrate email chain viewer

## New API Endpoints Required

```
GET /api/shipments/{id}/emails     - Get emails linked to shipment
GET /api/emails/{id}/thread        - Get full email conversation thread
GET /api/emails/{id}/body          - Get full email body content
POST /api/emails/{id}/link/{shipmentId} - Link email to shipment
DELETE /api/emails/{id}/link/{shipmentId} - Unlink email from shipment
GET /api/admin/email-scanner/status - Get scanner status and metrics
POST /api/admin/email-scanner/mode  - Switch between scan modes
```

## Technical Considerations

### Storage Requirements
- Email bodies can be large (especially HTML with images)
- Need efficient compression/storage strategy
- Consider storage limits and cleanup policies

### Performance Impact
- Time-based scanning may process more emails
- Need intelligent filtering to avoid processing spam/irrelevant emails
- Caching strategy for email thread queries

### Privacy and Security
- Email content storage requires careful handling
- Consider encryption for sensitive email data
- Implement proper access controls for email viewing

### Rate Limiting
- Gmail API has strict rate limits
- Time-based scanning must respect these limits
- Need exponential backoff for API failures

## Implementation Strategy

**Phase 1: Database Schema**
1. Add new tables without disrupting existing functionality
2. Create migration functions following existing patterns
3. Add indexes for time-based queries

**Phase 2: Backend Changes**
1. Implement time-based Gmail scanning alongside existing search
2. Add hybrid mode that uses both approaches
3. Extend email state storage for body content

**Phase 3: API Extensions**
1. Add new email-related endpoints
2. Extend shipment endpoints with email data
3. Maintain backward compatibility

**Phase 4: Frontend Integration**
1. Add email chain components
2. Integrate with shipment detail pages
3. Add admin controls for scan mode switching