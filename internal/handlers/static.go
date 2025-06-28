package handlers

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// Note: embed directive should be at package level and point to correct path
// For now, we'll use the filesystem directly in development

// StaticHandler handles serving static files and SPA routing
type StaticHandler struct {
	fileSystem http.FileSystem
}

// NewStaticHandler creates a new static file handler
func NewStaticHandler() *StaticHandler {
	// For development, serve from filesystem
	// In production, this would use embedded files
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
			
			// Set content type for HTML
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
	
	// Set appropriate content type based on file extension
	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	}
	
	// Serve the file
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)
}

// getModTime returns a default modification time for embedded files
func getModTime() time.Time {
	return time.Time{} // Zero time for embedded files
}