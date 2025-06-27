package carriers

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// USPS API XML structures
type USPSTrackRequest struct {
	XMLName xml.Name `xml:"TrackRequest"`
	UserID  string   `xml:"USERID,attr"`
	TrackIDs []USPSTrackID `xml:"TrackID"`
}

type USPSTrackID struct {
	ID string `xml:"ID,attr"`
}

type USPSTrackResponse struct {
	XMLName   xml.Name        `xml:"TrackResponse"`
	TrackInfos []USPSTrackInfo `xml:"TrackInfo"`
}

type USPSTrackInfo struct {
	ID           string             `xml:"ID,attr"`
	TrackSummary *USPSTrackSummary  `xml:"TrackSummary"`
	TrackDetails []USPSTrackDetail  `xml:"TrackDetail"`
	Error        *USPSError         `xml:"Error"`
}

type USPSTrackSummary struct {
	EventTime    string `xml:"EventTime"`
	EventDate    string `xml:"EventDate"`
	Event        string `xml:"Event"`
	EventCity    string `xml:"EventCity"`
	EventState   string `xml:"EventState"`
	EventZIPCode string `xml:"EventZIPCode"`
	EventCountry string `xml:"EventCountry"`
}

type USPSTrackDetail struct {
	EventTime    string `xml:"EventTime"`
	EventDate    string `xml:"EventDate"`
	Event        string `xml:"Event"`
	EventCity    string `xml:"EventCity"`
	EventState   string `xml:"EventState"`
	EventZIPCode string `xml:"EventZIPCode"`
	EventCountry string `xml:"EventCountry"`
}

type USPSError struct {
	Number      string `xml:"Number"`
	Description string `xml:"Description"`
}

// USPSClient implements the Client interface for USPS API
type USPSClient struct {
	userID    string
	baseURL   string
	client    *http.Client
	rateLimit *RateLimitInfo
}

// NewUSPSClient creates a new USPS API client
func NewUSPSClient(userID string, useSandbox bool) *USPSClient {
	baseURL := "https://secure.shippingapis.com/shippingapi.dll"
	if useSandbox {
		baseURL = "https://stg-secure.shippingapis.com/shippingapi.dll"
	}
	
	return &USPSClient{
		userID:  userID,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
		rateLimit: &RateLimitInfo{
			Limit:     35, // USPS allows max 35 packages per transaction
			Remaining: 35,
			ResetTime: time.Now().Add(time.Hour),
		},
	}
}

// GetCarrierName returns the carrier name
func (c *USPSClient) GetCarrierName() string {
	return "usps"
}

// ValidateTrackingNumber validates USPS tracking number formats
func (c *USPSClient) ValidateTrackingNumber(trackingNumber string) bool {
	if trackingNumber == "" {
		return false
	}
	
	// Remove spaces and convert to uppercase
	cleaned := strings.ToUpper(strings.ReplaceAll(trackingNumber, " ", ""))
	
	// USPS tracking number patterns
	patterns := []string{
		`^94\d{20}$`,           // Priority Mail Express & Priority Mail: 94001234567890123456
		`^93\d{20}$`,           // Signature Confirmation: 93012345678901234567
		`^92\d{20}$`,           // Certified Mail: 92012345678901234567
		`^91\d{20}$`,           // Signature Confirmation: 91012345678901234567
		`^82\d{8}$`,            // Priority Mail Express International: 82123456
		`^[A-Z]{2}\d{9}US$`,    // Priority Mail Express International: EK123456789US
		`^7\d{19}$`,            // Certified Mail: 7012345678901234567
		`^LC\d{9}US$`,          // Priority Mail Express International: LC123456789US
		`^LK\d{9}US$`,          // Priority Mail Express International: LK123456789US
		`^EA\d{9}US$`,          // Priority Mail Express International: EA123456789US
		`^CP\d{9}US$`,          // Priority Mail Express International: CP123456789US
		`^RA\d{9}US$`,          // Registered Mail International: RA123456789US
		`^RB\d{9}US$`,          // Registered Mail International: RB123456789US
		`^RC\d{9}US$`,          // Registered Mail International: RC123456789US
		`^RD\d{9}US$`,          // Registered Mail International: RD123456789US
	}
	
	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, cleaned)
		if matched {
			return true
		}
	}
	
	return false
}

// GetRateLimit returns current rate limit information
func (c *USPSClient) GetRateLimit() *RateLimitInfo {
	return c.rateLimit
}

// Track retrieves tracking information for the given tracking numbers
func (c *USPSClient) Track(ctx context.Context, req *TrackingRequest) (*TrackingResponse, error) {
	if len(req.TrackingNumbers) == 0 {
		return nil, fmt.Errorf("no tracking numbers provided")
	}
	
	// USPS allows up to 10 tracking numbers per request
	maxPerRequest := 10
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
	}
	
	return &TrackingResponse{
		Results:   results,
		Errors:    errors,
		RateLimit: c.rateLimit,
	}, nil
}

func (c *USPSClient) trackBatch(ctx context.Context, trackingNumbers []string) ([]TrackingInfo, []CarrierError, error) {
	// Build XML request
	trackRequest := USPSTrackRequest{
		UserID: c.userID,
	}
	
	for _, trackingNumber := range trackingNumbers {
		trackRequest.TrackIDs = append(trackRequest.TrackIDs, USPSTrackID{
			ID: trackingNumber,
		})
	}
	
	xmlData, err := xml.Marshal(trackRequest)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal XML request: %w", err)
	}
	
	// Build URL with query parameters
	params := url.Values{}
	params.Set("API", "TrackV2")
	params.Set("XML", string(xmlData))
	
	requestURL := c.baseURL + "?" + params.Encode()
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	// Make request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	
	// Parse XML response
	var trackResponse USPSTrackResponse
	if err := xml.NewDecoder(resp.Body).Decode(&trackResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to parse XML response: %w", err)
	}
	
	// Process results
	var results []TrackingInfo
	var errors []CarrierError
	
	for _, trackInfo := range trackResponse.TrackInfos {
		if trackInfo.Error != nil {
			errors = append(errors, CarrierError{
				Carrier:   "usps",
				Code:      trackInfo.Error.Number,
				Message:   trackInfo.Error.Description,
				Retryable: false,
				RateLimit: false,
			})
			continue
		}
		
		trackingInfo := c.parseTrackingInfo(trackInfo)
		results = append(results, trackingInfo)
	}
	
	return results, errors, nil
}

func (c *USPSClient) parseTrackingInfo(trackInfo USPSTrackInfo) TrackingInfo {
	info := TrackingInfo{
		TrackingNumber: trackInfo.ID,
		Carrier:        "usps",
		Events:         []TrackingEvent{},
		LastUpdated:    time.Now(),
	}
	
	// Process track summary (most recent event)
	if trackInfo.TrackSummary != nil {
		event := c.parseTrackingEvent(*trackInfo.TrackSummary)
		info.Events = append(info.Events, event)
		info.Status = event.Status
		
		if event.Status == StatusDelivered {
			info.ActualDelivery = &event.Timestamp
		}
	}
	
	// Process track details (historical events)
	for _, detail := range trackInfo.TrackDetails {
		event := c.parseTrackingEventFromDetail(detail)
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
	
	return info
}

func (c *USPSClient) parseTrackingEvent(summary USPSTrackSummary) TrackingEvent {
	timestamp := c.parseUSPSDateTime(summary.EventDate, summary.EventTime)
	status := c.mapUSPSStatus(summary.Event)
	location := c.formatLocation(summary.EventCity, summary.EventState, summary.EventZIPCode, summary.EventCountry)
	
	return TrackingEvent{
		Timestamp:   timestamp,
		Status:      status,
		Location:    location,
		Description: summary.Event,
	}
}

func (c *USPSClient) parseTrackingEventFromDetail(detail USPSTrackDetail) TrackingEvent {
	timestamp := c.parseUSPSDateTime(detail.EventDate, detail.EventTime)
	status := c.mapUSPSStatus(detail.Event)
	location := c.formatLocation(detail.EventCity, detail.EventState, detail.EventZIPCode, detail.EventCountry)
	
	return TrackingEvent{
		Timestamp:   timestamp,
		Status:      status,
		Location:    location,
		Description: detail.Event,
	}
}

func (c *USPSClient) parseUSPSDateTime(dateStr, timeStr string) time.Time {
	// USPS date format: "May 11, 2016"
	// USPS time format: "11:07 am"
	
	if dateStr == "" {
		return time.Now()
	}
	
	// Combine date and time
	dateTimeStr := dateStr
	if timeStr != "" {
		dateTimeStr += " " + timeStr
	}
	
	// Try different time formats
	layouts := []string{
		"January 2, 2006 3:04 pm",
		"January 2, 2006 3:04:05 pm",
		"January 2, 2006",
		"Jan 2, 2006 3:04 pm",
		"Jan 2, 2006 3:04:05 pm",
		"Jan 2, 2006",
	}
	
	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateTimeStr); err == nil {
			return t
		}
	}
	
	// Fallback to current time if parsing fails
	return time.Now()
}

func (c *USPSClient) mapUSPSStatus(eventDescription string) TrackingStatus {
	event := strings.ToLower(eventDescription)
	
	switch {
	case strings.Contains(event, "delivered"):
		return StatusDelivered
	case strings.Contains(event, "out for delivery"):
		return StatusOutForDelivery
	case strings.Contains(event, "in transit"):
		return StatusInTransit
	case strings.Contains(event, "departed"):
		return StatusInTransit
	case strings.Contains(event, "arrived"):
		return StatusInTransit
	case strings.Contains(event, "processed"):
		return StatusInTransit
	case strings.Contains(event, "acceptance"):
		return StatusPreShip
	case strings.Contains(event, "pre-shipment"):
		return StatusPreShip
	case strings.Contains(event, "exception"):
		return StatusException
	case strings.Contains(event, "returned"):
		return StatusReturned
	case strings.Contains(event, "return"):
		return StatusReturned
	default:
		return StatusUnknown
	}
}

func (c *USPSClient) formatLocation(city, state, zipCode, country string) string {
	var parts []string
	
	if city != "" && state != "" {
		parts = append(parts, city+", "+state)
	} else if city != "" {
		parts = append(parts, city)
	} else if state != "" {
		parts = append(parts, state)
	}
	
	if zipCode != "" {
		parts = append(parts, zipCode)
	}
	
	if country != "" && country != "US" {
		parts = append(parts, country)
	}
	
	return strings.Join(parts, " ")
}