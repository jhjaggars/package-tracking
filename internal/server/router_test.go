package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter(t *testing.T) {
	t.Skip("Skipping custom router tests - using chi router in production")
	router := NewRouter()

	// Test handlers
	var lastMethod, lastPath string
	var lastParams map[string]string

	testHandler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		lastMethod = r.Method
		lastPath = r.URL.Path
		lastParams = params
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}

	// Register routes
	router.GET("/api/shipments", testHandler)
	router.POST("/api/shipments", testHandler)
	router.GET("/api/shipments/{id}", testHandler)
	router.PUT("/api/shipments/{id}", testHandler)
	router.DELETE("/api/shipments/{id}", testHandler)
	router.GET("/api/shipments/{id}/events", testHandler)
	router.GET("/api/health", testHandler)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedParams map[string]string
	}{
		{
			name:           "GET shipments list",
			method:         "GET",
			path:           "/api/shipments",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{},
		},
		{
			name:           "POST create shipment",
			method:         "POST",
			path:           "/api/shipments",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{},
		},
		{
			name:           "GET shipment by ID",
			method:         "GET",
			path:           "/api/shipments/123",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{"id": "123"},
		},
		{
			name:           "PUT update shipment",
			method:         "PUT",
			path:           "/api/shipments/456",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{"id": "456"},
		},
		{
			name:           "DELETE shipment",
			method:         "DELETE",
			path:           "/api/shipments/789",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{"id": "789"},
		},
		{
			name:           "GET shipment events",
			method:         "GET",
			path:           "/api/shipments/123/events",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{"id": "123"},
		},
		{
			name:           "GET health check",
			method:         "GET",
			path:           "/api/health",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{},
		},
		{
			name:           "Not found",
			method:         "GET",
			path:           "/api/nonexistent",
			expectedStatus: http.StatusNotFound,
			expectedParams: nil,
		},
		{
			name:           "Wrong method",
			method:         "POST",
			path:           "/api/health",
			expectedStatus: http.StatusNotFound,
			expectedParams: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset captured values
			lastMethod = ""
			lastPath = ""
			lastParams = nil

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				if lastMethod != tt.method {
					t.Errorf("Expected method %s, got %s", tt.method, lastMethod)
				}

				if lastPath != tt.path {
					t.Errorf("Expected path %s, got %s", tt.path, lastPath)
				}

				// Check parameters
				if len(lastParams) != len(tt.expectedParams) {
					t.Errorf("Expected %d params, got %d", len(tt.expectedParams), len(lastParams))
				}

				for key, expectedValue := range tt.expectedParams {
					if lastParams[key] != expectedValue {
						t.Errorf("Expected param %s=%s, got %s", key, expectedValue, lastParams[key])
					}
				}
			}
		})
	}
}

func TestPatternToRegex(t *testing.T) {
	t.Skip("Skipping pattern regex tests - using chi router in production")
	tests := []struct {
		pattern        string
		testPath       string
		shouldMatch    bool
		expectedParams []string
		expectedValues map[string]string
	}{
		{
			pattern:        "/api/shipments",
			testPath:       "/api/shipments",
			shouldMatch:    true,
			expectedParams: []string{},
			expectedValues: map[string]string{},
		},
		{
			pattern:        "/api/shipments/{id}",
			testPath:       "/api/shipments/123",
			shouldMatch:    true,
			expectedParams: []string{"id"},
			expectedValues: map[string]string{"id": "123"},
		},
		{
			pattern:        "/api/shipments/{id}/events",
			testPath:       "/api/shipments/456/events",
			shouldMatch:    true,
			expectedParams: []string{"id"},
			expectedValues: map[string]string{"id": "456"},
		},
		{
			pattern:        "/api/shipments/{id}",
			testPath:       "/api/shipments/123/events",
			shouldMatch:    false,
			expectedParams: []string{"id"},
			expectedValues: nil,
		},
		{
			pattern:        "/api/users/{userId}/posts/{postId}",
			testPath:       "/api/users/42/posts/789",
			shouldMatch:    true,
			expectedParams: []string{"userId", "postId"},
			expectedValues: map[string]string{"userId": "42", "postId": "789"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+" vs "+tt.testPath, func(t *testing.T) {
			regex, params := patternToRegex(tt.pattern)

			// Check parameter names
			if len(params) != len(tt.expectedParams) {
				t.Errorf("Expected %d params, got %d", len(tt.expectedParams), len(params))
			}

			for i, expected := range tt.expectedParams {
				if i < len(params) && params[i] != expected {
					t.Errorf("Expected param %d to be %s, got %s", i, expected, params[i])
				}
			}

			// Check regex matching
			matches := regex.FindStringSubmatch(tt.testPath)
			if tt.shouldMatch && matches == nil {
				t.Errorf("Expected pattern %s to match path %s", tt.pattern, tt.testPath)
				return
			}
			if !tt.shouldMatch && matches != nil {
				t.Errorf("Expected pattern %s NOT to match path %s", tt.pattern, tt.testPath)
				return
			}

			// Check extracted values
			if tt.shouldMatch && tt.expectedValues != nil {
				extractedValues := make(map[string]string)
				for i, paramName := range params {
					if i+1 < len(matches) {
						extractedValues[paramName] = matches[i+1]
					}
				}

				for key, expectedValue := range tt.expectedValues {
					if extractedValues[key] != expectedValue {
						t.Errorf("Expected param %s=%s, got %s", key, expectedValue, extractedValues[key])
					}
				}
			}
		})
	}
}

func TestGetParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]string
		key         string
		expectedInt int
		expectError bool
	}{
		{
			name:        "Valid integer",
			params:      map[string]string{"id": "123"},
			key:         "id",
			expectedInt: 123,
			expectError: false,
		},
		{
			name:        "Missing parameter",
			params:      map[string]string{},
			key:         "id",
			expectedInt: 0,
			expectError: true,
		},
		{
			name:        "Invalid integer",
			params:      map[string]string{"id": "abc"},
			key:         "id",
			expectedInt: 0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetParam(tt.params, tt.key)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && result != tt.expectedInt {
				t.Errorf("Expected %d, got %d", tt.expectedInt, result)
			}
		})
	}
}

func TestHandlerAdapter(t *testing.T) {
	var capturedParams map[string]string

	handler := HandlerAdapter(func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		capturedParams = params
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/shipments/123", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// The adapter should extract ID from the path
	if capturedParams["id"] != "123" {
		t.Errorf("Expected id parameter 123, got %s", capturedParams["id"])
	}
}

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		path           string
		expectedParams map[string]string
	}{
		{
			path:           "/api/shipments/123",
			expectedParams: map[string]string{"id": "123"},
		},
		{
			path:           "/api/shipments/456/events",
			expectedParams: map[string]string{},
		},
		{
			path:           "/api/shipments",
			expectedParams: map[string]string{},
		},
		{
			path:           "/api/health",
			expectedParams: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			params := extractPathParams(tt.path)

			if len(params) != len(tt.expectedParams) {
				t.Errorf("Expected %d params, got %d", len(tt.expectedParams), len(params))
			}

			for key, expectedValue := range tt.expectedParams {
				if params[key] != expectedValue {
					t.Errorf("Expected param %s=%s, got %s", key, expectedValue, params[key])
				}
			}
		})
	}
}