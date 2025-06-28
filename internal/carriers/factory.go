package carriers

import (
	"fmt"
	"strings"
)

// ClientType represents the type of carrier client
type ClientType string

const (
	ClientTypeAPI       ClientType = "api"
	ClientTypeScraping  ClientType = "scraping"
	ClientTypeHeadless  ClientType = "headless"
)

// CarrierConfig holds configuration for carrier clients
type CarrierConfig struct {
	// API credentials
	APIKey       string
	ClientID     string
	ClientSecret string
	UserID       string
	
	// Scraping configuration
	UserAgent    string
	UseSandbox   bool
	
	// Headless browser configuration
	UseHeadless  bool
	
	// Preferred client type (can be overridden by availability)
	PreferredType ClientType
}

// ClientFactory creates carrier clients with automatic fallback
type ClientFactory struct {
	configs map[string]*CarrierConfig
}

// NewClientFactory creates a new client factory
func NewClientFactory() *ClientFactory {
	return &ClientFactory{
		configs: make(map[string]*CarrierConfig),
	}
}

// SetCarrierConfig sets configuration for a specific carrier
func (f *ClientFactory) SetCarrierConfig(carrier string, config *CarrierConfig) {
	f.configs[strings.ToLower(carrier)] = config
}

// CreateClient creates the appropriate client for a carrier
func (f *ClientFactory) CreateClient(carrier string) (Client, ClientType, error) {
	carrier = strings.ToLower(carrier)
	config := f.configs[carrier]
	
	// If no config exists, create default scraping config
	if config == nil {
		config = &CarrierConfig{
			PreferredType: ClientTypeScraping,
			UserAgent:     "Mozilla/5.0 (compatible; PackageTracker/1.0)",
		}
	}
	
	// Try to create API client first if credentials are available
	if config.PreferredType == ClientTypeAPI || config.PreferredType == "" {
		if apiClient, err := f.createAPIClient(carrier, config); err == nil {
			return apiClient, ClientTypeAPI, nil
		}
	}
	
	// Try headless client if requested or needed for specific carriers
	if config.PreferredType == ClientTypeHeadless || config.UseHeadless || f.requiresHeadless(carrier) {
		if headlessClient, err := f.createHeadlessClient(carrier, config); err == nil {
			return headlessClient, ClientTypeHeadless, nil
		}
	}
	
	// Fall back to scraping client
	scrapingClient, err := f.createScrapingClient(carrier, config)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create client for %s: %w", carrier, err)
	}
	
	return scrapingClient, ClientTypeScraping, nil
}

// createAPIClient creates an API client if credentials are available
func (f *ClientFactory) createAPIClient(carrier string, config *CarrierConfig) (Client, error) {
	switch carrier {
	case "usps":
		if config.UserID == "" {
			return nil, fmt.Errorf("USPS User ID not configured")
		}
		return NewUSPSClient(config.UserID, config.UseSandbox), nil
		
	case "ups":
		if config.ClientID == "" || config.ClientSecret == "" {
			return nil, fmt.Errorf("UPS Client ID/Secret not configured")
		}
		return NewUPSClient(config.ClientID, config.ClientSecret, config.UseSandbox), nil
		
	case "fedex":
		if config.ClientID == "" || config.ClientSecret == "" {
			return nil, fmt.Errorf("FedEx Client ID/Secret not configured")
		}
		return NewFedExClient(config.ClientID, config.ClientSecret, config.UseSandbox), nil
		
	case "dhl":
		if config.APIKey == "" {
			return nil, fmt.Errorf("DHL API Key not configured")
		}
		return NewDHLClient(config.APIKey, config.UseSandbox), nil
		
	default:
		return nil, fmt.Errorf("unsupported carrier: %s", carrier)
	}
}

// createScrapingClient creates a web scraping client
func (f *ClientFactory) createScrapingClient(carrier string, config *CarrierConfig) (Client, error) {
	userAgent := config.UserAgent
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (compatible; PackageTracker/1.0)"
	}
	
	switch carrier {
	case "usps":
		return NewUSPSScrapingClient(userAgent), nil
	case "ups":
		return NewUPSScrapingClient(userAgent), nil
	case "fedex":
		return NewFedExScrapingClient(userAgent), nil
	case "dhl":
		return NewDHLScrapingClient(userAgent), nil
	default:
		return nil, fmt.Errorf("unsupported carrier for scraping: %s", carrier)
	}
}

// createHeadlessClient creates a headless browser client
func (f *ClientFactory) createHeadlessClient(carrier string, config *CarrierConfig) (Client, error) {
	switch carrier {
	case "fedex":
		return NewFedExHeadlessClient(), nil
	// Other carriers can be added here as they get headless implementations
	// case "ups":
	//     return NewUPSHeadlessClient(), nil
	default:
		return nil, fmt.Errorf("headless client not available for carrier: %s", carrier)
	}
}

// requiresHeadless returns true for carriers that require headless browsing
func (f *ClientFactory) requiresHeadless(carrier string) bool {
	switch carrier {
	case "fedex":
		return true // FedEx now requires headless due to SPA
	default:
		return false
	}
}

// GetAvailableCarriers returns a list of supported carriers
func (f *ClientFactory) GetAvailableCarriers() []string {
	return []string{"usps", "ups", "fedex", "dhl"}
}

// IsAPIConfigured checks if API credentials are configured for a carrier
func (f *ClientFactory) IsAPIConfigured(carrier string) bool {
	config := f.configs[strings.ToLower(carrier)]
	if config == nil {
		return false
	}
	
	switch strings.ToLower(carrier) {
	case "usps":
		return config.UserID != ""
	case "ups", "fedex":
		return config.ClientID != "" && config.ClientSecret != ""
	case "dhl":
		return config.APIKey != ""
	default:
		return false
	}
}