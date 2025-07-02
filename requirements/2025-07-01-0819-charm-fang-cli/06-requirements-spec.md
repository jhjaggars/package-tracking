# Requirements Specification - Charm Fang CLI Enhancement

**Project:** Package Tracking CLI
**Feature:** Charm Fang Library Integration
**Date:** 2025-07-01
**Status:** Complete

## Problem Statement

The current CLI interface for the package tracking system is functional but lacks modern terminal UI features. Users interact with plain text output that doesn't leverage color coding, progress indicators, or other visual enhancements that could improve usability and user experience. The goal is to enhance the CLI with the Charm ecosystem tools while maintaining complete backward compatibility.

## Solution Overview

Implement a phased enhancement of the CLI using Charm's styling libraries (lipgloss, termenv) to add colors, styled output, and progress indicators. The initial phase will enhance the existing urfave/cli implementation without breaking changes. A future phase may consider full migration to Cobra/Fang for additional features.

## Functional Requirements

### 1. Visual Enhancements
- **Color-coded statuses**: Each package status must have a distinct color
  - `delivered`: Green
  - `in-transit`: Yellow
  - `pending`: Blue
  - `failed`/`error`: Red
  - `unknown`: Gray
- **Styled messages**: Enhance success (✓), error (✗), and info (ℹ) messages with appropriate colors
- **Table formatting**: Improve table borders and alignment while maintaining current column structure
- **Progress indicators**: Add spinners/progress bars for long operations (refresh, API calls)

### 2. Backward Compatibility
- **Output formats**: Maintain exact same table and JSON formats when colors are disabled
- **Command structure**: No changes to command names, arguments, or flags
- **Quiet mode**: Preserve current quiet mode behavior
- **Script compatibility**: Ensure output remains parseable by existing scripts

### 3. Smart Output Detection
- **TTY detection**: Automatically disable styling when output is piped or redirected
- **Environment respect**: Honor `NO_COLOR` environment variable
- **CI/CD compatibility**: Detect and disable styling in continuous integration environments
- **Terminal capability**: Check terminal color support before applying styles

### 4. User Control
- **Global flag**: Add `--no-color` flag to explicitly disable all styling
- **Configuration**: Add color preference to configuration file (`~/.package-tracker.json`)
- **Environment variable**: Support `PACKAGE_TRACKER_NO_COLOR` environment variable
- **Flag precedence**: CLI flag > environment variable > config file > auto-detection

### 5. Enhanced Feedback
- **Operation progress**: Show visual feedback during long operations
  - Spinner for indeterminate operations (API calls)
  - Progress bar for operations with known duration
  - Elapsed time indicator for refresh operations
- **Better errors**: Style error messages with context and suggestions
- **Command help**: Enhanced help text with examples (future phase with Fang)

## Technical Requirements

### Phase 1: Enhance Current Implementation

#### 1. Dependencies to Add
```go
// go.mod additions
github.com/charmbracelet/lipgloss v0.9.1
github.com/muesli/termenv v0.15.2
github.com/mattn/go-isatty v0.0.20
github.com/charmbracelet/bubbles v0.18.0
```

#### 2. Files to Modify

**`internal/cli/output.go`**
- Add `StyleConfig` struct for color configuration
- Enhance `OutputFormatter` with style support
- Add `isColorEnabled()` method for detection logic
- Update all Print methods to use lipgloss styles
- Maintain backward compatibility when colors disabled

**`cmd/cli/main.go`**
- Add `--no-color` global flag
- Initialize color detection on startup
- Pass color preference to OutputFormatter

**`internal/cli/config.go`**
- Add `NoColor` field to Config struct
- Support `PACKAGE_TRACKER_NO_COLOR` environment variable
- Add color preference to JSON config

**`go.mod`**
- Add Charm dependencies
- Run `go mod tidy`

#### 3. Implementation Patterns

```go
// Color detection logic
func (f *OutputFormatter) isColorEnabled() bool {
    // Priority order:
    // 1. Explicit --no-color flag
    // 2. NO_COLOR environment variable
    // 3. PACKAGE_TRACKER_NO_COLOR environment variable
    // 4. Config file setting
    // 5. TTY detection
    // 6. CI environment detection
}

// Status color mapping
var statusColors = map[string]lipgloss.Color{
    "delivered":  lipgloss.Color("10"), // Bright green
    "in-transit": lipgloss.Color("11"), // Bright yellow
    "pending":    lipgloss.Color("12"), // Bright blue
    "failed":     lipgloss.Color("9"),  // Bright red
    "unknown":    lipgloss.Color("8"),  // Gray
}
```

### Phase 2: Future Cobra/Fang Migration (Not in initial scope)

- Create parallel implementation in `cmd/cli-v2/`
- Full Cobra command structure
- Fang wrapper for enhanced help/errors
- Maintain feature parity with Phase 1

## Acceptance Criteria

### Phase 1 Completion
- [ ] Colors appear in interactive terminal sessions
- [ ] Colors automatically disabled when piping output
- [ ] `--no-color` flag disables all styling
- [ ] JSON output remains unchanged
- [ ] Table structure remains identical (only colors added)
- [ ] Progress spinners appear for refresh operations
- [ ] Status fields show appropriate colors
- [ ] All existing tests pass
- [ ] Documentation updated with new flags

### Testing Requirements
- [ ] Unit tests for color detection logic
- [ ] Integration tests with various terminal environments
- [ ] Script compatibility tests (pipe, redirect)
- [ ] CI/CD environment tests
- [ ] Cross-platform testing (macOS, Linux, Windows)

## Assumptions

1. Users want visual enhancements but value stability over features
2. Most users run the CLI in modern terminals with color support
3. The phased approach is preferred over a big-bang migration
4. Progress indicators are wanted only for operations > 1 second
5. The current 180-second timeout for refresh is appropriate

## Out of Scope

- Full migration to Cobra/Fang (future phase)
- Interactive table navigation
- Mouse support
- Custom themes
- Emoji additions (unless in future phase)
- Web UI integration

## Success Metrics

- Zero breaking changes for existing users
- Improved user satisfaction with visual feedback
- Faster issue identification through color coding
- Maintained script compatibility
- Positive user feedback on enhanced experience