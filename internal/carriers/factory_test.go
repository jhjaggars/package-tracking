package carriers

import (
	"testing"
)

func TestClientFactory_CreateClient_API(t *testing.T) {
	factory := NewClientFactory()
	
	// Test USPS API client creation
	factory.SetCarrierConfig("usps", &CarrierConfig{
		UserID:        "test_user_id",
		PreferredType: ClientTypeAPI,
	})
	
	client, clientType, err := factory.CreateClient("usps")
	if err != nil {
		t.Fatalf("Failed to create USPS client: %v", err)
	}
	
	if clientType != ClientTypeAPI {
		t.Errorf("Expected API client, got %s", clientType)
	}
	
	if client.GetCarrierName() != "usps" {
		t.Errorf("Expected carrier name 'usps', got '%s'", client.GetCarrierName())
	}
}

func TestClientFactory_CreateClient_FallbackToScraping(t *testing.T) {
	factory := NewClientFactory()
	
	// Test fallback to scraping when no API config
	factory.SetCarrierConfig("usps", &CarrierConfig{
		PreferredType: ClientTypeAPI, // Prefer API but no credentials
		UserAgent:     "test-agent",
	})
	
	client, clientType, err := factory.CreateClient("usps")
	if err != nil {
		t.Fatalf("Failed to create USPS scraping client: %v", err)
	}
	
	if clientType != ClientTypeScraping {
		t.Errorf("Expected scraping client, got %s", clientType)
	}
	
	if client.GetCarrierName() != "usps" {
		t.Errorf("Expected carrier name 'usps', got '%s'", client.GetCarrierName())
	}
}

func TestClientFactory_CreateClient_USPSMissingCredentials(t *testing.T) {
	factory := NewClientFactory()
	
	// Test USPS missing user ID - should fall back to scraping
	factory.SetCarrierConfig("usps", &CarrierConfig{
		PreferredType: ClientTypeAPI,
	})
	
	client, clientType, err := factory.CreateClient("usps")
	if err != nil {
		t.Fatalf("Failed to create USPS scraping fallback client: %v", err)
	}
	
	if clientType != ClientTypeScraping {
		t.Errorf("Expected scraping client as fallback, got %s", clientType)
	}
	
	if client.GetCarrierName() != "usps" {
		t.Errorf("Expected carrier name 'usps', got '%s'", client.GetCarrierName())
	}
}

func TestClientFactory_CreateClient_UPSMissingCredentials(t *testing.T) {
	factory := NewClientFactory()
	
	// Test UPS missing credentials - should fall back to scraping
	factory.SetCarrierConfig("ups", &CarrierConfig{
		PreferredType: ClientTypeAPI,
	})
	
	client, clientType, err := factory.CreateClient("ups")
	if err != nil {
		t.Fatalf("Failed to create UPS scraping fallback client: %v", err)
	}
	
	if clientType != ClientTypeScraping {
		t.Errorf("Expected scraping client as fallback, got %s", clientType)
	}
	
	if client.GetCarrierName() != "ups" {
		t.Errorf("Expected carrier name 'ups', got '%s'", client.GetCarrierName())
	}
}

func TestClientFactory_CreateClient_FedExMissingCredentials(t *testing.T) {
	factory := NewClientFactory()
	
	// Test FedEx missing credentials - should fall back to scraping
	factory.SetCarrierConfig("fedex", &CarrierConfig{
		PreferredType: ClientTypeAPI,
	})
	
	client, clientType, err := factory.CreateClient("fedex")
	if err != nil {
		t.Fatalf("Failed to create FedEx scraping fallback client: %v", err)
	}
	
	if clientType != ClientTypeScraping {
		t.Errorf("Expected scraping client as fallback, got %s", clientType)
	}
	
	if client.GetCarrierName() != "fedex" {
		t.Errorf("Expected carrier name 'fedex', got '%s'", client.GetCarrierName())
	}
}

func TestClientFactory_CreateClient_DHLMissingCredentials(t *testing.T) {
	factory := NewClientFactory()
	
	// Test DHL missing credentials - should fall back to scraping
	factory.SetCarrierConfig("dhl", &CarrierConfig{
		PreferredType: ClientTypeAPI,
	})
	
	client, clientType, err := factory.CreateClient("dhl")
	if err != nil {
		t.Fatalf("Failed to create DHL scraping fallback client: %v", err)
	}
	
	if clientType != ClientTypeScraping {
		t.Errorf("Expected scraping client as fallback, got %s", clientType)
	}
	
	if client.GetCarrierName() != "dhl" {
		t.Errorf("Expected carrier name 'dhl', got '%s'", client.GetCarrierName())
	}
}

func TestClientFactory_CreateClient_UPS(t *testing.T) {
	factory := NewClientFactory()
	
	// Test UPS API client creation
	factory.SetCarrierConfig("ups", &CarrierConfig{
		ClientID:      "test_client_id",
		ClientSecret:  "test_client_secret",
		PreferredType: ClientTypeAPI,
	})
	
	client, clientType, err := factory.CreateClient("ups")
	if err != nil {
		t.Fatalf("Failed to create UPS client: %v", err)
	}
	
	if clientType != ClientTypeAPI {
		t.Errorf("Expected API client, got %s", clientType)
	}
	
	if client.GetCarrierName() != "ups" {
		t.Errorf("Expected carrier name 'ups', got '%s'", client.GetCarrierName())
	}
}

func TestClientFactory_CreateClient_FedEx(t *testing.T) {
	factory := NewClientFactory()
	
	// Test FedEx API client creation
	factory.SetCarrierConfig("fedex", &CarrierConfig{
		ClientID:      "test_client_id",
		ClientSecret:  "test_client_secret",
		PreferredType: ClientTypeAPI,
	})
	
	client, clientType, err := factory.CreateClient("fedex")
	if err != nil {
		t.Fatalf("Failed to create FedEx client: %v", err)
	}
	
	if clientType != ClientTypeAPI {
		t.Errorf("Expected API client, got %s", clientType)
	}
	
	if client.GetCarrierName() != "fedex" {
		t.Errorf("Expected carrier name 'fedex', got '%s'", client.GetCarrierName())
	}
}

func TestClientFactory_CreateClient_DHL(t *testing.T) {
	factory := NewClientFactory()
	
	// Test DHL API client creation
	factory.SetCarrierConfig("dhl", &CarrierConfig{
		APIKey:        "test_api_key",
		PreferredType: ClientTypeAPI,
	})
	
	client, clientType, err := factory.CreateClient("dhl")
	if err != nil {
		t.Fatalf("Failed to create DHL client: %v", err)
	}
	
	if clientType != ClientTypeAPI {
		t.Errorf("Expected API client, got %s", clientType)
	}
	
	if client.GetCarrierName() != "dhl" {
		t.Errorf("Expected carrier name 'dhl', got '%s'", client.GetCarrierName())
	}
}

func TestClientFactory_CreateClient_MissingCredentials(t *testing.T) {
	factory := NewClientFactory()
	
	tests := []struct {
		name    string
		carrier string
		config  *CarrierConfig
	}{
		{
			name:    "UPS missing credentials",
			carrier: "ups",
			config: &CarrierConfig{
				ClientID:      "test_id", // Missing ClientSecret
				PreferredType: ClientTypeAPI,
			},
		},
		{
			name:    "FedEx missing credentials",
			carrier: "fedex",
			config: &CarrierConfig{
				ClientSecret:  "test_secret", // Missing ClientID
				PreferredType: ClientTypeAPI,
			},
		},
		{
			name:    "DHL missing API key",
			carrier: "dhl",
			config: &CarrierConfig{
				PreferredType: ClientTypeAPI,
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory.SetCarrierConfig(tt.carrier, tt.config)
			
			if tt.carrier == "ups" || tt.carrier == "fedex" || tt.carrier == "dhl" {
				// UPS, FedEx, and DHL should fall back to scraping successfully
				client, clientType, err := factory.CreateClient(tt.carrier)
				if err != nil {
					t.Fatalf("Failed to create %s scraping fallback client: %v", tt.carrier, err)
				}
				
				if clientType != ClientTypeScraping {
					t.Errorf("Expected scraping client as fallback, got %s", clientType)
				}
				
				if client.GetCarrierName() != tt.carrier {
					t.Errorf("Expected carrier name '%s', got '%s'", tt.carrier, client.GetCarrierName())
				}
			} else {
				// Other carriers should still panic since scraping clients aren't implemented yet
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic for missing credentials and unimplemented scraping")
					}
				}()
				
				factory.CreateClient(tt.carrier)
			}
		})
	}
}

func TestClientFactory_IsAPIConfigured(t *testing.T) {
	factory := NewClientFactory()
	
	// Test with no configuration
	if factory.IsAPIConfigured("usps") {
		t.Error("Expected USPS API to not be configured")
	}
	
	// Test with proper USPS configuration
	factory.SetCarrierConfig("usps", &CarrierConfig{
		UserID: "test_user_id",
	})
	
	if !factory.IsAPIConfigured("usps") {
		t.Error("Expected USPS API to be configured")
	}
	
	// Test with proper UPS configuration
	factory.SetCarrierConfig("ups", &CarrierConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
	})
	
	if !factory.IsAPIConfigured("ups") {
		t.Error("Expected UPS API to be configured")
	}
	
	// Test with incomplete UPS configuration
	factory.SetCarrierConfig("ups", &CarrierConfig{
		ClientID: "test_client_id", // Missing ClientSecret
	})
	
	if factory.IsAPIConfigured("ups") {
		t.Error("Expected UPS API to not be configured with incomplete credentials")
	}
}

func TestClientFactory_GetAvailableCarriers(t *testing.T) {
	factory := NewClientFactory()
	carriers := factory.GetAvailableCarriers()
	
	expected := []string{"usps", "ups", "fedex", "dhl"}
	
	if len(carriers) != len(expected) {
		t.Errorf("Expected %d carriers, got %d", len(expected), len(carriers))
	}
	
	for _, expectedCarrier := range expected {
		found := false
		for _, carrier := range carriers {
			if carrier == expectedCarrier {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected carrier '%s' not found in available carriers", expectedCarrier)
		}
	}
}

func TestClientFactory_CreateClient_UnsupportedCarrier(t *testing.T) {
	factory := NewClientFactory()
	
	_, _, err := factory.CreateClient("unsupported")
	if err == nil {
		t.Error("Expected error for unsupported carrier")
	}
	
	if err.Error() != "failed to create client for unsupported: unsupported carrier for scraping: unsupported" {
		t.Errorf("Expected 'unsupported carrier' error, got: %v", err)
	}
}