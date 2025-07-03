package cmd

import (
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:     "delete <shipment-id>",
	Aliases: []string{"del", "rm"},
	Short:   "Delete a shipment",
	Long:    `Delete a shipment from the tracking system.`,
	Args:    cobra.ExactArgs(1),
	RunE:    runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	config, formatter, client, err := initializeClient()
	if err != nil {
		return err
	}

	id, err := validateAndParseID(args[0], formatter)
	if err != nil {
		return err
	}

	err = client.DeleteShipment(id)
	if err != nil {
		formatter.PrintError(err)
		return err
	}

	if !config.Quiet {
		formatter.PrintSuccess("Shipment deleted successfully")
	}

	return nil
}