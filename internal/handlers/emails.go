package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"package-tracking/internal/database"
)

// EmailHandler handles email-related HTTP requests
type EmailHandler struct {
	db *database.DB
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(db *database.DB) *EmailHandler {
	return &EmailHandler{db: db}
}

// GetShipmentEmails retrieves all emails linked to a specific shipment
func (h *EmailHandler) GetShipmentEmails(w http.ResponseWriter, r *http.Request) {
	// Extract shipment ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	shipmentIDStr := pathParts[3] // /api/shipments/{id}/emails
	shipmentID, err := strconv.Atoi(shipmentIDStr)
	if err != nil {
		http.Error(w, "Invalid shipment ID", http.StatusBadRequest)
		return
	}

	// Get emails linked to the shipment
	emails, err := h.db.Emails.GetByShipmentID(shipmentID)
	if err != nil {
		// Log error but return empty array instead of error
		// This is more user-friendly for non-existent shipments
		emails = []database.EmailBodyEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(emails)
}

// GetEmailThread retrieves all emails in a conversation thread
func (h *EmailHandler) GetEmailThread(w http.ResponseWriter, r *http.Request) {
	// Extract thread ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	threadID := pathParts[3] // /api/emails/{thread_id}/thread

	// Get thread information
	thread, err := h.db.Emails.GetThreadByGmailThreadID(threadID)
	if err != nil {
		http.Error(w, "Thread not found", http.StatusNotFound)
		return
	}

	// Get all emails in the thread
	emails, err := h.db.Emails.GetEmailsByThreadID(threadID)
	if err != nil {
		http.Error(w, "Failed to get thread emails", http.StatusInternalServerError)
		return
	}

	response := struct {
		Thread database.EmailThread           `json:"thread"`
		Emails []database.EmailBodyEntry `json:"emails"`
	}{
		Thread: *thread,
		Emails: emails,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetEmailBody retrieves the full body content of a specific email
func (h *EmailHandler) GetEmailBody(w http.ResponseWriter, r *http.Request) {
	// Extract email ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	emailID := pathParts[3] // /api/emails/{email_id}/body

	// Get email by Gmail message ID
	email, err := h.db.Emails.GetByGmailMessageID(emailID)
	if err != nil {
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}

	// Decompress body if it's compressed
	bodyText := email.BodyText
	bodyHTML := email.BodyHTML

	if len(email.BodyCompressed) > 0 && bodyText == "" {
		decompressed, err := database.DecompressEmailBody(email.BodyCompressed)
		if err != nil {
			http.Error(w, "Failed to decompress email body", http.StatusInternalServerError)
			return
		}
		bodyText = decompressed
	}

	response := struct {
		PlainText string `json:"plain_text"`
		HTMLText  string `json:"html_text"`
		Subject   string `json:"subject"`
		From      string `json:"from"`
		Date      string `json:"date"`
	}{
		PlainText: bodyText,
		HTMLText:  bodyHTML,
		Subject:   email.Subject,
		From:      email.From,
		Date:      email.Date.Format("2006-01-02T15:04:05Z07:00"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// LinkEmailToShipment creates a link between an email and a shipment
func (h *EmailHandler) LinkEmailToShipment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract email ID and shipment ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 6 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	emailIDStr := pathParts[3]   // /api/emails/{email_id}/link/{shipment_id}
	shipmentIDStr := pathParts[5]

	emailID, err := strconv.Atoi(emailIDStr)
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	shipmentID, err := strconv.Atoi(shipmentIDStr)
	if err != nil {
		http.Error(w, "Invalid shipment ID", http.StatusBadRequest)
		return
	}

	// Parse request body for link details
	var linkData struct {
		LinkType       string `json:"link_type"`
		TrackingNumber string `json:"tracking_number"`
		CreatedBy      string `json:"created_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&linkData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if linkData.LinkType == "" {
		linkData.LinkType = "manual"
	}
	if linkData.CreatedBy == "" {
		linkData.CreatedBy = "user"
	}

	// Create the link
	err = h.db.Emails.LinkEmailToShipment(emailID, shipmentID, linkData.LinkType, linkData.TrackingNumber, linkData.CreatedBy)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create link: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Email linked to shipment successfully",
	})
}

// UnlinkEmailFromShipment removes the link between an email and a shipment
func (h *EmailHandler) UnlinkEmailFromShipment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract email ID and shipment ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 6 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	emailIDStr := pathParts[3]   // /api/emails/{email_id}/link/{shipment_id}
	shipmentIDStr := pathParts[5]

	emailID, err := strconv.Atoi(emailIDStr)
	if err != nil {
		http.Error(w, "Invalid email ID", http.StatusBadRequest)
		return
	}

	shipmentID, err := strconv.Atoi(shipmentIDStr)
	if err != nil {
		http.Error(w, "Invalid shipment ID", http.StatusBadRequest)
		return
	}

	// Remove the link
	err = h.db.Emails.UnlinkEmailFromShipment(emailID, shipmentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove link: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RegisterEmailRoutes registers all email-related routes with the given mux
func RegisterEmailRoutes(mux *http.ServeMux, handler *EmailHandler) {
	// Shipment email endpoints
	mux.HandleFunc("/api/shipments/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/emails") && r.Method == http.MethodGet {
			handler.GetShipmentEmails(w, r)
			return
		}
		// Let other handlers handle non-email shipment endpoints
		http.NotFound(w, r)
	})

	// Email-specific endpoints
	mux.HandleFunc("/api/emails/", func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) < 4 {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}

		if len(pathParts) >= 5 && pathParts[4] == "thread" && r.Method == http.MethodGet {
			handler.GetEmailThread(w, r)
		} else if len(pathParts) >= 5 && pathParts[4] == "body" && r.Method == http.MethodGet {
			handler.GetEmailBody(w, r)
		} else if len(pathParts) >= 6 && pathParts[4] == "link" {
			if r.Method == http.MethodPost {
				handler.LinkEmailToShipment(w, r)
			} else if r.Method == http.MethodDelete {
				handler.UnlinkEmailFromShipment(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else {
			http.NotFound(w, r)
		}
	})
}