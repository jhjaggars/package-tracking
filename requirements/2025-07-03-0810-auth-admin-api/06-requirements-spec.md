# Requirements Specification: Admin API Authentication

## Problem Statement
The package tracking system currently exposes admin API endpoints without any authentication or authorization controls. This creates a security vulnerability where anyone can access administrative functions like pausing/resuming the tracking updater.

## Solution Overview
Implement API key-based authentication for admin routes with the ability to disable authentication via configuration. This provides secure-by-default protection while allowing flexibility for development or trusted environments.

## Functional Requirements

### 1. Authentication Mechanism
- **FR1.1**: Admin API routes must require authentication using Bearer token format
- **FR1.2**: API key must be provided in Authorization header as "Bearer <api-key>"
- **FR1.3**: Missing or invalid API keys must result in 401 Unauthorized response
- **FR1.4**: Authentication can be disabled via DISABLE_ADMIN_AUTH environment variable

### 2. Configuration
- **FR2.1**: API key must be configured via ADMIN_API_KEY environment variable
- **FR2.2**: System must fail to start if auth is enabled but no API key is provided
- **FR2.3**: DISABLE_ADMIN_AUTH=true must bypass all authentication checks
- **FR2.4**: API key must be redacted when logging configuration

### 3. Security
- **FR3.1**: Failed authentication attempts must be logged at WARN level
- **FR3.2**: Response must not reveal why authentication failed (generic message only)
- **FR3.3**: API key comparison must be constant-time to prevent timing attacks

### 4. Protected Routes
- **FR4.1**: All routes under /api/admin must be protected
- **FR4.2**: Non-admin routes must remain publicly accessible
- **FR4.3**: Health check endpoint must remain public

## Technical Requirements

### 1. Configuration Changes (internal/config/config.go)
- Add `DisableAdminAuth bool` field to Config struct
- Add `AdminAPIKey string` field to Config struct
- Load DISABLE_ADMIN_AUTH with default false
- Load ADMIN_API_KEY from environment
- Validate that API key exists when auth is enabled
- Add getter methods: GetDisableAdminAuth() and GetAdminAPIKey()
- Implement redaction for API key in logs

### 2. Middleware Implementation (internal/server/middleware.go)
- Create new AuthMiddleware function that accepts API key
- Extract Bearer token from Authorization header
- Use crypto/subtle.ConstantTimeCompare for secure comparison
- Log failed attempts with request details (path, method, IP)
- Return 401 with generic "Unauthorized" message

### 3. Route Protection (cmd/server/main.go)
- Apply AuthMiddleware to admin route group
- Check DisableAdminAuth flag before applying middleware
- Maintain existing middleware order

### 4. Testing Requirements
- Unit tests for AuthMiddleware with various scenarios
- Configuration validation tests
- Integration tests for protected endpoints
- Test that disabled auth allows access

## Implementation Hints

### Middleware Pattern
```go
func AuthMiddleware(apiKey string) func(http.Handler) http.Handler {
    expectedKey := []byte(apiKey)
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract Bearer token
            // Constant-time comparison
            // Log failures
            // Return 401 or continue
        })
    }
}
```

### Configuration Pattern
Follow existing patterns from DISABLE_RATE_LIMIT and DISABLE_CACHE:
- Use getEnvBoolOrDefault for boolean flags
- Add validation in validate() method
- Implement interface methods for getters

### Route Protection Pattern
```go
r.Route("/api/admin", func(r chi.Router) {
    if !cfg.GetDisableAdminAuth() {
        r.Use(server.AuthMiddleware(cfg.GetAdminAPIKey()))
    }
    // Existing admin routes...
})
```

## Acceptance Criteria

1. **Security**
   - [ ] Admin routes return 401 without valid API key
   - [ ] Failed auth attempts are logged
   - [ ] API key is not exposed in logs
   - [ ] Timing attacks are prevented

2. **Configuration**
   - [ ] Server starts with valid ADMIN_API_KEY
   - [ ] Server fails to start without API key (when auth enabled)
   - [ ] DISABLE_ADMIN_AUTH=true allows unrestricted access
   - [ ] Configuration is logged safely at startup

3. **Functionality**
   - [ ] Valid API key allows access to admin routes
   - [ ] Public routes remain accessible without auth
   - [ ] Bearer token format is properly parsed
   - [ ] Generic error messages for auth failures

4. **Testing**
   - [ ] All auth scenarios are tested
   - [ ] Configuration validation is tested
   - [ ] Middleware ordering is verified
   - [ ] Integration tests pass

## Assumptions
- Single admin user is sufficient (no multi-user support needed)
- API key rotation is handled manually by changing environment variable
- No need for API key expiration or refresh tokens
- Authentication is sufficient (no fine-grained authorization needed)
- Standard Bearer token format is acceptable