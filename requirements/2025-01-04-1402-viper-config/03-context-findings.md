# Context Findings

## Current Configuration Architecture

### 1. Configuration Types
- **Server Config** (`internal/config/config.go`) - 40+ configuration fields
- **Email Config** (`internal/config/email_config.go`) - Nested structure with 5 sub-configs
- **CLI Config** (`internal/cli/config.go`) - Simple 3-field configuration

### 2. Key Files to Modify

#### Core Configuration Files
- `internal/config/config.go` - Main server configuration (needs Viper integration)
- `internal/config/email_config.go` - Email tracker config (needs Viper integration)
- `internal/config/helpers.go` - Helper functions (will be replaced by Viper)
- `internal/cli/config.go` - CLI configuration (needs Viper integration)

#### Application Entry Points
- `cmd/server/main.go` - Uses `config.Load()`
- `cmd/cli/cmd/root.go` - Uses `cliapi.LoadConfig()` with CLI flag precedence
- `cmd/email-tracker/cmd/root.go` - Uses `config.LoadEmailConfig()` with --config flag support

#### Configuration Files
- `.env.example` - Template with all configuration options
- `.env` - Active configuration (must remain supported)
- `~/.package-tracker.json` - CLI configuration file (migrate to Viper format)

### 3. Current Patterns to Preserve

#### Configuration Precedence
1. CLI flags (highest priority)
2. Environment variables
3. .env file values
4. Default values (lowest priority)

#### Security Features
- API key redaction in logs
- Path validation for .env files
- Sensitive field masking in JSON output

#### Validation Pattern
Each config struct has a `validate()` method that performs:
- Required field checks
- Value range validation
- Type validation
- Dependency validation

#### Interface Pattern
Handlers use Config interface to avoid circular imports:
```go
type Config interface {
    GetDisableRateLimit() bool
    GetDisableCache() bool
    // ... getter methods
}
```

### 4. Migration Considerations

#### Backward Compatibility Requirements
- Must continue to load existing .env files
- Environment variable names must remain unchanged
- CLI flags must maintain same behavior
- Existing deployments should work without changes

#### Viper Integration Points
1. Replace `LoadEnvFile()` with Viper's config file loading
2. Replace `getEnvOrDefault()` helpers with Viper's Get methods
3. Maintain validation at application level (not using Viper validation)
4. Support multiple config formats while defaulting to .env

#### Configuration Structure Mapping
Need to map flat environment variables to nested Viper structure:
- `SERVER_PORT` → `server.port`
- `USPS_API_KEY` → `carriers.usps.apiKey`
- `GMAIL_CLIENT_ID` → `email.gmail.clientId`

### 5. Technical Constraints

#### Circular Import Prevention
- Config package cannot import from handlers
- Must maintain interface pattern for config access
- Getter methods required for encapsulation

#### Type Safety
- Current system uses strongly typed config structs
- Need to maintain type safety with Viper unmarshaling
- Custom types (Duration, int64) need proper handling

#### Testing Considerations
- 20+ config-related tests exist
- Tests use environment variable manipulation
- Must maintain test compatibility

### 6. Similar Features Analyzed

#### Environment Loading Pattern
The `LoadEnvFile()` function in `internal/config/helpers.go`:
- Validates file path security
- Parses key=value pairs
- Handles quoted values
- Skips comments
- Only sets if not already in environment

#### Config Passing Pattern
Components receive config via constructor:
```go
trackingUpdater := workers.NewTrackingUpdater(cfg, db.Shipments, carrierFactory, cacheManager, logger)
```

#### CLI Config File Pattern
CLI uses JSON config file with struct unmarshaling:
```go
type CLIConfig struct {
    ServerURL string `json:"server_url"`
    Format    string `json:"format"`
    Quiet     bool   `json:"quiet"`
}
```

### 7. Viper Best Practices for This Project

Based on research and codebase analysis:

1. **Use struct unmarshaling** for type safety
2. **Set up automatic env binding** with prefix (e.g., "PKG_TRACKER_")
3. **Support multiple formats** but default to .env for compatibility
4. **Use SetDefault()** for all current default values
5. **Implement custom decoders** for Duration and other custom types
6. **Create separate Viper instances** for each app (server, CLI, email)
7. **Maintain getter methods** for interface compatibility
8. **Use mapstructure tags** for field mapping

### 8. Integration Points

#### Database Store Access
Configuration is passed to:
- Database stores
- HTTP handlers
- Background workers
- Carrier factories
- Cache managers

#### Middleware Dependencies
Server middleware uses config for:
- CORS settings
- Rate limiting
- Admin authentication
- Cache behavior

### 9. File Format Considerations

Current formats in use:
- `.env` files (key=value format)
- JSON for CLI config
- No YAML/TOML currently used

Viper migration should:
- Default to .env format for server/email-tracker
- Support YAML/TOML as alternatives
- Migrate CLI from JSON to YAML (more readable)

### 10. Related Features

- **Cobra CLI** already in use for email-tracker (integrates well with Viper)
- **Fang framework** for CLI enhancement
- **Structured logging** preparation mentions
- **Graceful shutdown** with config timeout values