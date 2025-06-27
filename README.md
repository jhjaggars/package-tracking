# Package Tracking System

A comprehensive two-part system for tracking shipments to your home, built with Go and SQLite using minimal dependencies and test-driven development.

## Overview

**Part 1: Core Tracking System** ✅ **COMPLETE**
- Manual shipment entry with comprehensive CRUD operations
- RESTful API with custom HTTP router and middleware
- Production-ready server with graceful shutdown and signal handling
- Carrier API integration for USPS, UPS, FedEx, and DHL
- Unified tracking interface with standardized error handling
- Comprehensive test coverage with TDD methodology

**Part 2: AI Email Processor** 🚧 **PLANNED**
- Automated extraction of tracking numbers from emails
- AI-powered email analysis and validation
- User approval workflow for auto-detected shipments

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- SQLite support (included via CGO)

### Running the Server
```bash
# Clone the repository
git clone git@github.com:jhjaggars/package-tracking.git
cd package-tracking

# Run the server (creates database.db automatically)
go run cmd/server/main.go

# Server starts on http://localhost:8080
```

### Testing the API
```bash
# Health check
curl http://localhost:8080/api/health

# List carriers
curl http://localhost:8080/api/carriers

# Create a shipment
curl -X POST http://localhost:8080/api/shipments \
  -H "Content-Type: application/json" \
  -d '{"tracking_number":"1Z999AA1234567890","carrier":"ups","description":"Test Package"}'

# List shipments
curl http://localhost:8080/api/shipments
```

## 🏗️ Architecture

### Technology Stack
- **Backend**: Go with standard library (minimal dependencies)
- **Database**: SQLite for persistence with automatic migrations
- **Router**: Custom HTTP router with path parameter extraction
- **Middleware**: Logging, CORS, security headers, panic recovery
- **Carrier APIs**: USPS (XML), UPS/FedEx (OAuth 2.0 JSON), DHL (API key JSON)
- **Testing**: Comprehensive TDD with mock HTTP servers
- **Deployment**: Single binary + SQLite database file

### Project Structure
```
package-tracking/
├── cmd/server/main.go           # Application entry point
├── internal/
│   ├── config/                  # Configuration management
│   ├── database/                # Database models and operations
│   ├── handlers/                # HTTP request handlers
│   ├── carriers/                # Carrier API clients (USPS, UPS, FedEx, DHL)
│   └── server/                  # Router, middleware, and server logic
├── go.mod                       # Go module definition
└── database.db                  # SQLite database (auto-created)
```

## 📊 Data Models

### Shipment
```go
type Shipment struct {
    ID               int        `json:"id"`
    TrackingNumber   string     `json:"tracking_number"`
    Carrier          string     `json:"carrier"`
    Description      string     `json:"description"`
    Status           string     `json:"status"`
    CreatedAt        time.Time  `json:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at"`
    ExpectedDelivery *time.Time `json:"expected_delivery,omitempty"`
    IsDelivered      bool       `json:"is_delivered"`
}
```

### TrackingEvent
```go
type TrackingEvent struct {
    ID          int       `json:"id"`
    ShipmentID  int       `json:"shipment_id"`
    Timestamp   time.Time `json:"timestamp"`
    Location    string    `json:"location"`
    Status      string    `json:"status"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
}
```

## 🌐 API Endpoints

### Shipments
- `GET /api/shipments` - List all shipments
- `POST /api/shipments` - Create new shipment
- `GET /api/shipments/{id}` - Get shipment by ID
- `PUT /api/shipments/{id}` - Update shipment
- `DELETE /api/shipments/{id}` - Delete shipment
- `GET /api/shipments/{id}/events` - Get tracking events for shipment

### System
- `GET /api/health` - Health check with database connectivity
- `GET /api/carriers` - List supported carriers
- `GET /api/carriers?active=true` - List only active carriers

## ⚙️ Configuration

Environment variables with sensible defaults:

```bash
# Server configuration
SERVER_HOST=localhost          # Server host
SERVER_PORT=8080              # Server port
DB_PATH=./database.db         # SQLite database path

# Feature configuration
UPDATE_INTERVAL=1h            # Background update interval
LOG_LEVEL=info               # Logging level (debug, info, warn, error)

# Carrier API keys (optional)
USPS_API_KEY=your_key
UPS_API_KEY=your_key
FEDEX_API_KEY=your_key
DHL_API_KEY=your_key
```

## 🧪 Testing

### Comprehensive Test Suite
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run integration tests only
go test ./internal/server -run TestIntegration
```

### Test Coverage
- ✅ **Unit Tests**: All handlers, router, middleware, configuration
- ✅ **Integration Tests**: Full HTTP workflows with real server
- ✅ **Database Tests**: CRUD operations with in-memory SQLite
- ✅ **Error Handling**: Validation, edge cases, and failure scenarios

### Live Server Testing
```bash
# Test script with full API workflow
./test_server.sh

# Signal handling demonstration
./test_signal_comprehensive.sh
```

## 🔒 Security Features

- **Input Validation**: All required fields validated with proper error messages
- **SQL Injection Prevention**: Prepared statements throughout
- **Security Headers**: XSS protection, frame options, content sniffing prevention
- **CORS Support**: Configurable cross-origin resource sharing
- **Graceful Shutdown**: Proper signal handling (SIGTERM, SIGINT)
- **Error Recovery**: Panic recovery middleware with safe error responses

## 🚦 Signal Handling

The server handles shutdown signals gracefully:

- **SIGTERM**: Graceful shutdown with 30-second timeout
- **SIGINT** (Ctrl+C): Graceful shutdown with cleanup
- **SIGKILL**: Immediate termination (uncatchable by design)

```bash
# Graceful shutdown
kill -TERM <pid>

# Force shutdown (not recommended)
kill -9 <pid>
```

## 📈 Current Implementation Status

### ✅ **COMPLETED (Part 1: Core System)**

**Database Layer**
- ✅ SQLite schema with proper relationships and indexes
- ✅ Store pattern for clean database operations
- ✅ Automatic migrations and default data seeding
- ✅ Transaction support and error handling

**HTTP Layer**
- ✅ Custom router with path parameter extraction (`/api/shipments/{id}`)
- ✅ RESTful API design with proper HTTP status codes
- ✅ Comprehensive middleware stack (logging, CORS, security, recovery)
- ✅ JSON request/response handling with validation

**Server Infrastructure**
- ✅ Production-ready server with timeouts and graceful shutdown
- ✅ Environment-based configuration with validation
- ✅ Structured logging and error handling
- ✅ Signal handling for deployment environments

**Testing & Quality**
- ✅ 100% test coverage for all components
- ✅ Integration testing with real HTTP server
- ✅ Database testing with in-memory SQLite
- ✅ Error scenario and edge case testing

**Deployment**
- ✅ Single binary deployment
- ✅ SQLite database with automatic setup
- ✅ Environment variable configuration
- ✅ Docker-ready architecture

**Carrier API Integration** ✅ **COMPLETE**
- ✅ HTTP clients for USPS, UPS, FedEx, DHL APIs
- ✅ Unified Client interface with standardized error handling
- ✅ Comprehensive authentication (OAuth 2.0, API keys, user IDs)
- ✅ Rate limiting and quota tracking for all carriers
- ✅ Retry logic with automatic token refresh
- ✅ Multiple data format support (XML for USPS, JSON for others)
- ✅ Tracking number validation for all carrier formats
- ✅ Status mapping to standardized tracking states
- ✅ Batch processing where supported (USPS: 10, FedEx: 30, UPS/DHL: 1)
- ✅ Rich metadata extraction (weight, dimensions, service types)
- ✅ Event timeline parsing with location and timestamp data

### 🚧 **PLANNED (Future Phases)**

**Phase 2: Background Services**
- Automatic tracking updates for active shipments
- Configurable update intervals and scheduling
- Database integration for tracking data persistence
- Notification system for status changes
- Retry logic and failure handling for API outages

**Phase 3: Web Interface** 
- HTML templates with Go's `html/template`
- Responsive design with vanilla CSS/JS
- Dashboard and shipment management forms
- Real-time updates and notifications

**Phase 4: AI Email Processing (Part 2)**
- Email monitoring (Gmail/Outlook/IMAP)
- AI-powered tracking number extraction
- User approval workflow for auto-detected shipments
- Integration with core tracking system

## 🤝 Contributing

This project follows test-driven development (TDD) principles:

1. **Write tests first** for new functionality
2. **Implement** to make tests pass
3. **Refactor** while maintaining test coverage
4. **Document** changes in commit messages

### Development Workflow
```bash
# Run tests continuously during development
go test ./... -watch

# Check test coverage
go test -cover ./...

# Build and test the server
go build -o bin/server cmd/server/main.go
./bin/server
```

## 📝 Implementation Notes

### Design Principles
- **Minimal Dependencies**: Only SQLite driver beyond standard library
- **Test-Driven Development**: Comprehensive test suite written first
- **Clean Architecture**: Separate layers for database, handlers, and server logic
- **Production Ready**: Proper error handling, logging, and graceful shutdown
- **Extensible Design**: Easy to add new carriers, endpoints, and features

### Key Technical Decisions
- **SQLite over PostgreSQL**: Simpler deployment, sufficient for single-user system
- **Custom Router over Gorilla/Chi**: Educational value and zero dependencies
- **Standard Library HTTP**: Reliable, well-tested, and lightweight
- **In-Memory Testing**: Fast test execution without external dependencies
- **Unified Client Interface**: Consistent API across all carriers despite different authentication methods
- **Comprehensive Error Handling**: CarrierError type with retry and rate limit flags
- **Test-Driven Development**: All carrier clients built with failing tests first

---

**Built with Test-Driven Development using Go 1.21+ and SQLite**

🤖 *This project was developed with [Claude Code](https://claude.ai/code) assistance*