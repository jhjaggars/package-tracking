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
	"package-tracking/internal/server"
	"package-tracking/internal/workers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Production builds will embed static files here
// For development, we'll use filesystem fallback
var embeddedFiles embed.FS

func main() {
	// Load configuration
	cfg, err := config.Load()
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

	// Initialize cache manager
	cacheManager := cache.NewManager(db.RefreshCache, cfg.GetDisableCache(), 5*time.Minute)
	defer cacheManager.Close()

	if cfg.GetDisableCache() {
		log.Printf("Cache disabled via configuration")
	} else {
		log.Printf("Cache initialized with 5 minute TTL")
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

	// Initialize structured logger for workers
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Initialize tracking updater
	trackingUpdater := workers.NewTrackingUpdater(cfg, db.Shipments, carrierFactory, logger)
	defer trackingUpdater.Stop()
	
	// Start the tracking updater
	trackingUpdater.Start()
	
	if cfg.AutoUpdateEnabled {
		log.Printf("Automatic tracking updates enabled (interval: %v, cutoff: %d days)", 
			cfg.UpdateInterval, cfg.AutoUpdateCutoffDays)
	} else {
		log.Printf("Automatic tracking updates disabled")
	}

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
	adminHandler := handlers.NewAdminHandler(trackingUpdater)
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
		r.Get("/health", healthHandler.HealthCheck)
		r.Get("/carriers", carrierHandler.GetCarriers)
		r.Get("/dashboard/stats", dashboardHandler.GetStats)
		
		// Admin routes
		r.Route("/admin", func(r chi.Router) {
			r.Get("/tracking-updater/status", adminHandler.GetTrackingUpdaterStatus)
			r.Post("/tracking-updater/pause", adminHandler.PauseTrackingUpdater)
			r.Post("/tracking-updater/resume", adminHandler.ResumeTrackingUpdater)
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