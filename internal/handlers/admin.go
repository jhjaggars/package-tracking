package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"package-tracking/internal/workers"
)

// AdminHandler handles administrative operations
type AdminHandler struct {
	trackingUpdater *workers.TrackingUpdater
	logger          *slog.Logger
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(trackingUpdater *workers.TrackingUpdater, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		trackingUpdater: trackingUpdater,
		logger:          logger,
	}
}

// TrackingUpdaterStatusResponse represents the status of the tracking updater
type TrackingUpdaterStatusResponse struct {
	Running bool `json:"running"`
	Paused  bool `json:"paused"`
}

// GetTrackingUpdaterStatus handles GET /api/admin/tracking-updater/status
func (h *AdminHandler) GetTrackingUpdaterStatus(w http.ResponseWriter, r *http.Request) {
	status := TrackingUpdaterStatusResponse{
		Running: h.trackingUpdater.IsRunning(),
		Paused:  h.trackingUpdater.IsPaused(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// PauseTrackingUpdater handles POST /api/admin/tracking-updater/pause
func (h *AdminHandler) PauseTrackingUpdater(w http.ResponseWriter, r *http.Request) {
	h.trackingUpdater.Pause()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "paused",
		"message": "Tracking updater has been paused",
	})
}

// ResumeTrackingUpdater handles POST /api/admin/tracking-updater/resume
func (h *AdminHandler) ResumeTrackingUpdater(w http.ResponseWriter, r *http.Request) {
	h.trackingUpdater.Resume()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "resumed",
		"message": "Tracking updater has been resumed",
	})
}

