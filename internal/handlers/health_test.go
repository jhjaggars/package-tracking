package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	t.Run("HealthyDatabase", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(db)

		handler := NewHealthHandler(db)

		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()

		handler.HealthCheck(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response struct {
			Status   string `json:"status"`
			Database string `json:"database"`
			Message  string `json:"message,omitempty"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response.Status)
		}

		if response.Database != "ok" {
			t.Errorf("Expected database 'ok', got '%s'", response.Database)
		}
	})

	t.Run("UnhealthyDatabase", func(t *testing.T) {
		db := setupTestDB(t)
		db.Close() // Close database to simulate unhealthy state

		handler := NewHealthHandler(db)

		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()

		handler.HealthCheck(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", w.Code)
		}

		var response struct {
			Status   string `json:"status"`
			Database string `json:"database"`
			Message  string `json:"message,omitempty"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Status != "unhealthy" {
			t.Errorf("Expected status 'unhealthy', got '%s'", response.Status)
		}

		if response.Database != "error" {
			t.Errorf("Expected database 'error', got '%s'", response.Database)
		}
	})
}