package email

import (
	"time"
)

// EmailClient defines the interface for email providers
type EmailClient interface {
	// Search performs a Gmail search query and returns matching messages
	Search(query string) ([]EmailMessage, error)
	
	// GetMessage retrieves the full content of a specific message
	GetMessage(id string) (*EmailMessage, error)
	
	// HealthCheck verifies the client connection is working
	HealthCheck() error
	
	// Close cleans up resources
	Close() error
}

// EmailMessage represents an email message with parsed content
type EmailMessage struct {
	ID       string            `json:"id"`
	ThreadID string            `json:"thread_id"`
	From     string            `json:"from"`
	Subject  string            `json:"subject"`
	Date     time.Time         `json:"date"`
	Headers  map[string]string `json:"headers"`
	
	// Content in different formats
	PlainText string `json:"plain_text"`
	HTMLText  string `json:"html_text"`
	
	// Gmail-specific fields
	Labels []string `json:"labels,omitempty"`
}

// EmailContent represents preprocessed email content for parsing
type EmailContent struct {
	PlainText string
	HTMLText  string
	Subject   string
	From      string
	Headers   map[string]string
	
	// Metadata for processing
	MessageID string
	ThreadID  string
	Date      time.Time
}

// TrackingInfo represents extracted tracking information
type TrackingInfo struct {
	Number      string    `json:"number"`
	Carrier     string    `json:"carrier"`
	Description string    `json:"description"`
	Merchant    string    `json:"merchant"`     // Store/retailer name for internal processing
	Confidence  float64   `json:"confidence"`
	Source      string    `json:"source"`       // "regex", "llm", "hybrid"
	Context     string    `json:"context"`      // Where it was found in email
	ExtractedAt time.Time `json:"extracted_at"`
	
	// Source email information
	SourceEmail EmailMessage `json:"source_email"`
}

// CarrierHint provides confidence scoring for carrier identification
type CarrierHint struct {
	Carrier    string  `json:"carrier"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"` // "sender", "subject", "content"
	Reason     string  `json:"reason"` // Why this carrier was identified
}

// TrackingCandidate represents a potential tracking number found in email
type TrackingCandidate struct {
	Text       string  `json:"text"`
	Position   int     `json:"position"`   // Character position in email
	Context    string  `json:"context"`    // Surrounding text
	Carrier    string  `json:"carrier"`    // Suggested carrier
	Confidence float64 `json:"confidence"`
	Method     string  `json:"method"`     // "direct", "labeled", "table"
}

// ProcessingResult represents the outcome of processing an email
type ProcessingResult struct {
	EmailID         string         `json:"email_id"`
	ProcessedAt     time.Time      `json:"processed_at"`
	TrackingNumbers []TrackingInfo `json:"tracking_numbers"`
	Success         bool           `json:"success"`
	Error           string         `json:"error,omitempty"`
	ProcessingTime  time.Duration  `json:"processing_time"`
}

// SearchQuery represents a Gmail search configuration
type SearchQuery struct {
	Query          string        `json:"query"`
	MaxResults     int           `json:"max_results"`
	AfterDate      *time.Time    `json:"after_date,omitempty"`
	BeforeDate     *time.Time    `json:"before_date,omitempty"`
	UnreadOnly     bool          `json:"unread_only"`
	IncludeLabels  []string      `json:"include_labels,omitempty"`
	ExcludeLabels  []string      `json:"exclude_labels,omitempty"`
}

// EmailMetrics tracks processing statistics
type EmailMetrics struct {
	TotalEmails        int           `json:"total_emails"`
	ProcessedEmails    int           `json:"processed_emails"`
	SkippedEmails      int           `json:"skipped_emails"`
	ErrorEmails        int           `json:"error_emails"`
	TrackingnumbersFound int         `json:"tracking_numbers_found"`
	ShipmentsCreated   int           `json:"shipments_created"`
	ProcessingDuration time.Duration `json:"processing_duration"`
	LastProcessed      time.Time     `json:"last_processed"`
}

// StateEntry represents a processed email record
type StateEntry struct {
	ID              int       `json:"id"`
	GmailMessageID  string    `json:"gmail_message_id"`
	GmailThreadID   string    `json:"gmail_thread_id"`
	ProcessedAt     time.Time `json:"processed_at"`
	TrackingNumbers string    `json:"tracking_numbers"` // JSON encoded
	Status          string    `json:"status"`           // "processed", "failed", "skipped"
	Sender          string    `json:"sender"`
	Subject         string    `json:"subject"`
	ErrorMessage    string    `json:"error_message,omitempty"`
}