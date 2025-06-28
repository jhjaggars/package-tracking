package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"package-tracking/internal/carriers"
	"package-tracking/internal/database"

	"github.com/go-chi/chi/v5"
)

// ShipmentHandler handles HTTP requests for shipments
type ShipmentHandler struct {
	db      *database.DB
	factory *carriers.ClientFactory
}

// NewShipmentHandler creates a new shipment handler
func NewShipmentHandler(db *database.DB) *ShipmentHandler {
	return &ShipmentHandler{
		db:      db,
		factory: carriers.NewClientFactory(),
	}
}

// GetShipments handles GET /api/shipments
func (h *ShipmentHandler) GetShipments(w http.ResponseWriter, r *http.Request) {
	shipments, err := h.db.Shipments.GetAll()
	if err != nil {
		log.Printf("ERROR: Failed to get shipments: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get shipments: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(shipments)
}

// CreateShipment handles POST /api/shipments
func (h *ShipmentHandler) CreateShipment(w http.ResponseWriter, r *http.Request) {
	var shipment database.Shipment
	if err := json.NewDecoder(r.Body).Decode(&shipment); err != nil {
		log.Printf("ERROR: Invalid JSON in CreateShipment: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if err := validateShipment(&shipment); err != nil {
		log.Printf("ERROR: Validation failed for shipment: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set default status if not provided
	if shipment.Status == "" {
		shipment.Status = "pending"
	}

	// Create the shipment
	if err := h.db.Shipments.Create(&shipment); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			log.Printf("ERROR: Duplicate tracking number: %s", shipment.TrackingNumber)
			http.Error(w, "Tracking number already exists", http.StatusConflict)
			return
		}
		log.Printf("ERROR: Failed to create shipment: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create shipment: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(shipment)
}

// GetShipmentByID handles GET /api/shipments/{id}
func (h *ShipmentHandler) GetShipmentByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid shipment ID", http.StatusBadRequest)
		return
	}

	shipment, err := h.db.Shipments.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Shipment not found", http.StatusNotFound)
			return
		}
		log.Printf("ERROR: Failed to get shipment %d: %v", id, err)
		http.Error(w, fmt.Sprintf("Failed to get shipment: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(shipment)
}

// UpdateShipment handles PUT /api/shipments/{id}
func (h *ShipmentHandler) UpdateShipment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid shipment ID", http.StatusBadRequest)
		return
	}

	var shipment database.Shipment
	if err := json.NewDecoder(r.Body).Decode(&shipment); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if err := validateShipment(&shipment); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update the shipment
	if err := h.db.Shipments.Update(id, &shipment); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Shipment not found", http.StatusNotFound)
			return
		}
		log.Printf("ERROR: Failed to update shipment %d: %v", id, err)
		http.Error(w, fmt.Sprintf("Failed to update shipment: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(shipment)
}

// DeleteShipment handles DELETE /api/shipments/{id}
func (h *ShipmentHandler) DeleteShipment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid shipment ID", http.StatusBadRequest)
		return
	}

	if err := h.db.Shipments.Delete(id); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Shipment not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to delete shipment: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetShipmentEvents handles GET /api/shipments/{id}/events
func (h *ShipmentHandler) GetShipmentEvents(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid shipment ID", http.StatusBadRequest)
		return
	}

	// Check if shipment exists
	_, err = h.db.Shipments.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Shipment not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get shipment: %v", err), http.StatusInternalServerError)
		return
	}

	// Get tracking events
	events, err := h.db.TrackingEvents.GetByShipmentID(id)
	if err != nil {
		log.Printf("ERROR: Failed to get tracking events for shipment %d: %v", id, err)
		http.Error(w, fmt.Sprintf("Failed to get tracking events: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(events)
}

// validateShipment validates shipment data
func validateShipment(shipment *database.Shipment) error {
	if shipment.TrackingNumber == "" {
		return fmt.Errorf("tracking number is required")
	}
	if shipment.Carrier == "" {
		return fmt.Errorf("carrier is required")
	}
	if shipment.Description == "" {
		return fmt.Errorf("description is required")
	}

	// Validate carrier
	validCarriers := []string{"ups", "usps", "fedex", "dhl"}
	validCarrier := false
	for _, c := range validCarriers {
		if shipment.Carrier == c {
			validCarrier = true
			break
		}
	}
	if !validCarrier {
		return fmt.Errorf("invalid carrier: must be one of %v", validCarriers)
	}

	return nil
}


// RefreshResponse represents the response from a manual refresh request
type RefreshResponse struct {
	ShipmentID  int                      `json:"shipment_id"`
	UpdatedAt   time.Time                `json:"updated_at"`
	EventsAdded int                      `json:"events_added"`
	TotalEvents int                      `json:"total_events"`
	Events      []database.TrackingEvent `json:"events"`
}

// RefreshShipment handles POST /api/shipments/{id}/refresh
func (h *ShipmentHandler) RefreshShipment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid shipment ID", http.StatusBadRequest)
		return
	}

	// Get the shipment
	shipment, err := h.db.Shipments.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Shipment not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get shipment: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if shipment is already delivered (409)
	if shipment.IsDelivered {
		http.Error(w, "Shipment already delivered - no need to refresh", http.StatusConflict)
		return
	}

	// Check rate limiting - 5 minutes between refreshes
	if shipment.LastManualRefresh != nil {
		timeSinceLastRefresh := time.Since(*shipment.LastManualRefresh)
		if timeSinceLastRefresh < 5*time.Minute {
			remainingTime := 5*time.Minute - timeSinceLastRefresh
			http.Error(w, fmt.Sprintf("Rate limit exceeded. Please wait %v before refreshing again", remainingTime.Truncate(time.Second)), http.StatusTooManyRequests)
			return
		}
	}

	// Force scraping client (bypass API)
	config := &carriers.CarrierConfig{
		PreferredType: carriers.ClientTypeScraping,
		UserAgent:     "Mozilla/5.0 (compatible; PackageTracker/1.0)",
	}
	h.factory.SetCarrierConfig(shipment.Carrier, config)

	// Create scraping client
	client, clientType, err := h.factory.CreateClient(shipment.Carrier)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create client for carrier %s: %v", shipment.Carrier, err), http.StatusServiceUnavailable)
		return
	}

	// Ensure we're using scraping
	if clientType != carriers.ClientTypeScraping {
		http.Error(w, "Scraping client not available for this carrier", http.StatusServiceUnavailable)
		return
	}

	// Get existing events count
	existingEvents, err := h.db.TrackingEvents.GetByShipmentID(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get existing events: %v", err), http.StatusInternalServerError)
		return
	}

	// Track the shipment using scraping
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &carriers.TrackingRequest{
		TrackingNumbers: []string{shipment.TrackingNumber},
		Carrier:         shipment.Carrier,
	}

	resp, err := client.Track(ctx, req)
	if err != nil {
		// Handle carrier errors
		if carrierErr, ok := err.(*carriers.CarrierError); ok {
			if carrierErr.RateLimit {
				http.Error(w, "Carrier rate limit exceeded. Please try again later", http.StatusTooManyRequests)
				return
			}
		}
		http.Error(w, fmt.Sprintf("Failed to scrape tracking data: %v", err), http.StatusBadGateway)
		return
	}

	// Process results
	eventsAdded := 0
	if len(resp.Results) > 0 {
		trackingInfo := resp.Results[0]

		// Update shipment status if changed
		if trackingInfo.Status != "" && string(trackingInfo.Status) != shipment.Status {
			shipment.Status = string(trackingInfo.Status)
			if trackingInfo.Status == carriers.StatusDelivered {
				shipment.IsDelivered = true
				if trackingInfo.ActualDelivery != nil {
					shipment.ExpectedDelivery = trackingInfo.ActualDelivery
				}
			}
		}

		// Add new tracking events
		for _, event := range trackingInfo.Events {
			dbEvent := &database.TrackingEvent{
				ShipmentID:  id,
				Timestamp:   event.Timestamp,
				Location:    event.Location,
				Status:      string(event.Status),
				Description: event.Description,
			}

			// CreateEvent has deduplication logic
			err := h.db.TrackingEvents.CreateEvent(dbEvent)
			if err != nil {
				// Log error but continue processing other events
				continue
			}
			eventsAdded++
		}

		// Update shipment in database
		err = h.db.Shipments.Update(id, shipment)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to update shipment: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Update refresh tracking
	err = h.db.Shipments.UpdateRefreshTracking(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update refresh tracking: %v", err), http.StatusInternalServerError)
		return
	}

	// Get updated events
	updatedEvents, err := h.db.TrackingEvents.GetByShipmentID(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get updated events: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate actual events added (in case some were deduplicated)
	actualEventsAdded := len(updatedEvents) - len(existingEvents)
	if actualEventsAdded < 0 {
		actualEventsAdded = 0
	}

	// Create response
	response := RefreshResponse{
		ShipmentID:  id,
		UpdatedAt:   time.Now(),
		EventsAdded: actualEventsAdded,
		TotalEvents: len(updatedEvents),
		Events:      updatedEvents,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}