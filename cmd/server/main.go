// Copyright 2024 Package Tracking System
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"embed"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"package-tracking/internal/cache"
	"package-tracking/internal/carriers"
	"package-tracking/internal/config"
	"package-tracking/internal/database"
	"package-tracking/internal/handlers"
	"package-tracking/internal/parser"
	"package-tracking/internal/server"
	"package-tracking/internal/services"
	"package-tracking/internal/workers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Production builds will embed static files here
// For development, we'll use filesystem fallback
var embeddedFiles embed.FS

func main() {
	// Load configuration
	cfg, err := config.LoadServerConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	log.Printf("Database initialized at %s", cfg.DBPath)

	// Initialize cache manager with configurable TTL
	cacheManager := cache.NewManager(db.RefreshCache, cfg.GetDisableCache(), cfg.GetCacheTTL())
	defer cacheManager.Close()

	if cfg.GetDisableCache() {
		log.Printf("Cache disabled via configuration")
	} else {
		log.Printf("Cache initialized with %v TTL", cfg.GetCacheTTL())
	}

	// Initialize carrier factory
	carrierFactory := carriers.NewClientFactory()
	
	// Configure carriers with available API credentials
	if cfg.USPSAPIKey != "" {
		uspsConfig := &carriers.CarrierConfig{
			UserID:        cfg.USPSAPIKey,
			PreferredType: carriers.ClientTypeAPI,
		}
		carrierFactory.SetCarrierConfig("usps", uspsConfig)
		log.Printf("USPS API credentials configured")
	}

	// Configure UPS with OAuth credentials (preferred) or legacy API key
	if cfg.GetUPSClientID() != "" && cfg.GetUPSClientSecret() != "" {
		upsConfig := &carriers.CarrierConfig{
			ClientID:      cfg.GetUPSClientID(),
			ClientSecret:  cfg.GetUPSClientSecret(),
			PreferredType: carriers.ClientTypeAPI,
		}
		carrierFactory.SetCarrierConfig("ups", upsConfig)
		log.Printf("UPS OAuth credentials configured")
	} else if cfg.UPSAPIKey != "" {
		log.Printf("WARNING: UPS_API_KEY is deprecated. Please use UPS_CLIENT_ID and UPS_CLIENT_SECRET instead.")
		upsConfig := &carriers.CarrierConfig{
			UserID:        cfg.UPSAPIKey,
			PreferredType: carriers.ClientTypeAPI,
		}
		carrierFactory.SetCarrierConfig("ups", upsConfig)
		log.Printf("UPS legacy API credentials configured")
	}

	if cfg.FedExAPIKey != "" && cfg.FedExSecretKey != "" {
		fedexConfig := &carriers.CarrierConfig{
			ClientID:      cfg.FedExAPIKey,
			ClientSecret:  cfg.FedExSecretKey,
			BaseURL:       cfg.FedExAPIURL,
			PreferredType: carriers.ClientTypeAPI,
		}
		carrierFactory.SetCarrierConfig("fedex", fedexConfig)
		log.Printf("FedEx API credentials configured")
	}

	// Configure Amazon carrier (email-based tracking, no API credentials needed)
	amazonConfig := &carriers.CarrierConfig{
		PreferredType: carriers.ClientTypeScraping,
	}
	carrierFactory.SetCarrierConfig("amazon", amazonConfig)
	log.Printf("Amazon carrier configured (email-based tracking)")

	// Initialize structured logger for workers
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Initialize tracking updater with cache manager for unified rate limiting
	trackingUpdater := workers.NewTrackingUpdater(cfg, db.Shipments, carrierFactory, cacheManager, logger)
	defer trackingUpdater.Stop()
	
	// Start the tracking updater
	trackingUpdater.Start()
	
	if cfg.AutoUpdateEnabled {
		log.Printf("Automatic tracking updates enabled (interval: %v, cutoff: %d days)", 
			cfg.UpdateInterval, cfg.AutoUpdateCutoffDays)
		if cfg.UPSAutoUpdateEnabled {
			log.Printf("UPS auto-updates enabled (cutoff: %d days)", cfg.UPSAutoUpdateCutoffDays)
		} else {
			log.Printf("UPS auto-updates disabled")
		}
	} else {
		log.Printf("Automatic tracking updates disabled")
	}

	// Initialize description enhancer for admin API
	extractorConfig := &parser.ExtractorConfig{
		EnableLLM:           false, // LLM can be enabled via environment variables
		MinConfidence:       0.5,
		MaxCandidates:       10,
		UseHybridValidation: true,
		DebugMode:           false,
	}
	extractor := parser.NewTrackingExtractor(carrierFactory, extractorConfig, nil)
	descriptionEnhancer := services.NewDescriptionEnhancer(db.Shipments, db.Emails, extractor, logger)

	// Create chi router
	r := chi.NewRouter()

	// Add middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(server.CORSMiddleware)
	r.Use(server.ContentTypeMiddleware)
	r.Use(server.SecurityMiddleware)

	// Create embedded file system for static assets
	// For development, use filesystem fallback
	var staticFS fs.FS = nil

	// Create handlers
	shipmentHandler := handlers.NewShipmentHandlerWithFactory(db, cfg, cacheManager, carrierFactory)
	healthHandler := handlers.NewHealthHandler(db)
	carrierHandler := handlers.NewCarrierHandler(db)
	dashboardHandler := handlers.NewDashboardHandler(db)
	adminHandler := handlers.NewAdminHandler(trackingUpdater, descriptionEnhancer, logger)
	emailHandler := handlers.NewEmailHandler(db)
	staticHandler := handlers.NewStaticHandler(staticFS)

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/shipments", shipmentHandler.GetShipments)
		r.Post("/shipments", shipmentHandler.CreateShipment)
		r.Get("/shipments/{id}", shipmentHandler.GetShipmentByID)
		r.Put("/shipments/{id}", shipmentHandler.UpdateShipment)
		r.Delete("/shipments/{id}", shipmentHandler.DeleteShipment)
		r.Get("/shipments/{id}/events", shipmentHandler.GetShipmentEvents)
		r.Post("/shipments/{id}/refresh", shipmentHandler.RefreshShipment)
		
		// Email-related routes
		r.Get("/shipments/{id}/emails", emailHandler.GetShipmentEmails)
		r.Get("/emails/{thread_id}/thread", emailHandler.GetEmailThread)
		r.Get("/emails/{email_id}/body", emailHandler.GetEmailBody)
		r.Post("/emails/{email_id}/link/{shipment_id}", emailHandler.LinkEmailToShipment)
		r.Delete("/emails/{email_id}/link/{shipment_id}", emailHandler.UnlinkEmailFromShipment)
		
		r.Get("/health", healthHandler.HealthCheck)
		r.Get("/carriers", carrierHandler.GetCarriers)
		r.Get("/dashboard/stats", dashboardHandler.GetStats)
		
		// Admin routes
		r.Route("/admin", func(r chi.Router) {
			// Apply authentication middleware if not disabled
			if !cfg.GetDisableAdminAuth() {
				r.Use(server.AuthMiddleware(cfg.GetAdminAPIKey()))
				log.Printf("Admin API authentication enabled")
			} else {
				log.Printf("Admin API authentication disabled")
			}
			
			r.Get("/tracking-updater/status", adminHandler.GetTrackingUpdaterStatus)
			r.Post("/tracking-updater/pause", adminHandler.PauseTrackingUpdater)
			r.Post("/tracking-updater/resume", adminHandler.ResumeTrackingUpdater)
			r.Post("/enhance-descriptions", adminHandler.EnhanceDescriptions)
		})
	})

	// Static file routes (catch-all for SPA)
	r.Get("/*", staticHandler.ServeHTTP)

	srv := &http.Server{
		Addr:    cfg.Address(),
		Handler: r,
		
		// Timeouts
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Handle server startup and graceful shutdown
	shutdownTimeout := 30 * time.Second
	if err := server.HandleSignals(srv, shutdownTimeout); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}