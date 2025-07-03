package cmd

import (
	"github.com/spf13/cobra"

	cliapi "package-tracking/internal/cli"
)

var updateCmd = &cobra.Command{
	Use:     "update <shipment-id>",
	Aliases: []string{"edit", "modify"},
	Short:   "Update shipment description",
	Long:    `Update the description of an existing shipment.`,
	Args:    cobra.ExactArgs(1),
	RunE:    runUpdate,
}

var updateDescription string

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringVarP(&updateDescription, "description", "d", "", "New description (required)")
	updateCmd.MarkFlagRequired("description")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	config, formatter, client, err := initializeClient()
	if err != nil {
		return err
	}

	id, err := validateAndParseID(args[0])
	if err != nil {
		formatter.PrintError(err)
		return err
	}

	req := &cliapi.UpdateShipmentRequest{
		Description: updateDescription,
	}

	shipment, err := client.UpdateShipment(id, req)
	if err != nil {
		formatter.PrintError(err)
		return err
	}

	if config.Quiet {
		formatter.PrintShipment(shipment)
	} else {
		formatter.PrintSuccess("Shipment updated successfully")
		formatter.PrintShipment(shipment)
	}

	return nil
}