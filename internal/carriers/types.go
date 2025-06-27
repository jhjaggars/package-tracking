package carriers

import (
	"context"
	"time"
)

// TrackingStatus represents the current status of a shipment
type TrackingStatus string

const (
	StatusUnknown    TrackingStatus = "unknown"
	StatusPreShip    TrackingStatus = "pre_ship"
	StatusInTransit  TrackingStatus = "in_transit"
	StatusOutForDelivery TrackingStatus = "out_for_delivery"
	StatusDelivered  TrackingStatus = "delivered"
	StatusException  TrackingStatus = "exception"
	StatusReturned   TrackingStatus = "returned"
)

// TrackingEvent represents a single tracking event in the shipment's journey
type TrackingEvent struct {
	Timestamp   time.Time      `json:"timestamp"`
	Status      TrackingStatus `json:"status"`
	Location    string         `json:"location"`
	Description string         `json:"description"`
	Details     string         `json:"details,omitempty"`
}

// TrackingInfo represents the complete tracking information for a shipment
type TrackingInfo struct {
	TrackingNumber   string           `json:"tracking_number"`
	Carrier          string           `json:"carrier"`
	Status           TrackingStatus   `json:"status"`
	EstimatedDelivery *time.Time      `json:"estimated_delivery,omitempty"`
	ActualDelivery   *time.Time       `json:"actual_delivery,omitempty"`
	Events           []TrackingEvent  `json:"events"`
	ServiceType      string           `json:"service_type,omitempty"`
	Weight           string           `json:"weight,omitempty"`
	Dimensions       string           `json:"dimensions,omitempty"`
	LastUpdated      time.Time        `json:"last_updated"`
}

// CarrierError represents errors from carrier APIs
type CarrierError struct {
	Carrier    string `json:"carrier"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Retryable  bool   `json:"retryable"`
	RateLimit  bool   `json:"rate_limit"`
}

func (e *CarrierError) Error() string {
	return e.Carrier + ": " + e.Message
}

// RateLimitInfo contains rate limiting information
type RateLimitInfo struct {
	Limit       int           `json:"limit"`
	Remaining   int           `json:"remaining"`
	ResetTime   time.Time     `json:"reset_time"`
	RetryAfter  time.Duration `json:"retry_after,omitempty"`
}

// TrackingRequest represents a request to track one or more shipments
type TrackingRequest struct {
	TrackingNumbers []string `json:"tracking_numbers"`
	Carrier         string   `json:"carrier"`
}

// TrackingResponse represents the response from a carrier tracking API
type TrackingResponse struct {
	Results     []TrackingInfo  `json:"results"`
	Errors      []CarrierError  `json:"errors"`
	RateLimit   *RateLimitInfo  `json:"rate_limit,omitempty"`
}

// Client interface that all carrier implementations must satisfy
type Client interface {
	// Track retrieves tracking information for the given tracking numbers
	Track(ctx context.Context, req *TrackingRequest) (*TrackingResponse, error)
	
	// GetCarrierName returns the name of the carrier this client handles
	GetCarrierName() string
	
	// ValidateTrackingNumber checks if a tracking number format is valid for this carrier
	ValidateTrackingNumber(trackingNumber string) bool
	
	// GetRateLimit returns current rate limit information
	GetRateLimit() *RateLimitInfo
}

// Config contains configuration for carrier clients
type Config struct {
	// USPS Configuration
	USPSUserID string `json:"usps_user_id"`
	
	// UPS Configuration
	UPSClientID     string `json:"ups_client_id"`
	UPSClientSecret string `json:"ups_client_secret"`
	
	// FedEx Configuration
	FedExClientID     string `json:"fedex_client_id"`
	FedExClientSecret string `json:"fedex_client_secret"`
	
	// DHL Configuration
	DHLAPIKey string `json:"dhl_api_key"`
	
	// Global Configuration
	Timeout     time.Duration `json:"timeout"`
	MaxRetries  int          `json:"max_retries"`
	UseSandbox  bool         `json:"use_sandbox"`
}