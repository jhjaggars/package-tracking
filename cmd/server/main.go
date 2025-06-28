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
	"log"
	"net/http"
	"time"

	"package-tracking/internal/config"
	"package-tracking/internal/database"
	"package-tracking/internal/handlers"
	"package-tracking/internal/server"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

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

	// Create chi router
	r := chi.NewRouter()

	// Add middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(server.CORSMiddleware)
	r.Use(server.ContentTypeMiddleware)
	r.Use(server.SecurityMiddleware)

	// Create handlers
	shipmentHandler := handlers.NewShipmentHandler(db)
	healthHandler := handlers.NewHealthHandler(db)
	carrierHandler := handlers.NewCarrierHandler(db)
	staticHandler := handlers.NewStaticHandler()

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