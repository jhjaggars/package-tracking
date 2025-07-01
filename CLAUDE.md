# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview
Package tracking system built in Go with SQLite. This is "System 1" of a planned two-part system - a core tracking system with manual entry and real-time carrier API integration. System 2 (future) will add AI-powered email processing for automatic shipment detection.

## Commands

### Build and Run
```bash
# Build the server
go build -o bin/server cmd/server/main.go

# Build the CLI client
go build -o bin/package-tracker cmd/cli/main.go

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
- `internal/config/` - Configuration management with environment variable support
- `internal/database/` - SQLite database layer with models and stores
- `internal/handlers/` - HTTP handlers for REST API endpoints
- `internal/server/` - HTTP server setup, routing, and middleware
- `internal/cli/` - CLI client configuration, HTTP client, and output formatting

### Core Components
1. **Config System**: Environment-based configuration with validation
2. **Database Layer**: SQLite with prepared statements and proper error handling
3. **HTTP Server**: REST API with middleware chain (logging, recovery, CORS, security)
4. **Graceful Shutdown**: Signal handling for clean server termination

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
- Cache entries automatically expire after 5 minutes
- Background cleanup removes expired entries every minute
- Cache invalidation on shipment updates ensures data consistency

### Environment Variables
Configuration via environment variables with sensible defaults.

The server automatically loads variables from a `.env` file if present. Environment variables take precedence over `.env` file values:
- `SERVER_PORT` (default: 8080)
- `SERVER_HOST` (default: localhost)
- `DB_PATH` (default: ./database.db)
- `UPDATE_INTERVAL` (default: 1h)
- `USPS_API_KEY`, `UPS_API_KEY`, `FEDEX_API_KEY`, `FEDEX_SECRET_KEY`, `FEDEX_API_URL`, `DHL_API_KEY` (optional)
- `LOG_LEVEL` (default: info)
- `DISABLE_CACHE` (default: false) - Disable refresh response caching
- `DISABLE_RATE_LIMIT` (default: false) - Disable rate limiting for development/testing

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

## Future Plans
This is Phase 1 of a larger system. Future phases will add:
- Web frontend with HTML templates
- Carrier API integrations for real-time tracking
- Background service for automatic updates
- AI email processing system (System 2)
