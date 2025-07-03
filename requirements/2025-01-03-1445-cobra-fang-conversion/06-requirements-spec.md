# Requirements Specification: CLI Conversion to Cobra/Fang

## Problem Statement
The current CLI implementation uses urfave/cli/v2 framework. The goal is to completely convert it to use Cobra with Fang enhancement from Charm.sh, providing a more modern and feature-rich CLI experience.

## Solution Overview
Migrate the entire CLI from urfave/cli to Cobra/Fang while:
- Leveraging Fang's batteries-included features for better UX
- Maintaining all current functionality
- Enhancing the user experience with Charm's styling capabilities
- Adding new features like shell completions and smart error messages
- No requirement for backward compatibility

## Functional Requirements

### Command Structure
1. **Reorganize commands** following Cobra best practices:
   - Create separate files for each command in `/cmd/cli/cmd/` directory
   - Implement root command with global flags
   - Maintain current command names but allow flexibility in structure

2. **Commands to implement**:
   - `add` - Create new shipment
   - `list` - Display all shipments  
   - `get` - Show shipment details
   - `update` - Modify shipment
   - `delete` - Remove shipment
   - `events` - View tracking history
   - `refresh` - Update tracking data
   - `completion` - Generate shell completions
   - `version` - Show version (automatic with Fang)

### Enhanced Features
1. **Progress Indicators**:
   - Enhance refresh command with detailed progress using Charm components
   - Show steps, elapsed time, and current operation
   - Graceful degradation in non-TTY environments

2. **Shell Completions**:
   - Auto-generate completions for bash, zsh, fish, PowerShell
   - Include completion command for easy installation

3. **Error Handling**:
   - Enable Cobra's suggestion system ("did you mean...?")
   - Style errors with Fang's theming
   - Provide helpful error messages

4. **Help System**:
   - Leverage Fang's styled help pages
   - Include examples for each command
   - Generate man pages with Fang's Mango

## Technical Requirements

### Dependencies
1. **Add dependencies**:
   - `github.com/spf13/cobra` (latest version)
   - Update `github.com/charmbracelet/fang` to latest

2. **Keep existing**:
   - All current Charm libraries (lipgloss, bubbletea, bubbles)
   - HTTP client implementation
   - Configuration management logic

### File Structure
```
cmd/cli/
├── main.go              # Entry point with Fang.Execute
├── cmd/
│   ├── root.go         # Root command and global flags
│   ├── add.go          # Add command implementation
│   ├── list.go         # List command
│   ├── get.go          # Get command
│   ├── update.go       # Update command
│   ├── delete.go       # Delete command
│   ├── events.go       # Events command
│   ├── refresh.go      # Refresh command with enhanced progress
│   └── completion.go   # Shell completion generation
```

### Implementation Patterns
1. **Command Creation**:
   ```go
   var addCmd = &cobra.Command{
       Use:   "add",
       Short: "Add a new shipment",
       RunE:  runAdd,
   }
   ```

2. **Flag Management**:
   - Use Cobra's flag system
   - Maintain current flag names
   - Global flags on root command

3. **Configuration**:
   - Adapt current config loading to Cobra's PreRun
   - Maintain three-tier precedence
   - Remove legacy config file support

4. **Output Formatting**:
   - Enhance with Fang's theming
   - Maintain table/JSON/quiet modes
   - Keep color detection logic

## Implementation Hints

### Specific Patterns to Follow
1. **Use Cobra's context pattern**:
   ```go
   func runAdd(cmd *cobra.Command, args []string) error {
       // Implementation
   }
   ```

2. **Global flag initialization**:
   ```go
   func init() {
       rootCmd.PersistentFlags().StringP("server", "s", "http://localhost:8080", "API server")
   }
   ```

3. **Fang integration**:
   ```go
   func main() {
       if err := fang.Execute(context.Background(), rootCmd); err != nil {
           os.Exit(1)
       }
   }
   ```

### Files to Modify
1. **Complete rewrite**:
   - `/cmd/cli/main.go` → New Fang-based entry
   - Create all files in `/cmd/cli/cmd/`

2. **Minor updates**:
   - `/internal/cli/client.go` - Adjust for new context
   - `/internal/cli/output.go` - Enhance with Fang themes
   - `/internal/cli/config.go` - Adapt flag parsing

3. **Update tests**:
   - Adapt all CLI tests for Cobra patterns
   - Test completion generation
   - Test suggestion system

## Acceptance Criteria
1. ✓ All current commands work with same functionality
2. ✓ Shell completions generate correctly for all shells
3. ✓ Progress indicators show enhanced information
4. ✓ Error messages include suggestions for typos
5. ✓ Help pages are styled with Fang
6. ✓ All tests pass with new implementation
7. ✓ Version command works automatically
8. ✓ Man pages can be generated
9. ✓ No backward compatibility issues (not required)
10. ✓ Code follows Cobra best practices

## Assumptions
1. **Command names** - Can be restructured since no backward compatibility needed
2. **Config file** - Legacy ~/.package-tracker.json support can be removed
3. **Output formats** - Current formats preserved but can be enhanced
4. **Interactive mode** - Not needed per requirements
5. **Global flags** - Will use Cobra's persistent flag system