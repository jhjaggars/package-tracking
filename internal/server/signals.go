package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// SignalHandler manages graceful shutdown of the HTTP server
type SignalHandler struct {
	server          *http.Server
	shutdownTimeout time.Duration
}

// NewSignalHandler creates a new signal handler
func NewSignalHandler(server *http.Server, shutdownTimeout time.Duration) *SignalHandler {
	return &SignalHandler{
		server:          server,
		shutdownTimeout: shutdownTimeout,
	}
}

// WaitForShutdown waits for shutdown signals and handles graceful shutdown
func (sh *SignalHandler) WaitForShutdown() {
	// Create channel to receive OS signals
	quit := make(chan os.Signal, 1)

	// Register the channel to receive specific signals
	// SIGINT - typically sent by Ctrl+C
	// SIGTERM - standard termination signal sent by process managers
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive a signal
	sig := <-quit
	log.Printf("Received signal: %v", sig)
	log.Println("Initiating graceful shutdown...")

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), sh.shutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := sh.server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown due to timeout: %v", err)
	} else {
		log.Println("Server gracefully shut down")
	}
}

// HandleSignals is a convenience function that combines server start and signal handling
func HandleSignals(server *http.Server, shutdownTimeout time.Duration) error {
	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for shutdown signal
	handler := NewSignalHandler(server, shutdownTimeout)
	handler.WaitForShutdown()

	return nil
}

// Notes about signal handling:
//
// Catchable signals (can be handled gracefully):
// - SIGINT (2): Interrupt signal, typically sent by Ctrl+C
// - SIGTERM (15): Termination signal, standard way to request program termination
// - SIGHUP (1): Hangup signal, often used to reload configuration
// - SIGUSR1 (10): User-defined signal 1
// - SIGUSR2 (12): User-defined signal 2
//
// Uncatchable signals (cannot be handled):
// - SIGKILL (9): Kill signal, immediately terminates the process
// - SIGSTOP (19): Stop signal, suspends the process (cannot be caught or ignored)
//
// Our server handles SIGINT and SIGTERM gracefully, but SIGKILL will
// immediately terminate the process without any cleanup.