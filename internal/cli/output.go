package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"package-tracking/internal/database"
)

// OutputFormatter handles different output formats
type OutputFormatter struct {
	format string
	quiet  bool
}

// NewOutputFormatter creates a new output formatter
func NewOutputFormatter(format string, quiet bool) *OutputFormatter {
	return &OutputFormatter{
		format: format,
		quiet:  quiet,
	}
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

// PrintSuccess prints a success message
func (f *OutputFormatter) PrintSuccess(message string) {
	if !f.quiet {
		fmt.Printf("✓ %s\n", message)
	}
}

// PrintError prints an error message
func (f *OutputFormatter) PrintError(err error) {
	if !f.quiet {
		fmt.Fprintf(os.Stderr, "✗ Error: %v\n", err)
	}
}

// PrintInfo prints an informational message
func (f *OutputFormatter) PrintInfo(message string) {
	if !f.quiet {
		fmt.Printf("ℹ %s\n", message)
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

	// Header
	fmt.Fprintln(w, "ID\tTRACKING\tCARRIER\tSTATUS\tDESCRIPTION\tCREATED")

	// Data
	for _, shipment := range shipments {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			shipment.ID,
			truncate(shipment.TrackingNumber, 15),
			strings.ToUpper(shipment.Carrier),
			shipment.Status,
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
	fmt.Printf("Status: %s\n", shipment.Status)
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

	// Header
	fmt.Fprintln(w, "TIMESTAMP\tLOCATION\tSTATUS\tDESCRIPTION")

	// Data
	for _, event := range events {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			event.Timestamp.Format("2006-01-02 15:04"),
			truncate(event.Location, 20),
			event.Status,
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