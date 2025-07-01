# Context Findings - Charm Fang CLI Integration

**Date:** 2025-07-01
**Phase:** Targeted Context Gathering

## Current CLI Implementation Analysis

### Architecture Overview
- **Framework**: urfave/cli/v2 (not Cobra)
- **Entry Point**: `cmd/cli/main.go`
- **Output Handling**: `internal/cli/output.go` with OutputFormatter abstraction
- **Configuration**: Three-tier system (file → env → flags)
- **Commands**: add, list, get, update, delete, events, refresh

### Key Files Requiring Modification
1. `cmd/cli/main.go` - Complete rewrite for Cobra/Fang
2. `internal/cli/output.go` - Enhance with Charm styling libraries
3. `internal/cli/config.go` - Minimal changes, mostly compatible
4. `internal/cli/client.go` - No changes needed
5. `go.mod` - Add Charm dependencies

### Current Output Patterns
- Table format using `text/tabwriter`
- JSON format for programmatic use
- Unicode symbols: ✓ (success), ✗ (error), ℹ (info)
- Truncation of long strings (15 chars for tracking numbers)
- Quiet mode for minimal output

### Charm Fang Requirements & Implications

#### Major Framework Change Required
- **Current**: urfave/cli/v2
- **Required**: spf13/cobra (Fang is built on Cobra)
- **Impact**: Complete rewrite of command definitions

#### Fang Features to Leverage
1. Styled help pages and errors
2. Automatic version handling
3. Man page generation
4. Shell completions
5. Theming capabilities
6. Better error UX (silent usage after errors)

#### Backward Compatibility Challenges
1. **Command Structure**: Must maintain exact same command interface
2. **Output Format**: Table and JSON must remain identical
3. **Environment Detection**: Need to add terminal capability detection
4. **Scripts**: Existing automation must continue working

### Integration Strategy

#### Phase 1: Enhance Current CLI (Without Framework Change)
- Add `github.com/charmbracelet/lipgloss` for styling
- Add `github.com/muesli/termenv` for color detection
- Enhance OutputFormatter with color support
- Keep urfave/cli for now
- Estimated effort: Low

#### Phase 2: Full Fang Migration
- Rewrite all commands using Cobra structure
- Wrap with Fang for enhanced UX
- Maintain backward compatibility flags
- Add theme configuration
- Estimated effort: High

#### Phase 3: Interactive Features
- Add `github.com/charmbracelet/bubbles/progress` for refresh operations
- Add `github.com/charmbracelet/bubbles/spinner` for API calls
- Add interactive table selection for list command
- Estimated effort: Medium

### Technical Constraints Found

1. **Framework Lock-in**: Fang requires Cobra, no way around it
2. **Testing**: Need comprehensive tests to ensure backward compatibility
3. **Environment Detection**: Must handle:
   - `NO_COLOR` environment variable
   - Dumb terminals
   - CI/CD environments (GitHub Actions, Jenkins)
   - Piped output detection

### Similar Features in Codebase
- None found - this would be the first terminal UI enhancement
- Current focus is on simplicity and portability

### Best Practices from Charm Ecosystem

1. **Graceful Degradation**:
   ```go
   if !termenv.HasDarkBackground() {
       // Use light theme
   }
   if os.Getenv("NO_COLOR") != "" {
       // Disable all styling
   }
   ```

2. **Output Detection**:
   ```go
   if !isatty.IsTerminal(os.Stdout.Fd()) {
       // Plain output for pipes
   }
   ```

3. **Progress Feedback**:
   - Use spinners for indeterminate operations
   - Use progress bars when duration is known
   - Always provide fallback text output

### Recommended Approach

Given the significant framework change required and backward compatibility needs, I recommend:

1. **Immediate**: Enhance current CLI with Charm styling libraries (lipgloss, termenv) without changing framework
2. **Future**: Create a parallel `cmd/cli-v2/` implementation using Fang
3. **Migration**: Provide transition period with both CLIs available
4. **Documentation**: Clear migration guide for users

This approach minimizes risk while providing immediate visual improvements.