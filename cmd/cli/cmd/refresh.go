package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	cliapi "package-tracking/internal/cli"
)

var refreshCmd = &cobra.Command{
	Use:   "refresh <shipment-id>",
	Short: "Manually refresh tracking data for a shipment",
	Long:  `Manually refresh the tracking data for a specific shipment by fetching the latest information from the carrier.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRefresh,
}

var (
	refreshVerbose bool
	refreshForce   bool
)

func init() {
	rootCmd.AddCommand(refreshCmd)

	refreshCmd.Flags().BoolVar(&refreshVerbose, "verbose", false, "Show detailed refresh information")
	refreshCmd.Flags().BoolVar(&refreshForce, "force", false, "Force refresh by bypassing cache")
}

func runRefresh(cmd *cobra.Command, args []string) error {
	config, formatter, client, err := initializeClient()
	if err != nil {
		return err
	}

	id, err := validateAndParseID(args[0], formatter)
	if err != nil {
		return err
	}

	// Show progress spinner for refresh operation
	var spinner *cliapi.ProgressSpinner
	if !config.Quiet {
		spinnerText := "Refreshing tracking data"
		if refreshForce {
			spinnerText = "Force refreshing tracking data (bypassing cache)"
		}
		spinner = cliapi.NewProgressSpinner(spinnerText, noColor)
		spinner.Start()
	}

	response, err := client.RefreshShipmentWithForce(id, refreshForce)
	
	// Stop spinner before printing results
	if spinner != nil {
		spinner.Stop()
	}
	
	if err != nil {
		formatter.PrintError(err)
		return err
	}

	if config.Quiet {
		// In quiet mode, just show the events
		return formatter.PrintEvents(response.Events)
	} else {
		// Show refresh details
		if refreshVerbose {
			formatter.PrintSuccess(fmt.Sprintf("Refresh completed successfully"))
			formatter.PrintInfo(fmt.Sprintf("Shipment ID: %d", response.ShipmentID))
			formatter.PrintInfo(fmt.Sprintf("Updated at: %s", response.UpdatedAt.Format("2006-01-02 15:04:05")))
			formatter.PrintInfo(fmt.Sprintf("Events added: %d", response.EventsAdded))
			formatter.PrintInfo(fmt.Sprintf("Total events: %d", response.TotalEvents))
			
			// Show cache information
			if response.CacheStatus != "" {
				formatter.PrintInfo(fmt.Sprintf("Cache status: %s", response.CacheStatus))
			}
			if response.RefreshDuration != "" {
				formatter.PrintInfo(fmt.Sprintf("Refresh duration: %s", response.RefreshDuration))
			}
			if response.PreviousCacheAge != "" {
				formatter.PrintInfo(fmt.Sprintf("Previous cache age: %s", response.PreviousCacheAge))
			}
			
			if response.EventsAdded > 0 {
				formatter.PrintInfo("New tracking events:")
			} else {
				formatter.PrintInfo("No new tracking events found")
			}
		} else {
			// Show basic status with cache info for force refresh
			var successMsg string
			if response.EventsAdded > 0 {
				successMsg = fmt.Sprintf("Refresh successful - %d new events found", response.EventsAdded)
			} else {
				successMsg = "Refresh successful - no new events"
			}
			
			// Add cache status for force refresh
			if refreshForce && response.PreviousCacheAge != "" {
				successMsg += fmt.Sprintf(" (invalidated %s old cache)", response.PreviousCacheAge)
			} else if response.CacheStatus == "hit" {
				successMsg += " (from cache)"
			}
			
			formatter.PrintSuccess(successMsg)
		}

		// Show all events
		return formatter.PrintEvents(response.Events)
	}
}