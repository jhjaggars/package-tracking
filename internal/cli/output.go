package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"package-tracking/internal/database"
	
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
)

// StyleConfig holds color and styling configuration
type StyleConfig struct {
	// Status colors
	DeliveredColor  lipgloss.Color
	InTransitColor  lipgloss.Color
	PendingColor    lipgloss.Color
	FailedColor     lipgloss.Color
	UnknownColor    lipgloss.Color
	
	// Message colors
	SuccessColor    lipgloss.Color
	ErrorColor      lipgloss.Color
	InfoColor       lipgloss.Color
	
	// Table styling
	HeaderStyle     lipgloss.Style
	CellStyle       lipgloss.Style
}

// DefaultStyleConfig returns the default style configuration
func DefaultStyleConfig() *StyleConfig {
	return &StyleConfig{
		DeliveredColor:  lipgloss.Color("10"), // Bright green
		InTransitColor:  lipgloss.Color("11"), // Bright yellow
		PendingColor:    lipgloss.Color("12"), // Bright blue
		FailedColor:     lipgloss.Color("9"),  // Bright red
		UnknownColor:    lipgloss.Color("8"),  // Gray
		SuccessColor:    lipgloss.Color("10"), // Green
		ErrorColor:      lipgloss.Color("9"),  // Red
		InfoColor:       lipgloss.Color("12"), // Blue
		HeaderStyle:     lipgloss.NewStyle().Bold(true),
		CellStyle:       lipgloss.NewStyle(),
	}
}

// OutputFormatter handles different output formats
type OutputFormatter struct {
	format      string
	quiet       bool
	noColor     bool
	styles      *StyleConfig
	colorOutput termenv.Profile
}

// NewOutputFormatter creates a new output formatter
func NewOutputFormatter(format string, quiet bool) *OutputFormatter {
	return NewOutputFormatterWithColor(format, quiet, false)
}

// NewOutputFormatterWithColor creates a new output formatter with color support
func NewOutputFormatterWithColor(format string, quiet bool, noColor bool) *OutputFormatter {
	f := &OutputFormatter{
		format:      format,
		quiet:       quiet,
		noColor:     noColor,
		styles:      DefaultStyleConfig(),
		colorOutput: termenv.ColorProfile(),
	}
	
	// Detect if colors should be disabled
	if !f.shouldUseColor() {
		f.noColor = true
	}
	
	return f
}

// shouldUseColor determines if colors should be used based on environment
func (f *OutputFormatter) shouldUseColor() bool {
	// If explicitly disabled, don't use color
	if f.noColor {
		return false
	}
	
	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	
	// Check if output is being piped
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return false
	}
	
	// Check if we're in a CI environment
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		return false
	}
	
	// Check terminal color support
	if f.colorOutput == termenv.Ascii {
		return false
	}
	
	return true
}

// PrintShipments prints a list of shipments
func (f *OutputFormatter) PrintShipments(shipments []database.Shipment) error {
	if f.quiet {
		for _, shipment := range shipments {
			fmt.Printf("%d\n", shipment.ID)
		}
		return nil
	}

	switch f.format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(shipments)
	case "table":
		return f.printShipmentsTable(shipments)
	default:
		return fmt.Errorf("unsupported format: %s", f.format)
	}
}

// PrintShipment prints a single shipment
func (f *OutputFormatter) PrintShipment(shipment *database.Shipment) error {
	if f.quiet {
		fmt.Printf("%d\n", shipment.ID)
		return nil
	}

	switch f.format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(shipment)
	case "table":
		return f.printShipmentTable(shipment)
	default:
		return fmt.Errorf("unsupported format: %s", f.format)
	}
}

// PrintEvents prints tracking events
func (f *OutputFormatter) PrintEvents(events []database.TrackingEvent) error {
	if f.quiet {
		for _, event := range events {
			fmt.Printf("%d\n", event.ID)
		}
		return nil
	}

	switch f.format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(events)
	case "table":
		return f.printEventsTable(events)
	default:
		return fmt.Errorf("unsupported format: %s", f.format)
	}
}

// getStatusStyle returns the appropriate style for a status
func (f *OutputFormatter) getStatusStyle(status string) lipgloss.Style {
	if f.noColor {
		return lipgloss.NewStyle()
	}
	
	var color lipgloss.Color
	switch strings.ToLower(status) {
	case "delivered":
		color = f.styles.DeliveredColor
	case "in transit", "in-transit", "transit":
		color = f.styles.InTransitColor
	case "pending":
		color = f.styles.PendingColor
	case "failed", "error", "exception":
		color = f.styles.FailedColor
	default:
		color = f.styles.UnknownColor
	}
	
	return lipgloss.NewStyle().Foreground(color)
}

// PrintSuccess prints a success message
func (f *OutputFormatter) PrintSuccess(message string) {
	if !f.quiet {
		if f.noColor {
			fmt.Printf("✓ %s\n", message)
		} else {
			style := lipgloss.NewStyle().Foreground(f.styles.SuccessColor)
			fmt.Printf("%s %s\n", style.Render("✓"), message)
		}
	}
}

// PrintError prints an error message
func (f *OutputFormatter) PrintError(err error) {
	if !f.quiet {
		if f.noColor {
			fmt.Fprintf(os.Stderr, "✗ Error: %v\n", err)
		} else {
			style := lipgloss.NewStyle().Foreground(f.styles.ErrorColor)
			fmt.Fprintf(os.Stderr, "%s Error: %v\n", style.Render("✗"), err)
		}
	}
}

// PrintInfo prints an informational message
func (f *OutputFormatter) PrintInfo(message string) {
	if !f.quiet {
		if f.noColor {
			fmt.Printf("ℹ %s\n", message)
		} else {
			style := lipgloss.NewStyle().Foreground(f.styles.InfoColor)
			fmt.Printf("%s %s\n", style.Render("ℹ"), message)
		}
	}
}

// printShipmentsTable prints shipments in table format
func (f *OutputFormatter) printShipmentsTable(shipments []database.Shipment) error {
	if len(shipments) == 0 {
		fmt.Println("No shipments found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Always use plain headers for tabwriter alignment, style them afterwards if needed
	fmt.Fprintln(w, "ID\tTRACKING\tCARRIER\tSTATUS\tDESCRIPTION\tCREATED")

	// Data rows
	for _, shipment := range shipments {
		status := shipment.Status
		if !f.noColor {
			statusStyle := f.getStatusStyle(shipment.Status)
			status = statusStyle.Render(shipment.Status)
		}
		
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			shipment.ID,
			truncate(shipment.TrackingNumber, 15),
			strings.ToUpper(shipment.Carrier),
			status,
			truncate(shipment.Description, 25),
			shipment.CreatedAt.Format("2006-01-02"))
	}

	return nil
}

// printShipmentTable prints a single shipment in table format
func (f *OutputFormatter) printShipmentTable(shipment *database.Shipment) error {
	fmt.Printf("Shipment ID: %d\n", shipment.ID)
	fmt.Printf("Tracking Number: %s\n", shipment.TrackingNumber)
	fmt.Printf("Carrier: %s\n", strings.ToUpper(shipment.Carrier))
	fmt.Printf("Description: %s\n", shipment.Description)
	
	// Style the status field
	if f.noColor {
		fmt.Printf("Status: %s\n", shipment.Status)
	} else {
		statusStyle := f.getStatusStyle(shipment.Status)
		fmt.Printf("Status: %s\n", statusStyle.Render(shipment.Status))
	}
	
	fmt.Printf("Created: %s\n", shipment.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", shipment.UpdatedAt.Format("2006-01-02 15:04:05"))
	
	if shipment.ExpectedDelivery != nil {
		fmt.Printf("Expected Delivery: %s\n", shipment.ExpectedDelivery.Format("2006-01-02"))
	}
	
	fmt.Printf("Delivered: %v\n", shipment.IsDelivered)
	
	return nil
}

// printEventsTable prints events in table format
func (f *OutputFormatter) printEventsTable(events []database.TrackingEvent) error {
	if len(events) == 0 {
		fmt.Println("No tracking events found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header - always plain for tabwriter alignment
	fmt.Fprintln(w, "TIMESTAMP\tLOCATION\tSTATUS\tDESCRIPTION")

	// Data
	for _, event := range events {
		status := event.Status
		if !f.noColor {
			statusStyle := f.getStatusStyle(event.Status)
			status = statusStyle.Render(event.Status)
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			event.Timestamp.Format("2006-01-02 15:04"),
			truncate(event.Location, 20),
			status,
			truncate(event.Description, 40))
	}

	return nil
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}