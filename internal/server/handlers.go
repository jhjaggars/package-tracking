package server

import (
	"net/http"

	"package-tracking/internal/database"
	"package-tracking/internal/handlers"
)

// HandlerWrappers adapts our existing handlers to work with the router
type HandlerWrappers struct {
	shipmentHandler *handlers.ShipmentHandler
	healthHandler   *handlers.HealthHandler
	carrierHandler  *handlers.CarrierHandler
}

// NewHandlerWrappers creates new handler wrappers
func NewHandlerWrappers(db *database.DB) *HandlerWrappers {
	return &HandlerWrappers{
		shipmentHandler: handlers.NewShipmentHandler(db),
		healthHandler:   handlers.NewHealthHandler(db),
		carrierHandler:  handlers.NewCarrierHandler(db),
	}
}

// GetShipments wraps the shipment list handler
func (hw *HandlerWrappers) GetShipments(w http.ResponseWriter, r *http.Request, params map[string]string) {
	hw.shipmentHandler.GetShipments(w, r)
}

// CreateShipment wraps the create shipment handler
func (hw *HandlerWrappers) CreateShipment(w http.ResponseWriter, r *http.Request, params map[string]string) {
	hw.shipmentHandler.CreateShipment(w, r)
}

// GetShipmentByID wraps the get shipment by ID handler
func (hw *HandlerWrappers) GetShipmentByID(w http.ResponseWriter, r *http.Request, params map[string]string) {
	// Extract ID from parameters and add to request path for existing handler
	if _, ok := params["id"]; ok {
		// Our existing handler uses extractIDFromPath, so we need to ensure the path is correct
		// The router already handles this, so we can call the handler directly
		hw.shipmentHandler.GetShipmentByID(w, r)
	} else {
		http.Error(w, "Missing shipment ID", http.StatusBadRequest)
	}
}

// UpdateShipment wraps the update shipment handler
func (hw *HandlerWrappers) UpdateShipment(w http.ResponseWriter, r *http.Request, params map[string]string) {
	if _, ok := params["id"]; ok {
		hw.shipmentHandler.UpdateShipment(w, r)
	} else {
		http.Error(w, "Missing shipment ID", http.StatusBadRequest)
	}
}

// DeleteShipment wraps the delete shipment handler
func (hw *HandlerWrappers) DeleteShipment(w http.ResponseWriter, r *http.Request, params map[string]string) {
	if _, ok := params["id"]; ok {
		hw.shipmentHandler.DeleteShipment(w, r)
	} else {
		http.Error(w, "Missing shipment ID", http.StatusBadRequest)
	}
}

// GetShipmentEvents wraps the get shipment events handler
func (hw *HandlerWrappers) GetShipmentEvents(w http.ResponseWriter, r *http.Request, params map[string]string) {
	if _, ok := params["id"]; ok {
		hw.shipmentHandler.GetShipmentEvents(w, r)
	} else {
		http.Error(w, "Missing shipment ID", http.StatusBadRequest)
	}
}

// HealthCheck wraps the health check handler
func (hw *HandlerWrappers) HealthCheck(w http.ResponseWriter, r *http.Request, params map[string]string) {
	hw.healthHandler.HealthCheck(w, r)
}

// GetCarriers wraps the get carriers handler
func (hw *HandlerWrappers) GetCarriers(w http.ResponseWriter, r *http.Request, params map[string]string) {
	hw.carrierHandler.GetCarriers(w, r)
}

// RegisterRoutes registers all routes with the router
func (hw *HandlerWrappers) RegisterRoutes(router *Router) {
	// Shipment routes
	router.GET("/api/shipments", hw.GetShipments)
	router.POST("/api/shipments", hw.CreateShipment)
	router.GET("/api/shipments/{id}", hw.GetShipmentByID)
	router.PUT("/api/shipments/{id}", hw.UpdateShipment)
	router.DELETE("/api/shipments/{id}", hw.DeleteShipment)
	router.GET("/api/shipments/{id}/events", hw.GetShipmentEvents)

	// Other routes
	router.GET("/api/health", hw.HealthCheck)
	router.GET("/api/carriers", hw.GetCarriers)
}