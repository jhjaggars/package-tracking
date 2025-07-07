package server

import (
	"net/http"
	"time"

	"package-tracking/internal/cache"
	"package-tracking/internal/database"
	"package-tracking/internal/handlers"

	"github.com/go-chi/chi/v5"
)

// TestConfig implements the Config interface for testing
type TestConfig struct {
	DisableRateLimit bool
	DisableCache     bool
}

func (tc *TestConfig) GetDisableRateLimit() bool {
	return tc.DisableRateLimit
}

func (tc *TestConfig) GetDisableCache() bool {
	return tc.DisableCache
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
	emailHandler    *handlers.EmailHandler
}

// NewHandlerWrappers creates new handler wrappers
func NewHandlerWrappers(db *database.DB) *HandlerWrappers {
	// Use default test config for backward compatibility
	config := &TestConfig{DisableRateLimit: false, DisableCache: true} // Disable cache in tests
	
	// Create a disabled cache manager for tests
	cacheManager := cache.NewManager(db.RefreshCache, true, 5*time.Minute)
	
	return &HandlerWrappers{
		shipmentHandler: handlers.NewShipmentHandler(db, config, cacheManager),
		healthHandler:   handlers.NewHealthHandler(db),
		carrierHandler:  handlers.NewCarrierHandler(db),
		staticHandler:   handlers.NewStaticHandler(nil), // Use filesystem fallback
		emailHandler:    handlers.NewEmailHandler(db),
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

// GetShipmentEmails wraps the get shipment emails handler
func (hw *HandlerWrappers) GetShipmentEmails(w http.ResponseWriter, r *http.Request, params map[string]string) {
	if _, ok := params["id"]; ok {
		hw.emailHandler.GetShipmentEmails(w, r)
	} else {
		http.Error(w, "Missing shipment ID", http.StatusBadRequest)
	}
}

// GetEmailThread wraps the get email thread handler
func (hw *HandlerWrappers) GetEmailThread(w http.ResponseWriter, r *http.Request, params map[string]string) {
	if _, ok := params["thread_id"]; ok {
		hw.emailHandler.GetEmailThread(w, r)
	} else {
		http.Error(w, "Missing thread ID", http.StatusBadRequest)
	}
}

// GetEmailBody wraps the get email body handler
func (hw *HandlerWrappers) GetEmailBody(w http.ResponseWriter, r *http.Request, params map[string]string) {
	if _, ok := params["email_id"]; ok {
		hw.emailHandler.GetEmailBody(w, r)
	} else {
		http.Error(w, "Missing email ID", http.StatusBadRequest)
	}
}

// LinkEmailToShipment wraps the link email to shipment handler
func (hw *HandlerWrappers) LinkEmailToShipment(w http.ResponseWriter, r *http.Request, params map[string]string) {
	if _, ok := params["email_id"]; ok {
		if _, ok := params["shipment_id"]; ok {
			hw.emailHandler.LinkEmailToShipment(w, r)
		} else {
			http.Error(w, "Missing shipment ID", http.StatusBadRequest)
		}
	} else {
		http.Error(w, "Missing email ID", http.StatusBadRequest)
	}
}

// UnlinkEmailFromShipment wraps the unlink email from shipment handler
func (hw *HandlerWrappers) UnlinkEmailFromShipment(w http.ResponseWriter, r *http.Request, params map[string]string) {
	if _, ok := params["email_id"]; ok {
		if _, ok := params["shipment_id"]; ok {
			hw.emailHandler.UnlinkEmailFromShipment(w, r)
		} else {
			http.Error(w, "Missing shipment ID", http.StatusBadRequest)
		}
	} else {
		http.Error(w, "Missing email ID", http.StatusBadRequest)
	}
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
	
	// Email-related routes (protected endpoints)
	router.GET("/api/shipments/{id}/emails", hw.GetShipmentEmails)
	router.GET("/api/emails/{thread_id}/thread", hw.GetEmailThread)
	router.GET("/api/emails/{email_id}/body", hw.GetEmailBody)
	router.POST("/api/emails/{email_id}/link/{shipment_id}", hw.LinkEmailToShipment)
	router.DELETE("/api/emails/{email_id}/link/{shipment_id}", hw.UnlinkEmailFromShipment)
	
	router.GET("/api/health", hw.HealthCheck)
	router.GET("/api/carriers", hw.GetCarriers)

	// Static file routes (catch-all for SPA)
	router.GET("/{path:.*}", hw.ServeStatic)
}

// RegisterChiRoutes registers all routes with a chi router
func (hw *HandlerWrappers) RegisterChiRoutes(r chi.Router) {
	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/shipments", hw.shipmentHandler.GetShipments)
		r.Post("/shipments", hw.shipmentHandler.CreateShipment)
		r.Get("/shipments/{id}", hw.shipmentHandler.GetShipmentByID)
		r.Put("/shipments/{id}", hw.shipmentHandler.UpdateShipment)
		r.Delete("/shipments/{id}", hw.shipmentHandler.DeleteShipment)
		r.Get("/shipments/{id}/events", hw.shipmentHandler.GetShipmentEvents)
		r.Post("/shipments/{id}/refresh", hw.shipmentHandler.RefreshShipment)
		
		// Email-related routes
		r.Get("/shipments/{id}/emails", hw.emailHandler.GetShipmentEmails)
		r.Get("/emails/{thread_id}/thread", hw.emailHandler.GetEmailThread)
		r.Get("/emails/{email_id}/body", hw.emailHandler.GetEmailBody)
		r.Post("/emails/{email_id}/link/{shipment_id}", hw.emailHandler.LinkEmailToShipment)
		r.Delete("/emails/{email_id}/link/{shipment_id}", hw.emailHandler.UnlinkEmailFromShipment)
		
		r.Get("/health", hw.healthHandler.HealthCheck)
		r.Get("/carriers", hw.carrierHandler.GetCarriers)
	})

	// Static file routes (catch-all for SPA)
	r.Get("/*", hw.staticHandler.ServeHTTP)
}