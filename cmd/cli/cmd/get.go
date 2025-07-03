package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <shipment-id>",
	Short: "Get shipment details by ID",
	Long:  `Get detailed information about a specific shipment by its ID.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runGet,
}

func init() {
	rootCmd.AddCommand(getCmd)
}

func runGet(cmd *cobra.Command, args []string) error {
	_, formatter, client, err := initializeClient()
	if err != nil {
		return err
	}

	id, err := validateAndParseID(args[0], formatter)
	if err != nil {
		return err
	}

	shipment, err := client.GetShipment(id)
	if err != nil {
		formatter.PrintError(err)
		return err
	}

	return formatter.PrintShipment(shipment)
}

// validateAndParseID validates that the argument is a non-empty, valid integer ID
func validateAndParseID(arg string, formatter interface{}) (int, error) {
	if strings.TrimSpace(arg) == "" {
		err := fmt.Errorf("ID cannot be empty")
		return 0, err
	}
	
	id, err := strconv.Atoi(arg)
	if err != nil {
		err = fmt.Errorf("invalid ID '%s': must be a positive integer", arg)
		return 0, err
	}
	
	if id <= 0 {
		err := fmt.Errorf("invalid ID '%d': must be a positive integer", id)
		return 0, err
	}
	
	return id, nil
}