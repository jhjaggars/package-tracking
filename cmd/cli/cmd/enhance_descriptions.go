package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"package-tracking/internal/carriers"
	"package-tracking/internal/config"
	"package-tracking/internal/database"
	"package-tracking/internal/parser"
	"package-tracking/internal/services"
)

var enhanceDescriptionsCmd = &cobra.Command{
	Use:   "enhance-descriptions",
	Short: "Retroactively enhance shipment descriptions using email content and LLM extraction",
	Long: `This command improves existing shipment descriptions by:
1. Finding emails associated with tracking numbers
2. Using LLM extraction to get enhanced product descriptions
3. Updating shipment records with improved descriptions

This is useful for fixing shipments that have poor descriptions like "Package from " or empty descriptions.`,
	RunE: runEnhanceDescriptions,
}

var (
	enhanceAll        bool
	enhanceShipmentID int
	enhanceLimit      int
	enhanceDryRun     bool
	enhanceFormat     string
	enhanceAssociate  bool
)

func init() {
	enhanceDescriptionsCmd.Flags().BoolVar(&enhanceAll, "all", false, "Process all shipments with poor descriptions")
	enhanceDescriptionsCmd.Flags().IntVar(&enhanceShipmentID, "shipment-id", 0, "Process a specific shipment by ID")
	enhanceDescriptionsCmd.Flags().IntVar(&enhanceLimit, "limit", 0, "Limit number of shipments to process (0 = no limit)")
	enhanceDescriptionsCmd.Flags().BoolVar(&enhanceDryRun, "dry-run", false, "Show what would be changed without making updates")
	enhanceDescriptionsCmd.Flags().StringVar(&enhanceFormat, "format", "table", "Output format: table, json")
	enhanceDescriptionsCmd.Flags().BoolVar(&enhanceAssociate, "associate", false, "First associate existing emails with shipments")

	rootCmd.AddCommand(enhanceDescriptionsCmd)
}

func runEnhanceDescriptions(cmd *cobra.Command, args []string) error {
	// Validate flags
	if !enhanceAll && enhanceShipmentID == 0 {
		return fmt.Errorf("must specify either --all or --shipment-id")
	}

	if enhanceAll && enhanceShipmentID != 0 {
		return fmt.Errorf("cannot specify both --all and --shipment-id")
	}

	// Load server configuration to get database path
	cfg, err := config.LoadServerConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Setup database
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Get stores from the database struct
	shipmentStore := db.Shipments
	emailStore := db.Emails

	// Setup LLM extractor - try to load LLM config from environment
	carrierFactory := &carriers.ClientFactory{}
	
	// Check if LLM is enabled via environment variables
	llmEnabled := os.Getenv("LLM_ENABLED") == "true"
	var llmConfig *parser.LLMConfig
	
	if llmEnabled {
		// Load LLM configuration from environment variables
		timeout, _ := time.ParseDuration(getEnvOrDefault("LLM_TIMEOUT", "120s"))
		maxTokens := 1000
		if tokens := os.Getenv("LLM_MAX_TOKENS"); tokens != "" {
			if parsed, err := strconv.Atoi(tokens); err == nil {
				maxTokens = parsed
			}
		}
		temperature := 0.1
		if temp := os.Getenv("LLM_TEMPERATURE"); temp != "" {
			if parsed, err := strconv.ParseFloat(temp, 64); err == nil {
				temperature = parsed
			}
		}
		retryCount := 2
		if retries := os.Getenv("LLM_RETRY_COUNT"); retries != "" {
			if parsed, err := strconv.Atoi(retries); err == nil {
				retryCount = parsed
			}
		}
		
		llmConfig = &parser.LLMConfig{
			Provider:    getEnvOrDefault("LLM_PROVIDER", "disabled"),
			Model:       getEnvOrDefault("LLM_MODEL", ""),
			APIKey:      os.Getenv("LLM_API_KEY"),
			Endpoint:    os.Getenv("LLM_ENDPOINT"),
			MaxTokens:   maxTokens,
			Temperature: temperature,
			Timeout:     timeout,
			RetryCount:  retryCount,
			Enabled:     true,
		}
	}

	extractorConfig := &parser.ExtractorConfig{
		EnableLLM:           llmEnabled,
		MinConfidence:       0.5,
		MaxCandidates:       10,
		UseHybridValidation: true,
		DebugMode:           false,
	}
	extractor := parser.NewTrackingExtractor(carrierFactory, extractorConfig, llmConfig)

	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Create description enhancer service
	enhancer := services.NewDescriptionEnhancer(shipmentStore, emailStore, extractor, logger)

	// Associate emails with shipments if requested
	if enhanceAssociate {
		fmt.Println("Associating existing emails with shipments...")
		err := enhancer.AssociateEmailsWithShipments()
		if err != nil {
			return fmt.Errorf("failed to associate emails with shipments: %w", err)
		}
		fmt.Println("Email-shipment association completed.")
		
		// If only association was requested, exit
		if !enhanceAll && enhanceShipmentID == 0 {
			return nil
		}
	}

	// Run enhancement operation
	if enhanceShipmentID != 0 {
		// Process specific shipment
		result, err := enhancer.EnhanceSpecificShipment(enhanceShipmentID, enhanceDryRun)
		if err != nil {
			return fmt.Errorf("failed to enhance shipment %d: %w", enhanceShipmentID, err)
		}

		if enhanceFormat == "json" {
			output, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(output))
		} else {
			printSingleResult(*result, enhanceDryRun)
		}
	} else {
		// Process all shipments with poor descriptions
		summary, err := enhancer.EnhanceAllShipmentsWithPoorDescriptions(enhanceLimit, enhanceDryRun)
		if err != nil {
			return fmt.Errorf("failed to enhance shipments: %w", err)
		}

		if enhanceFormat == "json" {
			output, _ := json.MarshalIndent(summary, "", "  ")
			fmt.Println(string(output))
		} else {
			printSummary(summary, enhanceDryRun)
		}
	}

	return nil
}

func printSingleResult(result services.DescriptionEnhancementResult, dryRun bool) {
	action := "Updated"
	if dryRun {
		action = "Would update"
	}

	fmt.Printf("Shipment Enhancement Result:\n")
	fmt.Printf("  Shipment ID: %d\n", result.ShipmentID)
	fmt.Printf("  Tracking Number: %s\n", result.TrackingNumber)
	fmt.Printf("  Emails Found: %d\n", result.EmailsFound)
	
	if result.Success {
		fmt.Printf("  Status: ✅ Success\n")
		fmt.Printf("  Old Description: %q\n", result.OldDescription)
		fmt.Printf("  New Description: %q\n", result.NewDescription)
		if result.NewDescription != result.OldDescription && result.NewDescription != "" {
			fmt.Printf("  %s description successfully\n", action)
		} else {
			fmt.Printf("  No change needed\n")
		}
	} else {
		fmt.Printf("  Status: ❌ Failed\n")
		fmt.Printf("  Error: %s\n", result.Error)
	}
	fmt.Printf("  Processed At: %s\n", result.ProcessedAt.Format(time.RFC3339))
}

func printSummary(summary *services.DescriptionEnhancementSummary, dryRun bool) {
	action := "Enhancement"
	if dryRun {
		action = "Dry Run Enhancement"
	}

	fmt.Printf("%s Summary:\n", action)
	fmt.Printf("  Total Shipments: %d\n", summary.TotalShipments)
	fmt.Printf("  Successful: %d\n", summary.SuccessCount)
	fmt.Printf("  Failed: %d\n", summary.FailureCount)
	fmt.Printf("  Processing Time: %v\n", summary.ProcessingTime)
	fmt.Printf("  Started: %s\n", summary.StartedAt.Format(time.RFC3339))
	fmt.Printf("  Completed: %s\n", summary.CompletedAt.Format(time.RFC3339))
	fmt.Println()

	if len(summary.Results) > 0 {
		fmt.Println("Individual Results:")
		for i, result := range summary.Results {
			status := "✅"
			if !result.Success {
				status = "❌"
			}
			
			fmt.Printf("  %d. %s Shipment %d (%s)\n", i+1, status, result.ShipmentID, result.TrackingNumber)
			if result.Success && result.NewDescription != result.OldDescription && result.NewDescription != "" {
				fmt.Printf("      %q → %q\n", result.OldDescription, result.NewDescription)
			} else if !result.Success {
				fmt.Printf("      Error: %s\n", result.Error)
			} else {
				fmt.Printf("      No change needed\n")
			}
		}
	}

	if dryRun {
		fmt.Println("\nNote: This was a dry run. No changes were made to the database.")
		fmt.Println("Run without --dry-run to actually update descriptions.")
	}
}