package server

import (
	"net/http"

	"package-tracking/internal/database"
	"package-tracking/internal/handlers"
)

// TestConfig implements the Config interface for testing
type TestConfig struct {
	DisableRateLimit bool
}

func (tc *TestConfig) GetDisableRateLimit() bool {
	return tc.DisableRateLimit
}

func (tc *TestConfig) GetFedExAPIKey() string {
	return ""
}

func (tc *TestConfig) GetFedExSecretKey() string {
	return ""
}

func (tc *TestConfig) GetFedExAPIURL() string {
	return "https://apis.fedex.com"
}

// HandlerWrappers adapts our existing handlers to work with the router
type HandlerWrappers struct {
	shipmentHandler *handlers.ShipmentHandler
	healthHandler   *handlers.HealthHandler
	carrierHandler  *handlers.CarrierHandler
	staticHandler   *handlers.StaticHandler
}

// NewHandlerWrappers creates new handler wrappers
func NewHandlerWrappers(db *database.DB) *HandlerWrappers {
	// Use default test config for backward compatibility
	config := &TestConfig{DisableRateLimit: false}
	return &HandlerWrappers{
		shipmentHandler: handlers.NewShipmentHandler(db, config),
		healthHandler:   handlers.NewHealthHandler(db),
		carrierHandler:  handlers.NewCarrierHandler(db),
		staticHandler:   handlers.NewStaticHandler(nil), // Use filesystem fallback
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

// RefreshShipment wraps the refresh shipment handler
func (hw *HandlerWrappers) RefreshShipment(w http.ResponseWriter, r *http.Request, params map[string]string) {
	if _, ok := params["id"]; ok {
		hw.shipmentHandler.RefreshShipment(w, r)
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

// ServeStatic wraps the static file handler
func (hw *HandlerWrappers) ServeStatic(w http.ResponseWriter, r *http.Request, params map[string]string) {
	hw.staticHandler.ServeHTTP(w, r)
}

// RegisterRoutes registers all routes with the router
func (hw *HandlerWrappers) RegisterRoutes(router *Router) {
	// API routes
	router.GET("/api/shipments", hw.GetShipments)
	router.POST("/api/shipments", hw.CreateShipment)
	router.GET("/api/shipments/{id}", hw.GetShipmentByID)
	router.PUT("/api/shipments/{id}", hw.UpdateShipment)
	router.DELETE("/api/shipments/{id}", hw.DeleteShipment)
	router.GET("/api/shipments/{id}/events", hw.GetShipmentEvents)
	router.POST("/api/shipments/{id}/refresh", hw.RefreshShipment)
	router.GET("/api/health", hw.HealthCheck)
	router.GET("/api/carriers", hw.GetCarriers)

	// Static file routes (catch-all for SPA)
	router.GET("/{path:.*}", hw.ServeStatic)
}