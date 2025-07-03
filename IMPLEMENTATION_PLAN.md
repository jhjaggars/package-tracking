# Email Tracking Workflow Implementation Plan

**GitHub Issue**: #29  
**Branch**: feature/email-tracking-workflow-issue-29  
**Implementation Date**: 2025-07-02

## Overview
Implement a standalone daemon process that monitors Gmail accounts using Gmail's search capabilities to automatically extract tracking numbers and create shipments in the package tracking system.

## Phase 1: Core Architecture (High Priority)

### 1.1 Design Interfaces and Types
**File**: `internal/email/types.go`
```go
type EmailClient interface {
    Search(query string) ([]EmailMessage, error)
    GetMessage(id string) (*EmailMessage, error)
    Close() error
}

type EmailMessage struct {
    ID       string
    ThreadID string
    From     string
    Subject  string
    Body     string
    Date     time.Time
    Headers  map[string]string
}

type TrackingInfo struct {
    Number      string
    Carrier     string
    Description string
    Source      EmailMessage
}
```

### 1.2 Gmail Client Implementation
**File**: `internal/email/gmail.go`
- OAuth2 authentication flow
- Gmail API client initialization
- Search query execution with pagination
- Message retrieval with content parsing
- Rate limiting and error handling

**File**: `internal/email/gmail_search.go`
- Query builder for Gmail search syntax
- Predefined queries for common carriers
- Date range filtering
- Result filtering and deduplication

### 1.3 IMAP Fallback
**File**: `internal/email/imap_gmail.go`
- Gmail IMAP configuration
- App password authentication
- IMAP search equivalent to Gmail API queries
- Message parsing from IMAP format

## Phase 2: Parsing and Extraction (High Priority)

### 2.1 Tracking Number Parser
**File**: `internal/parser/extractor.go`
```go
type TrackingExtractor struct {
    carriers map[string]*CarrierParser
}

func (e *TrackingExtractor) Extract(msg EmailMessage) ([]TrackingInfo, error)
```

**File**: `internal/parser/patterns.go`
- Regex patterns for each carrier
- Email format recognition (HTML vs plain text)
- Metadata extraction (description, expected delivery)
- Integration with existing `internal/carriers/` validation

**File**: `internal/parser/validator.go`
- Reuse existing carrier validation logic
- Cross-validate extracted numbers
- Filter false positives

## Phase 3: Worker Process (High Priority)

### 3.1 Email Processor Worker
**File**: `internal/workers/email_processor.go`
```go
type EmailProcessor struct {
    client    email.EmailClient
    extractor *parser.TrackingExtractor
    apiClient *api.Client
    state     *state.Manager
    config    *config.Config
    logger    *slog.Logger
}

func (p *EmailProcessor) Start(ctx context.Context) error
func (p *EmailProcessor) Stop() error
func (p *EmailProcessor) processEmails() error
```

Following pattern from `internal/workers/tracking_updater.go`:
- Context-based lifecycle management
- Configurable check intervals
- Graceful shutdown support
- Error handling and retry logic
- Structured logging

## Phase 4: State Management (Medium Priority)

### 4.1 Email State Tracking
**File**: `internal/email/state.go`
```go
type StateManager struct {
    db *sql.DB
}

func (s *StateManager) IsProcessed(messageID string) (bool, error)
func (s *StateManager) MarkProcessed(messageID string, trackingNumbers []string) error
func (s *StateManager) Cleanup(olderThan time.Time) error
```

**Database Schema**:
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
```

## Phase 5: Configuration and Setup (Medium Priority)

### 5.1 Configuration Extension
**File**: `internal/config/email_config.go`
```go
type EmailConfig struct {
    // Gmail API
    ClientID     string
    ClientSecret string
    RefreshToken string
    TokenFile    string
    
    // Gmail IMAP fallback
    Username    string
    AppPassword string
    
    // Search configuration
    SearchQuery     string
    SearchAfterDays int
    MaxResults      int
    
    // Processing
    CheckInterval time.Duration
    StateDBPath   string
    DryRun        bool
    
    // API client
    APIURL       string
    RetryCount   int
    RetryDelay   time.Duration
}
```

**Environment Variables**:
- `GMAIL_CLIENT_ID`, `GMAIL_CLIENT_SECRET`
- `GMAIL_REFRESH_TOKEN`, `GMAIL_TOKEN_FILE`
- `GMAIL_SEARCH_QUERY`, `GMAIL_SEARCH_AFTER_DAYS`
- `EMAIL_CHECK_INTERVAL`, `EMAIL_STATE_DB_PATH`
- `EMAIL_DRY_RUN`, `EMAIL_API_URL`

### 5.2 Command Entry Point
**File**: `cmd/email-tracker/main.go`
```go
func main() {
    // Load configuration
    cfg, err := config.LoadEmailConfig()
    
    // Initialize components
    emailClient := email.NewGmailClient(cfg.Gmail)
    extractor := parser.NewTrackingExtractor()
    stateManager := state.NewManager(cfg.StateDBPath)
    apiClient := api.NewClient(cfg.API)
    
    // Create and start processor
    processor := workers.NewEmailProcessor(emailClient, extractor, stateManager, apiClient, cfg)
    
    // Handle signals and graceful shutdown
    server.HandleSignals(processor)
}
```

## Phase 6: API Integration (Medium Priority)

### 6.1 API Client
**File**: `internal/api/client.go`
```go
type Client struct {
    baseURL    string
    httpClient *http.Client
    retryCount int
    retryDelay time.Duration
}

func (c *Client) CreateShipment(tracking TrackingInfo) error
func (c *Client) retryRequest(req *http.Request) (*http.Response, error)
```

- HTTP client for REST API calls
- Retry logic for transient failures
- Proper error handling for 409 (duplicate) responses
- Request/response logging

## Phase 7: Testing (Medium Priority)

### 7.1 Unit Tests
- `internal/email/gmail_test.go` - Mock Gmail API responses
- `internal/parser/extractor_test.go` - Test parsing various email formats
- `internal/workers/email_processor_test.go` - Worker lifecycle testing
- `internal/api/client_test.go` - API client error handling

### 7.2 Integration Tests
- End-to-end email processing flow
- Gmail API integration testing
- Database state management testing
- Error recovery testing

### 7.3 Test Data
- Sample emails from each carrier
- Various email formats (HTML, plain text)
- Edge cases and malformed emails

## Phase 8: Documentation (Low Priority)

### 8.1 Setup Documentation
**File**: `docs/email-tracker-setup.md`
- Gmail API setup and OAuth2 configuration
- Environment variable configuration
- Running the email tracker daemon
- Troubleshooting common issues

### 8.2 Code Documentation
- Comprehensive godoc comments
- Architecture decision records
- Configuration examples

## Implementation Order

1. **Start with interfaces** (`internal/email/types.go`)
2. **Build Gmail client** (`internal/email/gmail.go`)
3. **Implement parser** (`internal/parser/extractor.go`)
4. **Create worker** (`internal/workers/email_processor.go`)
5. **Add state management** (`internal/email/state.go`)
6. **Build main command** (`cmd/email-tracker/main.go`)
7. **Add configuration** (`internal/config/email_config.go`)
8. **Implement API client** (`internal/api/client.go`)
9. **Write comprehensive tests**
10. **Create documentation**

## Testing Strategy

- **TDD Approach**: Write tests first for new functionality
- **Mock external dependencies**: Gmail API, REST API
- **Test error conditions**: Network failures, parsing errors
- **Integration testing**: Full workflow testing
- **Performance testing**: Memory usage, processing speed

## Dependencies to Add

```go
// Gmail API
"google.golang.org/api/gmail/v1"
"golang.org/x/oauth2"
"golang.org/x/oauth2/google"

// Email parsing
"github.com/emersion/go-imap/v2"  // IMAP fallback
"github.com/jhillyerd/enmime"     // Email parsing

// HTML parsing
"golang.org/x/net/html"
```

## Acceptance Criteria Checklist

- [ ] Gmail API authentication working
- [ ] Search queries return relevant emails
- [ ] Tracking numbers extracted accurately
- [ ] Shipments created via API
- [ ] Duplicate processing avoided
- [ ] Graceful error handling
- [ ] Dry-run mode functional
- [ ] Comprehensive logging
- [ ] Configuration validation
- [ ] Tests passing

---

*Implementation plan created: 2025-07-02*