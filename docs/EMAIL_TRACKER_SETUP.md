# Email Tracker Setup Guide

The Email Tracker is a standalone service that monitors Gmail accounts for shipping emails and automatically extracts tracking numbers to create shipments in the package tracking system.

## Quick Start

1. **Build the binary:**
   ```bash
   go build -o bin/email-tracker cmd/email-tracker/main.go
   ```

2. **Set up Gmail credentials** (see Gmail Setup section below)

3. **Configure environment variables:**
   ```bash
   export GMAIL_CLIENT_ID="your-client-id"
   export GMAIL_CLIENT_SECRET="your-client-secret"
   export GMAIL_REFRESH_TOKEN="your-refresh-token"
   export EMAIL_API_URL="http://localhost:8080"
   ```

4. **Run the service:**
   ```bash
   ./bin/email-tracker
   ```

## Gmail API Setup

### Option 1: OAuth2 (Recommended)

1. **Create Google Cloud Project:**
   - Go to [Google Cloud Console](https://console.cloud.google.com)
   - Create a new project or select existing one
   - Enable the Gmail API

2. **Create OAuth2 Credentials:**
   - Go to Credentials → Create Credentials → OAuth 2.0 Client IDs
   - Application type: Desktop application
   - Download the JSON file

3. **Get Refresh Token:**
   - Use the OAuth2 playground or custom script to get refresh token
   - Requires one-time authorization flow

### Option 2: IMAP Fallback (Not yet implemented)

1. **Enable 2-Factor Authentication** on your Google account
2. **Generate App Password:**
   - Go to Google Account settings
   - Security → 2-Step Verification → App passwords
   - Generate password for "Mail"

## Configuration

### Environment Variables

#### Gmail Configuration
```bash
# OAuth2 (Primary method)
GMAIL_CLIENT_ID=your-client-id.apps.googleusercontent.com
GMAIL_CLIENT_SECRET=your-client-secret
GMAIL_REFRESH_TOKEN=your-refresh-token
GMAIL_ACCESS_TOKEN=your-access-token
GMAIL_TOKEN_FILE=./gmail-token.json

# IMAP Fallback (Future)
GMAIL_USERNAME=user@gmail.com
GMAIL_APP_PASSWORD=app-specific-password
```

#### Search Configuration
```bash
# Custom search query (optional)
GMAIL_SEARCH_QUERY="from:(ups.com OR fedex.com) subject:tracking"

# Search parameters
GMAIL_SEARCH_AFTER_DAYS=30          # Only emails from last 30 days
GMAIL_SEARCH_UNREAD_ONLY=false      # Process all emails, not just unread
GMAIL_SEARCH_MAX_RESULTS=100        # Maximum emails per search
```

#### Processing Configuration
```bash
EMAIL_CHECK_INTERVAL=5m              # Check every 5 minutes
EMAIL_MAX_PER_RUN=50                 # Process max 50 emails per run
EMAIL_DRY_RUN=false                  # Set to true for testing
EMAIL_STATE_DB_PATH=./email-state.db # SQLite database for state
EMAIL_MIN_CONFIDENCE=0.5             # Minimum extraction confidence
EMAIL_DEBUG_MODE=false               # Enable debug logging
```

#### API Configuration
```bash
EMAIL_API_URL=http://localhost:8080  # Package tracking API
EMAIL_API_TIMEOUT=30s                # Request timeout
EMAIL_API_RETRY_COUNT=3              # Number of retries
EMAIL_API_RETRY_DELAY=1s             # Delay between retries
```

#### LLM Configuration (Optional)
```bash
LLM_ENABLED=false                    # Enable LLM parsing
LLM_PROVIDER=openai                  # openai, anthropic, local
LLM_MODEL=gpt-4                      # Model name
LLM_API_KEY=sk-...                   # API key
LLM_MAX_TOKENS=1000                  # Response limit
LLM_TEMPERATURE=0.1                  # Sampling temperature
```

### Configuration File

You can also use a `.env` file in the working directory:

```bash
# .env file
GMAIL_CLIENT_ID=your-client-id
GMAIL_CLIENT_SECRET=your-client-secret
GMAIL_REFRESH_TOKEN=your-refresh-token
EMAIL_API_URL=http://localhost:8080
EMAIL_CHECK_INTERVAL=5m
EMAIL_DRY_RUN=false
```

## Usage Examples

### Basic Usage
```bash
# Start with minimal configuration
export GMAIL_CLIENT_ID="your-id"
export GMAIL_CLIENT_SECRET="your-secret" 
export GMAIL_REFRESH_TOKEN="your-token"
./bin/email-tracker
```

### Testing Mode
```bash
# Run in dry-run mode with debug logging
export EMAIL_DRY_RUN=true
export EMAIL_DEBUG_MODE=true
export EMAIL_CHECK_INTERVAL=1m
./bin/email-tracker
```

### Custom Search
```bash
# Only process Amazon shipping emails
export GMAIL_SEARCH_QUERY="from:amazon.com subject:(shipped OR tracking)"
export EMAIL_CHECK_INTERVAL=10m
./bin/email-tracker
```

### Production Setup
```bash
# Production configuration
export GMAIL_CLIENT_ID="production-id"
export GMAIL_CLIENT_SECRET="production-secret"
export GMAIL_REFRESH_TOKEN="production-token"
export EMAIL_API_URL="https://tracking.company.com"
export EMAIL_CHECK_INTERVAL=5m
export EMAIL_MAX_PER_RUN=100
export EMAIL_STATE_DB_PATH=/var/lib/email-tracker/state.db

# Run as systemd service
./bin/email-tracker
```

## Monitoring

### Logs
The service provides structured logging:
```
2025/07/02 20:00:00 Starting email tracker service version=1.0.0
2025/07/02 20:00:00 Configuration loaded successfully dry_run=false check_interval=5m0s
2025/07/02 20:00:00 Email client initialized successfully
2025/07/02 20:00:00 Starting email processing run
2025/07/02 20:00:01 Found emails to process count=5
2025/07/02 20:00:01 Found tracking numbers count=3
2025/07/02 20:00:01 Created shipments count=3 total_tracking=3
```

### State Database
The service maintains a SQLite database with processing history:
- Processed email IDs to avoid duplicates
- Tracking numbers found
- Processing status and errors
- Statistics and metrics

### Health Check
Monitor the service by checking:
1. **API connectivity**: The service tests API connection on startup
2. **Gmail connectivity**: Validates Gmail access during initialization
3. **Processing metrics**: Check logs for processing statistics
4. **State database**: Verify database file exists and is accessible

## Troubleshooting

### Common Issues

1. **Gmail Authentication Failed**
   - Verify OAuth2 credentials are correct
   - Check if Gmail API is enabled in Google Cloud Console
   - Ensure refresh token is valid

2. **No Emails Found**
   - Check search query syntax
   - Verify date range (GMAIL_SEARCH_AFTER_DAYS)
   - Test search query in Gmail web interface

3. **API Connection Failed**
   - Verify API URL is correct and accessible
   - Check if package tracking server is running
   - Test API health endpoint manually

4. **No Tracking Numbers Extracted**
   - Enable debug mode to see extraction details
   - Check if email content contains valid tracking numbers
   - Verify tracking number formats match carrier patterns

### Debug Mode
Enable detailed logging:
```bash
export EMAIL_DEBUG_MODE=true
./bin/email-tracker
```

This will show:
- Email content being processed
- Tracking number candidates found
- Validation results
- API requests and responses

### Testing
Use dry-run mode to test without creating shipments:
```bash
export EMAIL_DRY_RUN=true
export EMAIL_DEBUG_MODE=true
./bin/email-tracker
```

## Architecture

The Email Tracker consists of:

1. **Gmail Client** (`internal/email/gmail.go`)
   - OAuth2 authentication
   - Gmail API integration
   - Search query execution

2. **Tracking Extractor** (`internal/parser/extractor.go`)
   - Regex-based pattern matching
   - Carrier-specific validation
   - LLM integration (optional)

3. **Email Processor** (`internal/workers/email_processor.go`)
   - Background processing loop
   - State management
   - API integration

4. **State Manager** (`internal/email/state.go`)
   - SQLite database
   - Duplicate detection
   - Processing history

5. **API Client** (`internal/api/client.go`)
   - HTTP client for shipment creation
   - Retry logic
   - Error handling

## Security Considerations

1. **OAuth2 Tokens**: Store securely, rotate regularly
2. **API Access**: Consider adding authentication to package tracking API
3. **Email Content**: Logs may contain sensitive information
4. **State Database**: Contains email metadata and tracking numbers
5. **Network**: Use HTTPS for API communication in production

## Performance

The service is designed for efficiency:
- **Rate Limiting**: Respects Gmail API limits
- **Batch Processing**: Processes multiple emails per run
- **Caching**: Avoids reprocessing emails
- **Memory Efficient**: Processes emails individually
- **Fast Validation**: Uses existing carrier validation logic

Typical performance:
- Gmail API: ~100ms per email search
- Extraction: ~1ms per email
- API calls: ~50ms per shipment creation
- Overall: ~5-10 emails per second

## Integration with Main System

The Email Tracker integrates with the main package tracking system via:

1. **REST API**: Creates shipments using existing endpoints
2. **Shared Database**: Main system sees created shipments immediately  
3. **Compatible Format**: Uses same shipment structure
4. **Notifications**: Main system handles notifications (not email tracker)

The Email Tracker is completely independent and can be:
- Run on separate servers
- Deployed independently
- Scaled horizontally
- Stopped/started without affecting main system