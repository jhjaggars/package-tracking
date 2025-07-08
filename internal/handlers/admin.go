package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"package-tracking/internal/services"
	"package-tracking/internal/workers"
)

// AdminHandler handles administrative operations
type AdminHandler struct {
	trackingUpdater     *workers.TrackingUpdater
	descriptionEnhancer *services.DescriptionEnhancer
	logger              *slog.Logger
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(trackingUpdater *workers.TrackingUpdater, descriptionEnhancer *services.DescriptionEnhancer, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		trackingUpdater:     trackingUpdater,
		descriptionEnhancer: descriptionEnhancer,
		logger:              logger,
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

// EnhanceDescriptionsRequest represents the request body for description enhancement
type EnhanceDescriptionsRequest struct {
	ShipmentID *int `json:"shipment_id,omitempty"`
	Limit      int  `json:"limit,omitempty"`
	DryRun     bool `json:"dry_run,omitempty"`
	Associate  bool `json:"associate,omitempty"`
}

// EnhanceDescriptionsResponse represents the response from description enhancement
type EnhanceDescriptionsResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Summary interface{} `json:"summary,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// EnhanceDescriptions handles POST /api/admin/enhance-descriptions
func (h *AdminHandler) EnhanceDescriptions(w http.ResponseWriter, r *http.Request) {
	if h.descriptionEnhancer == nil {
		h.logger.Error("Description enhancer not configured")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(EnhanceDescriptionsResponse{
			Success: false,
			Error:   "Description enhancement service not available",
		})
		return
	}

	var req EnhanceDescriptionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body for description enhancement", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(EnhanceDescriptionsResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	h.logger.Info("Starting description enhancement via API",
		"shipment_id", req.ShipmentID,
		"limit", req.Limit,
		"dry_run", req.DryRun,
		"associate", req.Associate)

	// Handle email-shipment association if requested
	if req.Associate {
		if err := h.descriptionEnhancer.AssociateEmailsWithShipments(); err != nil {
			h.logger.Error("Failed to associate emails with shipments", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(EnhanceDescriptionsResponse{
				Success: false,
				Error:   "Failed to associate emails with shipments: " + err.Error(),
			})
			return
		}
	}

	var response EnhanceDescriptionsResponse

	// Process based on request type
	if req.ShipmentID != nil {
		// Process specific shipment
		result, err := h.descriptionEnhancer.EnhanceSpecificShipment(*req.ShipmentID, req.DryRun)
		if err != nil {
			h.logger.Error("Failed to enhance specific shipment",
				"shipment_id", *req.ShipmentID,
				"error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(EnhanceDescriptionsResponse{
				Success: false,
				Error:   "Failed to enhance shipment: " + err.Error(),
			})
			return
		}

		response = EnhanceDescriptionsResponse{
			Success: result.Success,
			Summary: result,
		}

		if result.Success {
			if req.DryRun {
				response.Message = "Dry run completed successfully"
			} else {
				response.Message = "Shipment description enhanced successfully"
			}
		} else {
			response.Message = "Enhancement failed"
			response.Error = result.Error
		}
	} else {
		// Process all shipments with poor descriptions
		summary, err := h.descriptionEnhancer.EnhanceAllShipmentsWithPoorDescriptions(req.Limit, req.DryRun)
		if err != nil {
			h.logger.Error("Failed to enhance shipment descriptions",
				"limit", req.Limit,
				"error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(EnhanceDescriptionsResponse{
				Success: false,
				Error:   "Failed to enhance descriptions: " + err.Error(),
			})
			return
		}

		response = EnhanceDescriptionsResponse{
			Success: summary.SuccessCount > 0 || summary.FailureCount == 0,
			Summary: summary,
		}

		if req.DryRun {
			response.Message = "Dry run completed successfully"
		} else {
			response.Message = "Description enhancement completed"
		}
	}

	h.logger.Info("Description enhancement completed via API",
		"success", response.Success,
		"message", response.Message)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}