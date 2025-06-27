# Package Tracking System 1 - Go Design

## Project Structure
```
package-tracking/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── database/
│   │   ├── db.go               # Database connection and setup
│   │   ├── migrations.go       # SQLite schema migrations
│   │   └── models.go           # Data models and queries
│   ├── handlers/
│   │   ├── shipments.go        # HTTP handlers for shipments
│   │   └── health.go           # Health check endpoint
│   ├── services/
│   │   ├── tracking.go         # Carrier API integration
│   │   └── scheduler.go        # Background tracking updates
│   └── templates/
│       ├── index.html          # Dashboard page
│       ├── add.html            # Add shipment form
│       └── detail.html         # Shipment details
├── web/
│   ├── static/
│   │   ├── style.css           # Simple CSS
│   │   └── script.js           # Basic JavaScript
└── database.db                 # SQLite database file
```

## Core Dependencies (Standard Library Only)
- `database/sql` + `github.com/mattn/go-sqlite3` (CGO required for SQLite)
- `net/http` for web server
- `html/template` for templating
- `encoding/json` for JSON handling
- `time` for scheduling
- `context` for request handling

## Data Models

### Shipment
```go
type Shipment struct {
    ID              int       `json:"id"`
    TrackingNumber  string    `json:"tracking_number"`
    Carrier         string    `json:"carrier"`
    Description     string    `json:"description"`
    Status          string    `json:"status"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
    ExpectedDelivery *time.Time `json:"expected_delivery,omitempty"`
    IsDelivered     bool      `json:"is_delivered"`
}
```

### TrackingEvent
```go
type TrackingEvent struct {
    ID         int       `json:"id"`
    ShipmentID int       `json:"shipment_id"`
    Timestamp  time.Time `json:"timestamp"`
    Location   string    `json:"location"`
    Status     string    `json:"status"`
    Description string   `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
}
```

### Carrier
```go
type Carrier struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Code     string `json:"code"`
    APIEndpoint string `json:"api_endpoint"`
    Active   bool   `json:"active"`
}
```

## SQLite Schema
```sql
CREATE TABLE IF NOT EXISTS shipments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tracking_number TEXT NOT NULL UNIQUE,
    carrier TEXT NOT NULL,
    description TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expected_delivery DATETIME,
    is_delivered BOOLEAN DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS tracking_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    shipment_id INTEGER NOT NULL,
    timestamp DATETIME NOT NULL,
    location TEXT,
    status TEXT NOT NULL,
    description TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (shipment_id) REFERENCES shipments(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS carriers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    api_endpoint TEXT,
    active BOOLEAN DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_shipments_status ON shipments(status);
CREATE INDEX IF NOT EXISTS idx_shipments_carrier ON shipments(carrier);
CREATE INDEX IF NOT EXISTS idx_tracking_events_shipment ON tracking_events(shipment_id);
```

## HTTP API Endpoints

### REST API
- `GET /api/shipments` - List all shipments
- `POST /api/shipments` - Add new shipment
- `GET /api/shipments/{id}` - Get shipment details
- `PUT /api/shipments/{id}` - Update shipment
- `DELETE /api/shipments/{id}` - Delete shipment
- `GET /api/shipments/{id}/events` - Get tracking events
- `POST /api/shipments/{id}/refresh` - Force tracking update
- `GET /api/carriers` - List supported carriers
- `GET /api/health` - Health check

### Web Pages
- `GET /` - Dashboard with shipment list
- `GET /add` - Add shipment form
- `GET /shipments/{id}` - Shipment detail page

## Carrier Integration Strategy

### Standard Library HTTP Client
```go
type TrackingService struct {
    client *http.Client
    timeout time.Duration
}

func (ts *TrackingService) Track(carrier, trackingNumber string) (*TrackingInfo, error) {
    switch carrier {
    case "usps":
        return ts.trackUSPS(trackingNumber)
    case "ups":
        return ts.trackUPS(trackingNumber)
    case "fedex":
        return ts.trackFedEx(trackingNumber)
    default:
        return nil, fmt.Errorf("unsupported carrier: %s", carrier)
    }
}
```

### Supported Carriers (Phase 1)
1. **USPS** - USPS Tracking API
2. **UPS** - UPS Tracking API  
3. **FedEx** - FedEx Track API
4. **Generic** - Manual status updates only

## Background Service Design

### Simple Scheduler
```go
type Scheduler struct {
    db *sql.DB
    tracking *TrackingService
    interval time.Duration
    quit chan bool
}

func (s *Scheduler) Start() {
    ticker := time.NewTicker(s.interval)
    for {
        select {
        case <-ticker.C:
            s.updateAllActiveShipments()
        case <-s.quit:
            ticker.Stop()
            return
        }
    }
}
```

## Web Frontend (Simple HTML/CSS/JS)

### Features
- Responsive design with CSS Grid/Flexbox
- HTMX or vanilla JS for dynamic updates
- No external CSS frameworks
- Progressive enhancement

### Pages
1. **Dashboard**: Table view of active shipments
2. **Add Form**: Simple form with carrier dropdown
3. **Detail View**: Timeline of tracking events
4. **Settings**: Basic configuration options

## Configuration

### Environment Variables
```bash
DB_PATH=./database.db
SERVER_PORT=8080
UPDATE_INTERVAL=1h
USPS_API_KEY=your_key
UPS_API_KEY=your_key
FEDEX_API_KEY=your_key
```

### Config Struct
```go
type Config struct {
    DBPath        string
    ServerPort    string
    UpdateInterval time.Duration
    USPSAPIKey    string
    UPSAPIKey     string
    FedExAPIKey   string
}
```

## Error Handling Strategy
- Structured logging with `log/slog`
- Graceful degradation when APIs fail
- Retry logic for transient failures
- User-friendly error messages
- Database transaction rollbacks

## Security Considerations
- Input validation and sanitization
- SQL injection prevention with prepared statements
- Rate limiting for API endpoints
- Basic authentication for admin functions
- HTTPS in production (reverse proxy)

## Deployment
- Single binary with embedded templates
- SQLite database file
- Systemd service file
- Docker container (optional)
- Reverse proxy configuration (nginx/caddy)