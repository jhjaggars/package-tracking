package cmd

import (
	"github.com/spf13/cobra"
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
}

func runList(cmd *cobra.Command, args []string) error {
	_, formatter, client, err := initializeClient()
	if err != nil {
		return err
	}

	shipments, err := client.GetShipments()
	if err != nil {
		formatter.PrintError(err)
		return err
	}

	return formatter.PrintShipments(shipments)
}