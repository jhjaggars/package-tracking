package carriers

import (
	"bytes"
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
	if len(req.TrackingNumbers) == 0 {
		return nil, fmt.Errorf("no tracking numbers provided")
	}
	
	// Ensure we have a valid access token
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}
	
	// FedEx supports batch tracking - up to 30 tracking numbers recommended per request
	maxPerRequest := 30
	var results []TrackingInfo
	var errors []CarrierError
	
	for i := 0; i < len(req.TrackingNumbers); i += maxPerRequest {
		end := i + maxPerRequest
		if end > len(req.TrackingNumbers) {
			end = len(req.TrackingNumbers)
		}
		
		batch := req.TrackingNumbers[i:end]
		batchResults, batchErrors, err := c.trackBatch(ctx, batch)
		if err != nil {
			return nil, err
		}
		
		results = append(results, batchResults...)
		errors = append(errors, batchErrors...)
		
		// Check if any error is a rate limit error and return immediately
		for _, batchError := range batchErrors {
			if batchError.RateLimit {
				return nil, &batchError
			}
		}
	}
	
	return &TrackingResponse{
		Results:   results,
		Errors:    errors,
		RateLimit: c.rateLimit,
	}, nil
}

func (c *FedExClient) ensureAuthenticated(ctx context.Context) error {
	// Only authenticate if we don't have a token at all
	if c.accessToken == "" {
		return c.authenticate(ctx)
	}
	return nil
}

func (c *FedExClient) authenticate(ctx context.Context) error {
	// Prepare OAuth request
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create OAuth request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Make request
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("OAuth request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read OAuth response: %w", err)
	}
	
	// Check for error response
	if resp.StatusCode != http.StatusOK {
		var oauthError struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if err := json.Unmarshal(body, &oauthError); err == nil {
			return fmt.Errorf("OAuth error: %s - %s", oauthError.Error, oauthError.ErrorDescription)
		}
		return fmt.Errorf("OAuth failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse success response
	var oauthResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		Scope       string `json:"scope"`
	}
	if err := json.Unmarshal(body, &oauthResp); err != nil {
		return fmt.Errorf("failed to parse OAuth response: %w", err)
	}
	
	// Store token and expiry
	c.accessToken = oauthResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(oauthResp.ExpiresIn) * time.Second)
	
	return nil
}

func (c *FedExClient) trackBatch(ctx context.Context, trackingNumbers []string) ([]TrackingInfo, []CarrierError, error) {
	// Build tracking request
	requestBody := map[string]interface{}{
		"includeDetailedScans": true,
		"trackingInfo": make([]map[string]interface{}, 0, len(trackingNumbers)),
	}
	
	// Add tracking numbers to request
	for _, trackingNumber := range trackingNumbers {
		trackingInfo := map[string]interface{}{
			"trackingNumberInfo": map[string]interface{}{
				"trackingNumber": trackingNumber,
			},
		}
		requestBody["trackingInfo"] = append(requestBody["trackingInfo"].([]map[string]interface{}), trackingInfo)
	}
	
	// Marshal request body
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/track/v1/trackingnumbers", bytes.NewReader(jsonData))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create tracking request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-locale", "en_US")
	
	// Make request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("tracking request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read tracking response: %w", err)
	}
	
	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		c.updateRateLimitFromHeaders(resp.Header)
		return nil, []CarrierError{{
			Carrier:   "fedex",
			Code:      "RATE.LIMIT.EXCEEDED",
			Message:   "Rate limit exceeded",
			Retryable: true,
			RateLimit: true,
		}}, nil
	}
	
	// Handle authentication errors (token expired)
	if resp.StatusCode == http.StatusUnauthorized {
		// Try to refresh token and retry once
		if err := c.authenticate(ctx); err != nil {
			return nil, nil, fmt.Errorf("failed to refresh token: %w", err)
		}
		
		// Create a new request with updated token
		newReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/track/v1/trackingnumbers", bytes.NewReader(jsonData))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create retry request: %w", err)
		}
		newReq.Header.Set("Authorization", "Bearer "+c.accessToken)
		newReq.Header.Set("Content-Type", "application/json")
		newReq.Header.Set("X-locale", "en_US")
		
		// Close the original response first
		resp.Body.Close()
		
		resp, err = c.client.Do(newReq)
		if err != nil {
			return nil, nil, fmt.Errorf("tracking request retry failed: %w", err)
		}
		defer resp.Body.Close()
		
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read tracking response on retry: %w", err)
		}
	}
	
	// Check for other HTTP errors
	if resp.StatusCode != http.StatusOK {
		// Try to parse FedEx error response
		var fedexError struct {
			Errors []struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(body, &fedexError); err == nil && len(fedexError.Errors) > 0 {
			return nil, []CarrierError{{
				Carrier:   "fedex",
				Code:      fedexError.Errors[0].Code,
				Message:   fedexError.Errors[0].Message,
				Retryable: false,
				RateLimit: strings.Contains(fedexError.Errors[0].Code, "RATE.LIMIT"),
			}}, nil
		}
		return nil, nil, fmt.Errorf("tracking request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Update rate limit info
	c.updateRateLimitFromHeaders(resp.Header)
	
	// Parse tracking response
	var trackResp struct {
		Output struct {
			CompleteTrackResults []struct {
				TrackingNumber string `json:"trackingNumber"`
				TrackResults   []struct {
					TrackingNumberInfo struct {
						TrackingNumber string `json:"trackingNumber"`
						CarrierCode    string `json:"carrierCode"`
					} `json:"trackingNumberInfo"`
					ShipmentDetails struct {
						Weight []struct {
							Value string `json:"value"`
							Unit  string `json:"unit"`
						} `json:"weight"`
						PackagingDescription   string `json:"packagingDescription"`
						PhysicalPackagingType  string `json:"physicalPackagingType"`
					} `json:"shipmentDetails"`
					ScanEvents []struct {
						Date             string `json:"date"`
						EventType        string `json:"eventType"`
						EventDescription string `json:"eventDescription"`
						ScanLocation     struct {
							City               string `json:"city"`
							StateOrProvinceCode string `json:"stateOrProvinceCode"`
							PostalCode         string `json:"postalCode"`
							CountryCode        string `json:"countryCode"`
						} `json:"scanLocation"`
					} `json:"scanEvents"`
					DateAndTimes []struct {
						Type     string `json:"type"`
						DateTime string `json:"dateTime"`
					} `json:"dateAndTimes"`
				} `json:"trackResults"`
			} `json:"completeTrackResults"`
		} `json:"output"`
	}
	
	if err := json.Unmarshal(body, &trackResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse tracking response: %w", err)
	}
	
	// Process results
	var results []TrackingInfo
	var errors []CarrierError
	
	for _, completeResult := range trackResp.Output.CompleteTrackResults {
		if len(completeResult.TrackResults) == 0 {
			errors = append(errors, CarrierError{
				Carrier:   "fedex",
				Code:      "NO_RESULTS",
				Message:   "No tracking results found for " + completeResult.TrackingNumber,
				Retryable: false,
				RateLimit: false,
			})
			continue
		}
		
		trackResult := completeResult.TrackResults[0]
		trackingInfo := c.parseFedExTrackingInfo(trackResult, completeResult.TrackingNumber)
		results = append(results, trackingInfo)
	}
	
	return results, errors, nil
}

func (c *FedExClient) updateRateLimitFromHeaders(headers http.Header) {
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
}

func (c *FedExClient) parseFedExTrackingInfo(trackResult struct {
	TrackingNumberInfo struct {
		TrackingNumber string `json:"trackingNumber"`
		CarrierCode    string `json:"carrierCode"`
	} `json:"trackingNumberInfo"`
	ShipmentDetails struct {
		Weight []struct {
			Value string `json:"value"`
			Unit  string `json:"unit"`
		} `json:"weight"`
		PackagingDescription   string `json:"packagingDescription"`
		PhysicalPackagingType  string `json:"physicalPackagingType"`
	} `json:"shipmentDetails"`
	ScanEvents []struct {
		Date             string `json:"date"`
		EventType        string `json:"eventType"`
		EventDescription string `json:"eventDescription"`
		ScanLocation     struct {
			City               string `json:"city"`
			StateOrProvinceCode string `json:"stateOrProvinceCode"`
			PostalCode         string `json:"postalCode"`
			CountryCode        string `json:"countryCode"`
		} `json:"scanLocation"`
	} `json:"scanEvents"`
	DateAndTimes []struct {
		Type     string `json:"type"`
		DateTime string `json:"dateTime"`
	} `json:"dateAndTimes"`
}, trackingNumber string) TrackingInfo {
	info := TrackingInfo{
		TrackingNumber: trackingNumber,
		Carrier:        "fedex",
		Events:         []TrackingEvent{},
		LastUpdated:    time.Now(),
		Status:         StatusUnknown,
	}
	
	// Set service type and weight
	if trackResult.ShipmentDetails.PackagingDescription != "" {
		info.ServiceType = trackResult.ShipmentDetails.PackagingDescription
	}
	
	if len(trackResult.ShipmentDetails.Weight) > 0 {
		weight := trackResult.ShipmentDetails.Weight[0]
		if weight.Value != "" && weight.Unit != "" {
			info.Weight = weight.Value + " " + weight.Unit
		}
	}
	
	// Process delivery dates
	for _, dateTime := range trackResult.DateAndTimes {
		if dateTime.Type == "ACTUAL_DELIVERY" {
			if deliveryTime, err := c.parseFedExDateTime(dateTime.DateTime); err == nil {
				info.ActualDelivery = &deliveryTime
			}
		} else if dateTime.Type == "ESTIMATED_DELIVERY" && info.EstimatedDelivery == nil {
			if estimatedTime, err := c.parseFedExDateTime(dateTime.DateTime); err == nil {
				info.EstimatedDelivery = &estimatedTime
			}
		}
	}
	
	// Process scan events
	for _, event := range trackResult.ScanEvents {
		trackingEvent := c.parseFedExScanEvent(event)
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
	
	// Set current status from most recent event
	if len(info.Events) > 0 {
		info.Status = info.Events[0].Status
	}
	
	return info
}

func (c *FedExClient) parseFedExScanEvent(event struct {
	Date             string `json:"date"`
	EventType        string `json:"eventType"`
	EventDescription string `json:"eventDescription"`
	ScanLocation     struct {
		City               string `json:"city"`
		StateOrProvinceCode string `json:"stateOrProvinceCode"`
		PostalCode         string `json:"postalCode"`
		CountryCode        string `json:"countryCode"`
	} `json:"scanLocation"`
}) TrackingEvent {
	// Parse timestamp
	timestamp, _ := c.parseFedExDateTime(event.Date)
	
	// Map status
	status := c.mapFedExStatus(event.EventType, event.EventDescription)
	
	// Format location
	location := c.formatFedExLocation(event.ScanLocation)
	
	return TrackingEvent{
		Timestamp:   timestamp,
		Status:      status,
		Location:    location,
		Description: event.EventDescription,
	}
}

func (c *FedExClient) parseFedExDateTime(dateTimeStr string) (time.Time, error) {
	// FedEx date format: "2023-05-15T14:45:00-05:00"
	layouts := []string{
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05",
	}
	
	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateTimeStr); err == nil {
			return t, nil
		}
	}
	
	return time.Now(), fmt.Errorf("unable to parse FedEx datetime: %s", dateTimeStr)
}

func (c *FedExClient) mapFedExStatus(eventType, description string) TrackingStatus {
	switch strings.ToUpper(eventType) {
	case "DL":
		return StatusDelivered
	case "OD":
		return StatusOutForDelivery
	case "IT", "DP", "AR", "AF":
		return StatusInTransit
	case "PU":
		return StatusPreShip
	case "DE", "CA", "DY":
		return StatusException
	default:
		// Check description for additional clues
		desc := strings.ToLower(description)
		switch {
		case strings.Contains(desc, "delivered"):
			return StatusDelivered
		case strings.Contains(desc, "out for delivery"), strings.Contains(desc, "on fedex vehicle"):
			return StatusOutForDelivery
		case strings.Contains(desc, "in transit"), strings.Contains(desc, "departed"), strings.Contains(desc, "arrived"):
			return StatusInTransit
		case strings.Contains(desc, "picked up"), strings.Contains(desc, "shipment information"):
			return StatusPreShip
		case strings.Contains(desc, "exception"), strings.Contains(desc, "delay"):
			return StatusException
		case strings.Contains(desc, "returned"):
			return StatusReturned
		default:
			return StatusUnknown
		}
	}
}

func (c *FedExClient) formatFedExLocation(location struct {
	City               string `json:"city"`
	StateOrProvinceCode string `json:"stateOrProvinceCode"`
	PostalCode         string `json:"postalCode"`
	CountryCode        string `json:"countryCode"`
}) string {
	// Format: "ATLANTA, GA 30309, US"
	var result string
	
	if location.City != "" && location.StateOrProvinceCode != "" {
		result = location.City + ", " + location.StateOrProvinceCode
	} else if location.City != "" {
		result = location.City
	} else if location.StateOrProvinceCode != "" {
		result = location.StateOrProvinceCode
	}
	
	if location.PostalCode != "" {
		if result != "" {
			result += " " + location.PostalCode
		} else {
			result = location.PostalCode
		}
	}
	
	if location.CountryCode != "" {
		if result != "" {
			result += ", " + location.CountryCode
		} else {
			result = location.CountryCode
		}
	}
	
	return result
}