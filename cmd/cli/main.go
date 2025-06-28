package main

import (
	"log"
	"os"
	"strconv"

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

	id, err := strconv.Atoi(c.Args().Get(0))
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

func updateShipment(c *cli.Context) error {
	config, formatter, client, err := initializeClient(c)
	if err != nil {
		return err
	}

	if c.NArg() != 1 {
		return cli.ShowCommandHelp(c, "update")
	}

	id, err := strconv.Atoi(c.Args().Get(0))
	if err != nil {
		formatter.PrintError(err)
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

	id, err := strconv.Atoi(c.Args().Get(0))
	if err != nil {
		formatter.PrintError(err)
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

	id, err := strconv.Atoi(c.Args().Get(0))
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
	client := cliapi.NewClient(config.ServerURL)

	// Test connectivity
	if err := client.HealthCheck(); err != nil {
		formatter.PrintError(err)
		return nil, nil, nil, err
	}

	return config, formatter, client, nil
}