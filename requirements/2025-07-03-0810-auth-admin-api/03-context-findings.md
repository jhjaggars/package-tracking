# Context Findings

## Architecture Overview
- **Router**: Chi router v5 with middleware chain pattern
- **Configuration**: Environment-based with `.env` file support
- **Admin Routes**: Currently unprotected under `/api/admin`
- **Logging**: Structured logging with slog for workers, standard log for HTTP

## Key Files to Modify
1. `internal/config/config.go` - Add auth configuration
2. `internal/server/middleware.go` - Add authentication middleware
3. `cmd/server/main.go` - Apply auth middleware to admin routes
4. `internal/handlers/admin.go` - Admin route handlers (no changes needed)

## Patterns to Follow

### Configuration Pattern
```go
// In Config struct
DisableAdminAuth bool
AdminAPIKey      string

// Environment loading
cfg.DisableAdminAuth = getEnvBoolOrDefault("DISABLE_ADMIN_AUTH", false)
cfg.AdminAPIKey = os.Getenv("ADMIN_API_KEY")

// Validation
if !cfg.DisableAdminAuth && cfg.AdminAPIKey == "" {
    return errors.New("ADMIN_API_KEY is required when admin auth is enabled")
}

// Getter methods
GetDisableAdminAuth() bool
GetAdminAPIKey() string
```

### Middleware Pattern
```go
func AuthMiddleware(apiKey string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Check Authorization header
            // Log failures
            // Return 401 if unauthorized
            next.ServeHTTP(w, r)
        })
    }
}
```

### Route Protection Pattern
```go
r.Route("/api/admin", func(r chi.Router) {
    if !cfg.GetDisableAdminAuth() {
        r.Use(server.AuthMiddleware(cfg.GetAdminAPIKey()))
    }
    // Admin routes...
})
```

## Similar Features Analyzed

### DISABLE_RATE_LIMIT Implementation
- Config field: `DisableRateLimit bool`
- Loaded with: `getEnvBoolOrDefault("DISABLE_RATE_LIMIT", false)`
- Applied conditionally in refresh handler
- No validation needed (boolean with default)

### DISABLE_CACHE Implementation
- Config field: `DisableCache bool`
- Loaded with: `getEnvBoolOrDefault("DISABLE_CACHE", false)`
- Checked in cache service before caching
- Similar pattern to follow for DISABLE_ADMIN_AUTH

## Security Considerations
1. API key should be validated in constant time to prevent timing attacks
2. Failed auth attempts should be logged at WARN level
3. Use Authorization header with Bearer token format
4. Return 401 Unauthorized for invalid/missing credentials
5. Security headers already set by SecurityMiddleware

## Testing Requirements
1. Test middleware with valid/invalid/missing API keys
2. Test that disabled auth allows access
3. Test logging of failed attempts
4. Test middleware ordering
5. Test configuration validation

## Integration Points
- Admin routes already grouped under `/api/admin`
- CORS already allows Authorization header
- Logging infrastructure in place
- Error handling patterns established