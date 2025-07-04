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
	}
}

// InteractiveTable represents the interactive table model
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
	showHelp    bool
	quitting    bool
	config      *cliapi.Config
	useColor    bool
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

	// Show table
	b.WriteString(m.table.View())
	b.WriteString("\n")

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
	m.message = fmt.Sprintf("Fetching events for shipment %d...", shipment.ID)
	
	// Note: This is a simplified implementation. In a real application,
	// you would fetch events from the API and display them properly.
	// For now, we'll just show a placeholder message.
	m.message = fmt.Sprintf("Events view for shipment %d would be displayed here", shipment.ID)
	return m, nil
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

	// Note: This is a simplified implementation. In a real application,
	// you would show a confirmation dialog.
	// For now, we'll just show a placeholder message.
	m.message = "Delete functionality not yet implemented"
	return m, nil
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