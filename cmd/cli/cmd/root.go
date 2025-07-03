package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	cliapi "package-tracking/internal/cli"
)

var (
	serverURL string
	format    string
	quiet     bool
	noColor   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "package-tracker",
	Short: "CLI client for package tracking API",
	Long: `Package Tracker CLI allows you to manage and track shipments through 
a REST API. You can add new shipments, list existing ones, update descriptions,
and view tracking events.`,
	Version: "1.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	fang.Execute(context.Background(), rootCmd)
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&serverURL, "server", "s", "http://localhost:8080", "API server address")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "table", "Output format (table, json)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Quiet mode (minimal output)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color output")

	// Bind environment variables
	rootCmd.PersistentFlags().Lookup("server").DefValue = getEnvOrDefault("PACKAGE_TRACKER_SERVER", "http://localhost:8080")
	rootCmd.PersistentFlags().Lookup("format").DefValue = getEnvOrDefault("PACKAGE_TRACKER_FORMAT", "table")
	rootCmd.PersistentFlags().Lookup("quiet").DefValue = getEnvOrDefault("PACKAGE_TRACKER_QUIET", "false")
	rootCmd.PersistentFlags().Lookup("no-color").DefValue = getEnvOrDefault("NO_COLOR", "false")
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

	// Test connectivity
	if err := client.HealthCheck(); err != nil {
		formatter.PrintError(err)
		return nil, nil, nil, err
	}

	return config, formatter, client, nil
}