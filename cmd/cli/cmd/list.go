package cmd

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	cliapi "package-tracking/internal/cli"
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
	
	// Add flags for interactive mode and field selection
	listCmd.Flags().BoolVarP(&interactiveMode, "interactive", "i", false, "Interactive table mode")
	listCmd.Flags().StringVar(&fieldsFlag, "fields", "", "Comma-separated list of fields to display (id,tracking,carrier,status,description,created,updated,delivery,delivered)")
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
	if shouldUseInteractiveMode(config, interactiveMode, isatty.IsTerminal(os.Stdout.Fd())) {
		return runInteractiveTable(shipments, client, formatter, fieldsFlag, config)
	}

	return formatter.PrintShipments(shipments)
}

// shouldUseInteractiveMode determines if interactive mode should be activated
func shouldUseInteractiveMode(config *cliapi.Config, explicit bool, isTTY bool) bool {
	// Interactive mode when:
	// - Explicitly requested, OR
	// - No format flags (table) AND stdout is TTY AND not quiet mode
	return explicit || (config.Format == "table" && !config.Quiet && isTTY)
}