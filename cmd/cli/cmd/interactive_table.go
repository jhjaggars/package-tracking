package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"

	cliapi "package-tracking/internal/cli"
	"package-tracking/internal/database"
)

// KeyMap represents the key bindings for the interactive table
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Refresh  key.Binding
	Update   key.Binding
	Delete   key.Binding
	Details  key.Binding
	Events   key.Binding
	Help     key.Binding
	Quit     key.Binding
	Confirm  key.Binding
	Cancel   key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Update: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "update"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Details: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "details"),
		),
		Events: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "events"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y", "Y"),
			key.WithHelp("y", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n", "N", "esc"),
			key.WithHelp("n/esc", "cancel"),
		),
	}
}

// InteractiveTable represents the interactive table model
type InteractiveTable struct {
	table             table.Model
	shipments         []database.Shipment
	client            *cliapi.Client
	formatter         *cliapi.OutputFormatter
	fields            []string
	keys              KeyMap
	loading           bool
	spinner           spinner.Model
	err               error
	message           string
	showHelp          bool
	quitting          bool
	config            *cliapi.Config
	useColor          bool
	showDeleteConfirm bool
	deleteTarget      int // ID of shipment to delete
	showEvents        bool
	eventsData        []database.TrackingEvent
	eventsShipmentID  int
	eventsScroll      int
}

// NewInteractiveTable creates a new interactive table
func NewInteractiveTable(shipments []database.Shipment, client *cliapi.Client, formatter *cliapi.OutputFormatter, fieldsFlag string, config *cliapi.Config) (*InteractiveTable, error) {
	// Parse and validate fields
	fields := parseFields(fieldsFlag)
	if err := validateFields(fields); err != nil {
		return nil, err
	}

	// Create table columns
	columns := make([]table.Column, len(fields))
	for i, field := range fields {
		columns[i] = table.Column{
			Title: getFieldDisplayName(field),
			Width: calculateColumnWidth(field, shipments),
		}
	}

	// Create table rows
	rows := make([]table.Row, len(shipments))
	for i, shipment := range shipments {
		rows[i] = shipmentToRow(shipment, fields)
	}

	// Create table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Determine if colors should be used
	useColor := !config.NoColor && isatty.IsTerminal(os.Stdout.Fd())

	// Apply styling
	if useColor {
		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(false)
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(false)
		t.SetStyles(s)
	}

	return &InteractiveTable{
		table:     t,
		shipments: shipments,
		client:    client,
		formatter: formatter,
		fields:    fields,
		keys:      DefaultKeyMap(),
		spinner:   s,
		config:    config,
		useColor:  useColor,
	}, nil
}

// Init initializes the interactive table
func (m InteractiveTable) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m InteractiveTable) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle confirmation dialog first
		if m.showDeleteConfirm {
			switch {
			case key.Matches(msg, m.keys.Confirm):
				return m.confirmDelete()
			case key.Matches(msg, m.keys.Cancel):
				m.showDeleteConfirm = false
				m.deleteTarget = 0
				m.message = "Delete cancelled"
				return m, nil
			}
			// Don't process other keys when in confirmation mode
			return m, nil
		}

		// Handle events view navigation
		if m.showEvents {
			switch {
			case key.Matches(msg, m.keys.Up):
				if m.eventsScroll > 0 {
					m.eventsScroll--
				}
				return m, nil
			case key.Matches(msg, m.keys.Down):
				maxScroll := len(m.eventsData) - 10 // Show 10 events at a time
				if maxScroll < 0 {
					maxScroll = 0
				}
				if m.eventsScroll < maxScroll {
					m.eventsScroll++
				}
				return m, nil
			case key.Matches(msg, m.keys.Cancel), key.Matches(msg, m.keys.Quit):
				// Close events view
				m.showEvents = false
				m.eventsData = nil
				m.eventsShipmentID = 0
				m.eventsScroll = 0
				m.message = ""
				return m, nil
			}
			// Don't process other keys when in events view
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			return m.handleRefresh()

		case key.Matches(msg, m.keys.Up):
			m.table, cmd = m.table.Update(msg)
			return m, cmd

		case key.Matches(msg, m.keys.Down):
			m.table, cmd = m.table.Update(msg)
			return m, cmd

		case key.Matches(msg, m.keys.Details):
			return m.handleDetails()

		case key.Matches(msg, m.keys.Events):
			return m.handleEvents()

		case key.Matches(msg, m.keys.Update):
			return m.handleUpdateDescription()

		case key.Matches(msg, m.keys.Delete):
			return m.handleDelete()
		}

	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width)
		return m, nil

	case refreshCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.message = fmt.Sprintf("Error refreshing shipment: %v", msg.err)
		} else {
			m.message = fmt.Sprintf("Refreshed successfully - %d events added", msg.response.EventsAdded)
			// We need to fetch the updated shipment data since refresh response doesn't include it
			// For now, just show the success message
		}
		return m, nil

	case deleteCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.message = fmt.Sprintf("Error deleting shipment: %v", msg.err)
		} else {
			// Remove the deleted shipment from the table
			m = m.removeShipmentFromTable(msg.shipmentID)
			m.message = "Shipment deleted successfully"
		}
		return m, nil

	case eventsCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.message = fmt.Sprintf("Error fetching events: %v", msg.err)
		} else {
			// Show the events view
			m.showEvents = true
			m.eventsData = msg.events
			m.eventsShipmentID = msg.shipmentID
			m.eventsScroll = 0
			m.message = ""
			m.err = nil
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// View renders the interactive table
func (m InteractiveTable) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var b strings.Builder

	// Show help if requested
	if m.showHelp {
		b.WriteString(m.helpView())
		b.WriteString("\n")
	}

	// Show spinner if loading
	if m.loading {
		b.WriteString(fmt.Sprintf("%s Loading...\n", m.spinner.View()))
	}

	// Show events view if active
	if m.showEvents {
		b.WriteString(m.eventsView())
		b.WriteString("\n")
	} else {
		// Show table
		b.WriteString(m.table.View())
		b.WriteString("\n")
	}

	// Show confirmation dialog if needed
	if m.showDeleteConfirm {
		confirmMsg := fmt.Sprintf("Delete shipment ID %d? (y/N): ", m.deleteTarget)
		if m.useColor {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render(confirmMsg))
		} else {
			b.WriteString(confirmMsg)
		}
		b.WriteString("\n")
	}

	// Show message if any
	if m.message != "" {
		if m.err != nil {
			if m.useColor {
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(m.message))
			} else {
				b.WriteString(m.message)
			}
		} else {
			if m.useColor {
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render(m.message))
			} else {
				b.WriteString(m.message)
			}
		}
		b.WriteString("\n")
	}

	// Show status line
	b.WriteString(m.statusLine())

	return b.String()
}

// helpView returns the help view
func (m InteractiveTable) helpView() string {
	help := strings.Builder{}
	help.WriteString("Help:\n")
	help.WriteString("  ↑/k         - Move up\n")
	help.WriteString("  ↓/j         - Move down\n")
	help.WriteString("  r           - Refresh selected shipment\n")
	help.WriteString("  u           - Update description\n")
	help.WriteString("  d           - Delete shipment\n")
	help.WriteString("  enter       - View details\n")
	help.WriteString("  e           - View events\n")
	help.WriteString("  ?           - Toggle help\n")
	help.WriteString("  q/ctrl+c    - Quit\n")
	return help.String()
}

// statusLine returns the status line
func (m InteractiveTable) statusLine() string {
	if m.showEvents {
		return "Events View | Press q/esc to return to shipments list"
	}
	
	if len(m.shipments) == 0 {
		return "No shipments found"
	}

	selected := m.table.Cursor()
	total := len(m.shipments)
	return fmt.Sprintf("Shipment %d of %d | Press ? for help", selected+1, total)
}

// calculateColumnWidth calculates the width for a column based on its content
func calculateColumnWidth(field string, shipments []database.Shipment) int {
	// Base width on field display name
	width := len(getFieldDisplayName(field))

	// Check a few sample rows to determine appropriate width
	samples := len(shipments)
	if samples > 10 {
		samples = 10
	}

	for i := 0; i < samples; i++ {
		value := getFieldValue(shipments[i], field)
		if len(value) > width {
			width = len(value)
		}
	}

	// Set reasonable limits
	if width < 8 {
		width = 8
	}
	if width > 50 {
		width = 50
	}

	return width
}

// shipmentToRow converts a shipment to a table row
func shipmentToRow(shipment database.Shipment, fields []string) table.Row {
	row := make(table.Row, len(fields))
	for i, field := range fields {
		row[i] = getFieldValue(shipment, field)
	}
	return row
}

// getFieldValue returns the value for a specific field from a shipment
func getFieldValue(shipment database.Shipment, field string) string {
	switch field {
	case "id":
		return strconv.Itoa(shipment.ID)
	case "tracking":
		return shipment.TrackingNumber
	case "carrier":
		return shipment.Carrier
	case "status":
		return shipment.Status
	case "description":
		return shipment.Description
	case "created":
		return shipment.CreatedAt.Format("2006-01-02")
	case "updated":
		return shipment.UpdatedAt.Format("2006-01-02")
	case "delivery":
		if shipment.ExpectedDelivery != nil {
			return shipment.ExpectedDelivery.Format("2006-01-02")
		}
		return ""
	case "delivered":
		if shipment.IsDelivered {
			return "Yes"
		}
		return "No"
	default:
		return ""
	}
}

// refreshCompleteMsg is sent when a refresh operation completes
type refreshCompleteMsg struct {
	response *cliapi.RefreshResponse
	err      error
}

// deleteCompleteMsg is sent when a delete operation completes
type deleteCompleteMsg struct {
	shipmentID int
	err        error
}

// eventsCompleteMsg is sent when an events fetch operation completes
type eventsCompleteMsg struct {
	shipmentID int
	events     []database.TrackingEvent
	err        error
}

// handleRefresh handles the refresh operation
func (m InteractiveTable) handleRefresh() (InteractiveTable, tea.Cmd) {
	if len(m.shipments) == 0 {
		m.message = "No shipments to refresh"
		return m, nil
	}

	selected := m.table.Cursor()
	if selected >= len(m.shipments) {
		m.message = "Invalid selection"
		return m, nil
	}

	shipment := m.shipments[selected]
	m.loading = true
	m.message = ""
	m.err = nil

	return m, tea.Batch(
		m.spinner.Tick,
		m.refreshShipment(shipment.ID),
	)
}

// refreshShipment refreshes a specific shipment
func (m InteractiveTable) refreshShipment(id int) tea.Cmd {
	return func() tea.Msg {
		// Use the client to refresh the shipment
		response, err := m.client.RefreshShipment(id)
		if err != nil {
			return refreshCompleteMsg{err: err}
		}
		return refreshCompleteMsg{response: response}
	}
}

// updateShipmentInTable updates a shipment in the table
// Note: This would require fetching updated shipment data from the API
// For now, we'll just show the refresh success message

// handleDetails handles viewing shipment details
func (m InteractiveTable) handleDetails() (InteractiveTable, tea.Cmd) {
	if len(m.shipments) == 0 {
		m.message = "No shipments to view"
		return m, nil
	}

	selected := m.table.Cursor()
	if selected >= len(m.shipments) {
		m.message = "Invalid selection"
		return m, nil
	}

	shipment := m.shipments[selected]
	
	// Format shipment details
	details := fmt.Sprintf(`
Shipment Details:
ID: %d
Tracking Number: %s
Carrier: %s
Status: %s
Description: %s
Created: %s
Updated: %s
Expected Delivery: %s
Delivered: %v
`,
		shipment.ID,
		shipment.TrackingNumber,
		shipment.Carrier,
		shipment.Status,
		shipment.Description,
		shipment.CreatedAt.Format("2006-01-02 15:04:05"),
		shipment.UpdatedAt.Format("2006-01-02 15:04:05"),
		func() string {
			if shipment.ExpectedDelivery != nil {
				return shipment.ExpectedDelivery.Format("2006-01-02 15:04:05")
			}
			return "N/A"
		}(),
		shipment.IsDelivered,
	)

	m.message = details
	return m, nil
}

// handleEvents handles viewing tracking events
func (m InteractiveTable) handleEvents() (InteractiveTable, tea.Cmd) {
	if len(m.shipments) == 0 {
		m.message = "No shipments to view events for"
		return m, nil
	}

	selected := m.table.Cursor()
	if selected >= len(m.shipments) {
		m.message = "Invalid selection"
		return m, nil
	}

	shipment := m.shipments[selected]
	m.loading = true
	m.message = ""
	m.err = nil

	return m, tea.Batch(
		m.spinner.Tick,
		m.fetchEvents(shipment.ID),
	)
}

// handleUpdateDescription handles updating the shipment description
func (m InteractiveTable) handleUpdateDescription() (InteractiveTable, tea.Cmd) {
	if len(m.shipments) == 0 {
		m.message = "No shipments to update"
		return m, nil
	}

	selected := m.table.Cursor()
	if selected >= len(m.shipments) {
		m.message = "Invalid selection"
		return m, nil
	}

	// Note: This is a simplified implementation. In a real application,
	// you would show a text input for the new description.
	// For now, we'll just show a placeholder message.
	m.message = "Update description functionality not yet implemented"
	return m, nil
}

// handleDelete handles deleting a shipment
func (m InteractiveTable) handleDelete() (InteractiveTable, tea.Cmd) {
	if len(m.shipments) == 0 {
		m.message = "No shipments to delete"
		return m, nil
	}

	selected := m.table.Cursor()
	if selected >= len(m.shipments) {
		m.message = "Invalid selection"
		return m, nil
	}

	shipment := m.shipments[selected]
	m.showDeleteConfirm = true
	m.deleteTarget = shipment.ID
	m.message = ""
	m.err = nil

	return m, nil
}

// confirmDelete executes the delete operation after confirmation
func (m InteractiveTable) confirmDelete() (InteractiveTable, tea.Cmd) {
	m.showDeleteConfirm = false
	m.loading = true
	m.message = ""
	m.err = nil

	return m, tea.Batch(
		m.spinner.Tick,
		m.deleteShipment(m.deleteTarget),
	)
}

// deleteShipment deletes a specific shipment
func (m InteractiveTable) deleteShipment(id int) tea.Cmd {
	return func() tea.Msg {
		err := m.client.DeleteShipment(id)
		return deleteCompleteMsg{shipmentID: id, err: err}
	}
}

// removeShipmentFromTable removes a shipment from the table after successful deletion
func (m InteractiveTable) removeShipmentFromTable(shipmentID int) InteractiveTable {
	// Find the shipment to remove
	newShipments := make([]database.Shipment, 0, len(m.shipments)-1)
	for _, shipment := range m.shipments {
		if shipment.ID != shipmentID {
			newShipments = append(newShipments, shipment)
		}
	}

	// Update the shipments slice
	m.shipments = newShipments

	// Recreate table rows
	rows := make([]table.Row, len(m.shipments))
	for i, shipment := range m.shipments {
		rows[i] = shipmentToRow(shipment, m.fields)
	}

	// Update the table
	m.table.SetRows(rows)

	// Adjust cursor if necessary
	if len(m.shipments) > 0 {
		cursor := m.table.Cursor()
		if cursor >= len(m.shipments) {
			// Move cursor to the last item if it's beyond the new range
			for cursor >= len(m.shipments) && cursor > 0 {
				cursor--
			}
			// We can't directly set cursor, so we'll let the natural navigation handle it
		}
	}

	return m
}

// fetchEvents fetches events for a specific shipment
func (m InteractiveTable) fetchEvents(shipmentID int) tea.Cmd {
	return func() tea.Msg {
		events, err := m.client.GetEvents(shipmentID)
		return eventsCompleteMsg{
			shipmentID: shipmentID,
			events:     events,
			err:        err,
		}
	}
}

// eventsView renders the events view
func (m InteractiveTable) eventsView() string {
	var b strings.Builder
	
	// Find shipment for header
	var shipmentDesc string
	for _, shipment := range m.shipments {
		if shipment.ID == m.eventsShipmentID {
			shipmentDesc = fmt.Sprintf("ID %d - %s (%s)", shipment.ID, shipment.TrackingNumber, shipment.Carrier)
			break
		}
	}
	
	// Header
	title := fmt.Sprintf("Tracking Events for %s", shipmentDesc)
	if m.useColor {
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
		b.WriteString(titleStyle.Render(title))
	} else {
		b.WriteString(title)
	}
	b.WriteString("\n")
	
	// Instructions
	instructions := "Use ↑/↓ to scroll, q/esc to close"
	if m.useColor {
		instrStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		b.WriteString(instrStyle.Render(instructions))
	} else {
		b.WriteString(instructions)
	}
	b.WriteString("\n\n")
	
	if len(m.eventsData) == 0 {
		b.WriteString("No tracking events found.\n")
		return b.String()
	}
	
	// Table header
	header := "TIMESTAMP         LOCATION              STATUS        DESCRIPTION"
	if m.useColor {
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))
		b.WriteString(headerStyle.Render(header))
	} else {
		b.WriteString(header)
	}
	b.WriteString("\n")
	
	// Add separator line
	separator := strings.Repeat("-", len(header))
	if m.useColor {
		sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		b.WriteString(sepStyle.Render(separator))
	} else {
		b.WriteString(separator)
	}
	b.WriteString("\n")
	
	// Show events with scrolling
	maxVisible := 10
	start := m.eventsScroll
	end := start + maxVisible
	if end > len(m.eventsData) {
		end = len(m.eventsData)
	}
	
	for i := start; i < end; i++ {
		event := m.eventsData[i]
		
		// Format timestamp
		timestamp := event.Timestamp.Format("2006-01-02 15:04")
		
		// Truncate location and description
		location := truncateString(event.Location, 20)
		description := truncateString(event.Description, 40)
		
		// Format status with color
		status := event.Status
		if m.useColor {
			status = m.getStatusColorForEvent(event.Status)
		}
		
		// Create row
		row := fmt.Sprintf("%-17s %-20s %-12s %s",
			timestamp,
			location,
			status,
			description)
		
		b.WriteString(row)
		b.WriteString("\n")
	}
	
	// Show scroll indicator if there are more events
	if len(m.eventsData) > maxVisible {
		scrollInfo := fmt.Sprintf("\nShowing %d-%d of %d events", start+1, end, len(m.eventsData))
		if m.useColor {
			scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
			b.WriteString(scrollStyle.Render(scrollInfo))
		} else {
			b.WriteString(scrollInfo)
		}
	}
	
	return b.String()
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// getStatusColorForEvent returns colored status text
func (m InteractiveTable) getStatusColorForEvent(status string) string {
	if m.useColor {
		var color lipgloss.Color
		switch strings.ToLower(status) {
		case "delivered":
			color = lipgloss.Color("82") // Green
		case "in transit", "in-transit", "transit":
			color = lipgloss.Color("226") // Yellow
		case "pending", "pre_ship":
			color = lipgloss.Color("75") // Blue
		case "failed", "error", "exception":
			color = lipgloss.Color("196") // Red
		default:
			color = lipgloss.Color("244") // Gray
		}
		return lipgloss.NewStyle().Foreground(color).Render(status)
	}
	return status
}

// runInteractiveTable runs the interactive table
func runInteractiveTable(shipments []database.Shipment, client *cliapi.Client, formatter *cliapi.OutputFormatter, fieldsFlag string, config *cliapi.Config) error {
	interactiveTable, err := NewInteractiveTable(shipments, client, formatter, fieldsFlag, config)
	if err != nil {
		return err
	}

	p := tea.NewProgram(interactiveTable, tea.WithAltScreen())
	_, err = p.Run()
	return err
}