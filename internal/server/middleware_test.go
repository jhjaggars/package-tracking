package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	middleware := LoggingMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "test" {
		t.Errorf("Expected body 'test', got '%s'", w.Body.String())
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORSMiddleware(handler)

	t.Run("Normal request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected CORS origin header to be set")
		}

		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Expected CORS methods header to be set")
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("OPTIONS request", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestRecoveryMiddleware(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	middleware := RecoveryMiddleware(panicHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic and should return 500
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Internal Server Error") {
		t.Error("Expected error message in response body")
	}
}

func TestContentTypeMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := ContentTypeMiddleware(handler)

	tests := []struct {
		path        string
		expectJSON  bool
		description string
	}{
		{"/api/shipments", true, "API route should get JSON content type"},
		{"/api/health", true, "API health route should get JSON content type"},
		{"/", false, "Non-API route should not get JSON content type"},
		{"/static/style.css", false, "Static route should not get JSON content type"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			contentType := w.Header().Get("Content-Type")
			if tt.expectJSON {
				if contentType != "application/json" {
					t.Errorf("Expected JSON content type for %s, got '%s'", tt.path, contentType)
				}
			} else {
				if contentType == "application/json" {
					t.Errorf("Did not expect JSON content type for %s, got '%s'", tt.path, contentType)
				}
			}
		})
	}
}

func TestSecurityMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := SecurityMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
	}

	for header, expectedValue := range expectedHeaders {
		if w.Header().Get(header) != expectedValue {
			t.Errorf("Expected header %s to be '%s', got '%s'", header, expectedValue, w.Header().Get(header))
		}
	}
}

func TestChain(t *testing.T) {
	var callOrder []string

	// Create test middleware that records call order
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "middleware1")
			next.ServeHTTP(w, r)
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "middleware2")
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callOrder = append(callOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})

	// Chain middleware
	chained := Chain(handler, middleware1, middleware2)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	chained.ServeHTTP(w, req)

	// Check call order
	expectedOrder := []string{"middleware1", "middleware2", "handler"}
	if len(callOrder) != len(expectedOrder) {
		t.Fatalf("Expected %d calls, got %d", len(expectedOrder), len(callOrder))
	}

	for i, expected := range expectedOrder {
		if callOrder[i] != expected {
			t.Errorf("Expected call %d to be '%s', got '%s'", i, expected, callOrder[i])
		}
	}
}

func TestIsAPIRoute(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/shipments", true},
		{"/api/health", true},
		{"/api", false}, // Too short
		{"/", false},
		{"/static/style.css", false},
		{"/about", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isAPIRoute(tt.path)
			if result != tt.expected {
				t.Errorf("Expected isAPIRoute(%s) to be %t, got %t", tt.path, tt.expected, result)
			}
		})
	}
}

func TestResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	wrapper := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// Test default status code
	if wrapper.statusCode != http.StatusOK {
		t.Errorf("Expected default status 200, got %d", wrapper.statusCode)
	}

	// Test WriteHeader
	wrapper.WriteHeader(http.StatusNotFound)
	if wrapper.statusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", wrapper.statusCode)
	}
}

func TestAuthMiddleware(t *testing.T) {
	testAPIKey := "test-secret-key-123"
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	})

	middleware := AuthMiddleware(testAPIKey)
	protectedHandler := middleware(handler)

	t.Run("ValidAPIKey", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/status", nil)
		req.Header.Set("Authorization", "Bearer "+testAPIKey)
		w := httptest.NewRecorder()

		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Body.String() != "authenticated" {
			t.Errorf("Expected body 'authenticated', got '%s'", w.Body.String())
		}
	})

	t.Run("MissingAuthorizationHeader", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/status", nil)
		w := httptest.NewRecorder()

		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}

		if !strings.Contains(w.Body.String(), "Unauthorized") {
			t.Error("Expected 'Unauthorized' in response body")
		}
	})

	t.Run("InvalidAuthorizationFormat", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/status", nil)
		req.Header.Set("Authorization", "Basic "+testAPIKey) // Wrong format
		w := httptest.NewRecorder()

		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("InvalidAPIKey", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/status", nil)
		req.Header.Set("Authorization", "Bearer wrong-key")
		w := httptest.NewRecorder()

		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("EmptyBearerToken", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/status", nil)
		req.Header.Set("Authorization", "Bearer ")
		w := httptest.NewRecorder()

		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("BearerOnly", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/status", nil)
		req.Header.Set("Authorization", "Bearer")
		w := httptest.NewRecorder()

		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("CaseSensitiveAPIKey", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/status", nil)
		req.Header.Set("Authorization", "Bearer "+strings.ToUpper(testAPIKey))
		w := httptest.NewRecorder()

		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for case-sensitive key, got %d", w.Code)
		}
	})
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name         string
		remoteAddr   string
		xForwardedFor string
		xRealIP      string
		expected     string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:          "X-Forwarded-For single IP",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.1",
			expected:      "203.0.113.1",
		},
		{
			name:          "X-Forwarded-For multiple IPs",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.1, 192.168.1.1, 10.0.0.1",
			expected:      "203.0.113.1",
		},
		{
			name:        "X-Real-IP",
			remoteAddr:  "10.0.0.1:12345",
			xRealIP:     "203.0.113.2",
			expected:    "203.0.113.2",
		},
		{
			name:          "X-Forwarded-For takes precedence over X-Real-IP",
			remoteAddr:    "10.0.0.1:12345",
			xForwardedFor: "203.0.113.1",
			xRealIP:       "203.0.113.2",
			expected:      "203.0.113.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			result := getClientIP(req)
			if result != tt.expected {
				t.Errorf("Expected IP '%s', got '%s'", tt.expected, result)
			}
		})
	}
}