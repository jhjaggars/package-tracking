package server

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// HandlerFunc is our custom handler function that includes path parameters
type HandlerFunc func(w http.ResponseWriter, r *http.Request, params map[string]string)

// Route represents a single route with method, pattern, and handler
type Route struct {
	Method  string
	Pattern string
	Handler HandlerFunc
	regex   *regexp.Regexp
	params  []string
}

// Router handles HTTP routing with path parameter extraction
type Router struct {
	routes []Route
}

// NewRouter creates a new router instance
func NewRouter() *Router {
	return &Router{
		routes: make([]Route, 0),
	}
}

// GET registers a GET route
func (r *Router) GET(pattern string, handler HandlerFunc) {
	r.addRoute("GET", pattern, handler)
}

// POST registers a POST route
func (r *Router) POST(pattern string, handler HandlerFunc) {
	r.addRoute("POST", pattern, handler)
}

// PUT registers a PUT route
func (r *Router) PUT(pattern string, handler HandlerFunc) {
	r.addRoute("PUT", pattern, handler)
}

// DELETE registers a DELETE route
func (r *Router) DELETE(pattern string, handler HandlerFunc) {
	r.addRoute("DELETE", pattern, handler)
}

// addRoute adds a route to the router
func (r *Router) addRoute(method, pattern string, handler HandlerFunc) {
	route := Route{
		Method:  method,
		Pattern: pattern,
		Handler: handler,
	}

	// Convert pattern to regex and extract parameter names
	route.regex, route.params = patternToRegex(pattern)
	r.routes = append(r.routes, route)
}

// ServeHTTP implements http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Find matching route
	route, params := r.findRoute(req.Method, req.URL.Path)
	if route == nil {
		http.NotFound(w, req)
		return
	}

	// Call the handler with extracted parameters
	route.Handler(w, req, params)
}

// findRoute finds a matching route and extracts parameters
func (r *Router) findRoute(method, path string) (*Route, map[string]string) {
	for _, route := range r.routes {
		if route.Method != method {
			continue
		}

		matches := route.regex.FindStringSubmatch(path)
		if matches == nil {
			continue
		}

		// Extract parameters
		params := make(map[string]string)
		for i, paramName := range route.params {
			if i+1 < len(matches) {
				params[paramName] = matches[i+1]
			}
		}

		return &route, params
	}

	return nil, nil
}

// patternToRegex converts a pattern like "/api/shipments/{id}" to a regex
func patternToRegex(pattern string) (*regexp.Regexp, []string) {
	var params []string
	
	// Escape special regex characters except {/}
	escaped := regexp.QuoteMeta(pattern)
	
	// Find parameter placeholders like {id}
	paramRegex := regexp.MustCompile(`\\{([^}]+)\\}`)
	
	// Replace each {param} with a capturing group
	regexPattern := paramRegex.ReplaceAllStringFunc(escaped, func(match string) string {
		// Extract parameter name (remove \{ and \})
		paramName := strings.Trim(match, "\\{}")
		params = append(params, paramName)
		return `([^/]+)` // Match any characters except slash
	})
	
	// Anchor the pattern to match the entire path
	regexPattern = "^" + regexPattern + "$"
	
	regex := regexp.MustCompile(regexPattern)
	return regex, params
}

// HandlerAdapter converts our HandlerFunc to standard http.HandlerFunc
func HandlerAdapter(handler HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract path parameters manually for adapted handlers
		params := extractPathParams(r.URL.Path)
		handler(w, r, params)
	}
}

// extractPathParams extracts parameters from common patterns (fallback for adapted handlers)
func extractPathParams(path string) map[string]string {
	params := make(map[string]string)
	
	// Handle /api/shipments/{id} pattern
	if strings.HasPrefix(path, "/api/shipments/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 4 && parts[3] != "" {
			// Check if it's a pure ID (not "events")
			if parts[3] != "events" {
				// For paths like /api/shipments/123/events, don't extract the ID
				// since this should be handled by proper routing
				if len(parts) == 4 {
					params["id"] = parts[3]
				}
			}
		}
	}
	
	return params
}

// GetParam extracts a parameter value and converts to int
func GetParam(params map[string]string, key string) (int, error) {
	value, exists := params[key]
	if !exists {
		return 0, fmt.Errorf("parameter %s not found", key)
	}
	
	id, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid parameter %s: %s", key, value)
	}
	
	return id, nil
}

// SetParam adds a parameter to the request context (alternative approach)
func SetParam(r *http.Request, key, value string) *http.Request {
	ctx := context.WithValue(r.Context(), key, value)
	return r.WithContext(ctx)
}

// GetParamFromContext extracts a parameter from request context
func GetParamFromContext(r *http.Request, key string) (string, bool) {
	value := r.Context().Value(key)
	if value == nil {
		return "", false
	}
	
	str, ok := value.(string)
	return str, ok
}