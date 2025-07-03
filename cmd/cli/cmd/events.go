package cmd

import (
	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events <shipment-id>",
	Short: "View tracking events for a shipment",
	Long:  `View the tracking history and events for a specific shipment.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runEvents,
}

func init() {
	rootCmd.AddCommand(eventsCmd)
}

func runEvents(cmd *cobra.Command, args []string) error {
	_, formatter, client, err := initializeClient()
	if err != nil {
		return err
	}

	id, err := validateAndParseID(args[0])
	if err != nil {
		formatter.PrintError(err)
		return err
	}

	events, err := client.GetEvents(id)
	if err != nil {
		formatter.PrintError(err)
		return err
	}

	return formatter.PrintEvents(events)
}