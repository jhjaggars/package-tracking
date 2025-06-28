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
	"package-tracking/internal/server"
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

	// Create router and register routes
	router := server.NewRouter()
	handlerWrappers := server.NewHandlerWrappers(db)
	handlerWrappers.RegisterRoutes(router)

	// Create HTTP server with middleware
	handler := server.Chain(
		router,
		server.LoggingMiddleware,
		server.RecoveryMiddleware,
		server.CORSMiddleware,
		server.ContentTypeMiddleware,
		server.SecurityMiddleware,
	)

	srv := &http.Server{
		Addr:    cfg.Address(),
		Handler: handler,
		
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