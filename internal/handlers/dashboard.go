package handlers

import (
	"encoding/json"
	"net/http"

	"package-tracking/internal/database"
)

// DashboardHandler handles dashboard-related HTTP requests
type DashboardHandler struct {
	db *database.DB
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(db *database.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// GetStats returns aggregated dashboard statistics
func (h *DashboardHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	shipmentStore := database.NewShipmentStore(h.db.DB)
	
	stats, err := shipmentStore.GetStats()
	if err != nil {
		http.Error(w, "Failed to get dashboard statistics", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}