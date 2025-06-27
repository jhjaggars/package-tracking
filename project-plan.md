# Package Tracking Application Plan

## Overview
A two-part system for tracking shipments to your home:
1. **Core Tracking System**: Manual entry with real-time tracking
2. **AI Email Processor**: Automated extraction of tracking numbers from emails

## Core Features & Requirements

### Part 1: Core Tracking System
- **Add shipments**: Manual entry with tracking number, carrier, description
- **Real-time tracking**: Automatic status updates from carrier APIs
- **Delivery notifications**: Alerts when packages arrive or status changes
- **Package history**: View past deliveries and tracking timeline
- **Multi-carrier support**: USPS, UPS, FedEx, Amazon, DHL
- **Expected delivery dates**: Show estimated arrival times
- **Package organization**: Categories, priorities, or tags

### Part 2: AI Email Processing System
- **Email monitoring**: Connect to Gmail/Outlook/IMAP accounts
- **Smart extraction**: AI identifies shipping confirmation emails
- **Tracking number detection**: Extract tracking numbers and carrier info
- **Auto-population**: Automatically add shipments to Part 1 system
- **User approval**: Review extracted shipments before adding
- **Duplicate prevention**: Avoid adding same shipment twice

## Technology Stack

### Part 1: Core Tracking System (Go + SQLite)
- **Backend**: Go with standard library (minimal dependencies)
- **Database**: SQLite for persistence
- **Web Server**: Built-in `net/http` package
- **Templates**: `html/template` for web interface
- **HTTP Client**: Standard library for carrier API calls
- **Deployment**: Single binary + SQLite database file

### Part 2: AI Email Processing System (Future)
- **Backend**: Node.js/Express + TypeScript (or Go with AI/ML integrations)
- **Database**: SQLite (shared with Part 1) + Redis for queues
- **AI/ML**: OpenAI API or local Hugging Face models
- **Email**: Gmail API + IMAP libraries
- **Frontend**: Simple HTML/CSS/JS or React (depending on complexity)
- **Deployment**: Docker containers or single binary

## Data Model
- **Shipments**: tracking_number, carrier, description, status, created_date, expected_delivery
- **Tracking_Events**: shipment_id, timestamp, location, status, description
- **Carriers**: name, api_endpoint, api_key_required

## Architecture

### System Components
1. **API Layer**: RESTful endpoints for CRUD operations
2. **Tracking Service**: Background job to poll carrier APIs
3. **Notification Service**: Email/SMS/push notifications
4. **Web Interface**: Dashboard showing active shipments
5. **Database**: Store shipment data and tracking history
6. **Email Processor**: AI-powered email analysis and extraction
7. **Approval Queue**: User review system for auto-detected shipments

## AI Components
- **Email Classification**: Identify shipping/delivery emails vs regular emails
- **Entity Extraction**: Pull tracking numbers, carriers, order details, delivery addresses
- **Pattern Recognition**: Learn from different email formats (Amazon, retailers, carriers)
- **Confidence Scoring**: Rate extraction accuracy for user review

## User Interface Plan
- **Dashboard**: Grid/list view of active shipments with status indicators
- **Add Shipment**: Simple form with tracking number and carrier selection
- **Shipment Details**: Timeline view showing tracking events and map
- **Approval Queue**: Review auto-detected shipments from email processing
- **Settings**: Notification preferences, API keys, delivery address, email connections

## Email Access & Security
- **Authentication**: OAuth2 for Gmail/Outlook, app passwords for IMAP
- **Permissions**: Read-only access, specific folder monitoring
- **Data Privacy**: Process emails in memory, don't store email content
- **Secure Storage**: Encrypted token storage, environment variables
- **User Control**: Easy disconnect, data deletion options

## Tracking Number Extraction & Validation
- **Pattern Recognition**: Regex patterns for USPS, UPS, FedEx, DHL, Amazon formats
- **AI Processing**: OpenAI/local LLM for context-aware extraction from email text
- **Validation**: Check extracted numbers against carrier format rules
- **Confidence Scoring**: ML model rates extraction accuracy (0-100%)
- **Attachment Processing**: OCR for PDF/image shipping labels
- **Carrier Detection**: Auto-identify carrier from tracking number format

## API Integration Requirements
- **Rate limiting**: Respect carrier API limits
- **Error handling**: Graceful failures when APIs are down
- **Data caching**: Store tracking data to reduce API calls
- **Webhook support**: Real-time updates where available

## System Integration
- **API Endpoints**: Email processor calls REST API to create pending shipments
- **Approval Queue**: User reviews auto-detected shipments before confirmation
- **Notification System**: Alert user of new detected shipments for approval
- **Duplicate Detection**: Check existing shipments before creating new ones
- **Background Processing**: Queue system for email processing (Redis/Bull)
- **Error Handling**: Retry failed extractions, log processing errors

## Implementation Phases

### Part 1: Go + SQLite Core System
1. **Phase 1**: Core CRUD API with SQLite (✅ Tests Written)
   - REST endpoints for shipments management
   - SQLite database with proper schema
   - Basic validation and error handling
   - Health check and carrier endpoints

2. **Phase 2**: Web interface and templates
   - HTML templates with Go's `html/template`
   - Basic CSS styling (no external frameworks)
   - JavaScript for dynamic interactions
   - Dashboard and forms for shipment management

3. **Phase 3**: Carrier API integrations
   - HTTP clients for USPS, UPS, FedEx APIs
   - Background service for tracking updates
   - Retry logic and error handling
   - Rate limiting and caching

4. **Phase 4**: Background services and notifications
   - Scheduled tracking updates
   - Simple notification system
   - Logging and monitoring

### Part 2: AI Email Processing System (Future)
5. **Phase 5**: Email processing system and AI extraction
6. **Phase 6**: User approval queue and system integration
7. **Phase 7**: Advanced features and optimizations

## Current Status
- ✅ **System 1 Design Complete**: Go + SQLite architecture defined
- ✅ **Comprehensive Test Suite**: Full CRUD API tests written and executable
- 🚧 **Next**: Implement CRUD API handlers to make tests pass