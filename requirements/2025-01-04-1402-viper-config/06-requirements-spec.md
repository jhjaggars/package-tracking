# Viper Configuration Migration Requirements Specification

## Problem Statement

The package tracking system currently uses a custom configuration management system with manual environment variable parsing, type conversion helpers, and limited file format support. This implementation requires significant boilerplate code and lacks advanced features like hot-reloading, multiple format support, and standardized patterns across the three applications (server, CLI, email-tracker).

## Solution Overview

Migrate all configuration management to Viper, a mature Go configuration library that provides:
- Automatic environment variable binding with nested key support
- Multiple configuration file format support (JSON, YAML, TOML, HCL, INI, .env)
- Type-safe unmarshaling to structs
- Built-in default value management
- Standardized configuration patterns
- Better integration with Cobra CLI (already used in email-tracker)

## Functional Requirements

### FR1: Configuration File Support
- Support multiple configuration file formats (YAML, TOML, JSON, .env)
- Maintain backward compatibility with existing .env files
- Email-tracker --config flag must accept any Viper-supported format
- Provide example configuration files in YAML format (config.example.yaml)
- Do not auto-create configuration files

### FR2: Environment Variable Handling
- Adopt Viper's nested naming convention (PKG_TRACKER_CARRIERS_USPS_API_KEY)
- Automatically bind environment variables to configuration keys
- Use "PKG_TRACKER" as the environment prefix
- Environment variables override configuration file values
- Update all documentation to reflect new naming convention

### FR3: Configuration Structure
- Maintain existing configuration structs in internal/config/
- Use mapstructure tags for Viper unmarshaling
- Preserve type safety with proper struct definitions
- Keep validation logic at application level (not Viper validation)
- Maintain Config interface pattern for handlers

### FR4: CLI Configuration
- Migrate CLI from JSON config (~/.package-tracker.json) to YAML
- Support reading legacy JSON files for backward compatibility
- Integrate with existing Cobra command structure
- Maintain CLI flag precedence over config file

### FR5: Hot-Reloading
- Do not implement hot-reloading (as per requirements)
- Configuration changes require application restart

## Technical Requirements

### TR1: Viper Integration Architecture
- Create separate Viper instances for each application:
  - `internal/config/viper_server.go` - Server configuration
  - `internal/config/viper_email.go` - Email tracker configuration
  - `internal/config/viper_cli.go` - CLI configuration
- Replace helper functions in `internal/config/helpers.go` with Viper methods
- Maintain existing public APIs in config packages

### TR2: Configuration Loading
- Replace `LoadEnvFile()` with Viper's config file loading
- Update `Load()` methods to use Viper for parsing
- Implement custom decoders for Duration and other custom types
- Support configuration search paths:
  - Current directory
  - `./config/` directory
  - Home directory for CLI

### TR3: Type Conversion
- Replace `getEnvOrDefault()` helper functions with Viper's Get methods:
  - `viper.GetString()` for string values
  - `viper.GetBool()` for boolean values
  - `viper.GetInt()` for integer values
  - `viper.GetDuration()` for duration values
  - Custom unmarshal for complex types

### TR4: Backward Compatibility
- Support reading existing .env files through Viper
- Provide migration path for old environment variable names
- Display deprecation warnings for old variable names
- Document migration steps in README

### TR5: Configuration Validation
- Keep existing `validate()` methods on config structs
- Perform validation after Viper unmarshaling
- Return same error types for compatibility
- Validate required fields, ranges, and dependencies

## Implementation Details

### File Modifications Required

1. **New Files to Create:**
   - `internal/config/viper_server.go` - Server Viper setup
   - `internal/config/viper_email.go` - Email Viper setup
   - `internal/config/viper_cli.go` - CLI Viper setup
   - `config.example.yaml` - Example server config
   - `email-tracker.example.yaml` - Example email config
   - `cli.example.yaml` - Example CLI config

2. **Files to Modify:**
   - `internal/config/config.go` - Update Load() to use Viper
   - `internal/config/email_config.go` - Update LoadEmailConfig() to use Viper
   - `internal/cli/config.go` - Update LoadConfig() to use Viper
   - `cmd/email-tracker/cmd/root.go` - Update config file handling
   - `.env.example` - Update with new variable names
   - `README.md` - Update configuration documentation

3. **Files to Deprecate:**
   - `internal/config/helpers.go` - Move to legacy support only

### Configuration Mapping

Example environment variable mapping:
```
Old Format → New Format
SERVER_PORT → PKG_TRACKER_SERVER_PORT
USPS_API_KEY → PKG_TRACKER_CARRIERS_USPS_API_KEY
GMAIL_CLIENT_ID → PKG_TRACKER_EMAIL_GMAIL_CLIENT_ID
UPDATE_INTERVAL → PKG_TRACKER_UPDATE_INTERVAL
```

### Example YAML Configuration

```yaml
# config.example.yaml
server:
  host: localhost
  port: 8080
  
database:
  path: ./database.db
  
logging:
  level: info
  
carriers:
  usps:
    api_key: ""
  ups:
    client_id: ""
    client_secret: ""
  fedex:
    api_key: ""
    secret_key: ""
    api_url: "https://apis.fedex.com"
    
update:
  interval: 1h
  auto_enabled: true
  failure_threshold: 10
  
cache:
  ttl: 5m
  disabled: false
  
admin:
  api_key: ""
  auth_disabled: false
```

## Acceptance Criteria

1. ✓ All three applications use Viper for configuration management
2. ✓ Existing .env files continue to work without modification
3. ✓ New nested environment variable format is supported
4. ✓ Configuration can be loaded from YAML, TOML, JSON, or .env files
5. ✓ All existing tests pass with minimal modifications
6. ✓ Config interface pattern remains unchanged for handlers
7. ✓ Documentation is updated with new configuration format
8. ✓ Example configuration files are provided for all formats
9. ✓ CLI configuration migrates from JSON to YAML
10. ✓ Validation logic remains at application level

## Assumptions

1. Remote configuration (etcd, consul) is not needed
2. Hot-reloading is not required
3. Users will migrate to new environment variable format over time
4. Default configuration examples are sufficient (no auto-generation)
5. Performance impact of Viper is acceptable
6. Breaking changes to environment variables are acceptable with documentation

## Migration Steps for Users

1. Update environment variable names to new format
2. Optionally convert .env files to YAML/TOML format
3. Update deployment scripts with new variable names
4. Test configuration loading with new format
5. Remove deprecated .env files once migrated

## Testing Strategy

1. Unit tests for each Viper configuration module
2. Integration tests for configuration loading
3. Backward compatibility tests for old .env files
4. CLI flag precedence tests
5. Validation error tests
6. Multi-format loading tests