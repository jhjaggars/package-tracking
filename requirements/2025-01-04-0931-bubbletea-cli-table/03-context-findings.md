# Context Findings

## Current CLI Architecture Analysis

### Data Flow Pattern
The current CLI follows a consistent pattern across all commands:
1. `initializeClient()` → returns `(config, formatter, client, error)`
2. `client.GetShipments()` → fetches `[]database.Shipment` from API
3. `formatter.PrintShipments(shipments)` → outputs formatted table

### Key Implementation Files

#### Core List Command
- **File**: `cmd/cli/cmd/list.go:32`
- **Current Logic**: Direct API call → formatter output
- **Integration Point**: `runList()` function needs conditional logic for interactive mode

#### Output Formatting System
- **File**: `internal/cli/output.go:114-131`
- **Current Method**: `PrintShipments()` with table/JSON format support
- **Styling**: Full `lipgloss` integration with color profiles and smart detection
- **Table Structure**: ID, TRACKING, CARRIER, STATUS, DESCRIPTION, CREATED
- **Color Coding**: Status-based coloring (delivered=green, in-transit=yellow, etc.)

#### BubbleTea Usage Pattern
- **File**: `internal/cli/progress.go:70-110`
- **Current Usage**: Simple spinner component for long operations
- **Pattern**: Implements full `tea.Model` interface with `Init()`, `Update()`, `View()`
- **Integration**: `tea.NewProgram(prog).Start()` pattern already established

### Available Operations for Interactive Mode

#### Individual Shipment Operations
1. **Refresh**: `client.RefreshShipmentWithForce(id, force)` - `/cmd/cli/cmd/refresh.go:54`
2. **Update**: `client.UpdateShipment(id, req)` - `/cmd/cli/cmd/update.go:43`
3. **Delete**: `client.DeleteShipment(id)` - `/cmd/cli/cmd/delete.go:32`
4. **View Details**: `client.GetShipment(id)` - Available for detailed view
5. **View Events**: `client.GetEvents(shipmentID)` - Available for event history

#### Operation Patterns
- **Progress Indication**: Spinner usage in refresh command `/cmd/cli/cmd/refresh.go:44-52`
- **Error Handling**: Consistent `formatter.PrintError(err)` usage
- **Success Feedback**: `formatter.PrintSuccess()` with descriptive messages
- **Quiet Mode**: All operations respect `config.Quiet` flag

### Data Model Structure

#### Shipment Fields Available
From `database.Shipment` struct:
- **Core Display**: ID, TrackingNumber, Carrier, Status, Description, CreatedAt
- **Extended Info**: UpdatedAt, ExpectedDelivery, IsDelivered
- **Refresh Tracking**: LastManualRefresh, AutoRefreshEnabled, AutoRefreshError
- **Metadata**: ManualRefreshCount, AutoRefreshCount, AutoRefreshFailCount

#### Status Values and Colors
- **Delivered**: Green (#10)
- **In Transit**: Yellow (#11) 
- **Pending**: Blue (#12)
- **Failed/Error**: Red (#9)
- **Unknown**: Gray (#8)

### BubbleTea Table Component Analysis

#### Available Table Features
- **Navigation**: Arrow keys, vim-style (j/k), page up/down, home/end
- **Selection**: `SelectedRow()` method for getting current selection
- **Customization**: `WithColumns()`, `WithRows()`, `WithHeight()`, `WithWidth()`
- **Styling**: `WithStyles()` for full visual customization
- **Key Mapping**: `WithKeyMap()` for custom keyboard shortcuts

#### Table Data Structure
```go
type Column struct {
    Title string
    Width int
}

type Row []string // Array of cell values
```

### Configuration Integration Points

#### Backward Compatibility Requirements
- **JSON Format**: `--format json` → skip interactive mode, use existing JSON output
- **Quiet Mode**: `--quiet` → skip interactive mode, use minimal output
- **No Color**: `--no-color` → disable styling in interactive mode
- **CI Detection**: Automatically disable interactive mode in CI environments

#### Interactive Mode Triggers
- **Default Behavior**: Interactive mode when no format flags specified
- **Explicit Flag**: Could add `--interactive` flag for explicit activation
- **TTY Detection**: Only enable interactive mode when stdout is a terminal

### Implementation Strategy

#### File Structure
- **New File**: `cmd/cli/cmd/interactive_table.go` - BubbleTea table implementation
- **Modified**: `cmd/cli/cmd/list.go` - Add conditional logic for interactive mode
- **Extended**: `internal/cli/output.go` - May need table styling integration

#### Key Integration Points
1. **Mode Detection**: Check for `--format`, `--quiet`, TTY status
2. **Data Passing**: Pass `[]database.Shipment` to interactive table
3. **Client Integration**: Pass API client for operations
4. **Style Integration**: Reuse existing `StyleConfig` and color detection

#### Operation Flow
```
List Command → Check Mode → Interactive Table → Selected Operation → API Call → Update Display
```

### Related Features and Patterns

#### Similar Interactive Components
- **Progress Spinner**: Pattern for long-running operations
- **Error Handling**: Consistent formatter-based error display
- **Configuration**: Viper-based config with environment variable support

#### Field Display Configuration
Based on Q4 answer about configurable fields:
- **Default Fields**: ID, TRACKING, CARRIER, STATUS, DESCRIPTION, CREATED
- **Additional Fields**: UpdatedAt, ExpectedDelivery, IsDelivered
- **Configuration**: Could use CLI flags or config file for field selection

### Technical Constraints

#### Environment Compatibility
- **NO_COLOR**: Must respect environment variable
- **CI Environments**: Must detect and disable interactive mode
- **Terminal Detection**: Must check TTY status
- **Piped Output**: Must disable interactive mode when output is piped

#### Performance Considerations
- **Static Snapshot**: No real-time updates (per Q5 answer)
- **Cache Integration**: Refresh operations use existing cache system
- **Error Handling**: Graceful handling of API failures during operations