package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	cliapi "package-tracking/internal/cli"
	"package-tracking/internal/database"
)

// Message types for tea.Cmd
type refreshCompleteMsg struct {
	response *cliapi.RefreshResponse
	err      error
}

type updateCompleteMsg struct {
	shipment *database.Shipment
	err      error
}

type deleteCompleteMsg struct {
	err error
}

type shipmentDetailsMsg struct {
	shipment *database.Shipment
	err      error
}

type eventsMsg struct {
	events []database.TrackingEvent
	err    error
}

// KeyMap defines keyboard shortcuts for the interactive table
type KeyMap struct {
	Refresh  key.Binding
	Update   key.Binding
	Delete   key.Binding
	Details  key.Binding
	Events   key.Binding
	Help     key.Binding
	Quit     key.Binding
}

// DefaultKeyMap returns the default key bindings
var DefaultKeyMap = KeyMap{
	Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Update:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update")),
	Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Details:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
	Events:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "events")),
	Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// InteractiveTable represents the BubbleTea model for the interactive table
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
	showDetails bool
	currentDetails interface{}
	showHelp    bool
}

// NewInteractiveTable creates a new interactive table model
func NewInteractiveTable(shipments []database.Shipment, client *cliapi.Client, formatter *cliapi.OutputFormatter, fields []string) *InteractiveTable {
	// Set up the table columns based on fields
	columns := []table.Column{}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if displayName, exists := availableFields[field]; exists {
			width := getColumnWidth(field)
			columns = append(columns, table.Column{
				Title: displayName,
				Width: width,
			})
		}
	}

	// Create table rows
	rows := []table.Row{}
	for _, shipment := range shipments {
		row := table.Row{}
		for _, field := range fields {
			field = strings.TrimSpace(field)
			value := getFieldValue(shipment, field)
			row = append(row, value)
		}
		rows = append(rows, row)
	}

	// Create the table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Create spinner for loading states
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &InteractiveTable{
		table:     t,
		shipments: shipments,
		client:    client,
		formatter: formatter,
		fields:    fields,
		keys:      DefaultKeyMap,
		spinner:   s,
	}
}

// Init initializes the interactive table
func (m InteractiveTable) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles key presses and updates the model
func (m InteractiveTable) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showDetails {
			// When showing details, any key except help goes back to table
			switch {
			case key.Matches(msg, m.keys.Help):
				m.showHelp = !m.showHelp
			default:
				m.showDetails = false
				m.currentDetails = nil
			}
			return m, nil
		}

		if m.showHelp {
			// When showing help, any key closes help
			m.showHelp = false
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = true
			return m, nil

		case key.Matches(msg, m.keys.Refresh):
			return m, m.handleRefresh()

		case key.Matches(msg, m.keys.Update):
			return m, m.handleUpdate()

		case key.Matches(msg, m.keys.Delete):
			return m, m.handleDelete()

		case key.Matches(msg, m.keys.Details):
			return m, m.handleDetails()

		case key.Matches(msg, m.keys.Events):
			return m, m.handleEvents()
		}

	case refreshCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.message = fmt.Sprintf("Refresh failed: %v", msg.err)
		} else {
			m.message = fmt.Sprintf("Refresh complete: %d events added", msg.response.EventsAdded)
			// Update the shipment in our local data
			m.updateLocalShipment(msg.response.ShipmentID)
		}
		return m, nil

	case updateCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.message = fmt.Sprintf("Update failed: %v", msg.err)
		} else {
			m.message = "Description updated successfully"
			// Update the shipment in our local data
			m.updateLocalShipmentData(*msg.shipment)
		}
		return m, nil

	case deleteCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.message = fmt.Sprintf("Delete failed: %v", msg.err)
		} else {
			m.message = "Shipment deleted successfully"
			// Remove the shipment from our local data
			m.removeLocalShipment()
		}
		return m, nil

	case shipmentDetailsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.message = fmt.Sprintf("Failed to get details: %v", msg.err)
		} else {
			m.showDetails = true
			m.currentDetails = msg.shipment
		}
		return m, nil

	case eventsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.message = fmt.Sprintf("Failed to get events: %v", msg.err)
		} else {
			m.showDetails = true
			m.currentDetails = msg.events
		}
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Update the table
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the interactive table
func (m InteractiveTable) View() string {
	if m.quitting {
		return ""
	}

	if m.showHelp {
		return m.helpView()
	}

	if m.showDetails {
		return m.detailsView()
	}

	var s strings.Builder

	// Title
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("Package Tracking - Interactive Mode"))
	s.WriteString("\n\n")

	// Table
	s.WriteString(m.table.View())
	s.WriteString("\n\n")

	// Status line
	s.WriteString(m.statusLine())
	s.WriteString("\n")

	// Message or error
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		s.WriteString("\n")
	} else if m.message != "" {
		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
		s.WriteString(infoStyle.Render(m.message))
		s.WriteString("\n")
	}

	// Loading indicator
	if m.loading {
		s.WriteString(m.spinner.View() + " Loading...")
		s.WriteString("\n")
	}

	return s.String()
}

// statusLine returns the status line with keyboard shortcuts
func (m InteractiveTable) statusLine() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	return style.Render("r: refresh • u: update • d: delete • enter: details • e: events • ?: help • q: quit")
}

// helpView renders the help screen
func (m InteractiveTable) helpView() string {
	var s strings.Builder
	
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	s.WriteString(titleStyle.Render("Interactive Table Help"))
	s.WriteString("\n\n")

	helpText := []string{
		"Navigation:",
		"  ↑/↓, j/k    Navigate up/down",
		"  Page Up/Down Page navigation",
		"  Home/End    Go to first/last row",
		"",
		"Operations:",
		"  r           Refresh selected shipment",
		"  u           Update description",
		"  d           Delete shipment (with confirmation)",
		"  enter       View shipment details",
		"  e           View tracking events",
		"",
		"General:",
		"  ?           Show/hide this help",
		"  q, Ctrl+C   Quit",
		"",
		"Press any key to return to the table.",
	}

	for _, line := range helpText {
		s.WriteString(line)
		s.WriteString("\n")
	}

	return s.String()
}

// detailsView renders the details screen
func (m InteractiveTable) detailsView() string {
	var s strings.Builder
	
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))

	switch details := m.currentDetails.(type) {
	case *database.Shipment:
		s.WriteString(titleStyle.Render("Shipment Details"))
		s.WriteString("\n\n")
		s.WriteString(m.formatShipmentDetails(*details))
		
	case []database.TrackingEvent:
		s.WriteString(titleStyle.Render("Tracking Events"))
		s.WriteString("\n\n")
		s.WriteString(m.formatTrackingEvents(details))
	}

	s.WriteString("\n\nPress any key to return to the table.")
	return s.String()
}

// Helper functions for operations
func (m InteractiveTable) handleRefresh() tea.Cmd {
	if len(m.shipments) == 0 || m.table.Cursor() >= len(m.shipments) {
		return nil
	}

	shipment := m.shipments[m.table.Cursor()]
	m.loading = true
	m.message = ""
	m.err = nil

	return tea.Cmd(func() tea.Msg {
		response, err := m.client.RefreshShipmentWithForce(shipment.ID, false)
		return refreshCompleteMsg{response: response, err: err}
	})
}

func (m InteractiveTable) handleUpdate() tea.Cmd {
	if len(m.shipments) == 0 || m.table.Cursor() >= len(m.shipments) {
		return nil
	}

	// For now, we'll implement a simple prompt-based update
	// In a real implementation, this could use a text input bubble
	fmt.Print("Enter new description: ")
	var newDescription string
	fmt.Scanln(&newDescription)

	if newDescription == "" {
		return nil
	}

	shipment := m.shipments[m.table.Cursor()]
	m.loading = true
	m.message = ""
	m.err = nil

	return tea.Cmd(func() tea.Msg {
		req := &cliapi.UpdateShipmentRequest{Description: newDescription}
		updatedShipment, err := m.client.UpdateShipment(shipment.ID, req)
		return updateCompleteMsg{shipment: updatedShipment, err: err}
	})
}

func (m InteractiveTable) handleDelete() tea.Cmd {
	if len(m.shipments) == 0 || m.table.Cursor() >= len(m.shipments) {
		return nil
	}

	// Simple confirmation
	fmt.Print("Are you sure you want to delete this shipment? (y/N): ")
	var confirm string
	fmt.Scanln(&confirm)

	if strings.ToLower(confirm) != "y" {
		return nil
	}

	shipment := m.shipments[m.table.Cursor()]
	m.loading = true
	m.message = ""
	m.err = nil

	return tea.Cmd(func() tea.Msg {
		err := m.client.DeleteShipment(shipment.ID)
		return deleteCompleteMsg{err: err}
	})
}

func (m InteractiveTable) handleDetails() tea.Cmd {
	if len(m.shipments) == 0 || m.table.Cursor() >= len(m.shipments) {
		return nil
	}

	shipment := m.shipments[m.table.Cursor()]
	m.loading = true
	m.message = ""
	m.err = nil

	return tea.Cmd(func() tea.Msg {
		detailedShipment, err := m.client.GetShipment(shipment.ID)
		return shipmentDetailsMsg{shipment: detailedShipment, err: err}
	})
}

func (m InteractiveTable) handleEvents() tea.Cmd {
	if len(m.shipments) == 0 || m.table.Cursor() >= len(m.shipments) {
		return nil
	}

	shipment := m.shipments[m.table.Cursor()]
	m.loading = true
	m.message = ""
	m.err = nil

	return tea.Cmd(func() tea.Msg {
		events, err := m.client.GetEvents(shipment.ID)
		return eventsMsg{events: events, err: err}
	})
}

// Helper functions for data management
func (m *InteractiveTable) updateLocalShipment(shipmentID int) {
	// Refresh the shipment data from the server
	go func() {
		if updatedShipment, err := m.client.GetShipment(shipmentID); err == nil {
			m.updateLocalShipmentData(*updatedShipment)
		}
	}()
}

func (m *InteractiveTable) updateLocalShipmentData(shipment database.Shipment) {
	// Find and update the shipment in our local data
	for i, s := range m.shipments {
		if s.ID == shipment.ID {
			m.shipments[i] = shipment
			// Update the table row
			m.updateTableRow(i, shipment)
			break
		}
	}
}

func (m *InteractiveTable) removeLocalShipment() {
	cursor := m.table.Cursor()
	if cursor >= len(m.shipments) {
		return
	}

	// Remove the shipment from our local data
	m.shipments = append(m.shipments[:cursor], m.shipments[cursor+1:]...)
	
	// Update the table
	rows := []table.Row{}
	for _, shipment := range m.shipments {
		row := table.Row{}
		for _, field := range m.fields {
			field = strings.TrimSpace(field)
			value := getFieldValue(shipment, field)
			row = append(row, value)
		}
		rows = append(rows, row)
	}
	m.table.SetRows(rows)
}

func (m *InteractiveTable) updateTableRow(index int, shipment database.Shipment) {
	if index >= len(m.shipments) {
		return
	}

	row := table.Row{}
	for _, field := range m.fields {
		field = strings.TrimSpace(field)
		value := getFieldValue(shipment, field)
		row = append(row, value)
	}

	// Get current rows and update the specific row
	rows := m.table.Rows()
	if index < len(rows) {
		rows[index] = row
		m.table.SetRows(rows)
	}
}

// Helper functions for field values and formatting
func getFieldValue(shipment database.Shipment, field string) string {
	switch field {
	case "id":
		return strconv.Itoa(shipment.ID)
	case "tracking":
		return truncateString(shipment.TrackingNumber, 15)
	case "carrier":
		return strings.ToUpper(shipment.Carrier)
	case "status":
		return shipment.Status
	case "description":
		return truncateString(shipment.Description, 25)
	case "created":
		return shipment.CreatedAt.Format("2006-01-02")
	case "updated":
		return shipment.UpdatedAt.Format("2006-01-02")
	case "delivery":
		if shipment.ExpectedDelivery != nil {
			return shipment.ExpectedDelivery.Format("2006-01-02")
		}
		return "-"
	case "delivered":
		if shipment.IsDelivered {
			return "Yes"
		}
		return "No"
	default:
		return ""
	}
}

func getColumnWidth(field string) int {
	switch field {
	case "id":
		return 5
	case "tracking":
		return 18
	case "carrier":
		return 8
	case "status":
		return 12
	case "description":
		return 30
	case "created", "updated", "delivery":
		return 12
	case "delivered":
		return 9
	default:
		return 10
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func (m InteractiveTable) formatShipmentDetails(shipment database.Shipment) string {
	var s strings.Builder
	
	s.WriteString(fmt.Sprintf("ID: %d\n", shipment.ID))
	s.WriteString(fmt.Sprintf("Tracking Number: %s\n", shipment.TrackingNumber))
	s.WriteString(fmt.Sprintf("Carrier: %s\n", strings.ToUpper(shipment.Carrier)))
	s.WriteString(fmt.Sprintf("Description: %s\n", shipment.Description))
	s.WriteString(fmt.Sprintf("Status: %s\n", shipment.Status))
	s.WriteString(fmt.Sprintf("Created: %s\n", shipment.CreatedAt.Format("2006-01-02 15:04:05")))
	s.WriteString(fmt.Sprintf("Updated: %s\n", shipment.UpdatedAt.Format("2006-01-02 15:04:05")))
	
	if shipment.ExpectedDelivery != nil {
		s.WriteString(fmt.Sprintf("Expected Delivery: %s\n", shipment.ExpectedDelivery.Format("2006-01-02")))
	}
	
	s.WriteString(fmt.Sprintf("Delivered: %v\n", shipment.IsDelivered))
	
	return s.String()
}

func (m InteractiveTable) formatTrackingEvents(events []database.TrackingEvent) string {
	if len(events) == 0 {
		return "No tracking events found."
	}

	var s strings.Builder
	
	for _, event := range events {
		s.WriteString(fmt.Sprintf("%s - %s\n", 
			event.Timestamp.Format("2006-01-02 15:04"), 
			event.Description))
		if event.Location != "" {
			s.WriteString(fmt.Sprintf("  Location: %s\n", event.Location))
		}
		if event.Status != "" {
			s.WriteString(fmt.Sprintf("  Status: %s\n", event.Status))
		}
		s.WriteString("\n")
	}
	
	return s.String()
}