package carriers

// Placeholder constructors for scraping clients
// These will be implemented in separate files

// NewUSPSScrapingClient creates a new USPS web scraping client
func NewUSPSScrapingClient(userAgent string) Client {
	return &USPSScrapingClient{
		ScrapingClient: NewScrapingClient("usps", userAgent),
		baseURL:        "https://tools.usps.com",
	}
}

// NewUPSScrapingClient creates a new UPS web scraping client
func NewUPSScrapingClient(userAgent string) Client {
	return &UPSScrapingClient{
		ScrapingClient: NewScrapingClient("ups", userAgent),
		baseURL:        "https://www.ups.com",
	}
}

// NewFedExScrapingClient creates a new FedEx web scraping client
func NewFedExScrapingClient(userAgent string) Client {
	// TODO: Implement FedEx scraping client
	panic("FedEx scraping client not yet implemented")
}

// NewDHLScrapingClient creates a new DHL web scraping client
func NewDHLScrapingClient(userAgent string) Client {
	// TODO: Implement DHL scraping client
	panic("DHL scraping client not yet implemented")
}