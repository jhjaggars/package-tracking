package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	cliapi "package-tracking/internal/cli"
)

var (
	serverURL       string
	format          string
	quiet           bool
	noColor         bool
	skipHealthCheck bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "package-tracker",
	Short: "CLI client for package tracking API",
	Long: `Package Tracker CLI allows you to manage and track shipments through 
a REST API. You can add new shipments, list existing ones, update descriptions,
and view tracking events.`,
	Version:                "1.0.0",
	SuggestionsMinimumDistance: 2,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	fang.Execute(context.Background(), rootCmd)
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&serverURL, "server", "s", "", "API server address")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "", "Output format (table, json)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Quiet mode (minimal output)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color output")
	rootCmd.PersistentFlags().BoolVar(&skipHealthCheck, "skip-health-check", false, "Skip API health check for faster execution")
}

// initConfig initializes configuration and environment variable binding
func initConfig() {
	// Set defaults and bind environment variables
	if serverURL == "" {
		serverURL = getEnvOrDefault("PACKAGE_TRACKER_SERVER", "http://localhost:8080")
	}
	if format == "" {
		format = getEnvOrDefault("PACKAGE_TRACKER_FORMAT", "table")
	}
	
	// Handle boolean environment variables
	if os.Getenv("PACKAGE_TRACKER_QUIET") == "true" && !rootCmd.PersistentFlags().Changed("quiet") {
		quiet = true
	}
	if (os.Getenv("NO_COLOR") != "" || os.Getenv("PACKAGE_TRACKER_NO_COLOR") == "true") && !rootCmd.PersistentFlags().Changed("no-color") {
		noColor = true
	}
	if os.Getenv("PACKAGE_TRACKER_SKIP_HEALTH_CHECK") == "true" && !rootCmd.PersistentFlags().Changed("skip-health-check") {
		skipHealthCheck = true
	}
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(envVar, defaultVal string) string {
	if val := os.Getenv(envVar); val != "" {
		return val
	}
	return defaultVal
}

// initializeClient sets up configuration, formatter, and API client
func initializeClient() (*cliapi.Config, *cliapi.OutputFormatter, *cliapi.Client, error) {
	config, err := cliapi.LoadConfig(serverURL, format, quiet)
	if err != nil {
		return nil, nil, nil, err
	}

	formatter := cliapi.NewOutputFormatterWithColor(config.Format, config.Quiet, noColor)
	client := cliapi.NewClientWithTimeout(config.ServerURL, config.RequestTimeout)

	// Test connectivity (unless skipped for performance)
	if !skipHealthCheck {
		if err := client.HealthCheck(); err != nil {
			formatter.PrintError(err)
			return nil, nil, nil, err
		}
	}

	return config, formatter, client, nil
}