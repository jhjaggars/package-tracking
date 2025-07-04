# Requirements Specification: Email-Tracker Cobra Conversion with .env Support

## Problem Statement

The email-tracker application currently:
1. Does not support reading configuration from .env files (only environment variables)
2. Is not implemented as a Cobra CLI application, making it inconsistent with the main CLI tool
3. Has duplicate configuration helper functions across config packages
4. Lacks CLI-level overrides for testing (like --dry-run flag)

## Solution Overview

Convert email-tracker to a Cobra-based CLI application that:
- Reads configuration from .env files with proper precedence
- Follows the same Cobra/Fang patterns as the main CLI application
- Supports CLI flags for common overrides
- Maintains backward compatibility with existing environment variables
- Refactors common configuration helpers to eliminate duplication

## Functional Requirements

### FR1: Cobra CLI Structure
- Email-tracker shall be converted to use Cobra with Fang integration
- The application shall run directly when executed (no subcommands)
- Version information shall be accessible via --version flag
- Help information shall preserve existing printUsage() content

### FR2: Configuration Loading
- Email-tracker shall load configuration from .env file in the current directory
- Environment variables shall take precedence over .env file values
- CLI flags shall take precedence over environment variables
- All existing environment variable names shall remain unchanged

### FR3: CLI Flags
- **--config**: Specify alternative .env file location (e.g., --config=.env.test)
- **--dry-run**: Override EMAIL_DRY_RUN environment variable
- **--version**: Display version information
- **--help**: Display comprehensive help with configuration details

### FR4: Configuration Helper Refactoring
- Common helper functions shall be extracted to internal/config/helpers.go
- Both config.go and email_config.go shall use the shared helpers
- Email-specific helpers shall remain in email_config.go

## Technical Requirements

### TR1: File Structure
```
cmd/email-tracker/
├── main.go          # Simplified entry point calling cmd.Execute()
└── cmd/
    └── root.go      # Cobra root command with all logic
```

### TR2: Shared Configuration Helpers
Create `internal/config/helpers.go` containing:
- getEnvOrDefault
- getEnvBoolOrDefault
- getEnvIntOrDefault
- getEnvDurationOrDefault
- loadEnvFile (moved from config.go)

### TR3: Email Configuration Updates
Modify `internal/config/email_config.go`:
- Remove duplicate helper functions
- Import shared helpers from config package
- Add LoadEmailConfigWithEnvFile(envFile string) function
- Keep email-specific helpers (getEnvInt64OrDefault, getEnvFloatOrDefault, etc.)

### TR4: Cobra Implementation Pattern
Follow existing patterns from cmd/cli:
- Use cobra.Command with Use, Short, Long, Version fields
- Implement cobra.OnInitialize for configuration loading
- Use fang.Execute for enhanced error handling
- Return errors from RunE function

### TR5: Configuration Precedence
1. CLI flags (highest priority)
2. Environment variables
3. .env file values
4. Default values (lowest priority)

## Implementation Hints

### 1. Main.go Simplification
```go
package main

import "package-tracking/cmd/email-tracker/cmd"

func main() {
    cmd.Execute()
}
```

### 2. Root Command Structure
```go
var rootCmd = &cobra.Command{
    Use:     "email-tracker",
    Short:   "Email tracking service for package tracking system",
    Long:    `[existing printUsage content here]`,
    Version: Version,
    RunE:    runEmailTracker,
}
```

### 3. Configuration Loading Flow
```go
func initConfig() {
    // Load .env file if specified or default
    if configFile != "" {
        config.LoadEnvFile(configFile)
    } else {
        config.LoadEnvFile(".env")
    }
    
    // Load email configuration
    cfg, err = config.LoadEmailConfig()
    
    // Override with CLI flags
    if dryRun {
        cfg.Processing.DryRun = true
    }
}
```

### 4. Import Updates
- Add imports for cobra and fang packages
- Update import paths after moving helper functions

## Acceptance Criteria

1. **Cobra Conversion**
   - [ ] Email-tracker uses Cobra framework with Fang integration
   - [ ] Application runs directly without subcommands
   - [ ] --version and --help flags work correctly

2. **.env File Support**
   - [ ] Email-tracker reads from .env file in current directory
   - [ ] --config flag allows specifying alternative .env file
   - [ ] Environment variables override .env values
   - [ ] Existing configurations continue to work unchanged

3. **CLI Overrides**
   - [ ] --dry-run flag overrides EMAIL_DRY_RUN environment variable
   - [ ] CLI flags take precedence in configuration hierarchy

4. **Code Refactoring**
   - [ ] Common helpers moved to internal/config/helpers.go
   - [ ] No duplicate helper functions remain
   - [ ] Both config packages use shared helpers

5. **Backward Compatibility**
   - [ ] All existing environment variable names unchanged
   - [ ] Existing deployment configurations work without modification
   - [ ] Container deployments continue to function

6. **Documentation**
   - [ ] CLAUDE.md updated with new CLI structure
   - [ ] EMAIL_TRACKER_SETUP.md updated with new usage examples
   - [ ] Help text includes all configuration options

## Assumptions

1. The existing email-tracker functionality remains unchanged
2. go.mod already includes cobra and fang dependencies (from CLI usage)
3. Version information can be set at build time or hardcoded
4. The .env file format follows the same conventions as the server
5. No changes needed to the email processing logic itself