package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"package-tracking/internal/email"
)

// Client handles HTTP requests to the package tracking API
type Client struct {
	baseURL    string
	httpClient *http.Client
	config     *ClientConfig
}

// ClientConfig configures the API client behavior
type ClientConfig struct {
	BaseURL       string
	Timeout       time.Duration
	RetryCount    int
	RetryDelay    time.Duration
	UserAgent     string
	MaxRetries    int
	BackoffFactor float64
}

// ShipmentRequest represents the request payload for creating a shipment
type ShipmentRequest struct {
	TrackingNumber   string `json:"tracking_number"`
	Carrier          string `json:"carrier"`
	Description      string `json:"description"`
	Status          string `json:"status,omitempty"`
	ExpectedDelivery string `json:"expected_delivery,omitempty"`
}

// ShipmentResponse represents the API response for shipment creation
type ShipmentResponse struct {
	ID               int    `json:"id"`
	TrackingNumber   string `json:"tracking_number"`
	Carrier          string `json:"carrier"`
	Description      string `json:"description"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// NewClient creates a new API client
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		config = &ClientConfig{
			BaseURL:       "http://localhost:8080",
			Timeout:       30 * time.Second,
			RetryCount:    3,
			RetryDelay:    1 * time.Second,
			UserAgent:     "email-tracker/1.0",
			MaxRetries:    3,
			BackoffFactor: 2.0,
		}
	}
	
	// Set defaults for missing fields
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.UserAgent == "" {
		config.UserAgent = "email-tracker/1.0"
	}
	if config.RetryCount == 0 {
		config.RetryCount = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}
	
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}
	
	return &Client{
		baseURL:    config.BaseURL,
		httpClient: httpClient,
		config:     config,
	}
}

// CreateShipment creates a new shipment via the API
func (c *Client) CreateShipment(tracking email.TrackingInfo) error {
	// Convert tracking info to API request format
	request := ShipmentRequest{
		TrackingNumber: tracking.Number,
		Carrier:        tracking.Carrier,
		Description:    tracking.Description,
		Status:         "pending", // Default status
	}
	
	// If description is empty, generate one with enhanced merchant support
	if request.Description == "" {
		// Check if we have merchant information for fallback
		if tracking.Merchant != "" {
			request.Description = fmt.Sprintf("Package from %s", tracking.Merchant)
		} else {
			// Legacy fallback: use email subject or sender
			if tracking.SourceEmail.Subject != "" {
				request.Description = tracking.SourceEmail.Subject
			} else {
				request.Description = fmt.Sprintf("Package from %s", tracking.SourceEmail.From)
			}
		}
	}
	
	url := fmt.Sprintf("%s/api/shipments", c.baseURL)
	
	// Marshal request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Execute request with retry logic
	var lastErr error
	for attempt := 0; attempt <= c.config.RetryCount; attempt++ {
		err := c.executeRequest("POST", url, requestBody, tracking.Number)
		if err == nil {
			return nil // Success
		}
		
		lastErr = err
		
		// Check if error is retryable
		if !c.isRetryableError(err) {
			return err // Don't retry for non-retryable errors
		}
		
		// Don't sleep after the last attempt
		if attempt < c.config.RetryCount {
			delay := c.calculateBackoffDelay(attempt)
			time.Sleep(delay)
		}
	}
	
	return fmt.Errorf("failed to create shipment after %d attempts: %w", c.config.RetryCount+1, lastErr)
}

// executeRequest executes a single HTTP request
func (c *Client) executeRequest(method, url string, body []byte, trackingNumber string) error {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Accept", "application/json")
	
	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Handle different status codes
	switch resp.StatusCode {
	case http.StatusCreated:
		// Success - parse response
		var shipmentResp ShipmentResponse
		if err := json.Unmarshal(respBody, &shipmentResp); err != nil {
			return fmt.Errorf("failed to parse success response: %w", err)
		}
		return nil
		
	case http.StatusConflict:
		// Duplicate tracking number - not an error for our purposes
		return nil
		
	case http.StatusBadRequest:
		// Parse error response
		var errorResp ErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			return fmt.Errorf("bad request: %s", errorResp.Error)
		}
		return fmt.Errorf("bad request: %s", string(respBody))
		
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		// Server errors - retryable
		var errorResp ErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			return &RetryableError{
				Message:    errorResp.Error,
				StatusCode: resp.StatusCode,
				Retryable:  true,
			}
		}
		return &RetryableError{
			Message:    fmt.Sprintf("server error: %s", string(respBody)),
			StatusCode: resp.StatusCode,
			Retryable:  true,
		}
		
	default:
		// Other errors
		var errorResp ErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, errorResp.Error)
		}
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}
}

// HealthCheck verifies the API is accessible
func (c *Client) HealthCheck() error {
	url := fmt.Sprintf("%s/api/health", c.baseURL)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	
	req.Header.Set("User-Agent", c.config.UserAgent)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// GetShipment retrieves a shipment by ID (for verification)
func (c *Client) GetShipment(id int) (*ShipmentResponse, error) {
	url := fmt.Sprintf("%s/api/shipments/%d", c.baseURL, id)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", c.config.UserAgent)
	req.Header.Set("Accept", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var shipment ShipmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&shipment); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &shipment, nil
}

// isRetryableError determines if an error should trigger a retry
func (c *Client) isRetryableError(err error) bool {
	if retryableErr, ok := err.(*RetryableError); ok {
		return retryableErr.Retryable
	}
	
	// Network errors are generally retryable
	return true
}

// calculateBackoffDelay calculates the delay for exponential backoff
func (c *Client) calculateBackoffDelay(attempt int) time.Duration {
	baseDelay := c.config.RetryDelay
	
	// Exponential backoff: delay = baseDelay * (backoffFactor ^ attempt)
	multiplier := 1.0
	for i := 0; i < attempt; i++ {
		multiplier *= c.config.BackoffFactor
	}
	
	delay := time.Duration(float64(baseDelay) * multiplier)
	
	// Cap the maximum delay at 30 seconds
	maxDelay := 30 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}
	
	return delay
}

// RetryableError represents an error that should be retried
type RetryableError struct {
	Message    string
	StatusCode int
	Retryable  bool
}

func (e *RetryableError) Error() string {
	return e.Message
}

// TestConnection tests the connection to the API
func (c *Client) TestConnection() error {
	return c.HealthCheck()
}

// GetBaseURL returns the configured base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// SetTimeout updates the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.config.Timeout = timeout
	c.httpClient.Timeout = timeout
}

// CreateShipmentBatch creates multiple shipments in batch (if API supports it)
func (c *Client) CreateShipmentBatch(trackingInfos []email.TrackingInfo) error {
	// For now, create shipments individually
	// TODO: Implement actual batch API if available
	
	var errors []error
	for _, tracking := range trackingInfos {
		if err := c.CreateShipment(tracking); err != nil {
			errors = append(errors, fmt.Errorf("failed to create shipment %s: %w", tracking.Number, err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("batch creation had %d errors: %v", len(errors), errors[0])
	}
	
	return nil
}

// Stats tracks API client statistics
type Stats struct {
	TotalRequests    int64
	SuccessfulRequests int64
	FailedRequests   int64
	RetryCount       int64
	AverageLatency   time.Duration
}

// GetStats returns client statistics (placeholder for future implementation)
func (c *Client) GetStats() *Stats {
	// TODO: Implement actual statistics tracking
	return &Stats{}
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	// For HTTP clients, there's typically nothing to close
	// This method is provided for interface compatibility
	return nil
}