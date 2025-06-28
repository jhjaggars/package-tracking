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
	return &FedExScrapingClient{
		ScrapingClient: NewScrapingClient("fedex", userAgent),
		baseURL:        "https://www.fedex.com",
	}
}

// NewDHLScrapingClient creates a new DHL web scraping client
func NewDHLScrapingClient(userAgent string) Client {
	return &DHLScrapingClient{
		ScrapingClient: NewScrapingClient("dhl", userAgent),
		baseURL:        "https://www.dhl.com",
	}
}