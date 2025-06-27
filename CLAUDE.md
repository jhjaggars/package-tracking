# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview
Package tracking system built in Go with SQLite. This is "System 1" of a planned two-part system - a core tracking system with manual entry and real-time carrier API integration. System 2 (future) will add AI-powered email processing for automatic shipment detection.

## Commands

### Build and Run
```bash
# Build the server
go build -o bin/server cmd/server/main.go

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

## Architecture

### Project Structure
- `cmd/server/main.go` - Application entry point with server setup and graceful shutdown
- `internal/config/` - Configuration management with environment variable support
- `internal/database/` - SQLite database layer with models and stores
- `internal/handlers/` - HTTP handlers for REST API endpoints
- `internal/server/` - HTTP server setup, routing, and middleware

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

### API Endpoints
REST API following `/api/` prefix:
- Shipments: GET/POST `/api/shipments`, GET/PUT/DELETE `/api/shipments/{id}`
- Events: GET `/api/shipments/{id}/events`
- Carriers: GET `/api/carriers`
- Health: GET `/api/health`

### Environment Variables
Configuration via environment variables with sensible defaults:
- `SERVER_PORT` (default: 8080)
- `SERVER_HOST` (default: localhost)
- `DB_PATH` (default: ./database.db)
- `UPDATE_INTERVAL` (default: 1h)
- `USPS_API_KEY`, `UPS_API_KEY`, `FEDEX_API_KEY`, `DHL_API_KEY` (optional)
- `LOG_LEVEL` (default: info)

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

## Future Plans
This is Phase 1 of a larger system. Future phases will add:
- Web frontend with HTML templates
- Carrier API integrations for real-time tracking
- Background service for automatic updates
- AI email processing system (System 2)