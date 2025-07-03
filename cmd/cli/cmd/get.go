package cmd

import (
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:     "get <shipment-id>",
	Aliases: []string{"show", "info"},
	Short:   "Get shipment details by ID",
	Long:    `Get detailed information about a specific shipment by its ID.`,
	Args:    cobra.ExactArgs(1),
	RunE:    runGet,
}

func init() {
	rootCmd.AddCommand(getCmd)
}

func runGet(cmd *cobra.Command, args []string) error {
	_, formatter, client, err := initializeClient()
	if err != nil {
		return err
	}

	id, err := validateAndParseID(args[0])
	if err != nil {
		formatter.PrintError(err)
		return err
	}

	shipment, err := client.GetShipment(id)
	if err != nil {
		formatter.PrintError(err)
		return err
	}

	return formatter.PrintShipment(shipment)
}