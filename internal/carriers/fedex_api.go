package carriers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// FedExAPIClient implements tracking using the official FedEx Track API
type FedExAPIClient struct {
	apiKey       string
	secretKey    string
	baseURL      string
	client       *http.Client
	accessToken  string
	tokenExpiry  time.Time
}

// NewFedExAPIClient creates a new FedEx API client
func NewFedExAPIClient(apiKey, secretKey string) *FedExAPIClient {
	return &FedExAPIClient{
		apiKey:    apiKey,
		secretKey: secretKey,
		baseURL:   "https://apis.fedex.com", // Production URL
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// NewFedExAPISandboxClient creates a new FedEx API client for sandbox testing
func NewFedExAPISandboxClient(apiKey, secretKey string) *FedExAPIClient {
	return &FedExAPIClient{
		apiKey:    apiKey,
		secretKey: secretKey,
		baseURL:   "https://apis-sandbox.fedex.com", // Sandbox URL
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// NewFedExAPIClientWithURL creates a new FedEx API client with custom base URL
func NewFedExAPIClientWithURL(apiKey, secretKey, baseURL string) *FedExAPIClient {
	return &FedExAPIClient{
		apiKey:    apiKey,
		secretKey: secretKey,
		baseURL:   baseURL,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// FedExTrackRequest represents the request structure for FedEx Track API
type FedExTrackRequest struct {
	TrackingInfo []FedExTrackingInfo `json:"trackingInfo"`
	IncludeDetailedScans bool        `json:"includeDetailedScans"`
}

// FedExTrackingInfo represents individual tracking info in the request
type FedExTrackingInfo struct {
	TrackingNumberInfo FedExTrackingNumberInfo `json:"trackingNumberInfo"`
	ShipDateBegin      string                  `json:"shipDateBegin,omitempty"`
	ShipDateEnd        string                  `json:"shipDateEnd,omitempty"`
}

// FedExTrackingNumberInfo represents tracking number details
type FedExTrackingNumberInfo struct {
	TrackingNumber string `json:"trackingNumber"`
}

// FedExTrackResponse represents the response from FedEx Track API
type FedExTrackResponse struct {
	TransactionID         string                    `json:"transactionId"`
	CustomerTransactionID string                    `json:"customerTransactionId"`
	Output                FedExTrackResponseOutput  `json:"output"`
}

// FedExTrackResponseOutput contains the tracking results
type FedExTrackResponseOutput struct {
	CompleteTrackResults []FedExCompleteTrackResult `json:"completeTrackResults"`
}

// FedExCompleteTrackResult contains tracking results for a single tracking number
type FedExCompleteTrackResult struct {
	TrackingNumber string             `json:"trackingNumber"`
	TrackResults   []FedExTrackResult `json:"trackResults"`
}

// FedExTrackResult contains detailed tracking information
type FedExTrackResult struct {
	TrackingNumberInfo     FedExAPITrackingNumberInfo `json:"trackingNumberInfo"`
	AdditionalTrackingInfo FedExAdditionalTrackingInfo `json:"additionalTrackingInfo,omitempty"`
	ShipmentDetails        FedExShipmentDetails        `json:"shipmentDetails,omitempty"`
	ScanEvents             []FedExScanEvent            `json:"scanEvents,omitempty"`
	DateAndTimes           []FedExDateAndTime          `json:"dateAndTimes,omitempty"`
	PackageDetails         FedExPackageDetails         `json:"packageDetails,omitempty"`
	GoodsClassificationCode string                     `json:"goodsClassificationCode,omitempty"`
	HoldAtLocationDetails  FedExHoldAtLocationDetails  `json:"holdAtLocationDetails,omitempty"`
	CustomDeliveryOptions  []FedExCustomDeliveryOption `json:"customDeliveryOptions,omitempty"`
	EstimatedDeliveryTimeWindow FedExEstimatedDeliveryTimeWindow `json:"estimatedDeliveryTimeWindow,omitempty"`
	DistanceToDestination  FedExDistanceToDestination  `json:"distanceToDestination,omitempty"`
	ConsolidationDetail    []FedExConsolidationDetail  `json:"consolidationDetail,omitempty"`
	MosterReference        FedExMosterReference        `json:"mosterReference,omitempty"`
	AvailableImages        []FedExAvailableImage       `json:"availableImages,omitempty"`
	SpecialHandlings       []FedExSpecialHandling      `json:"specialHandlings,omitempty"`
	DeliveryDetails        FedExDeliveryDetails        `json:"deliveryDetails,omitempty"`
	OriginLocation         FedExLocationDetail         `json:"originLocation,omitempty"`
	DestinationLocation    FedExLocationDetail         `json:"destinationLocation,omitempty"`
	LatestStatusDetail     FedExLatestStatusDetail     `json:"latestStatusDetail,omitempty"`
	ServiceDetail          FedExServiceDetail          `json:"serviceDetail,omitempty"`
	StandardTransitTimeWindow FedExStandardTransitTimeWindow `json:"standardTransitTimeWindow,omitempty"`
	Error                  *FedExAPIError              `json:"error,omitempty"`
}

// FedExAPITrackingNumberInfo contains tracking number details from API response
type FedExAPITrackingNumberInfo struct {
	TrackingNumber         string `json:"trackingNumber"`
	TrackingNumberUniqueID string `json:"trackingNumberUniqueId"`
	CarrierCode           string `json:"carrierCode"`
}

// FedExAdditionalTrackingInfo contains additional tracking details
type FedExAdditionalTrackingInfo struct {
	Nickname                string `json:"nickname"`
	PackageIdentifiers      []FedExPackageIdentifier `json:"packageIdentifiers,omitempty"`
	HasAssociatedShipments  bool   `json:"hasAssociatedShipments,omitempty"`
}

// FedExPackageIdentifier represents package identification details
type FedExPackageIdentifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// FedExShipmentDetails contains shipment information
type FedExShipmentDetails struct {
	PossessionStatus             bool                    `json:"possessionStatus,omitempty"`
	Weight                       []FedExWeight           `json:"weight,omitempty"`
	ContentPieceCount            int                     `json:"contentPieceCount,omitempty"`
	PackagingDescription         FedExPackagingDescription `json:"packagingDescription,omitempty"`
	PhysicalPackagingType        string                  `json:"physicalPackagingType,omitempty"`
	SequenceNumber               string                  `json:"sequenceNumber,omitempty"`
	UndeliveredCount             string                  `json:"undeliveredCount,omitempty"`
	CountInDestinationCountry    int                     `json:"countInDestinationCountry,omitempty"`
	WeightAndDimensions          FedExWeightAndDimensions `json:"weightAndDimensions,omitempty"`
}

// FedExWeight represents weight information
type FedExWeight struct {
	Units string  `json:"units"`
	Value float64 `json:"value"`
}

// FedExPackagingDescription represents packaging details
type FedExPackagingDescription struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// FedExWeightAndDimensions represents weight and dimension details
type FedExWeightAndDimensions struct {
	Weight     []FedExWeight    `json:"weight,omitempty"`
	Dimensions []FedExDimension `json:"dimensions,omitempty"`
}

// FedExDimension represents dimension information
type FedExDimension struct {
	Length int    `json:"length"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Units  string `json:"units"`
}

// FedExScanEvent represents a tracking scan event
type FedExScanEvent struct {
	Date                    string                    `json:"date"`
	EventType               string                    `json:"eventType"`
	EventDescription        string                    `json:"eventDescription"`
	ExceptionCode           string                    `json:"exceptionCode,omitempty"`
	ExceptionDescription    string                    `json:"exceptionDescription,omitempty"`
	ScanLocation            FedExScanLocation         `json:"scanLocation,omitempty"`
	LocationId              string                    `json:"locationId,omitempty"`
	LocationContactAndAddress FedExLocationContactAndAddress `json:"locationContactAndAddress,omitempty"`
	DerivedStatus           string                    `json:"derivedStatus,omitempty"`
}

// FedExScanLocation represents the location of a scan event
type FedExScanLocation struct {
	StreetLines             []string `json:"streetLines,omitempty"`
	City                    string   `json:"city,omitempty"`
	StateOrProvinceCode     string   `json:"stateOrProvinceCode,omitempty"`
	PostalCode              string   `json:"postalCode,omitempty"`
	CountryCode             string   `json:"countryCode,omitempty"`
	CountryName             string   `json:"countryName,omitempty"`
	Residential             bool     `json:"residential,omitempty"`
}

// FedExLocationContactAndAddress represents contact and address information
type FedExLocationContactAndAddress struct {
	Contact FedExContact `json:"contact,omitempty"`
	Address FedExAddress `json:"address,omitempty"`
}

// FedExContact represents contact information
type FedExContact struct {
	PersonName   string `json:"personName,omitempty"`
	PhoneNumber  string `json:"phoneNumber,omitempty"`
	CompanyName  string `json:"companyName,omitempty"`
}

// FedExAddress represents address information
type FedExAddress struct {
	StreetLines             []string `json:"streetLines,omitempty"`
	City                    string   `json:"city,omitempty"`
	StateOrProvinceCode     string   `json:"stateOrProvinceCode,omitempty"`
	PostalCode              string   `json:"postalCode,omitempty"`
	CountryCode             string   `json:"countryCode,omitempty"`
	CountryName             string   `json:"countryName,omitempty"`
	Residential             bool     `json:"residential,omitempty"`
}

// Additional struct definitions for completeness (abbreviated for brevity)
type FedExDateAndTime struct {
	Type     string `json:"type"`
	DateTime string `json:"dateTime"`
}

type FedExPackageDetails struct {
	PhysicalPackagingType string `json:"physicalPackagingType,omitempty"`
	SequenceNumber        string `json:"sequenceNumber,omitempty"`
}

type FedExHoldAtLocationDetails struct {
	LocationId   string       `json:"locationId,omitempty"`
	LocationContactAndAddress FedExLocationContactAndAddress `json:"locationContactAndAddress,omitempty"`
}

type FedExCustomDeliveryOption struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

type FedExEstimatedDeliveryTimeWindow struct {
	Description string `json:"description,omitempty"`
	Window      FedExTimeWindow `json:"window,omitempty"`
}

type FedExTimeWindow struct {
	Begin string `json:"begins,omitempty"`
	Ends  string `json:"ends,omitempty"`
}

type FedExDistanceToDestination struct {
	Units string  `json:"units,omitempty"`
	Value float64 `json:"value,omitempty"`
}

type FedExConsolidationDetail struct {
	TimeStamp            string `json:"timeStamp,omitempty"`
	ConsolidationCompletionDetail string `json:"consolidationCompletionDetail,omitempty"`
}

type FedExMosterReference struct {
	// Add fields as needed
}

type FedExAvailableImage struct {
	Size string `json:"size,omitempty"`
	Type string `json:"type,omitempty"`
}

type FedExSpecialHandling struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

type FedExDeliveryDetails struct {
	ReceiverInformation FedExReceiverInformation `json:"receiverInformation,omitempty"`
	LocationDescription string                   `json:"locationDescription,omitempty"`
	ActualDeliveryAddress FedExAddress           `json:"actualDeliveryAddress,omitempty"`
	DeliveryAttempts     string                  `json:"deliveryAttempts,omitempty"`
	DeliveryOptionEligibilityDetails []FedExDeliveryOptionEligibilityDetail `json:"deliveryOptionEligibilityDetails,omitempty"`
}

type FedExReceiverInformation struct {
	ContactAndAddress FedExLocationContactAndAddress `json:"contactAndAddress,omitempty"`
}

type FedExDeliveryOptionEligibilityDetail struct {
	Option      string `json:"option,omitempty"`
	Eligibility string `json:"eligibility,omitempty"`
}

type FedExLocationDetail struct {
	LocationContactAndAddress FedExLocationContactAndAddress `json:"locationContactAndAddress,omitempty"`
}

type FedExLatestStatusDetail struct {
	Code        string    `json:"code,omitempty"`
	Description string    `json:"description,omitempty"`
	ScanLocation FedExScanLocation `json:"scanLocation,omitempty"`
}

type FedExServiceDetail struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	ShortDescription string `json:"shortDescription,omitempty"`
}

type FedExStandardTransitTimeWindow struct {
	Window FedExTimeWindow `json:"window,omitempty"`
}

// FedExAPIError represents an API error response
type FedExAPIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// FedExOAuthRequest represents the OAuth token request
type FedExOAuthRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// FedExOAuthResponse represents the OAuth token response
type FedExOAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// ValidateTrackingNumber validates FedEx tracking number formats
func (c *FedExAPIClient) ValidateTrackingNumber(trackingNumber string) bool {
	if trackingNumber == "" {
		return false
	}
	
	// Remove spaces and keep only digits
	cleaned := strings.ReplaceAll(trackingNumber, " ", "")
	
	// Check if it's all digits
	if matched, _ := regexp.MatchString(`^\d+$`, cleaned); !matched {
		return false
	}
	
	// FedEx tracking number lengths: 12, 14, 15, 16, 18, 20, 22
	validLengths := []int{12, 14, 15, 16, 18, 20, 22}
	
	for _, length := range validLengths {
		if len(cleaned) == length {
			return true
		}
	}
	
	return false
}

// getAccessToken obtains an OAuth access token from FedEx
func (c *FedExAPIClient) getAccessToken(ctx context.Context) error {
	// Check if we have a valid token
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	// Request new token
	tokenURL := c.baseURL + "/oauth/token"
	
	// FedEx OAuth expects application/x-www-form-urlencoded format
	formData := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s",
		c.apiKey, c.secretKey)
	
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(formData))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}
	
	var tokenResponse FedExOAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}
	
	// Store token and calculate expiry (with 5-minute buffer)
	c.accessToken = tokenResponse.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResponse.ExpiresIn-300) * time.Second)
	
	return nil
}

// Track retrieves tracking information using the FedEx Track API
func (c *FedExAPIClient) Track(ctx context.Context, req *TrackingRequest) (*TrackingResponse, error) {
	if len(req.TrackingNumbers) == 0 {
		return nil, fmt.Errorf("no tracking numbers provided")
	}
	
	// Ensure we have a valid access token
	if err := c.getAccessToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to obtain access token: %w", err)
	}
	
	var results []TrackingInfo
	var errors []CarrierError
	
	// FedEx API supports up to 30 tracking numbers per request
	batchSize := 30
	for i := 0; i < len(req.TrackingNumbers); i += batchSize {
		end := i + batchSize
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
		RateLimit: c.GetRateLimit(),
	}, nil
}

// trackBatch processes a batch of tracking numbers
func (c *FedExAPIClient) trackBatch(ctx context.Context, trackingNumbers []string) ([]TrackingInfo, []CarrierError, error) {
	// Build track request
	trackingInfo := make([]FedExTrackingInfo, len(trackingNumbers))
	for i, trackingNumber := range trackingNumbers {
		trackingInfo[i] = FedExTrackingInfo{
			TrackingNumberInfo: FedExTrackingNumberInfo{
				TrackingNumber: trackingNumber,
			},
		}
	}
	
	apiRequest := FedExTrackRequest{
		TrackingInfo:         trackingInfo,
		IncludeDetailedScans: true,
	}
	
	jsonBody, err := json.Marshal(apiRequest)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal track request: %w", err)
	}
	
	// Make API request
	trackURL := c.baseURL + "/track/v1/trackingnumbers"
	req, err := http.NewRequestWithContext(ctx, "POST", trackURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create track request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("X-locale", "en_US")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("track request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("track request failed with status %d", resp.StatusCode)
	}
	
	var trackResponse FedExTrackResponse
	if err := json.NewDecoder(resp.Body).Decode(&trackResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to decode track response: %w", err)
	}
	
	// Process results
	return c.processTrackResults(trackResponse)
}

// processTrackResults converts FedEx API response to our internal format
func (c *FedExAPIClient) processTrackResults(response FedExTrackResponse) ([]TrackingInfo, []CarrierError, error) {
	var results []TrackingInfo
	var errors []CarrierError
	
	for _, completeResult := range response.Output.CompleteTrackResults {
		for _, trackResult := range completeResult.TrackResults {
			if trackResult.Error != nil {
				// Handle API errors
				carrierErr := CarrierError{
					Carrier:   "fedex",
					Code:      trackResult.Error.Code,
					Message:   trackResult.Error.Message,
					Retryable: c.isRetryableError(trackResult.Error.Code),
					RateLimit: false,
				}
				errors = append(errors, carrierErr)
				continue
			}
			
			// Convert to our internal tracking info format
			trackingInfo := c.convertToTrackingInfo(trackResult)
			results = append(results, trackingInfo)
		}
	}
	
	return results, errors, nil
}

// convertToTrackingInfo converts FedEx API result to our internal format
func (c *FedExAPIClient) convertToTrackingInfo(result FedExTrackResult) TrackingInfo {
	info := TrackingInfo{
		TrackingNumber: result.TrackingNumberInfo.TrackingNumber,
		Carrier:        "fedex",
		Events:         []TrackingEvent{},
		LastUpdated:    time.Now(),
		Status:         StatusUnknown,
	}
	
	// Convert scan events
	for _, scanEvent := range result.ScanEvents {
		event := c.convertScanEvent(scanEvent)
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
	
	// Set current status from latest event or latest status detail
	if result.LatestStatusDetail.Code != "" {
		info.Status = c.mapFedExStatusCode(result.LatestStatusDetail.Code)
	} else if len(info.Events) > 0 {
		info.Status = info.Events[0].Status
	}
	
	// Set delivery time if delivered
	if info.Status == StatusDelivered && len(info.Events) > 0 {
		info.ActualDelivery = &info.Events[0].Timestamp
	}
	
	return info
}

// convertScanEvent converts FedEx scan event to our internal format
func (c *FedExAPIClient) convertScanEvent(scanEvent FedExScanEvent) TrackingEvent {
	// Parse timestamp
	parsedTime, err := time.Parse("2006-01-02T15:04:05Z", scanEvent.Date)
	if err != nil {
		// Try alternative formats
		parsedTime, _ = time.Parse("2006-01-02T15:04:05-07:00", scanEvent.Date)
	}
	if err != nil {
		parsedTime = time.Now()
	}
	
	// Build location string
	location := c.buildLocationString(scanEvent.ScanLocation)
	
	// Map event type to our status
	status := c.mapFedExEventType(scanEvent.EventType, scanEvent.EventDescription)
	
	return TrackingEvent{
		Timestamp:   parsedTime,
		Status:      status,
		Location:    location,
		Description: scanEvent.EventDescription,
	}
}

// buildLocationString builds a location string from FedEx location data
func (c *FedExAPIClient) buildLocationString(location FedExScanLocation) string {
	var parts []string
	
	if location.City != "" {
		parts = append(parts, location.City)
	}
	if location.StateOrProvinceCode != "" {
		parts = append(parts, location.StateOrProvinceCode)
	}
	if location.CountryCode != "" && location.CountryCode != "US" {
		parts = append(parts, location.CountryCode)
	}
	
	return strings.Join(parts, ", ")
}

// mapFedExStatusCode maps FedEx status codes to our internal status
func (c *FedExAPIClient) mapFedExStatusCode(code string) TrackingStatus {
	switch strings.ToUpper(code) {
	case "DL", "DELIVERED":
		return StatusDelivered
	case "OD", "OUT_FOR_DELIVERY":
		return StatusOutForDelivery
	case "IT", "IN_TRANSIT":
		return StatusInTransit
	case "PU", "PICKED_UP":
		return StatusInTransit
	case "EX", "EXCEPTION":
		return StatusException
	case "HL", "HOLD_AT_LOCATION":
		return StatusException
	default:
		return StatusInTransit
	}
}

// mapFedExEventType maps FedEx event types to our internal status
func (c *FedExAPIClient) mapFedExEventType(eventType, description string) TrackingStatus {
	eventType = strings.ToUpper(eventType)
	description = strings.ToLower(description)
	
	if strings.Contains(description, "delivered") {
		return StatusDelivered
	}
	if strings.Contains(description, "out for delivery") {
		return StatusOutForDelivery
	}
	if strings.Contains(description, "exception") || strings.Contains(description, "delay") {
		return StatusException
	}
	
	switch eventType {
	case "DL":
		return StatusDelivered
	case "OD":
		return StatusOutForDelivery
	case "PU", "PK":
		return StatusInTransit
	case "EX":
		return StatusException
	default:
		return StatusInTransit
	}
}

// isRetryableError determines if a FedEx API error is retryable
func (c *FedExAPIClient) isRetryableError(code string) bool {
	retryableCodes := []string{
		"SYSTEM.UNAVAILABLE.EXCEPTION",
		"SERVICE.UNAVAILABLE.EXCEPTION", 
		"INTERNAL.SERVER.ERROR",
		"TIMEOUT.EXCEPTION",
	}
	
	for _, retryableCode := range retryableCodes {
		if code == retryableCode {
			return true
		}
	}
	
	return false
}

// GetCarrierName returns the carrier name
func (c *FedExAPIClient) GetCarrierName() string {
	return "fedex"
}

// GetRateLimit returns rate limit information (FedEx API has generous limits)
func (c *FedExAPIClient) GetRateLimit() *RateLimitInfo {
	return &RateLimitInfo{
		Limit:     1000, // FedEx API has generous limits
		Remaining: 1000,
		ResetTime: time.Now().Add(time.Hour),
	}
}