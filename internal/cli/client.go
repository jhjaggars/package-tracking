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
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIError represents an error from the API
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
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

// doRequest performs an HTTP request and handles errors
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
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
		return nil, fmt.Errorf("failed to decode response: %w", err)
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
		return nil, fmt.Errorf("failed to decode response: %w", err)
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
		return nil, fmt.Errorf("failed to decode response: %w", err)
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
		return nil, fmt.Errorf("failed to decode response: %w", err)
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
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return events, nil
}