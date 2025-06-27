package carriers

import (
	"context"
	"encoding/base64"
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

// UPS OAuth structures
type UPSOAuthRequest struct {
	GrantType string `json:"grant_type"`
}

type UPSOAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

type UPSOAuthError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// UPS Track API structures
type UPSTrackResponse struct {
	TrackResponse struct {
		Shipment []struct {
			Package []struct {
				TrackingNumber string `json:"trackingNumber"`
				DeliveryDate   []struct {
					Date string `json:"date"`
				} `json:"deliveryDate"`
				Activity []struct {
					Date     string `json:"date"`
					Time     string `json:"time"`
					Status   struct {
						Type        string `json:"type"`
						Description string `json:"description"`
						Code        string `json:"code"`
					} `json:"status"`
					Location struct {
						Address struct {
							City                string `json:"city"`
							StateProvinceCode   string `json:"stateProvinceCode"`
							PostalCode          string `json:"postalCode"`
							Country             string `json:"country"`
						} `json:"address"`
					} `json:"location"`
				} `json:"activity"`
			} `json:"package"`
		} `json:"shipment"`
	} `json:"trackResponse"`
}

// UPSClient implements the Client interface for UPS API
type UPSClient struct {
	clientID     string
	clientSecret string
	baseURL      string
	client       *http.Client
	accessToken  string
	tokenExpiry  time.Time
	rateLimit    *RateLimitInfo
}

// NewUPSClient creates a new UPS API client
func NewUPSClient(clientID, clientSecret string, useSandbox bool) *UPSClient {
	baseURL := "https://onlinetools.ups.com"
	if useSandbox {
		baseURL = "https://wwwcie.ups.com"
	}
	
	return &UPSClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      baseURL,
		client:       &http.Client{Timeout: 30 * time.Second},
		rateLimit: &RateLimitInfo{
			Limit:     100, // UPS allows up to 100 tracking numbers per request
			Remaining: 100,
			ResetTime: time.Now().Add(time.Hour),
		},
	}
}

// GetCarrierName returns the carrier name
func (c *UPSClient) GetCarrierName() string {
	return "ups"
}

// ValidateTrackingNumber validates UPS tracking number format
func (c *UPSClient) ValidateTrackingNumber(trackingNumber string) bool {
	if trackingNumber == "" {
		return false
	}
	
	// Remove spaces and convert to uppercase
	cleaned := strings.ToUpper(strings.ReplaceAll(trackingNumber, " ", ""))
	
	// UPS tracking number pattern: 1Z + 6 alphanumeric + 2 digits + 7 digits
	// Example: 1Z999AA1234567890
	pattern := `^1Z[A-Z0-9]{6}\d{2}\d{7}$`
	matched, _ := regexp.MatchString(pattern, cleaned)
	return matched
}

// GetRateLimit returns current rate limit information
func (c *UPSClient) GetRateLimit() *RateLimitInfo {
	return c.rateLimit
}

// Track retrieves tracking information for the given tracking numbers
func (c *UPSClient) Track(ctx context.Context, req *TrackingRequest) (*TrackingResponse, error) {
	if len(req.TrackingNumbers) == 0 {
		return nil, fmt.Errorf("no tracking numbers provided")
	}
	
	// Ensure we have a valid access token
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}
	
	var results []TrackingInfo
	var errors []CarrierError
	
	// UPS API handles one tracking number per request
	for _, trackingNumber := range req.TrackingNumbers {
		result, err := c.trackSingle(ctx, trackingNumber)
		if err != nil {
			if carrierErr, ok := err.(*CarrierError); ok {
				errors = append(errors, *carrierErr)
				// For rate limits, return immediately
				if carrierErr.RateLimit {
					return nil, err
				}
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

func (c *UPSClient) ensureAuthenticated(ctx context.Context) error {
	// Only authenticate if we don't have a token at all
	if c.accessToken == "" {
		return c.authenticate(ctx)
	}
	return nil
}

func (c *UPSClient) authenticate(ctx context.Context) error {
	// Prepare OAuth request
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/security/v1/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create OAuth request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Set Basic auth header
	auth := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	
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
		var oauthError UPSOAuthError
		if err := json.Unmarshal(body, &oauthError); err == nil {
			return fmt.Errorf("OAuth error: %s - %s", oauthError.Error, oauthError.ErrorDescription)
		}
		return fmt.Errorf("OAuth failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse success response
	var oauthResp UPSOAuthResponse
	if err := json.Unmarshal(body, &oauthResp); err != nil {
		return fmt.Errorf("failed to parse OAuth response: %w", err)
	}
	
	// Store token and expiry
	c.accessToken = oauthResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(oauthResp.ExpiresIn) * time.Second)
	
	return nil
}

func (c *UPSClient) trackSingle(ctx context.Context, trackingNumber string) (*TrackingInfo, error) {
	// Build tracking URL
	trackURL := fmt.Sprintf("%s/track/v1/details/%s", c.baseURL, trackingNumber)
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", trackURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracking request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	
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
			Carrier:   "ups",
			Code:      strconv.Itoa(resp.StatusCode),
			Message:   "Rate limit exceeded",
			Retryable: true,
			RateLimit: true,
		}
	}
	
	// Handle authentication errors (token expired)
	if resp.StatusCode == http.StatusUnauthorized {
		// Try to refresh token and retry once
		if err := c.authenticate(ctx); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
		
		// Create a new request with updated token
		newReq, err := http.NewRequestWithContext(ctx, "GET", trackURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create retry request: %w", err)
		}
		newReq.Header.Set("Authorization", "Bearer "+c.accessToken)
		newReq.Header.Set("Content-Type", "application/json")
		
		// Close the original response first
		resp.Body.Close()
		
		resp, err = c.client.Do(newReq)
		if err != nil {
			return nil, fmt.Errorf("tracking request retry failed: %w", err)
		}
		defer resp.Body.Close()
		
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read tracking response on retry: %w", err)
		}
	}
	
	// Check for other HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tracking request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Update rate limit info
	c.updateRateLimitFromHeaders(resp.Header)
	
	// Parse tracking response
	var trackResp UPSTrackResponse
	if err := json.Unmarshal(body, &trackResp); err != nil {
		return nil, fmt.Errorf("failed to parse tracking response: %w", err)
	}
	
	// Convert to our format
	return c.parseUPSTrackingInfo(trackResp, trackingNumber)
}

func (c *UPSClient) updateRateLimitFromHeaders(headers http.Header) {
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

func (c *UPSClient) parseUPSTrackingInfo(trackResp UPSTrackResponse, trackingNumber string) (*TrackingInfo, error) {
	info := &TrackingInfo{
		TrackingNumber: trackingNumber,
		Carrier:        "ups",
		Events:         []TrackingEvent{},
		LastUpdated:    time.Now(),
		Status:         StatusUnknown,
	}
	
	// UPS response structure: trackResponse -> shipment -> package
	if len(trackResp.TrackResponse.Shipment) == 0 {
		return info, nil
	}
	
	shipment := trackResp.TrackResponse.Shipment[0]
	if len(shipment.Package) == 0 {
		return info, nil
	}
	
	pkg := shipment.Package[0]
	
	// Process delivery date
	if len(pkg.DeliveryDate) > 0 {
		if deliveryTime, err := c.parseUPSDate(pkg.DeliveryDate[0].Date); err == nil {
			info.ActualDelivery = &deliveryTime
		}
	}
	
	// Process activities (tracking events)
	for _, activity := range pkg.Activity {
		event := c.parseUPSActivity(activity)
		info.Events = append(info.Events, event)
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
	
	return info, nil
}

func (c *UPSClient) parseUPSActivity(activity struct {
	Date     string `json:"date"`
	Time     string `json:"time"`
	Status   struct {
		Type        string `json:"type"`
		Description string `json:"description"`
		Code        string `json:"code"`
	} `json:"status"`
	Location struct {
		Address struct {
			City                string `json:"city"`
			StateProvinceCode   string `json:"stateProvinceCode"`
			PostalCode          string `json:"postalCode"`
			Country             string `json:"country"`
		} `json:"address"`
	} `json:"location"`
}) TrackingEvent {
	// Parse timestamp
	timestamp := c.parseUPSDateTime(activity.Date, activity.Time)
	
	// Map status
	status := c.mapUPSStatus(activity.Status.Type, activity.Status.Description)
	
	// Format location
	location := c.formatUPSLocation(activity.Location.Address)
	
	return TrackingEvent{
		Timestamp:   timestamp,
		Status:      status,
		Location:    location,
		Description: activity.Status.Description,
	}
}

func (c *UPSClient) parseUPSDate(dateStr string) (time.Time, error) {
	// UPS date format: "20230515"
	return time.Parse("20060102", dateStr)
}

func (c *UPSClient) parseUPSDateTime(dateStr, timeStr string) time.Time {
	// UPS date format: "20230515"
	// UPS time format: "144500" (HHMMSS)
	
	if dateStr == "" {
		return time.Now()
	}
	
	// Combine date and time
	dateTimeStr := dateStr
	if timeStr != "" {
		dateTimeStr += timeStr
	}
	
	// Try different formats
	layouts := []string{
		"20060102150405", // Full datetime
		"20060102",       // Date only
	}
	
	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateTimeStr); err == nil {
			return t
		}
	}
	
	// Fallback to current time
	return time.Now()
}

func (c *UPSClient) mapUPSStatus(statusType, description string) TrackingStatus {
	switch strings.ToUpper(statusType) {
	case "D":
		return StatusDelivered
	case "I":
		if strings.Contains(strings.ToLower(description), "out for delivery") {
			return StatusOutForDelivery
		}
		return StatusInTransit
	case "P":
		return StatusPreShip
	case "X":
		return StatusException
	default:
		// Check description for additional clues
		desc := strings.ToLower(description)
		switch {
		case strings.Contains(desc, "delivered"):
			return StatusDelivered
		case strings.Contains(desc, "out for delivery"):
			return StatusOutForDelivery
		case strings.Contains(desc, "in transit"):
			return StatusInTransit
		case strings.Contains(desc, "exception"):
			return StatusException
		case strings.Contains(desc, "returned"):
			return StatusReturned
		default:
			return StatusUnknown
		}
	}
}

func (c *UPSClient) formatUPSLocation(address struct {
	City                string `json:"city"`
	StateProvinceCode   string `json:"stateProvinceCode"`
	PostalCode          string `json:"postalCode"`
	Country             string `json:"country"`
}) string {
	// Format: "ATLANTA, GA 30309, US"
	var result string
	
	if address.City != "" && address.StateProvinceCode != "" {
		result = address.City + ", " + address.StateProvinceCode
	} else if address.City != "" {
		result = address.City
	} else if address.StateProvinceCode != "" {
		result = address.StateProvinceCode
	}
	
	if address.PostalCode != "" {
		if result != "" {
			result += " " + address.PostalCode
		} else {
			result = address.PostalCode
		}
	}
	
	if address.Country != "" {
		if result != "" {
			result += ", " + address.Country
		} else {
			result = address.Country
		}
	}
	
	return result
}