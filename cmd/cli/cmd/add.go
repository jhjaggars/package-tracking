package cmd

import (
	"github.com/spf13/cobra"

	cliapi "package-tracking/internal/cli"
)

var addCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"a", "create"},
	Short:   "Add a new shipment",
	Long:    `Add a new shipment to track with the specified tracking number and carrier.`,
	RunE:    runAdd,
}

var (
	addTrackingNumber string
	addCarrier        string
	addDescription    string
)

func init() {
	rootCmd.AddCommand(addCmd)

	// Required flags
	addCmd.Flags().StringVarP(&addTrackingNumber, "tracking", "t", "", "Tracking number (required)")
	addCmd.Flags().StringVarP(&addCarrier, "carrier", "c", "", "Carrier name (ups, fedex, usps, dhl) (required)")
	addCmd.Flags().StringVarP(&addDescription, "description", "d", "", "Package description")

	// Mark required flags
	addCmd.MarkFlagRequired("tracking")
	addCmd.MarkFlagRequired("carrier")
}

func runAdd(cmd *cobra.Command, args []string) error {
	config, formatter, client, err := initializeClient()
	if err != nil {
		return err
	}

	req := &cliapi.CreateShipmentRequest{
		TrackingNumber: addTrackingNumber,
		Carrier:        addCarrier,
		Description:    addDescription,
	}

	shipment, err := client.CreateShipment(req)
	if err != nil {
		formatter.PrintError(err)
		return err
	}

	if config.Quiet {
		formatter.PrintShipment(shipment)
	} else {
		formatter.PrintSuccess("Shipment added successfully")
		formatter.PrintShipment(shipment)
	}

	return nil
}