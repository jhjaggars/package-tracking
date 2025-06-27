package carriers

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// FedExClient implements the Client interface for FedEx API
type FedExClient struct {
	clientID     string
	clientSecret string
	baseURL      string
	client       *http.Client
	accessToken  string
	tokenExpiry  time.Time
	rateLimit    *RateLimitInfo
}

// NewFedExClient creates a new FedEx API client
func NewFedExClient(clientID, clientSecret string, useSandbox bool) *FedExClient {
	baseURL := "https://apis.fedex.com"
	if useSandbox {
		baseURL = "https://apis-sandbox.fedex.com"
	}
	
	return &FedExClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      baseURL,
		client:       &http.Client{Timeout: 30 * time.Second},
		rateLimit: &RateLimitInfo{
			Limit:     30, // FedEx recommends max 30 tracking numbers per request
			Remaining: 30,
			ResetTime: time.Now().Add(10 * time.Second), // FedEx rate limit is per 10 seconds
		},
	}
}

// GetCarrierName returns the carrier name
func (c *FedExClient) GetCarrierName() string {
	return "fedex"
}

// ValidateTrackingNumber validates FedEx tracking number formats
func (c *FedExClient) ValidateTrackingNumber(trackingNumber string) bool {
	if trackingNumber == "" {
		return false
	}
	
	// Remove spaces and keep only digits
	cleaned := strings.ReplaceAll(trackingNumber, " ", "")
	
	// Check if it's all digits
	if matched, _ := regexp.MatchString(`^\d+$`, cleaned); !matched {
		return false
	}
	
	// FedEx tracking number lengths
	validLengths := []int{12, 14, 15, 18, 20, 22}
	
	for _, length := range validLengths {
		if len(cleaned) == length {
			return true
		}
	}
	
	return false
}

// GetRateLimit returns current rate limit information
func (c *FedExClient) GetRateLimit() *RateLimitInfo {
	return c.rateLimit
}

// Track retrieves tracking information for the given tracking numbers
func (c *FedExClient) Track(ctx context.Context, req *TrackingRequest) (*TrackingResponse, error) {
	// TODO: Implement FedEx tracking API call
	// This is a stub for now
	var results []TrackingInfo
	
	for _, trackingNumber := range req.TrackingNumbers {
		results = append(results, TrackingInfo{
			TrackingNumber: trackingNumber,
			Carrier:        "fedex",
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

func (c *FedExClient) authenticate(ctx context.Context) error {
	// TODO: Implement OAuth 2.0 authentication
	// This is a stub for now
	c.accessToken = "mock_token"
	c.tokenExpiry = time.Now().Add(1 * time.Hour) // FedEx tokens expire after 1 hour
	return nil
}