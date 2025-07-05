package carriers

import (
	"context"
	"regexp"
	"strings"
	"time"
)

// Pre-compiled regular expressions for better performance
var (
	letterRegex = regexp.MustCompile(`[A-Za-z]`)
	numberRegex = regexp.MustCompile(`\d`)
)

// AmazonClient implements the Client interface for Amazon shipments
// Since Amazon doesn't provide public APIs, this client handles:
// 1. Amazon order numbers (###-#######-#######)
// 2. Amazon Logistics tracking numbers (TBA############)
// 3. Delegation to other carriers when Amazon uses UPS/FedEx/USPS/DHL
type AmazonClient struct {
	factory   *ClientFactory
	rateLimit *RateLimitInfo
}

// NewAmazonClient creates a new Amazon client
func NewAmazonClient(factory *ClientFactory) *AmazonClient {
	return &AmazonClient{
		factory: factory,
		rateLimit: &RateLimitInfo{
			Limit:     -1, // No rate limits for Amazon (email-based)
			Remaining: -1,
			ResetTime: time.Now().Add(24 * time.Hour),
		},
	}
}

// GetCarrierName returns the carrier name
func (c *AmazonClient) GetCarrierName() string {
	return "amazon"
}

// ValidateTrackingNumber validates Amazon tracking number formats
func (c *AmazonClient) ValidateTrackingNumber(trackingNumber string) bool {
	if trackingNumber == "" {
		return false
	}

	// Clean the tracking number
	cleaned := strings.ReplaceAll(trackingNumber, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	// Check for Amazon order number format: ###-#######-#######
	// After cleaning: 17 digits total
	if c.isAmazonOrderNumber(cleaned) {
		return true
	}

	// Check for Amazon Logistics tracking number: TBA############
	if c.isAmazonLogisticsNumber(trackingNumber) {
		return true
	}

	// Check for Amazon internal reference codes (flexible validation)
	if c.isAmazonInternalReference(trackingNumber) {
		return true
	}

	return false
}

// isAmazonOrderNumber checks if the cleaned string is a valid Amazon order number
func (c *AmazonClient) isAmazonOrderNumber(cleaned string) bool {
	// Amazon order numbers are 17 digits after removing dashes
	if len(cleaned) != 17 {
		return false
	}

	// Must be all digits
	if matched, _ := regexp.MatchString(`^\d{17}$`, cleaned); !matched {
		return false
	}

	return true
}

// isAmazonLogisticsNumber checks if the string is a valid Amazon Logistics tracking number
func (c *AmazonClient) isAmazonLogisticsNumber(trackingNumber string) bool {
	// Amazon Logistics format: TBA followed by 12 digits
	// Case insensitive
	if matched, _ := regexp.MatchString(`^(?i)TBA\d{12}$`, trackingNumber); matched {
		return true
	}

	return false
}

// isAmazonInternalReference checks if the string is a valid Amazon internal reference code
func (c *AmazonClient) isAmazonInternalReference(trackingNumber string) bool {
	// Amazon internal reference codes can have various formats:
	// 1. Mixed alphanumeric codes (e.g., "BqPz3RXRS")
	// 2. Amazon internal shipment IDs
	// 3. Amazon fulfillment center codes

	// Length constraints: Amazon internal codes are typically 6-20 characters
	if len(trackingNumber) < 6 || len(trackingNumber) > 20 {
		return false
	}

	// Must contain at least one letter and one number (mixed alphanumeric)
	hasLetter := letterRegex.MatchString(trackingNumber)
	hasNumber := numberRegex.MatchString(trackingNumber)

	if !hasLetter || !hasNumber {
		return false
	}

	// Must be alphanumeric only (no special characters except dashes)
	if matched, _ := regexp.MatchString(`^[A-Za-z0-9-]+$`, trackingNumber); !matched {
		return false
	}

	// Exclude known carrier patterns that should not be treated as Amazon internal codes
	knownCarrierPatterns := []string{
		`^1Z[A-Z0-9]{6}\d{2}\d{7}$`, // UPS format
		`^94\d{20}$`,                // USPS format
		`^\d{12}$`,                  // FedEx format
		`^\d{10,11}$`,               // DHL format
		`^TBA\d{12}$`,               // Amazon Logistics (handled separately)
		`^\d{3}-\d{7}-\d{7}$`,       // Amazon order format (handled separately)
		`^[A-Z]{3}\d{12}$`,          // Generic carrier format like ABC123456789012
		`^TBA\d{1,11}$`,             // Incomplete TBA formats (too short)
	}

	for _, pattern := range knownCarrierPatterns {
		if matched, _ := regexp.MatchString(pattern, trackingNumber); matched {
			return false
		}
	}

	// Exclude common false positives like email addresses, URLs, etc.
	falsePositives := []string{
		`@`, `\.com`, `\.org`, `\.net`, `http`, `www\.`, `mailto:`,
		`^(19|20)\d{2}$`,                                     // Years
		`^(mon|tue|wed|thu|fri|sat|sun)`,                     // Days
		`^(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)`, // Months
		`^invalid`,                                           // Obviously invalid patterns
		`^test\d+$`,                                          // Test patterns
		`^fake\d+$`,                                          // Fake patterns
		`^example`,                                           // Example patterns
	}
	for _, pattern := range falsePositives {
		if matched, _ := regexp.MatchString(pattern, strings.ToLower(trackingNumber)); matched {
			return false
		}
	}

	return true
}

// GetRateLimit returns current rate limit information
func (c *AmazonClient) GetRateLimit() *RateLimitInfo {
	return c.rateLimit
}

// Track retrieves tracking information for Amazon shipments
// Since Amazon doesn't provide public APIs, this method:
// 1. Validates tracking numbers
// 2. Returns basic tracking info with pre_ship status
// 3. In the future, will handle delegation to other carriers
func (c *AmazonClient) Track(ctx context.Context, req *TrackingRequest) (*TrackingResponse, error) {
	if len(req.TrackingNumbers) == 0 {
		return &TrackingResponse{
			Results: []TrackingInfo{},
			Errors:  []CarrierError{},
		}, nil
	}

	var results []TrackingInfo
	var errors []CarrierError

	for _, trackingNumber := range req.TrackingNumbers {
		if !c.ValidateTrackingNumber(trackingNumber) {
			errors = append(errors, CarrierError{
				Carrier:   "amazon",
				Code:      "INVALID_TRACKING_NUMBER",
				Message:   "Invalid Amazon tracking number format: " + trackingNumber,
				Retryable: false,
				RateLimit: false,
			})
			continue
		}

		// Create basic tracking info
		trackingInfo := c.createBasicTrackingInfo(trackingNumber)
		results = append(results, trackingInfo)
	}

	return &TrackingResponse{
		Results:   results,
		Errors:    errors,
		RateLimit: c.rateLimit,
	}, nil
}

// createBasicTrackingInfo creates a basic tracking info structure for Amazon shipments
func (c *AmazonClient) createBasicTrackingInfo(trackingNumber string) TrackingInfo {
	now := time.Now()

	// Determine the type of Amazon tracking number
	var serviceType string
	var description string

	if c.isAmazonLogisticsNumber(trackingNumber) {
		serviceType = "Amazon Logistics"
		description = "Amazon Logistics shipment created"
	} else if c.isAmazonOrderNumber(strings.ReplaceAll(strings.ReplaceAll(trackingNumber, " ", ""), "-", "")) {
		serviceType = "Amazon Standard"
		description = "Amazon order received"
	} else if c.isAmazonInternalReference(trackingNumber) {
		serviceType = "Amazon Internal"
		description = "Amazon shipment reference created"
	} else {
		serviceType = "Amazon Standard"
		description = "Amazon order received"
	}

	// Create initial tracking event
	event := TrackingEvent{
		Timestamp:   now,
		Status:      StatusPreShip,
		Location:    "",
		Description: description,
		Details:     "Tracking information will be updated when shipment is processed",
	}

	return TrackingInfo{
		TrackingNumber:    trackingNumber,
		Carrier:           "amazon",
		Status:            StatusPreShip,
		EstimatedDelivery: nil,
		ActualDelivery:    nil,
		Events:            []TrackingEvent{event},
		ServiceType:       serviceType,
		Weight:            "",
		Dimensions:        "",
		LastUpdated:       now,
	}
}

// DelegateToCarrier handles delegation to other carriers when Amazon uses third-party delivery
// This method would be called when the system detects that Amazon has delegated
// a shipment to UPS, FedEx, USPS, or DHL
func (c *AmazonClient) DelegateToCarrier(ctx context.Context, carrier string, trackingNumber string) (*TrackingInfo, error) {
	// Create the appropriate carrier client
	delegatedClient, _, err := c.factory.CreateClient(carrier)
	if err != nil {
		return nil, err
	}

	// Track using the delegated carrier
	req := &TrackingRequest{
		TrackingNumbers: []string{trackingNumber},
		Carrier:         carrier,
	}

	resp, err := delegatedClient.Track(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.Results) == 0 {
		return nil, &CarrierError{
			Carrier:   "amazon",
			Code:      "DELEGATION_FAILED",
			Message:   "No results from delegated carrier " + carrier,
			Retryable: true,
			RateLimit: false,
		}
	}

	// Return the first result from the delegated carrier
	// The calling code will update the Amazon shipment with this information
	result := resp.Results[0]
	return &result, nil
}
