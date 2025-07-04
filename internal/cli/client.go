package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"package-tracking/internal/database"
)

// Client represents an HTTP client for the package tracking API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	return NewClientWithTimeout(baseURL, 180*time.Second) // Extended for SPA scraping (3 minutes)
}

// NewClientWithTimeout creates a new API client with specified timeout
func NewClientWithTimeout(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// APIError represents an error from the API
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Code == 0 {
		return e.Message
	}
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

// CreateShipmentRequest represents a request to create a shipment
type CreateShipmentRequest struct {
	TrackingNumber string `json:"tracking_number"`
	Carrier        string `json:"carrier"`
	Description    string `json:"description"`
}

// UpdateShipmentRequest represents a request to update a shipment
type UpdateShipmentRequest struct {
	Description string `json:"description"`
}

// RefreshResponse represents the response from a manual refresh request
type RefreshResponse struct {
	ShipmentID       int                      `json:"shipment_id"`
	UpdatedAt        time.Time                `json:"updated_at"`
	EventsAdded      int                      `json:"events_added"`
	TotalEvents      int                      `json:"total_events"`
	Events           []database.TrackingEvent `json:"events"`
	CacheStatus      string                   `json:"cache_status,omitempty"`      // "hit", "miss", "forced", "disabled"
	RefreshDuration  string                   `json:"refresh_duration,omitempty"`  // How long the refresh took
	PreviousCacheAge string                   `json:"previous_cache_age,omitempty"` // Age of cache that was invalidated
}

// doRequest performs an HTTP request and handles errors
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, &APIError{
				Code:    0,
				Message: fmt.Sprintf("Invalid request data: %v", err),
			}
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, &APIError{
			Code:    0,
			Message: fmt.Sprintf("Invalid request: %v", err),
		}
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &APIError{
			Code:    0, // 0 indicates network error, not HTTP status
			Message: fmt.Sprintf("Network error: %v", err),
		}
	}

	// Handle API errors
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			// If we can't decode the error, create a generic one
			apiErr = APIError{
				Code:    resp.StatusCode,
				Message: resp.Status,
			}
		}
		return nil, &apiErr
	}

	return resp, nil
}

// HealthCheck checks if the API server is healthy
func (c *Client) HealthCheck() error {
	resp, err := c.doRequest("GET", "/api/health", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// CreateShipment creates a new shipment
func (c *Client) CreateShipment(req *CreateShipmentRequest) (*database.Shipment, error) {
	resp, err := c.doRequest("POST", "/api/shipments", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var shipment database.Shipment
	if err := json.NewDecoder(resp.Body).Decode(&shipment); err != nil {
		return nil, &APIError{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("Invalid response format: %v", err),
		}
	}

	return &shipment, nil
}

// GetShipments returns all shipments
func (c *Client) GetShipments() ([]database.Shipment, error) {
	resp, err := c.doRequest("GET", "/api/shipments", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var shipments []database.Shipment
	if err := json.NewDecoder(resp.Body).Decode(&shipments); err != nil {
		return nil, &APIError{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("Invalid response format: %v", err),
		}
	}

	return shipments, nil
}

// GetShipment returns a specific shipment by ID
func (c *Client) GetShipment(id int) (*database.Shipment, error) {
	path := "/api/shipments/" + strconv.Itoa(id)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var shipment database.Shipment
	if err := json.NewDecoder(resp.Body).Decode(&shipment); err != nil {
		return nil, &APIError{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("Invalid response format: %v", err),
		}
	}

	return &shipment, nil
}

// UpdateShipment updates a shipment
func (c *Client) UpdateShipment(id int, req *UpdateShipmentRequest) (*database.Shipment, error) {
	path := "/api/shipments/" + strconv.Itoa(id)
	resp, err := c.doRequest("PUT", path, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var shipment database.Shipment
	if err := json.NewDecoder(resp.Body).Decode(&shipment); err != nil {
		return nil, &APIError{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("Invalid response format: %v", err),
		}
	}

	return &shipment, nil
}

// DeleteShipment deletes a shipment
func (c *Client) DeleteShipment(id int) error {
	path := "/api/shipments/" + strconv.Itoa(id)
	resp, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// GetEvents returns tracking events for a shipment
func (c *Client) GetEvents(shipmentID int) ([]database.TrackingEvent, error) {
	path := "/api/shipments/" + strconv.Itoa(shipmentID) + "/events"
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var events []database.TrackingEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, &APIError{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("Invalid response format: %v", err),
		}
	}

	return events, nil
}

// RefreshShipment manually refreshes tracking data for a shipment
func (c *Client) RefreshShipment(shipmentID int) (*RefreshResponse, error) {
	return c.RefreshShipmentWithForce(shipmentID, false)
}

// RefreshShipmentWithForce manually refreshes tracking data for a shipment with optional force flag
func (c *Client) RefreshShipmentWithForce(shipmentID int, force bool) (*RefreshResponse, error) {
	path := "/api/shipments/" + strconv.Itoa(shipmentID) + "/refresh"
	if force {
		path += "?force=true"
	}
	resp, err := c.doRequest("POST", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var refreshResp RefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&refreshResp); err != nil {
		return nil, &APIError{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("Invalid response format: %v", err),
		}
	}

	return &refreshResp, nil
}