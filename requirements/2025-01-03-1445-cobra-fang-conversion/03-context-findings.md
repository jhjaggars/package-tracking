# Context Findings

## Current Implementation Analysis

### Framework and Structure
- **Current Framework**: urfave/cli/v2
- **Main Entry**: `/cmd/cli/main.go` 
- **Core Components**:
  - `/internal/cli/client.go` - HTTP client implementation
  - `/internal/cli/config.go` - Configuration management
  - `/internal/cli/output.go` - Output formatting with color support
  - `/internal/cli/progress.go` - Progress spinner implementation

### Command Inventory
1. **add** - Create new shipment (flags: --tracking, --carrier, --description)
2. **list** - Display all shipments
3. **get** - Show shipment details by ID
4. **update** - Modify shipment description
5. **delete** - Remove shipment
6. **events** - View tracking history
7. **refresh** - Update tracking data (flags: --verbose, --force)

### Existing Charm Integration
The project already uses several Charm libraries:
- **lipgloss** v1.1.0 - For styling and colors
- **bubbletea** v1.3.5 - For interactive components
- **bubbles** v0.18.0 - For spinner component
- **termenv** v0.16.0 - For terminal detection

### Key Patterns to Preserve
1. **Three-tier configuration**: CLI flags > Environment variables > Config file
2. **Smart color detection**: Auto-disable for pipes, CI, NO_COLOR
3. **Output formats**: Table (default), JSON, Quiet mode
4. **Progress spinners**: For long operations like refresh
5. **Comprehensive error handling**: With colored output

### Files Requiring Modification

#### Primary files to rewrite:
- `/cmd/cli/main.go` - Complete rewrite for Cobra/Fang
- `/internal/cli/client.go` - Minor updates for new command context
- `/internal/cli/output.go` - Enhance with Fang styling
- `/internal/cli/config.go` - Adapt for Cobra's flag system

#### Test files to update:
- `/internal/cli/client_test.go`
- `/internal/cli/config_test.go`
- `/internal/cli/output_test.go`

### Migration Strategy Insights

1. **Command Migration Map**:
   - urfave `cli.Command` → Cobra `cobra.Command`
   - urfave `Action` → Cobra `RunE`
   - urfave flag definitions → Cobra persistent/local flags

2. **Context Handling**:
   - urfave `*cli.Context` → Cobra command args and flags
   - Global flags via `rootCmd.PersistentFlags()`

3. **Fang Benefits to Leverage**:
   - Styled help pages (automatic)
   - Built-in version command
   - Manpage generation
   - Enhanced shell completions
   - Themeable interface

### Technical Constraints
- Must maintain API client compatibility
- HTTP timeout of 180 seconds for SPA scraping
- SQLite database interaction patterns unchanged
- REST API endpoints remain the same

### Opportunities for Enhancement
1. **Better command organization**: Cobra's command tree structure
2. **Improved help**: Fang's styled help pages
3. **Native completions**: Cobra's completion generation
4. **Consistent styling**: Fang's themeable interface
5. **Simplified main.go**: Fang handles boilerplate

### Related Features Analyzed
- Current progress spinner uses bubbletea - can be enhanced with Fang
- Color output uses lipgloss - integrates well with Fang's theming
- Configuration loading is well-structured - easily portable to Cobra

### Integration Points
- HTTP client remains unchanged
- API endpoints stay the same
- Database models not affected
- Only CLI layer needs modification