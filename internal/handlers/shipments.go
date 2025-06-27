package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"package-tracking/internal/database"
)

// ShipmentHandler handles HTTP requests for shipments
type ShipmentHandler struct {
	db *database.DB
}

// NewShipmentHandler creates a new shipment handler
func NewShipmentHandler(db *database.DB) *ShipmentHandler {
	return &ShipmentHandler{db: db}
}

// GetShipments handles GET /api/shipments
func (h *ShipmentHandler) GetShipments(w http.ResponseWriter, r *http.Request) {
	shipments, err := h.db.Shipments.GetAll()
	if err != nil {
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
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if err := validateShipment(&shipment); err != nil {
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
			http.Error(w, "Tracking number already exists", http.StatusConflict)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to create shipment: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(shipment)
}

// GetShipmentByID handles GET /api/shipments/{id}
func (h *ShipmentHandler) GetShipmentByID(w http.ResponseWriter, r *http.Request) {
	id, err := extractIDFromPath(r.URL.Path)
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
		http.Error(w, fmt.Sprintf("Failed to get shipment: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(shipment)
}

// UpdateShipment handles PUT /api/shipments/{id}
func (h *ShipmentHandler) UpdateShipment(w http.ResponseWriter, r *http.Request) {
	id, err := extractIDFromPath(r.URL.Path)
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
		http.Error(w, fmt.Sprintf("Failed to update shipment: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(shipment)
}

// DeleteShipment handles DELETE /api/shipments/{id}
func (h *ShipmentHandler) DeleteShipment(w http.ResponseWriter, r *http.Request) {
	id, err := extractIDFromPath(r.URL.Path)
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
	id, err := extractIDFromPath(r.URL.Path)
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

// extractIDFromPath extracts ID from URL path like /api/shipments/123
func extractIDFromPath(path string) (int, error) {
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		return 0, fmt.Errorf("invalid path")
	}
	
	idStr := parts[3]
	// Handle paths like /api/shipments/123/events
	if len(parts) > 4 && parts[4] == "events" {
		idStr = parts[3]
	}
	
	return strconv.Atoi(idStr)
}