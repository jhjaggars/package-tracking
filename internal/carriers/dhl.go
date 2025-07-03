package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
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
	
	// Remove spaces and normalize
	cleaned := strings.ReplaceAll(trackingNumber, " ", "")
	
	// Check basic alphanumeric pattern
	if matched, _ := regexp.MatchString(`^[A-Za-z0-9]+$`, cleaned); !matched {
		return false
	}
	
	// DHL tracking number lengths: 10-20 characters
	length := len(cleaned)
	if length < 10 || length > 20 {
		return false
	}
	
	// DHL tracking numbers must contain at least some digits
	// This prevents common words like "INFORMATION" from being validated
	if matched, _ := regexp.MatchString(`\d`, cleaned); !matched {
		return false
	}
	
	return true
}

// GetRateLimit returns current rate limit information
func (c *DHLClient) GetRateLimit() *RateLimitInfo {
	return c.rateLimit
}

// Track retrieves tracking information for the given tracking numbers
func (c *DHLClient) Track(ctx context.Context, req *TrackingRequest) (*TrackingResponse, error) {
	if len(req.TrackingNumbers) == 0 {
		return nil, fmt.Errorf("no tracking numbers provided")
	}
	
	var results []TrackingInfo
	var errors []CarrierError
	
	// DHL API handles one tracking number per request
	for _, trackingNumber := range req.TrackingNumbers {
		result, err := c.trackSingle(ctx, trackingNumber)
		if err != nil {
			if carrierErr, ok := err.(*CarrierError); ok {
				// For rate limits and auth errors, return immediately
				if carrierErr.RateLimit || carrierErr.Code == "401" || carrierErr.Code == "400" {
					return nil, err
				}
				errors = append(errors, *carrierErr)
			} else {
				return nil, err
			}
		} else {
			results = append(results, *result)
		}
	}
	
	return &TrackingResponse{
		Results:   results,
		Errors:    errors,
		RateLimit: c.rateLimit,
	}, nil
}

func (c *DHLClient) trackSingle(ctx context.Context, trackingNumber string) (*TrackingInfo, error) {
	// Build tracking URL with query parameters
	baseURL := c.baseURL + "/track/shipments"
	params := url.Values{}
	params.Set("trackingNumber", trackingNumber)
	params.Set("requesterCountryCode", "US")
	params.Set("language", "en")
	
	trackURL := baseURL + "?" + params.Encode()
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", trackURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracking request: %w", err)
	}
	
	// Set headers
	req.Header.Set("DHL-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	
	// Make request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tracking request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read tracking response: %w", err)
	}
	
	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		c.updateRateLimitFromHeaders(resp.Header)
		return nil, &CarrierError{
			Carrier:   "dhl",
			Code:      "429",
			Message:   "Rate limit exceeded",
			Retryable: true,
			RateLimit: true,
		}
	}
	
	// Handle authentication errors
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &CarrierError{
			Carrier:   "dhl",
			Code:      "401",
			Message:   "Invalid API key",
			Retryable: false,
			RateLimit: false,
		}
	}
	
	// Handle other HTTP errors
	if resp.StatusCode != http.StatusOK {
		// Try to parse DHL error response
		var dhlError struct {
			Title   string `json:"title"`
			Status  int    `json:"status"`
			Detail  string `json:"detail"`
			Instance string `json:"instance"`
		}
		if err := json.Unmarshal(body, &dhlError); err == nil {
			return nil, &CarrierError{
				Carrier:   "dhl",
				Code:      strconv.Itoa(dhlError.Status),
				Message:   dhlError.Detail,
				Retryable: dhlError.Status >= 500, // 5xx errors are potentially retryable
				RateLimit: dhlError.Status == 429,
			}
		}
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	
	// Update rate limit info
	c.updateRateLimitFromHeaders(resp.Header)
	
	// Parse tracking response
	var trackResp struct {
		Shipments []struct {
			ID      string `json:"id"`
			Service string `json:"service"`
			Origin  struct {
				Address struct {
					CountryCode      string `json:"countryCode"`
					PostalCode       string `json:"postalCode"`
					AddressLocality  string `json:"addressLocality"`
				} `json:"address"`
			} `json:"origin"`
			Destination struct {
				Address struct {
					CountryCode      string `json:"countryCode"`
					PostalCode       string `json:"postalCode"`
					AddressLocality  string `json:"addressLocality"`
				} `json:"address"`
			} `json:"destination"`
			Status struct {
				Timestamp   string `json:"timestamp"`
				Location    struct {
					Address struct {
						CountryCode      string `json:"countryCode"`
						PostalCode       string `json:"postalCode"`
						AddressLocality  string `json:"addressLocality"`
					} `json:"address"`
				} `json:"location"`
				StatusCode  string `json:"statusCode"`
				Status      string `json:"status"`
				Description string `json:"description"`
			} `json:"status"`
			EstimatedTimeOfDelivery string `json:"estimatedTimeOfDelivery"`
			EstimatedDeliveryTimeFrame struct {
				EstimatedFrom    string `json:"estimatedFrom"`
				EstimatedThrough string `json:"estimatedThrough"`
			} `json:"estimatedDeliveryTimeFrame"`
			Details struct {
				Carrier struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"carrier"`
				Product struct {
					ProductName string `json:"productName"`
				} `json:"product"`
				ProofOfDelivery struct {
					Timestamp string `json:"timestamp"`
					SignedBy  string `json:"signedBy"`
				} `json:"proofOfDelivery"`
				TotalNumberOfPieces int `json:"totalNumberOfPieces"`
				Weight struct {
					Value    float64 `json:"value"`
					UnitText string  `json:"unitText"`
				} `json:"weight"`
				Volume struct {
					Value    float64 `json:"value"`
					UnitText string  `json:"unitText"`
				} `json:"volume"`
			} `json:"details"`
			Events []struct {
				Timestamp   string `json:"timestamp"`
				Location    struct {
					Address struct {
						CountryCode      string `json:"countryCode"`
						PostalCode       string `json:"postalCode"`
						AddressLocality  string `json:"addressLocality"`
						StreetAddress    string `json:"streetAddress"`
					} `json:"address"`
				} `json:"location"`
				StatusCode  string `json:"statusCode"`
				Status      string `json:"status"`
				Description string `json:"description"`
				Remark      string `json:"remark"`
			} `json:"events"`
		} `json:"shipments"`
	}
	
	if err := json.Unmarshal(body, &trackResp); err != nil {
		return nil, fmt.Errorf("failed to parse tracking response: %w", err)
	}
	
	// Process results
	if len(trackResp.Shipments) == 0 {
		return nil, &CarrierError{
			Carrier:   "dhl",
			Code:      "NO_RESULTS",
			Message:   "No tracking results found for " + trackingNumber,
			Retryable: false,
			RateLimit: false,
		}
	}
	
	shipment := trackResp.Shipments[0]
	trackingInfo := c.parseDHLTrackingInfo(shipment, trackingNumber)
	
	return &trackingInfo, nil
}

func (c *DHLClient) updateRateLimitFromHeaders(headers http.Header) {
	if c.rateLimit == nil {
		c.rateLimit = &RateLimitInfo{}
	}
	
	if limit := headers.Get("X-RateLimit-Limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			c.rateLimit.Limit = l
		}
	}
	
	if remaining := headers.Get("X-RateLimit-Remaining"); remaining != "" {
		if r, err := strconv.Atoi(remaining); err == nil {
			c.rateLimit.Remaining = r
		}
	}
	
	if reset := headers.Get("X-RateLimit-Reset"); reset != "" {
		if r, err := strconv.ParseInt(reset, 10, 64); err == nil {
			c.rateLimit.ResetTime = time.Unix(r, 0)
		}
	}
	
	if retryAfter := headers.Get("Retry-After"); retryAfter != "" {
		if r, err := strconv.Atoi(retryAfter); err == nil {
			c.rateLimit.RetryAfter = time.Duration(r) * time.Second
		}
	}
}

func (c *DHLClient) parseDHLTrackingInfo(shipment struct {
	ID      string `json:"id"`
	Service string `json:"service"`
	Origin  struct {
		Address struct {
			CountryCode      string `json:"countryCode"`
			PostalCode       string `json:"postalCode"`
			AddressLocality  string `json:"addressLocality"`
		} `json:"address"`
	} `json:"origin"`
	Destination struct {
		Address struct {
			CountryCode      string `json:"countryCode"`
			PostalCode       string `json:"postalCode"`
			AddressLocality  string `json:"addressLocality"`
		} `json:"address"`
	} `json:"destination"`
	Status struct {
		Timestamp   string `json:"timestamp"`
		Location    struct {
			Address struct {
				CountryCode      string `json:"countryCode"`
				PostalCode       string `json:"postalCode"`
				AddressLocality  string `json:"addressLocality"`
			} `json:"address"`
		} `json:"location"`
		StatusCode  string `json:"statusCode"`
		Status      string `json:"status"`
		Description string `json:"description"`
	} `json:"status"`
	EstimatedTimeOfDelivery string `json:"estimatedTimeOfDelivery"`
	EstimatedDeliveryTimeFrame struct {
		EstimatedFrom    string `json:"estimatedFrom"`
		EstimatedThrough string `json:"estimatedThrough"`
	} `json:"estimatedDeliveryTimeFrame"`
	Details struct {
		Carrier struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"carrier"`
		Product struct {
			ProductName string `json:"productName"`
		} `json:"product"`
		ProofOfDelivery struct {
			Timestamp string `json:"timestamp"`
			SignedBy  string `json:"signedBy"`
		} `json:"proofOfDelivery"`
		TotalNumberOfPieces int `json:"totalNumberOfPieces"`
		Weight struct {
			Value    float64 `json:"value"`
			UnitText string  `json:"unitText"`
		} `json:"weight"`
		Volume struct {
			Value    float64 `json:"value"`
			UnitText string  `json:"unitText"`
		} `json:"volume"`
	} `json:"details"`
	Events []struct {
		Timestamp   string `json:"timestamp"`
		Location    struct {
			Address struct {
				CountryCode      string `json:"countryCode"`
				PostalCode       string `json:"postalCode"`
				AddressLocality  string `json:"addressLocality"`
				StreetAddress    string `json:"streetAddress"`
			} `json:"address"`
		} `json:"location"`
		StatusCode  string `json:"statusCode"`
		Status      string `json:"status"`
		Description string `json:"description"`
		Remark      string `json:"remark"`
	} `json:"events"`
}, trackingNumber string) TrackingInfo {
	info := TrackingInfo{
		TrackingNumber: trackingNumber,
		Carrier:        "dhl",
		Events:         []TrackingEvent{},
		LastUpdated:    time.Now(),
		Status:         StatusUnknown,
	}
	
	// Set service type and weight
	if shipment.Details.Product.ProductName != "" {
		info.ServiceType = shipment.Details.Product.ProductName
	}
	
	if shipment.Details.Weight.Value > 0 && shipment.Details.Weight.UnitText != "" {
		info.Weight = fmt.Sprintf("%.1f %s", shipment.Details.Weight.Value, shipment.Details.Weight.UnitText)
	}
	
	// Set dimensions if available
	if shipment.Details.Volume.Value > 0 && shipment.Details.Volume.UnitText != "" {
		info.Dimensions = fmt.Sprintf("%.3f %s", shipment.Details.Volume.Value, shipment.Details.Volume.UnitText)
	}
	
	// Process estimated delivery
	if shipment.EstimatedTimeOfDelivery != "" {
		if estimatedTime, err := c.parseDHLDateTime(shipment.EstimatedTimeOfDelivery); err == nil {
			info.EstimatedDelivery = &estimatedTime
		}
	}
	
	// Process proof of delivery (actual delivery)
	if shipment.Details.ProofOfDelivery.Timestamp != "" {
		if deliveryTime, err := c.parseDHLDateTime(shipment.Details.ProofOfDelivery.Timestamp); err == nil {
			info.ActualDelivery = &deliveryTime
		}
	}
	
	// Process events
	for _, event := range shipment.Events {
		trackingEvent := c.parseDHLEvent(event)
		info.Events = append(info.Events, trackingEvent)
	}
	
	// Sort events by timestamp (newest first)
	for i := 0; i < len(info.Events)-1; i++ {
		for j := i + 1; j < len(info.Events); j++ {
			if info.Events[i].Timestamp.Before(info.Events[j].Timestamp) {
				info.Events[i], info.Events[j] = info.Events[j], info.Events[i]
			}
		}
	}
	
	// Set current status from most recent event or shipment status
	if len(info.Events) > 0 {
		info.Status = info.Events[0].Status
	} else {
		info.Status = c.mapDHLStatus(shipment.Status.StatusCode, shipment.Status.Status)
	}
	
	return info
}

func (c *DHLClient) parseDHLEvent(event struct {
	Timestamp   string `json:"timestamp"`
	Location    struct {
		Address struct {
			CountryCode      string `json:"countryCode"`
			PostalCode       string `json:"postalCode"`
			AddressLocality  string `json:"addressLocality"`
			StreetAddress    string `json:"streetAddress"`
		} `json:"address"`
	} `json:"location"`
	StatusCode  string `json:"statusCode"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Remark      string `json:"remark"`
}) TrackingEvent {
	// Parse timestamp
	timestamp, _ := c.parseDHLDateTime(event.Timestamp)
	
	// Map status
	status := c.mapDHLStatus(event.StatusCode, event.Status)
	
	// Format location
	location := c.formatDHLLocation(event.Location.Address)
	
	// Use remark for additional details if available
	details := event.Remark
	
	return TrackingEvent{
		Timestamp:   timestamp,
		Status:      status,
		Location:    location,
		Description: event.Description,
		Details:     details,
	}
}

func (c *DHLClient) parseDHLDateTime(dateTimeStr string) (time.Time, error) {
	// DHL date format: "2023-05-15T14:45:00.000+02:00"
	layouts := []string{
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
	}
	
	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateTimeStr); err == nil {
			return t, nil
		}
	}
	
	return time.Now(), fmt.Errorf("unable to parse DHL datetime: %s", dateTimeStr)
}

func (c *DHLClient) mapDHLStatus(statusCode, status string) TrackingStatus {
	// Map based on status code first
	switch strings.ToLower(statusCode) {
	case "delivered":
		return StatusDelivered
	case "with-delivery-courier":
		return StatusOutForDelivery
	case "transit", "processed", "departed", "arrived":
		return StatusInTransit
	case "pre-transit", "pickup", "picked-up":
		return StatusPreShip
	case "exception", "customs-status", "held":
		return StatusException
	case "returned", "return":
		return StatusReturned
	default:
		// Fall back to status description
		desc := strings.ToLower(status + " " + statusCode)
		switch {
		case strings.Contains(desc, "delivered"):
			return StatusDelivered
		case strings.Contains(desc, "out for delivery"), strings.Contains(desc, "delivery courier"):
			return StatusOutForDelivery
		case strings.Contains(desc, "transit"), strings.Contains(desc, "departed"), strings.Contains(desc, "arrived"):
			return StatusInTransit
		case strings.Contains(desc, "picked"), strings.Contains(desc, "pickup"):
			return StatusPreShip
		case strings.Contains(desc, "exception"), strings.Contains(desc, "customs"), strings.Contains(desc, "held"):
			return StatusException
		case strings.Contains(desc, "returned"), strings.Contains(desc, "return"):
			return StatusReturned
		default:
			return StatusUnknown
		}
	}
}

func (c *DHLClient) formatDHLLocation(address struct {
	CountryCode      string `json:"countryCode"`
	PostalCode       string `json:"postalCode"`
	AddressLocality  string `json:"addressLocality"`
	StreetAddress    string `json:"streetAddress"`
}) string {
	// Format: "New York, 10001, US"
	var parts []string
	
	if address.AddressLocality != "" {
		parts = append(parts, address.AddressLocality)
	}
	
	if address.PostalCode != "" {
		parts = append(parts, address.PostalCode)
	}
	
	if address.CountryCode != "" {
		parts = append(parts, address.CountryCode)
	}
	
	return strings.Join(parts, ", ")
}