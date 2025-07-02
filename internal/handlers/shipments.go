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

	"package-tracking/internal/cache"
	"package-tracking/internal/carriers"
	"package-tracking/internal/database"

	"github.com/go-chi/chi/v5"
)

// Config interface to avoid circular imports
type Config interface {
	GetDisableRateLimit() bool
	GetDisableCache() bool
	// Add FedEx API configuration getters
	GetFedExAPIKey() string
	GetFedExSecretKey() string
	GetFedExAPIURL() string
}

// ShipmentHandler handles HTTP requests for shipments
type ShipmentHandler struct {
	db      *database.DB
	factory *carriers.ClientFactory
	config  Config
	cache   *cache.Manager
}

// NewShipmentHandler creates a new shipment handler
func NewShipmentHandler(db *database.DB, config Config, cacheManager *cache.Manager) *ShipmentHandler {
	factory := carriers.NewClientFactory()
	
	// Configure FedEx API if credentials are available
	if config.GetFedExAPIKey() != "" && config.GetFedExSecretKey() != "" {
		fedexConfig := &carriers.CarrierConfig{
			ClientID:      config.GetFedExAPIKey(),
			ClientSecret:  config.GetFedExSecretKey(),
			BaseURL:       config.GetFedExAPIURL(),
			PreferredType: carriers.ClientTypeAPI,
			UseSandbox:    false, // Use BaseURL for endpoint selection
		}
		factory.SetCarrierConfig("fedex", fedexConfig)
	}
	
	return &ShipmentHandler{
		db:      db,
		factory: factory,
		config:  config,
		cache:   cacheManager,
	}
}

// NewShipmentHandlerWithFactory creates a new shipment handler with an external carrier factory
func NewShipmentHandlerWithFactory(db *database.DB, config Config, cacheManager *cache.Manager, factory *carriers.ClientFactory) *ShipmentHandler {
	return &ShipmentHandler{
		db:      db,
		factory: factory,
		config:  config,
		cache:   cacheManager,
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

	// Invalidate cache for updated shipment
	if err := h.cache.Delete(id); err != nil {
		log.Printf("WARN: Failed to invalidate cache for shipment %d: %v", id, err)
		// Continue anyway - cache invalidation failure shouldn't break the response
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

	// Invalidate cache for deleted shipment
	if err := h.cache.Delete(id); err != nil {
		log.Printf("WARN: Failed to invalidate cache for deleted shipment %d: %v", id, err)
		// Continue anyway - cache invalidation failure shouldn't break the response
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
	ShipmentID       int                      `json:"shipment_id"`
	UpdatedAt        time.Time                `json:"updated_at"`
	EventsAdded      int                      `json:"events_added"`
	TotalEvents      int                      `json:"total_events"`
	Events           []database.TrackingEvent `json:"events"`
	CacheStatus      string                   `json:"cache_status"`      // "hit", "miss", "forced", "disabled"
	RefreshDuration  string                   `json:"refresh_duration"`  // How long the refresh took
	PreviousCacheAge string                   `json:"previous_cache_age"` // Age of cache that was invalidated
}

// RefreshShipment handles POST /api/shipments/{id}/refresh
func (h *ShipmentHandler) RefreshShipment(w http.ResponseWriter, r *http.Request) {
	refreshStart := time.Now()
	
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid shipment ID", http.StatusBadRequest)
		return
	}

	// Check for force parameter
	forceRefresh := r.URL.Query().Get("force") == "true"
	log.Printf("DEBUG: Force refresh parameter: %v", forceRefresh)
	
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

	var cacheStatus string
	var previousCacheAge string
	
	// Check if cache is disabled
	if !h.cache.IsEnabled() {
		cacheStatus = "disabled"
	} else if forceRefresh {
		// Handle force refresh - invalidate cache first
		log.Printf("INFO: Force refresh requested for shipment %d", id)
		cacheAge, err := h.cache.ForceInvalidate(id)
		if err != nil {
			log.Printf("WARN: Failed to invalidate cache for shipment %d: %v", id, err)
		}
		if cacheAge != nil {
			previousCacheAge = cacheAge.Truncate(time.Second).String()
			log.Printf("INFO: Invalidated cache for shipment %d (age: %s)", id, previousCacheAge)
		}
		cacheStatus = "forced"
	} else {
		// Check cache first - if we have fresh data, return it without rate limiting
		if cachedResponse, err := h.cache.Get(id); err == nil && cachedResponse != nil {
			log.Printf("DEBUG: Serving cached refresh response for shipment %d", id)
			
			// Convert database.RefreshResponse back to handlers.RefreshResponse
			response := RefreshResponse{
				ShipmentID:      cachedResponse.ShipmentID,
				UpdatedAt:       cachedResponse.UpdatedAt,
				EventsAdded:     cachedResponse.EventsAdded,
				TotalEvents:     cachedResponse.TotalEvents,
				Events:          cachedResponse.Events,
				CacheStatus:     "hit",
				RefreshDuration: time.Since(refreshStart).Truncate(time.Millisecond).String(),
			}
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		} else if err != nil {
			log.Printf("WARN: Cache error for shipment %d: %v", id, err)
			// Continue with normal flow if cache error
		}
		cacheStatus = "miss"
	}

	// Check rate limiting - 5 minutes between refreshes (unless disabled or forced)
	if !h.config.GetDisableRateLimit() && !forceRefresh && shipment.LastManualRefresh != nil {
		timeSinceLastRefresh := time.Since(*shipment.LastManualRefresh)
		if timeSinceLastRefresh < 5*time.Minute {
			remainingTime := 5*time.Minute - timeSinceLastRefresh
			http.Error(w, fmt.Sprintf("Rate limit exceeded. Please wait %v before refreshing again", remainingTime.Truncate(time.Second)), http.StatusTooManyRequests)
			return
		}
	}

	// Create client for tracking - prefer API for FedEx, fallback to headless/scraping for others
	var client carriers.Client
	var clientType carriers.ClientType
	
	// Check if we have an existing config that includes API credentials
	if shipment.Carrier == "fedex" && h.config.GetFedExAPIKey() != "" && h.config.GetFedExSecretKey() != "" {
		// Use existing FedEx API configuration
		client, clientType, err = h.factory.CreateClient(shipment.Carrier)
	} else {
		// Force fresh data collection (prefer headless/scraping)
		config := &carriers.CarrierConfig{
			PreferredType: carriers.ClientTypeHeadless, // Try headless first
			UseHeadless:   true,
			UserAgent:     "Mozilla/5.0 (compatible; PackageTracker/1.0)",
		}
		h.factory.SetCarrierConfig(shipment.Carrier, config)
		client, clientType, err = h.factory.CreateClient(shipment.Carrier)
		
		// For non-FedEx carriers, ensure we're not using API for "fresh" data collection
		if clientType == carriers.ClientTypeAPI && shipment.Carrier != "fedex" {
			http.Error(w, "Fresh data collection client not available for this carrier", http.StatusServiceUnavailable)
			return
		}
	}
	
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create client for carrier %s: %v", shipment.Carrier, err), http.StatusServiceUnavailable)
		return
	}

	// Get existing events count
	existingEvents, err := h.db.TrackingEvents.GetByShipmentID(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get existing events: %v", err), http.StatusInternalServerError)
		return
	}

	// Track the shipment using fresh data collection (extended timeout for SPA sites)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
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
		log.Printf("ERROR: Failed to fetch tracking data: %v", err)
		http.Error(w, fmt.Sprintf("Failed to fetch tracking data: %v", err), http.StatusBadGateway)
		return
	}

	// Debug: Log the tracking response
	log.Printf("DEBUG: Tracking response received - Results: %d, Errors: %d", len(resp.Results), len(resp.Errors))
	if len(resp.Results) > 0 {
		result := resp.Results[0]
		log.Printf("DEBUG: TrackingInfo - Status: %s, Events: %d, LastUpdated: %v", result.Status, len(result.Events), result.LastUpdated)
		for i, event := range result.Events {
			log.Printf("DEBUG: Event %d - %v: %s at %s (%s)", i, event.Timestamp, event.Description, event.Location, event.Status)
		}
	}
	for i, err := range resp.Errors {
		log.Printf("DEBUG: Error %d - %s: %s (Code: %s)", i, err.Carrier, err.Message, err.Code)
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
		ShipmentID:       id,
		UpdatedAt:        time.Now(),
		EventsAdded:      actualEventsAdded,
		TotalEvents:      len(updatedEvents),
		Events:           updatedEvents,
		CacheStatus:      cacheStatus,
		RefreshDuration:  time.Since(refreshStart).Truncate(time.Millisecond).String(),
		PreviousCacheAge: previousCacheAge,
	}

	// Convert to database.RefreshResponse for caching
	dbResponse := &database.RefreshResponse{
		ShipmentID:  response.ShipmentID,
		UpdatedAt:   response.UpdatedAt,
		EventsAdded: response.EventsAdded,
		TotalEvents: response.TotalEvents,
		Events:      response.Events,
	}

	// Store successful response in cache
	if err := h.cache.Set(id, dbResponse); err != nil {
		log.Printf("WARN: Failed to cache refresh response for shipment %d: %v", id, err)
		// Continue anyway - caching failure shouldn't break the response
	}

	// Debug: Log response summary (without sensitive data)
	log.Printf("DEBUG: Refresh response - ShipmentID: %d, EventsAdded: %d, CacheStatus: %s, Duration: %s", 
		response.ShipmentID, response.EventsAdded, response.CacheStatus, response.RefreshDuration)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}