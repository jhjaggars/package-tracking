package cmd

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	cliapi "package-tracking/internal/cli"
	"package-tracking/internal/database"
)

var (
	interactiveMode bool
	fieldsFlag      string
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all shipments",
	Long:    `List all shipments currently being tracked.`,
	RunE:    runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	
	// Add flags for interactive mode and field configuration
	listCmd.Flags().BoolVarP(&interactiveMode, "interactive", "i", false, "Interactive table mode")
	listCmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated list of fields to display")
}

func runList(cmd *cobra.Command, args []string) error {
	config, formatter, client, err := initializeClient()
	if err != nil {
		return err
	}

	shipments, err := client.GetShipments()
	if err != nil {
		formatter.PrintError(err)
		return err
	}

	// Determine if interactive mode should be used
	if shouldUseInteractiveMode(config, interactiveMode) {
		return runInteractiveTable(shipments, client, formatter, fieldsFlag)
	}

	return formatter.PrintShipments(shipments)
}

// shouldUseInteractiveMode determines if interactive mode should be used
func shouldUseInteractiveMode(config *cliapi.Config, explicit bool) bool {
	// Interactive mode when:
	// - Explicitly requested, OR
	// - No format flags AND stdout is TTY AND not quiet mode AND not in CI
	if explicit {
		return true
	}
	
	// Don't use interactive mode for non-table formats
	if config.Format != "table" {
		return false
	}
	
	// Don't use interactive mode in quiet mode
	if config.Quiet {
		return false
	}
	
	// Don't use interactive mode if not a terminal
	if !isTerminalFunc() {
		return false
	}
	
	// Don't use interactive mode in CI environments
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		return false
	}
	
	return true
}

// isTerminalFunc allows for testing by mocking terminal detection
var isTerminalFunc = func() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// Default field configuration
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

// parseFields parses the --fields flag input
func parseFields(fieldsFlag string) []string {
	if fieldsFlag == "" {
		return defaultFields
	}
	return strings.Split(fieldsFlag, ",")
}

// validateFields validates that all requested fields are available
func validateFields(fields []string) error {
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if _, exists := availableFields[field]; !exists {
			return fmt.Errorf("unknown field: %s", field)
		}
	}
	return nil
}

// runInteractiveTable runs the interactive table interface
func runInteractiveTable(shipments []database.Shipment, client *cliapi.Client, formatter *cliapi.OutputFormatter, fieldsFlag string) error {
	// Parse and validate fields
	fields := parseFields(fieldsFlag)
	if err := validateFields(fields); err != nil {
		return err
	}
	
	// Create and run the interactive table
	model := NewInteractiveTable(shipments, client, formatter, fields)
	p := tea.NewProgram(model)
	
	_, err := p.Run()
	return err
}