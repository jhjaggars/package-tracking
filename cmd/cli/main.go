package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
	
	cliapi "package-tracking/internal/cli"
)

func main() {
	app := &cli.App{
		Name:  "package-tracker",
		Usage: "CLI client for package tracking API",
		Version: "1.0.0",
		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "Add a new shipment",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "tracking",
						Aliases:  []string{"t"},
						Usage:    "Tracking number",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "carrier",
						Aliases:  []string{"c"},
						Usage:    "Carrier name (ups, fedex, usps, dhl)",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "description",
						Aliases: []string{"d"},
						Usage:   "Package description",
					},
				},
				Action: addShipment,
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List all shipments",
				Action:  listShipments,
			},
			{
				Name:      "get",
				Usage:     "Get shipment details by ID",
				ArgsUsage: "<shipment-id>",
				Action:    getShipment,
			},
			{
				Name:      "update",
				Usage:     "Update shipment description",
				ArgsUsage: "<shipment-id>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "description",
						Aliases:  []string{"d"},
						Usage:    "New description",
						Required: true,
					},
				},
				Action: updateShipment,
			},
			{
				Name:      "delete",
				Aliases:   []string{"del", "rm"},
				Usage:     "Delete a shipment",
				ArgsUsage: "<shipment-id>",
				Action:    deleteShipment,
			},
			{
				Name:      "events",
				Usage:     "View tracking events for a shipment",
				ArgsUsage: "<shipment-id>",
				Action:    getEvents,
			},
			{
				Name:      "refresh",
				Usage:     "Manually refresh tracking data for a shipment",
				ArgsUsage: "<shipment-id>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Usage:   "Show detailed refresh information",
					},
				},
				Action: refreshShipment,
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "server",
				Aliases: []string{"s"},
				Usage:   "API server address",
				Value:   "http://localhost:8080",
				EnvVars: []string{"PACKAGE_TRACKER_SERVER"},
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "Quiet mode (minimal output)",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// Command handlers
func addShipment(c *cli.Context) error {
	config, formatter, client, err := initializeClient(c)
	if err != nil {
		return err
	}

	tracking := c.String("tracking")
	carrier := c.String("carrier")
	description := c.String("description")

	req := &cliapi.CreateShipmentRequest{
		TrackingNumber: tracking,
		Carrier:        carrier,
		Description:    description,
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

func listShipments(c *cli.Context) error {
	_, formatter, client, err := initializeClient(c)
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

func getShipment(c *cli.Context) error {
	_, formatter, client, err := initializeClient(c)
	if err != nil {
		return err
	}

	if c.NArg() != 1 {
		return cli.ShowCommandHelp(c, "get")
	}

	id, err := validateAndParseID(c.Args().Get(0), formatter)
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

func updateShipment(c *cli.Context) error {
	config, formatter, client, err := initializeClient(c)
	if err != nil {
		return err
	}

	if c.NArg() != 1 {
		return cli.ShowCommandHelp(c, "update")
	}

	id, err := validateAndParseID(c.Args().Get(0), formatter)
	if err != nil {
		return err
	}

	description := c.String("description")
	req := &cliapi.UpdateShipmentRequest{
		Description: description,
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

func deleteShipment(c *cli.Context) error {
	config, formatter, client, err := initializeClient(c)
	if err != nil {
		return err
	}

	if c.NArg() != 1 {
		return cli.ShowCommandHelp(c, "delete")
	}

	id, err := validateAndParseID(c.Args().Get(0), formatter)
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

func getEvents(c *cli.Context) error {
	_, formatter, client, err := initializeClient(c)
	if err != nil {
		return err
	}

	if c.NArg() != 1 {
		return cli.ShowCommandHelp(c, "events")
	}

	id, err := validateAndParseID(c.Args().Get(0), formatter)
	if err != nil {
		return err
	}

	events, err := client.GetEvents(id)
	if err != nil {
		formatter.PrintError(err)
		return err
	}

	return formatter.PrintEvents(events)
}

// validateAndParseID validates that the argument is a non-empty, valid integer ID
func validateAndParseID(arg string, formatter *cliapi.OutputFormatter) (int, error) {
	if strings.TrimSpace(arg) == "" {
		err := fmt.Errorf("ID cannot be empty")
		formatter.PrintError(err)
		return 0, err
	}
	
	id, err := strconv.Atoi(arg)
	if err != nil {
		err = fmt.Errorf("invalid ID '%s': must be a positive integer", arg)
		formatter.PrintError(err)
		return 0, err
	}
	
	if id <= 0 {
		err := fmt.Errorf("invalid ID '%d': must be a positive integer", id)
		formatter.PrintError(err)
		return 0, err
	}
	
	return id, nil
}

// initializeClient sets up configuration, formatter, and API client
func initializeClient(c *cli.Context) (*cliapi.Config, *cliapi.OutputFormatter, *cliapi.Client, error) {
	config, err := cliapi.LoadConfig(
		c.String("server"),
		c.String("format"),
		c.Bool("quiet"),
	)
	if err != nil {
		return nil, nil, nil, err
	}

	formatter := cliapi.NewOutputFormatter(config.Format, config.Quiet)
	client := cliapi.NewClientWithTimeout(config.ServerURL, config.RequestTimeout)

	// Test connectivity
	if err := client.HealthCheck(); err != nil {
		formatter.PrintError(err)
		return nil, nil, nil, err
	}

	return config, formatter, client, nil
}

func refreshShipment(c *cli.Context) error {
	config, formatter, client, err := initializeClient(c)
	if err != nil {
		return err
	}

	if c.NArg() != 1 {
		return cli.ShowCommandHelp(c, "refresh")
	}

	id, err := validateAndParseID(c.Args().Get(0), formatter)
	if err != nil {
		return err
	}

	verbose := c.Bool("verbose")

	if verbose {
		formatter.PrintInfo("Refreshing tracking data...")
	}

	response, err := client.RefreshShipment(id)
	if err != nil {
		formatter.PrintError(err)
		return err
	}

	if config.Quiet {
		// In quiet mode, just show the events
		return formatter.PrintEvents(response.Events)
	} else {
		// Show refresh details
		if verbose {
			formatter.PrintSuccess(fmt.Sprintf("Refresh completed successfully"))
			formatter.PrintInfo(fmt.Sprintf("Shipment ID: %d", response.ShipmentID))
			formatter.PrintInfo(fmt.Sprintf("Updated at: %s", response.UpdatedAt.Format("2006-01-02 15:04:05")))
			formatter.PrintInfo(fmt.Sprintf("Events added: %d", response.EventsAdded))
			formatter.PrintInfo(fmt.Sprintf("Total events: %d", response.TotalEvents))
			
			if response.EventsAdded > 0 {
				formatter.PrintInfo("New tracking events:")
			} else {
				formatter.PrintInfo("No new tracking events found")
			}
		} else {
			if response.EventsAdded > 0 {
				formatter.PrintSuccess(fmt.Sprintf("Refresh successful - %d new events found", response.EventsAdded))
			} else {
				formatter.PrintSuccess("Refresh successful - no new events")
			}
		}

		// Show all events
		return formatter.PrintEvents(response.Events)
	}
}