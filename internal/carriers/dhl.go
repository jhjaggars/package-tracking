package carriers

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// DHLClient implements the Client interface for DHL API
type DHLClient struct {
	apiKey   string
	baseURL  string
	client   *http.Client
	rateLimit *RateLimitInfo
}

// NewDHLClient creates a new DHL API client
func NewDHLClient(apiKey string, useSandbox bool) *DHLClient {
	baseURL := "https://api-eu.dhl.com"
	if useSandbox {
		baseURL = "https://api-sandbox.dhl.com"
	}
	
	return &DHLClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
		rateLimit: &RateLimitInfo{
			Limit:     250, // DHL initial limit: 250 calls per day
			Remaining: 250,
			ResetTime: time.Now().Add(24 * time.Hour),
		},
	}
}

// GetCarrierName returns the carrier name
func (c *DHLClient) GetCarrierName() string {
	return "dhl"
}

// ValidateTrackingNumber validates DHL tracking number formats
func (c *DHLClient) ValidateTrackingNumber(trackingNumber string) bool {
	if trackingNumber == "" {
		return false
	}
	
	// Remove spaces and keep only digits
	cleaned := strings.ReplaceAll(trackingNumber, " ", "")
	
	// Check if it's all digits
	if matched, _ := regexp.MatchString(`^\d+$`, cleaned); !matched {
		return false
	}
	
	// DHL tracking numbers are typically 10 or 11 digits
	return len(cleaned) == 10 || len(cleaned) == 11
}

// GetRateLimit returns current rate limit information
func (c *DHLClient) GetRateLimit() *RateLimitInfo {
	return c.rateLimit
}

// Track retrieves tracking information for the given tracking numbers
func (c *DHLClient) Track(ctx context.Context, req *TrackingRequest) (*TrackingResponse, error) {
	// TODO: Implement DHL tracking API call
	// This is a stub for now
	var results []TrackingInfo
	
	for _, trackingNumber := range req.TrackingNumbers {
		results = append(results, TrackingInfo{
			TrackingNumber: trackingNumber,
			Carrier:        "dhl",
			Status:         StatusUnknown,
			Events:         []TrackingEvent{},
			LastUpdated:    time.Now(),
		})
	}
	
	return &TrackingResponse{
		Results:   results,
		Errors:    []CarrierError{},
		RateLimit: c.rateLimit,
	}, nil
}