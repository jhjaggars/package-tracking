package handlers

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// StaticHandler handles serving static files and SPA routing
type StaticHandler struct {
	fileSystem http.FileSystem
}

// NewStaticHandler creates a new static file handler
func NewStaticHandler(embeddedFS fs.FS) *StaticHandler {
	if embeddedFS != nil {
		// Use embedded files (production)
		return &StaticHandler{
			fileSystem: http.FS(embeddedFS),
		}
	}
	
	// Fall back to filesystem for development
	return &StaticHandler{
		fileSystem: http.Dir("./web/dist"),
	}
}

// ServeHTTP serves static files and handles SPA routing
func (h *StaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}
	
	// Try to serve the requested file
	file, err := h.fileSystem.Open(path)
	if err != nil {
		// If file doesn't exist and it's not an API route, serve index.html for SPA routing
		if !strings.HasPrefix(path, "/api/") {
			indexFile, indexErr := h.fileSystem.Open("/index.html")
			if indexErr != nil {
				http.NotFound(w, r)
				return
			}
			defer indexFile.Close()
			
			// Set security headers and content type for HTML
			h.setSecurityHeaders(w)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			http.ServeContent(w, r, "index.html", getModTime(), indexFile)
			return
		}
		
		// For API routes, return 404
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	
	// Get file info for proper serving
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	
	// Set security headers for all static content
	h.setSecurityHeaders(w)
	
	// Set appropriate content type and caching based on file extension
	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Don't cache HTML files to ensure fresh SPA routing
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
		w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
	case ".json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hour
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
		w.Header().Set("Cache-Control", "public, max-age=86400") // 1 day
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=86400") // 1 day
	}
	
	// Serve the file
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)
}

// setSecurityHeaders adds comprehensive security headers to responses
func (h *StaticHandler) setSecurityHeaders(w http.ResponseWriter) {
	// Content Security Policy - restrict resource loading
	csp := "default-src 'self'; " +
		"script-src 'self' 'unsafe-inline'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: https:; " +
		"font-src 'self' data:; " +
		"connect-src 'self'; " +
		"frame-ancestors 'none';"
	w.Header().Set("Content-Security-Policy", csp)
	
	// Prevent MIME sniffing
	w.Header().Set("X-Content-Type-Options", "nosniff")
	
	// Prevent clickjacking
	w.Header().Set("X-Frame-Options", "DENY")
	
	// XSS protection
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	
	// Referrer policy
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	
	// Strict Transport Security (HSTS) - only if HTTPS
	// w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
}

// getModTime returns a default modification time for embedded files
func getModTime() time.Time {
	return time.Time{} // Zero time for embedded files
}