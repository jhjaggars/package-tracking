package server

import (
	"crypto/subtle"
	"log"
	"net/http"
	"strings"
	"time"
)

// Middleware represents a middleware function
type Middleware func(http.Handler) http.Handler

// Chain applies multiple middleware functions to a handler
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapper := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		// Call the next handler
		next.ServeHTTP(wrapper, r)
		
		// Log the request with different levels based on status code
		duration := time.Since(start)
		if wrapper.statusCode >= 500 {
			log.Printf("ERROR: %s %s %d %v", r.Method, r.URL.Path, wrapper.statusCode, duration)
		} else if wrapper.statusCode >= 400 {
			log.Printf("WARN: %s %s %d %v", r.Method, r.URL.Path, wrapper.statusCode, duration)
		} else {
			log.Printf("INFO: %s %s %d %v", r.Method, r.URL.Path, wrapper.statusCode, duration)
		}
	})
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware recovers from panics and returns 500 error
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		
		next.ServeHTTP(w, r)
	})
}

// ContentTypeMiddleware sets JSON content type for API routes
func ContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set JSON content type for API routes
		if isAPIRoute(r.URL.Path) {
			w.Header().Set("Content-Type", "application/json")
		}
		
		next.ServeHTTP(w, r)
	})
}

// SecurityMiddleware adds basic security headers
func SecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		
		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware validates API key authentication for admin routes
func AuthMiddleware(apiKey string) func(http.Handler) http.Handler {
	expectedKey := []byte(apiKey)
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Printf("WARN: Unauthorized access attempt to %s %s from %s: missing authorization header", 
					r.Method, r.URL.Path, getClientIP(r))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			
			// Check for Bearer token format
			if !strings.HasPrefix(authHeader, "Bearer ") {
				log.Printf("WARN: Unauthorized access attempt to %s %s from %s: invalid authorization format", 
					r.Method, r.URL.Path, getClientIP(r))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			
			// Extract token
			token := strings.TrimPrefix(authHeader, "Bearer ")
			providedKey := []byte(token)
			
			// Use constant-time comparison to prevent timing attacks
			if len(providedKey) != len(expectedKey) || 
			   subtle.ConstantTimeCompare(providedKey, expectedKey) != 1 {
				log.Printf("WARN: Unauthorized access attempt to %s %s from %s: invalid API key", 
					r.Method, r.URL.Path, getClientIP(r))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			
			// Authentication successful, proceed to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	
	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// isAPIRoute checks if the path is an API route
func isAPIRoute(path string) bool {
	return strings.HasPrefix(path, "/api")
}

// MethodMiddleware converts router methods to standard middleware
func MethodMiddleware(router *Router) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to find a route in our router
			route, params := router.findRoute(r.Method, r.URL.Path)
			if route != nil {
				// Use our router's handler
				route.Handler(w, r, params)
				return
			}
			
			// Fall back to the next handler
			next.ServeHTTP(w, r)
		})
	}
}