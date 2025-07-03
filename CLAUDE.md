# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview
Package tracking system built in Go with SQLite. Comprehensive package tracking with manual entry, real-time carrier API integration, and automated email processing for tracking number extraction. The system includes both a core tracking service and an email processing daemon.

## Commands

### Build and Run
```bash
# Build the server
go build -o bin/server cmd/server/main.go

# Build the CLI client
go build -o bin/package-tracker cmd/cli/main.go

# Build the email tracker
go build -o bin/email-tracker cmd/email-tracker/main.go

# Run the server directly
go run cmd/server/main.go

# Run with custom configuration
SERVER_PORT=8081 DB_PATH=./test.db go run cmd/server/main.go
```

### Testing
```bash
# Run all tests
go test -v ./...

# Run tests for specific package
go test -v ./internal/handlers
go test -v ./internal/config

# Run integration test (starts actual server)
./test_server.sh
```

### Database Management
```bash
# The SQLite database is automatically created at ./database.db
# To use a different path: DB_PATH=./custom.db go run cmd/server/main.go
```

### CLI Usage
```bash
# Add a new shipment
./bin/package-tracker add --tracking "1Z999AA1234567890" --carrier "ups" --description "My Package"

# List all shipments (table format)
./bin/package-tracker list

# List shipments in JSON format
./bin/package-tracker list --format json

# Get specific shipment details
./bin/package-tracker get 1

# View tracking events for a shipment
./bin/package-tracker events 1

# Update shipment description
./bin/package-tracker update 1 --description "Updated description"

# Delete a shipment
./bin/package-tracker delete 1

# Use with custom server endpoint
./bin/package-tracker --server http://example.com:8080 list

# Quiet mode (minimal output)
./bin/package-tracker --quiet list

# Disable color output (for scripts/CI environments)
./bin/package-tracker --no-color list

# Can also disable colors via environment variable
NO_COLOR=1 ./bin/package-tracker list
```

## Architecture

### Project Structure
- `cmd/server/main.go` - Application entry point with server setup and graceful shutdown
- `cmd/cli/main.go` - CLI client entry point for interacting with the API
- `cmd/email-tracker/main.go` - Email processing daemon for automatic tracking number extraction
- `internal/config/` - Configuration management with environment variable support
- `internal/database/` - SQLite database layer with models and stores
- `internal/handlers/` - HTTP handlers for REST API endpoints
- `internal/server/` - HTTP server setup, routing, and middleware
- `internal/cli/` - CLI client configuration, HTTP client, and output formatting
- `internal/email/` - Email client interfaces and Gmail API integration
- `internal/parser/` - Tracking number extraction and validation
- `internal/workers/` - Background processing services (tracking updates, email processing)

### Core Components
1. **Config System**: Environment-based configuration with validation
2. **Database Layer**: SQLite with prepared statements and proper error handling
3. **HTTP Server**: REST API with middleware chain (logging, recovery, CORS, security)
4. **Email Processing**: Gmail API integration with automated tracking number extraction
5. **Tracking Parser**: Multi-carrier tracking number recognition and validation
6. **Graceful Shutdown**: Signal handling for clean server termination

### Database Schema
Main entities:
- `shipments` - Core shipment data with tracking numbers, carriers, status
- `tracking_events` - Historical tracking events for each shipment
- `carriers` - Supported carrier configurations
- `refresh_cache` - In-memory cache storage for refresh responses

### API Endpoints
REST API following `/api/` prefix:
- Shipments: GET/POST `/api/shipments`, GET/PUT/DELETE `/api/shipments/{id}`
- Events: GET `/api/shipments/{id}/events`
- Refresh: POST `/api/shipments/{id}/refresh` - Refresh tracking data with caching
- Carriers: GET `/api/carriers`
- Health: GET `/api/health`
- Admin: GET/POST `/api/admin/tracking-updater/*` - Admin endpoints (authentication required)

### Refresh Caching System
The system implements intelligent caching for refresh requests to improve performance and reduce carrier API load:

**Cache Behavior:**
- Refresh responses are cached for 5 minutes in both memory and SQLite database
- Cache persists across server restarts (loaded from database on startup)
- Cache is automatically invalidated when shipments are updated or deleted
- If cache entry exists and is fresh (< 5 minutes old), returns cached response immediately
- If cache entry is stale or missing, performs actual carrier refresh and caches the result

**Performance Benefits:**
- Cached responses return in < 10ms vs. typical carrier API calls (1-3 seconds)
- Prevents redundant carrier API calls within the 5-minute window
- Transparent to API consumers - same response format regardless of cache hit/miss

**Cache Management:**
- Set `DISABLE_CACHE=true` to disable caching entirely
- Cache entries automatically expire after configurable TTL (default: 5 minutes)
- Background cleanup removes expired entries every minute
- Cache invalidation on shipment updates ensures data consistency

### Admin Authentication
The system includes secure authentication for admin API endpoints to prevent unauthorized access to administrative functions:

**Features:**
- API key-based authentication using Bearer token format
- Configurable authentication (can be disabled for development)
- Secure constant-time API key comparison to prevent timing attacks
- Failed authentication attempts are logged with client IP details
- API key redaction in logs for security

**Usage:**
```bash
# Enable authentication (default behavior)
ADMIN_API_KEY="your-secret-key-here" go run cmd/server/main.go

# Disable authentication for development
DISABLE_ADMIN_AUTH=true go run cmd/server/main.go

# Access admin endpoints with authentication
curl -H "Authorization: Bearer your-secret-key-here" \
     http://localhost:8080/api/admin/tracking-updater/status
```

**Configuration:**
- Set `ADMIN_API_KEY` environment variable with a secure random key
- Use `DISABLE_ADMIN_AUTH=true` to bypass authentication (development only)
- Failed attempts are logged at WARN level with request details
- API keys are automatically redacted in configuration logs

**Protected Endpoints:**
- `GET /api/admin/tracking-updater/status` - Get tracking updater status
- `POST /api/admin/tracking-updater/pause` - Pause automatic updates
- `POST /api/admin/tracking-updater/resume` - Resume automatic updates

### UPS Automatic Updates
The system supports automatic tracking updates for UPS shipments alongside existing USPS auto-updates:

**Features:**
- Unified scheduling - UPS and USPS updates run in the same cycle
- OAuth 2.0 authentication support with `UPS_CLIENT_ID` and `UPS_CLIENT_SECRET`
- Backward compatibility with legacy `UPS_API_KEY` (deprecated)
- Configurable failure threshold to prevent excessive API calls
- Per-carrier cutoff day configuration with global fallback
- Automatic fallback from API to scraping when credentials unavailable

**Behavior:**
- UPS shipments are queried from database using carrier-specific filters
- Cache-based rate limiting applies consistently across all carriers
- Failed updates increment failure count; shipments disabled after threshold reached
- Cache TTL is configurable via `CACHE_TTL` environment variable
- Updates respect the same 5-minute rate limiting as manual refreshes

**Configuration:**
- Use `UPS_AUTO_UPDATE_ENABLED=false` to disable UPS auto-updates
- Configure `UPS_AUTO_UPDATE_CUTOFF_DAYS` for UPS-specific cutoff (defaults to global setting)
- Set `AUTO_UPDATE_FAILURE_THRESHOLD` to control when shipments are disabled due to failures

### Email Tracking Workflow
The system includes automated email processing for Gmail accounts to extract tracking numbers and create shipments:

**Features:**
- Gmail API integration with OAuth2 authentication
- Intelligent tracking number extraction using regex patterns and optional LLM enhancement
- Support for UPS, USPS, FedEx, and DHL tracking formats
- Duplicate email detection and processing state management
- Configurable search queries and filtering
- Dry-run mode for testing without creating shipments
- Graceful error handling and retry logic

**Email Tracker Daemon:**
```bash
# Build and run email tracker
go build -o bin/email-tracker cmd/email-tracker/main.go
./bin/email-tracker

# Example with configuration
export GMAIL_CLIENT_ID="your-client-id"
export GMAIL_CLIENT_SECRET="your-client-secret"
export GMAIL_REFRESH_TOKEN="your-refresh-token"
export EMAIL_API_URL="http://localhost:8080"
./bin/email-tracker
```

**Key Environment Variables:**
- `GMAIL_CLIENT_ID`, `GMAIL_CLIENT_SECRET`, `GMAIL_REFRESH_TOKEN` - Gmail OAuth2 credentials
- `GMAIL_SEARCH_QUERY` - Custom Gmail search query (default: shipping emails from major carriers)
- `EMAIL_CHECK_INTERVAL` - How often to check for new emails (default: 5m)
- `EMAIL_DRY_RUN` - Extract tracking numbers without creating shipments (default: false)
- `EMAIL_STATE_DB_PATH` - SQLite database for tracking processed emails (default: ./email-state.db)
- `EMAIL_API_URL` - Package tracking API endpoint (default: http://localhost:8080)

**Processing Workflow:**
1. Searches Gmail using configurable query patterns
2. Extracts tracking numbers from email content (subject, body, attachments)
3. Validates tracking numbers against carrier-specific patterns
4. Creates shipments via REST API
5. Marks emails as processed to avoid duplicates
6. Maintains processing statistics and error tracking

### Environment Variables
Configuration via environment variables with sensible defaults.

The server automatically loads variables from a `.env` file if present. Environment variables take precedence over `.env` file values:
- `SERVER_PORT` (default: 8080)
- `SERVER_HOST` (default: localhost)
- `DB_PATH` (default: ./database.db)
- `UPDATE_INTERVAL` (default: 1h)
- `USPS_API_KEY`, `UPS_API_KEY` (deprecated), `UPS_CLIENT_ID`, `UPS_CLIENT_SECRET`, `FEDEX_API_KEY`, `FEDEX_SECRET_KEY`, `FEDEX_API_URL`, `DHL_API_KEY` (optional)
- `LOG_LEVEL` (default: info)
- `AUTO_UPDATE_FAILURE_THRESHOLD` (default: 10) - Number of consecutive failures before disabling auto-updates for a shipment
- `UPS_AUTO_UPDATE_ENABLED` (default: true) - Enable/disable UPS automatic updates
- `UPS_AUTO_UPDATE_CUTOFF_DAYS` (default: 30) - Cutoff days for UPS shipments (falls back to AUTO_UPDATE_CUTOFF_DAYS if 0)
- `CACHE_TTL` (default: 5m) - Cache time-to-live duration
- `DISABLE_CACHE` (default: false) - Disable refresh response caching
- `DISABLE_RATE_LIMIT` (default: false) - Disable rate limiting for development/testing
- `DISABLE_ADMIN_AUTH` (default: false) - Disable admin API authentication for development/testing
- `ADMIN_API_KEY` (required when auth enabled) - API key for admin endpoints authentication

#### CLI Configuration
- `PACKAGE_TRACKER_SERVER` (default: http://localhost:8080)
- `PACKAGE_TRACKER_FORMAT` (default: table)
- `PACKAGE_TRACKER_QUIET` (default: false)

CLI also supports a configuration file at `~/.package-tracker.json`:
```json
{
  "server_url": "http://localhost:8080",
  "format": "table",
  "quiet": false
}
```

## Testing Strategy
- Unit tests for handlers using in-memory SQLite databases
- Integration tests via `test_server.sh` script that starts actual server
- Configuration tests with environment variable scenarios
- Tests use `httptest.ResponseRecorder` for HTTP testing

## Development Notes
- Uses minimal external dependencies (only go-sqlite3 driver)
- Standard library HTTP server with custom middleware chain
- Structured logging preparation (mentions slog usage)
- Signal handling for graceful shutdown with configurable timeout
- Middleware includes logging, recovery, CORS, content-type, and security headers

### Development Servers (Tmux-based)
```bash
# Start development environment in tmux session
./start-dev.sh [session-name]

# Examples:
./start-dev.sh                    # Creates session: package-tracker-dev
./start-dev.sh my-feature         # Creates session: my-feature

# Tmux session management:
tmux attach -t package-tracker-dev    # Connect to running servers
tmux list-sessions                     # List all sessions
tmux kill-session -t session-name     # Stop servers and session

# Inside tmux session:
# Ctrl+b then 0  -> Switch to backend window
# Ctrl+b then 1  -> Switch to frontend window  
# Ctrl+b then d  -> Detach from session (keeps servers running)
# Ctrl+C         -> Stop server in current window
```

### Manual Development Setup
- Use the start-dev.sh script for the recommended tmux-based workflow
- For debugging: servers run in separate tmux windows for easy log monitoring

### CLI Styling Features
The CLI includes enhanced visual styling using the Charm ecosystem:
- **Color-coded statuses**: Package statuses are displayed with appropriate colors (delivered=green, in-transit=yellow, pending=blue, failed=red)
- **Styled headers**: Table headers are displayed in bold
- **Progress indicators**: Long operations like refresh show progress spinners (disabled in --no-color mode)
- **Smart color detection**: Colors automatically disabled when output is piped, in CI environments, or when NO_COLOR is set
- **Backward compatibility**: All existing output formats and scripts continue to work unchanged

## Carrier Integration Notes

### FedEx API vs Scraping
- **API Preferred**: When `FEDEX_API_KEY` and `FEDEX_SECRET_KEY` are configured, the system automatically uses the official FedEx Track API
- **Configurable Endpoint**: Use `FEDEX_API_URL` to specify production (apis.fedex.com) or sandbox (apis-sandbox.fedex.com) endpoints
- **Scraping Fallback**: Without API credentials, the system uses enhanced headless browser scraping with bot detection avoidance
- **Error Handling**: Enhanced detection distinguishes between bot detection, server errors, and legitimate tracking failures
- **Performance**: API calls complete in ~2 seconds vs ~96 seconds for scraping

## Current System Features
The package tracking system includes:
- ✅ Core REST API for shipment management
- ✅ CLI client for command-line operations
- ✅ Multiple carrier support (UPS, USPS, FedEx, DHL)
- ✅ Background automatic tracking updates
- ✅ Email processing daemon with Gmail integration
- ✅ Intelligent tracking number extraction
- ✅ Web frontend with modern React UI
- ✅ Caching and rate limiting
- ✅ Admin authentication and controls

## Future Enhancements
Potential future improvements:
- IMAP fallback support for email processing
- Additional LLM providers for enhanced parsing
- Mobile application
- Advanced analytics and reporting
- Multi-language support
- Additional carrier integrations
