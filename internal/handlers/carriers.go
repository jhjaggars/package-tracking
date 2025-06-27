package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"package-tracking/internal/database"
)

// CarrierHandler handles HTTP requests for carriers
type CarrierHandler struct {
	db *database.DB
}

// NewCarrierHandler creates a new carrier handler
func NewCarrierHandler(db *database.DB) *CarrierHandler {
	return &CarrierHandler{db: db}
}

// GetCarriers handles GET /api/carriers
func (h *CarrierHandler) GetCarriers(w http.ResponseWriter, r *http.Request) {
	// Check if we should filter for active carriers only
	activeOnly := r.URL.Query().Get("active") == "true"

	carriers, err := h.db.Carriers.GetAll(activeOnly)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get carriers: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(carriers)
}