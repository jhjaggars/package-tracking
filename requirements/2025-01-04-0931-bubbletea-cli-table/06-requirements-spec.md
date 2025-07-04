# Requirements Specification: Interactive BubbleTea CLI Table

## Problem Statement

The current CLI `list` command outputs a static table that requires users to copy shipment IDs and run separate commands for operations like refresh, update, or delete. This creates a fragmented user experience where users must:

1. Run `list` to see shipments
2. Copy an ID from the output
3. Run separate commands like `refresh <id>`, `update <id>`, or `delete <id>`
4. Return to step 1 to see updated results

## Solution Overview

Convert the `list` command to use an interactive BubbleTea-powered table that allows users to:
- Navigate shipments with keyboard controls
- Perform operations directly on selected shipments
- Maintain a persistent interface for efficient workflow
- Preserve full backward compatibility with existing scripts

## Functional Requirements

### FR1: Interactive Mode Activation
- **Trigger**: Interactive mode activates by default when:
  - No `--format` flag is specified
  - No `--quiet` flag is specified
  - stdout is a TTY (terminal)
- **Fallback**: Use existing static table for scripts/pipes
- **Override**: Add `--interactive` flag for explicit activation

### FR2: Table Navigation
- **Keyboard Support**: 
  - Arrow keys (↑/↓) for row navigation
  - Vim-style keys (j/k) for up/down movement
  - Page Up/Down for page navigation
  - Home/End for first/last row
- **Visual Feedback**: Highlight selected row with cursor
- **Scrolling**: Support scrolling for tables with many rows

### FR3: Shipment Operations
- **Refresh** (r key): Refresh tracking data for selected shipment
  - Show progress spinner during operation
  - Update table with new status/events
  - Display success/error messages
- **Update** (u key): Update description of selected shipment
  - Prompt for new description
  - Update table with new description
  - Show confirmation message
- **Delete** (d key): Delete selected shipment
  - Show confirmation dialog
  - Remove from table upon confirmation
  - Show deletion confirmation
- **View Details** (Enter key): Show detailed shipment information
  - Display all shipment fields
  - Show in overlay or separate view
- **View Events** (e key): Show tracking events for selected shipment
  - Display event history
  - Show in overlay or separate view

### FR4: Display Configuration
- **Default Fields**: ID, TRACKING, CARRIER, STATUS, DESCRIPTION, CREATED
- **Field Selection**: Support `--fields` flag for custom field display
  - Format: `--fields=id,tracking,status,description`
  - Available fields: id, tracking, carrier, status, description, created, updated, delivery, delivered
- **Column Sizing**: Automatic column width adjustment based on content
- **Status Coloring**: Apply existing color scheme (delivered=green, in-transit=yellow, etc.)

### FR5: Help and Navigation
- **Help Display** (? key): Show keyboard shortcuts and available operations
- **Quit Operation** (q key or Ctrl+C): Exit interactive mode cleanly
- **Status Line**: Show current selection and available operations

## Technical Requirements

### TR1: Integration with Existing CLI
- **File Location**: Implement in new file `cmd/cli/cmd/interactive_table.go`
- **List Command**: Modify `cmd/cli/cmd/list.go:32` to add conditional logic
- **Client Reuse**: Use existing `*cliapi.Client` for all API operations
- **Formatter Integration**: Leverage existing `*cliapi.OutputFormatter` for messages

### TR2: BubbleTea Implementation
- **Component**: Use `github.com/charmbracelet/bubbles/table` component
- **Model Structure**: Implement `tea.Model` interface with:
  ```go
  type InteractiveTable struct {
      table     table.Model
      shipments []database.Shipment
      client    *cliapi.Client
      formatter *cliapi.OutputFormatter
      // ... other fields
  }
  ```
- **Key Handling**: Custom `KeyMap` for operation shortcuts
- **State Management**: Handle loading states, errors, and updates

### TR3: Styling Integration
- **Color System**: Reuse existing `StyleConfig` from `internal/cli/output.go`
- **Environment Detection**: Use existing `shouldUseColor()` logic
- **Status Colors**: Apply existing status color mapping
- **Consistency**: Match visual styling with current table output

### TR4: Error Handling
- **API Errors**: Display errors using existing `formatter.PrintError()` pattern
- **Network Issues**: Handle connection failures gracefully
- **Invalid Operations**: Prevent operations on invalid selections
- **Recovery**: Allow continuation after errors

### TR5: Performance Considerations
- **Static Data**: Work with snapshot data (no real-time updates)
- **Efficient Updates**: Update only modified rows after operations
- **Memory Management**: Proper cleanup of resources
- **Responsive UI**: Non-blocking operations where possible

## Implementation Hints

### File Modifications Required

#### 1. cmd/cli/cmd/list.go
```go
// Add interactive flag
var interactiveMode bool

func init() {
    // Add flag
    listCmd.Flags().BoolVarP(&interactiveMode, "interactive", "i", false, "Interactive table mode")
    listCmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated list of fields to display")
}

func runList(cmd *cobra.Command, args []string) error {
    // ... existing setup ...
    
    // Determine if interactive mode should be used
    if shouldUseInteractiveMode(config, interactiveMode) {
        return runInteractiveTable(shipments, client, formatter, fieldsFlag)
    }
    
    return formatter.PrintShipments(shipments)
}

func shouldUseInteractiveMode(config *cliapi.Config, explicit bool) bool {
    // Interactive mode when:
    // - Explicitly requested, OR
    // - No format flags AND stdout is TTY AND not quiet mode
    return explicit || (config.Format == "table" && !config.Quiet && isatty.IsTerminal(os.Stdout.Fd()))
}
```

#### 2. cmd/cli/cmd/interactive_table.go (new file)
```go
type InteractiveTable struct {
    table       table.Model
    shipments   []database.Shipment
    client      *cliapi.Client
    formatter   *cliapi.OutputFormatter
    fields      []string
    keys        KeyMap
    loading     bool
    spinner     spinner.Model
    err         error
    message     string
    quitting    bool
}

func (m InteractiveTable) Init() tea.Cmd
func (m InteractiveTable) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m InteractiveTable) View() string
```

#### 3. Key Bindings
```go
type KeyMap struct {
    Refresh  key.Binding
    Update   key.Binding
    Delete   key.Binding
    Details  key.Binding
    Events   key.Binding
    Help     key.Binding
    Quit     key.Binding
}

var DefaultKeyMap = KeyMap{
    Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
    Update:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update")),
    Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
    Details:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
    Events:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "events")),
    Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
    Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}
```

### Field Configuration
```go
var defaultFields = []string{"id", "tracking", "carrier", "status", "description", "created"}
var availableFields = map[string]string{
    "id":          "ID",
    "tracking":    "TRACKING", 
    "carrier":     "CARRIER",
    "status":      "STATUS",
    "description": "DESCRIPTION",
    "created":     "CREATED",
    "updated":     "UPDATED",
    "delivery":    "DELIVERY",
    "delivered":   "DELIVERED",
}

func parseFields(fieldsFlag string) []string {
    if fieldsFlag == "" {
        return defaultFields
    }
    return strings.Split(fieldsFlag, ",")
}
```

### Operation Patterns
```go
func (m InteractiveTable) handleRefresh() tea.Cmd {
    selectedRow := m.table.SelectedRow()
    if len(selectedRow) == 0 {
        return nil
    }
    
    id := selectedRow[0] // Assuming ID is first column
    return tea.Batch(
        m.showSpinner("Refreshing..."),
        m.refreshShipment(id),
    )
}

func (m InteractiveTable) refreshShipment(id string) tea.Cmd {
    return tea.Cmd(func() tea.Msg {
        // Use existing client method
        response, err := m.client.RefreshShipmentWithForce(id, false)
        return RefreshCompleteMsg{Response: response, Error: err}
    })
}
```

## Acceptance Criteria

### AC1: Backward Compatibility
- [ ] `--format json` outputs JSON without interactive mode
- [ ] `--quiet` mode outputs minimal text without interactive mode
- [ ] Piped output (`./cli list | grep something`) works as before
- [ ] CI environments automatically disable interactive mode

### AC2: Navigation and Selection
- [ ] Arrow keys navigate table rows
- [ ] Vim keys (j/k) navigate table rows
- [ ] Page Up/Down navigate by pages
- [ ] Home/End navigate to first/last row
- [ ] Selected row is visually highlighted

### AC3: Operations
- [ ] 'r' key refreshes selected shipment with progress indicator
- [ ] 'u' key prompts for description update
- [ ] 'd' key shows confirmation dialog for deletion
- [ ] 'Enter' key shows detailed shipment information
- [ ] 'e' key shows tracking events
- [ ] '?' key shows help with available operations
- [ ] 'q' or Ctrl+C exits cleanly

### AC4: Display Configuration
- [ ] Default display matches current table format
- [ ] `--fields` flag allows field selection
- [ ] Status colors match existing color scheme
- [ ] Color detection respects NO_COLOR and CI environments
- [ ] Field validation prevents invalid field names

### AC5: Error Handling
- [ ] API errors are displayed clearly
- [ ] Network failures don't crash the interface
- [ ] Invalid operations show appropriate messages
- [ ] User can continue after errors

### AC6: Performance
- [ ] Table loads quickly with reasonable number of shipments
- [ ] Operations complete without blocking the UI
- [ ] Memory usage is reasonable
- [ ] Graceful handling of large datasets

## Assumptions

1. **Table Size**: Interactive mode will handle up to 1000 shipments reasonably
2. **Network**: Operations may have network latency; progress indicators will be shown
3. **Terminal**: Users have terminals that support ANSI colors and key input
4. **Compatibility**: BubbleTea v1.3.5 API remains stable
5. **Usage**: Interactive mode is primarily for human users, not scripts
6. **Operations**: All operations use existing API client methods without modification