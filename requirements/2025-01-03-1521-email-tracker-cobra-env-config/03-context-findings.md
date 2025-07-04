# Context Findings

## Current State Analysis

### Email-Tracker Implementation
- **Location**: `cmd/email-tracker/main.go`
- **Current Design**: Standard Go application with main() function
- **Not using Cobra**: Currently a standalone application without CLI framework
- **Configuration**: Uses `LoadEmailConfig()` from `internal/config/email_config.go`
- **Environment Variables Only**: Reads configuration exclusively from environment variables
- **No .env Support**: Does not use the `loadEnvFile()` function available in the main config

### Configuration Loading Patterns

#### Main Server Configuration (`internal/config/config.go`)
- **Loads .env file**: Calls `loadEnvFile(".env")` at the start of `Load()`
- **Environment precedence**: Environment variables override .env file values
- **Helper functions**: Has standard helpers for parsing env vars

#### Email Configuration (`internal/config/email_config.go`)
- **No .env loading**: Only reads from environment variables
- **Duplicated helpers**: Contains duplicate implementations of helper functions
- **Additional helpers**: Has unique helpers for int64, float64, and slice parsing
- **Complex structure**: Nested configuration with 5 main sections (Gmail, Search, Processing, API, LLM)

### CLI Application Patterns (cmd/cli)

#### Cobra/Fang Structure
- **Root command**: Defines persistent flags and shared initialization
- **Subcommands**: Each in separate file with consistent patterns
- **Fang integration**: Used for enhanced error handling via `fang.Execute()`
- **Configuration hierarchy**: Flags > Environment > Config file > Defaults

#### Key Patterns to Follow
1. **Command structure**: Use, Short, Long, Args, RunE
2. **Flag naming**: Kebab-case with descriptive names
3. **Environment variables**: PREFIX_SNAKE_CASE format
4. **Error handling**: Return errors from RunE functions
5. **Progress indicators**: For long-running operations
6. **Output formatting**: Centralized through formatters

### Technical Constraints

1. **No external dependencies**: Email config uses only standard library
2. **Validation requirements**: Extensive validation logic that needs preservation
3. **Helper function duplication**: Should be refactored to avoid duplication
4. **Large number of config options**: ~40+ configuration parameters
5. **Backward compatibility**: Must maintain existing environment variable names

### Similar Features Analyzed

#### Server's .env Loading (`loadEnvFile`)
- Silent failure if file doesn't exist (intentional)
- Supports comments and empty lines
- Handles quoted values (single and double)
- Simple key=value format
- No variable expansion or complex features

#### CLI Configuration Loading
- Multiple sources with clear precedence
- Boolean environment variables handled specially
- Configuration file support (~/.package-tracker.json)
- Environment variable prefix for namespacing

### Integration Points Identified

1. **Shared configuration helpers**: Can be extracted to common package
2. **.env loading**: Can reuse existing `loadEnvFile` function
3. **Cobra patterns**: Well-established in CLI application
4. **Validation flow**: Can integrate with Cobra's PreRunE
5. **Help text**: Can leverage Cobra's built-in help generation

### Files That Need Modification

1. **Create new files**:
   - `cmd/email-tracker/cmd/root.go` - Root command definition
   - `cmd/email-tracker/cmd/run.go` - Main run command (moved from main.go)
   - `internal/config/helpers.go` - Shared helper functions

2. **Modify existing files**:
   - `cmd/email-tracker/main.go` - Simplified to just call cmd.Execute()
   - `internal/config/email_config.go` - Add .env loading support
   - `internal/config/config.go` - Extract shared helpers

3. **Update documentation**:
   - `CLAUDE.md` - Document new CLI structure and commands
   - `docs/EMAIL_TRACKER_SETUP.md` - Update usage instructions