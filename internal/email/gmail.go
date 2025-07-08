package email

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GmailClient implements EmailClient for Gmail API
type GmailClient struct {
	service *gmail.Service
	userID  string
	config  *GmailConfig
	ctx     context.Context
}

// GmailConfig holds Gmail API configuration
type GmailConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	AccessToken  string
	TokenFile    string
	UserEmail    string
	
	// Request limits
	MaxResults      int64
	RequestTimeout  time.Duration
	RateLimitDelay  time.Duration
}

// NewGmailClient creates a new Gmail API client
func NewGmailClient(config *GmailConfig) (*GmailClient, error) {
	ctx := context.Background()
	
	// Configure OAuth2
	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Scopes:       []string{gmail.GmailReadonlyScope},
		Endpoint:     google.Endpoint,
	}
	
	// Create token from configuration
	token := &oauth2.Token{
		AccessToken:  config.AccessToken,
		RefreshToken: config.RefreshToken,
		TokenType:    "Bearer",
	}
	
	// Create HTTP client with token
	httpClient := oauthConfig.Client(ctx, token)
	
	// Create Gmail service
	service, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}
	
	userID := "me"
	if config.UserEmail != "" {
		userID = config.UserEmail
	}
	
	client := &GmailClient{
		service: service,
		userID:  userID,
		config:  config,
		ctx:     ctx,
	}
	
	// Verify connection
	if err := client.HealthCheck(); err != nil {
		return nil, fmt.Errorf("Gmail client health check failed: %w", err)
	}
	
	return client, nil
}

// Search performs a Gmail search query
func (g *GmailClient) Search(query string) ([]EmailMessage, error) {
	log.Printf("Searching Gmail with query: %s", query)
	
	// Apply rate limiting
	time.Sleep(g.config.RateLimitDelay)
	
	// Execute search
	req := g.service.Users.Messages.List(g.userID).Q(query)
	if g.config.MaxResults > 0 {
		req = req.MaxResults(g.config.MaxResults)
	}
	
	resp, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf("Gmail search failed: %w", err)
	}
	
	log.Printf("Found %d messages", len(resp.Messages))
	
	var messages []EmailMessage
	for _, msg := range resp.Messages {
		// Rate limiting between requests
		time.Sleep(g.config.RateLimitDelay)
		
		fullMessage, err := g.GetMessage(msg.Id)
		if err != nil {
			log.Printf("Failed to get message %s: %v", msg.Id, err)
			continue
		}
		
		messages = append(messages, *fullMessage)
	}
	
	return messages, nil
}

// GetMessage retrieves the full content of a specific message
func (g *GmailClient) GetMessage(id string) (*EmailMessage, error) {
	msg, err := g.service.Users.Messages.Get(g.userID, id).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message %s: %w", id, err)
	}
	
	// Parse the message
	emailMsg, err := g.parseGmailMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message %s: %w", id, err)
	}
	
	return emailMsg, nil
}

// GetMessageMetadata retrieves only the metadata of a specific message (headers, snippet, no content)
func (g *GmailClient) GetMessageMetadata(id string) (*EmailMessage, error) {
	msg, err := g.service.Users.Messages.Get(g.userID, id).Format("metadata").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message metadata %s: %w", id, err)
	}
	
	// Parse the message metadata only
	emailMsg, err := g.parseGmailMessageMetadata(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message metadata %s: %w", id, err)
	}
	
	return emailMsg, nil
}

// GetMessagesSinceMetadataOnly retrieves messages since a specific time with metadata only
func (g *GmailClient) GetMessagesSinceMetadataOnly(since time.Time) ([]EmailMessage, error) {
	// Build search query for time-based search
	query := fmt.Sprintf("after:%d", since.Unix())
	log.Printf("Searching Gmail with metadata-only query: %s", query)
	
	// Execute search
	req := g.service.Users.Messages.List(g.userID).Q(query)
	if g.config.MaxResults > 0 {
		req = req.MaxResults(g.config.MaxResults)
	}
	
	resp, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf("Gmail metadata search failed: %w", err)
	}
	
	log.Printf("Found %d messages for metadata extraction", len(resp.Messages))
	
	var messages []EmailMessage
	for _, msg := range resp.Messages {
		// Rate limiting between requests
		time.Sleep(g.config.RateLimitDelay)
		
		metadataMessage, err := g.GetMessageMetadata(msg.Id)
		if err != nil {
			log.Printf("Failed to get message metadata %s: %v", msg.Id, err)
			continue
		}
		
		messages = append(messages, *metadataMessage)
	}
	
	return messages, nil
}

// parseGmailMessageMetadata converts Gmail API message metadata to EmailMessage (no content)
func (g *GmailClient) parseGmailMessageMetadata(msg *gmail.Message) (*EmailMessage, error) {
	emailMsg := &EmailMessage{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Headers:  make(map[string]string),
		Labels:   msg.LabelIds,
		// Note: PlainText and HTMLText will be empty for metadata-only
		PlainText: "",
		HTMLText:  "",
		Snippet:   msg.Snippet, // Gmail provides a snippet in metadata
	}
	
	// Parse headers
	if msg.Payload != nil && msg.Payload.Headers != nil {
		for _, header := range msg.Payload.Headers {
			emailMsg.Headers[header.Name] = header.Value
			
			switch header.Name {
			case "Subject":
				emailMsg.Subject = header.Value
			case "From":
				emailMsg.From = header.Value
			case "To":
				emailMsg.To = header.Value
			case "Date":
				if date, err := parseRFC2822Date(header.Value); err == nil {
					emailMsg.Date = date
				}
			}
		}
	}
	
	// Set internal timestamp
	if msg.InternalDate > 0 {
		emailMsg.InternalDate = time.Unix(0, msg.InternalDate*int64(time.Millisecond))
	}
	
	return emailMsg, nil
}

// parseGmailMessage converts Gmail API message to EmailMessage
func (g *GmailClient) parseGmailMessage(msg *gmail.Message) (*EmailMessage, error) {
	emailMsg := &EmailMessage{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Headers:  make(map[string]string),
		Labels:   msg.LabelIds,
	}
	
	// Parse headers
	for _, header := range msg.Payload.Headers {
		emailMsg.Headers[header.Name] = header.Value
		
		switch strings.ToLower(header.Name) {
		case "from":
			emailMsg.From = header.Value
		case "subject":
			emailMsg.Subject = header.Value
		case "date":
			if date, err := mail.ParseDate(header.Value); err == nil {
				emailMsg.Date = date
			}
		}
	}
	
	// Extract body content
	plainText, htmlText := g.extractContent(msg.Payload)
	emailMsg.PlainText = plainText
	emailMsg.HTMLText = htmlText
	
	return emailMsg, nil
}

// extractContent extracts plain text and HTML content from message payload
func (g *GmailClient) extractContent(payload *gmail.MessagePart) (plainText, htmlText string) {
	if payload.MimeType == "text/plain" && payload.Body.Data != "" {
		if decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data); err == nil {
			plainText = string(decoded)
		}
	} else if payload.MimeType == "text/html" && payload.Body.Data != "" {
		if decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data); err == nil {
			htmlText = string(decoded)
		}
	}
	
	// Recursively process parts for multipart messages
	for _, part := range payload.Parts {
		partPlain, partHTML := g.extractContent(part)
		if partPlain != "" && plainText == "" {
			plainText = partPlain
		}
		if partHTML != "" && htmlText == "" {
			htmlText = partHTML
		}
	}
	
	// Convert HTML to plain text if no plain text version
	if plainText == "" && htmlText != "" {
		plainText = g.htmlToText(htmlText)
	}
	
	return plainText, htmlText
}

// htmlToText converts HTML content to plain text (basic implementation)
func (g *GmailClient) htmlToText(html string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, " ")
	
	// Decode HTML entities (basic ones)
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	
	// Normalize whitespace
	re = regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")
	
	return strings.TrimSpace(text)
}

// parseRFC2822Date parses an RFC2822 date string commonly found in email headers
func parseRFC2822Date(dateStr string) (time.Time, error) {
	// Common RFC2822 date formats found in email headers
	formats := []string{
		time.RFC1123Z,     // "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC1123,      // "Mon, 02 Jan 2006 15:04:05 MST"
		time.RFC822Z,      // "02 Jan 06 15:04 -0700"
		time.RFC822,       // "02 Jan 06 15:04 MST"
		"Mon, 2 Jan 2006 15:04:05 -0700",   // Single digit day
		"Mon, 02 Jan 2006 15:04:05 -0700 (MST)", // With timezone name in parentheses
		"2 Jan 2006 15:04:05 -0700",        // Without day of week
		"02 Jan 2006 15:04:05 -0700",       // Without day of week, zero-padded
	}
	
	for _, format := range formats {
		if parsed, err := time.Parse(format, dateStr); err == nil {
			return parsed, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// HealthCheck verifies the Gmail connection is working
func (g *GmailClient) HealthCheck() error {
	profile, err := g.service.Users.GetProfile(g.userID).Do()
	if err != nil {
		return fmt.Errorf("failed to get Gmail profile: %w", err)
	}
	
	log.Printf("Connected to Gmail account: %s", profile.EmailAddress)
	return nil
}

// Close cleans up resources
func (g *GmailClient) Close() error {
	// Gmail API client doesn't require explicit cleanup
	return nil
}

// BuildSearchQuery constructs a Gmail search query from components
func BuildSearchQuery(carriers []string, afterDays int, unreadOnly bool, customQuery string) string {
	if customQuery != "" {
		return customQuery
	}
	
	var parts []string
	
	// Add carrier-specific sender filters
	if len(carriers) > 0 {
		var senders []string
		for _, carrier := range carriers {
			switch carrier {
			case "ups":
				senders = append(senders, "noreply@ups.com", "quantum@ups.com", "pkginfo@ups.com")
			case "usps":
				senders = append(senders, "usps.com", "informeddelivery@email.usps.com")
			case "fedex":
				senders = append(senders, "fedex.com", "tracking@fedex.com", "shipment@fedex.com")
			case "dhl":
				senders = append(senders, "dhl.com", "noreply@dhl.com")
			}
		}
		
		if len(senders) > 0 {
			parts = append(parts, fmt.Sprintf("from:(%s)", strings.Join(senders, " OR ")))
		}
	} else {
		// Default: search common shipping senders
		parts = append(parts, "from:(ups.com OR usps.com OR fedex.com OR dhl.com OR amazon.com OR shopify.com)")
	}
	
	// Add subject filters for shipping-related terms
	parts = append(parts, "subject:(tracking OR shipment OR package OR delivery OR shipped OR \"tracking number\")")
	
	// Add date filter
	if afterDays > 0 {
		afterDate := time.Now().AddDate(0, 0, -afterDays).Format("2006/1/2")
		parts = append(parts, fmt.Sprintf("after:%s", afterDate))
	}
	
	// Add unread filter
	if unreadOnly {
		parts = append(parts, "is:unread")
	}
	
	return strings.Join(parts, " ")
}

// SearchWithDefaults performs a search with common shipping email patterns
func (g *GmailClient) SearchWithDefaults(afterDays int, unreadOnly bool) ([]EmailMessage, error) {
	query := BuildSearchQuery(nil, afterDays, unreadOnly, "")
	return g.Search(query)
}

// BuildSearchQueryForLLMProcessing constructs a broader Gmail search query for LLM processing
// This function creates searches without sender/subject filtering to capture more emails
// for LLM analysis, allowing the LLM to identify tracking information from a wider range of emails
func BuildSearchQueryForLLMProcessing(afterDays int, unreadOnly bool) string {
	var parts []string
	
	// Add date filter - this is the primary constraint for LLM processing
	if afterDays > 0 {
		afterDate := time.Now().AddDate(0, 0, -afterDays).Format("2006/1/2")
		parts = append(parts, fmt.Sprintf("after:%s", afterDate))
	}
	
	// Add unread filter to focus on new emails
	if unreadOnly {
		parts = append(parts, "is:unread")
	}
	
	// Return broader search query without sender/subject restrictions
	return strings.Join(parts, " ")
}

// SearchCarrierEmails searches for emails from specific carriers
func (g *GmailClient) SearchCarrierEmails(carriers []string, afterDays int) ([]EmailMessage, error) {
	query := BuildSearchQuery(carriers, afterDays, false, "")
	return g.Search(query)
}

// GetMessagesSince retrieves all messages since a specific timestamp using time-based scanning
func (g *GmailClient) GetMessagesSince(since time.Time) ([]EmailMessage, error) {
	log.Printf("Getting messages since: %v", since)

	// Build time-based query
	query := fmt.Sprintf("after:%s", since.Format("2006/1/2"))
	
	var allMessages []EmailMessage
	pageToken := ""
	
	for {
		// Apply rate limiting
		time.Sleep(g.config.RateLimitDelay)
		
		// Build request
		req := g.service.Users.Messages.List(g.userID).Q(query)
		if g.config.MaxResults > 0 {
			req = req.MaxResults(g.config.MaxResults)
		}
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		
		// Execute request
		resp, err := req.Do()
		if err != nil {
			return nil, fmt.Errorf("Gmail messages list failed: %w", err)
		}
		
		log.Printf("Found %d messages in page (total so far: %d)", len(resp.Messages), len(allMessages))
		
		// Process messages in this page
		for _, msg := range resp.Messages {
			// Rate limiting between requests
			time.Sleep(g.config.RateLimitDelay)
			
			fullMessage, err := g.GetEnhancedMessage(msg.Id)
			if err != nil {
				log.Printf("Failed to get enhanced message %s: %v", msg.Id, err)
				continue
			}
			
			allMessages = append(allMessages, *fullMessage)
		}
		
		// Check for more pages
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
		
		log.Printf("Fetching next page with token: %s", pageToken)
	}
	
	log.Printf("Total messages retrieved since %v: %d", since, len(allMessages))
	return allMessages, nil
}

// GetEnhancedMessage retrieves a message with full body content for storage
func (g *GmailClient) GetEnhancedMessage(id string) (*EmailMessage, error) {
	msg, err := g.service.Users.Messages.Get(g.userID, id).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get enhanced message %s: %w", id, err)
	}
	
	// Parse the message with enhanced body extraction
	emailMsg, err := g.parseEnhancedGmailMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse enhanced message %s: %w", id, err)
	}
	
	return emailMsg, nil
}

// GetThreadMessages retrieves all messages in a Gmail thread
func (g *GmailClient) GetThreadMessages(threadID string) ([]EmailMessage, error) {
	log.Printf("Getting thread messages for thread: %s", threadID)
	
	// Apply rate limiting
	time.Sleep(g.config.RateLimitDelay)
	
	// Get the thread
	thread, err := g.service.Users.Threads.Get(g.userID, threadID).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get thread %s: %w", threadID, err)
	}
	
	var messages []EmailMessage
	for _, msg := range thread.Messages {
		// Parse each message in the thread
		emailMsg, err := g.parseEnhancedGmailMessage(msg)
		if err != nil {
			log.Printf("Failed to parse message %s in thread %s: %v", msg.Id, threadID, err)
			continue
		}
		
		messages = append(messages, *emailMsg)
	}
	
	log.Printf("Retrieved %d messages from thread %s", len(messages), threadID)
	return messages, nil
}

// parseEnhancedGmailMessage converts Gmail API message to EmailMessage with full body content
func (g *GmailClient) parseEnhancedGmailMessage(msg *gmail.Message) (*EmailMessage, error) {
	emailMsg := &EmailMessage{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Headers:  make(map[string]string),
		Labels:   msg.LabelIds,
	}
	
	// Parse headers
	for _, header := range msg.Payload.Headers {
		emailMsg.Headers[header.Name] = header.Value
		
		switch strings.ToLower(header.Name) {
		case "from":
			emailMsg.From = header.Value
		case "subject":
			emailMsg.Subject = header.Value
		case "date":
			if date, err := mail.ParseDate(header.Value); err == nil {
				emailMsg.Date = date
			}
		}
	}
	
	// Extract body content with enhanced parsing for storage
	plainText, htmlText := g.extractEnhancedContent(msg.Payload)
	emailMsg.PlainText = plainText
	emailMsg.HTMLText = htmlText
	
	return emailMsg, nil
}

// extractEnhancedContent extracts both plain text and HTML content with better handling
func (g *GmailClient) extractEnhancedContent(payload *gmail.MessagePart) (plainText, htmlText string) {
	// Handle direct content
	if payload.Body != nil && payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			content := string(decoded)
			switch payload.MimeType {
			case "text/plain":
				plainText = content
			case "text/html":
				htmlText = content
			}
		}
	}
	
	// Handle multipart content recursively
	for _, part := range payload.Parts {
		partPlain, partHTML := g.extractEnhancedContent(part)
		
		// Prefer the first non-empty content found
		if partPlain != "" && plainText == "" {
			plainText = partPlain
		}
		if partHTML != "" && htmlText == "" {
			htmlText = partHTML
		}
	}
	
	// Convert HTML to plain text if no plain text version exists
	if plainText == "" && htmlText != "" {
		plainText = g.htmlToText(htmlText)
	}
	
	return plainText, htmlText
}

// GetMessagesSinceWithPagination retrieves messages with custom pagination parameters
func (g *GmailClient) GetMessagesSinceWithPagination(since time.Time, maxResults int64, pageToken string) (*EmailPage, error) {
	query := fmt.Sprintf("after:%s", since.Format("2006/1/2"))
	
	// Apply rate limiting
	time.Sleep(g.config.RateLimitDelay)
	
	// Build request
	req := g.service.Users.Messages.List(g.userID).Q(query)
	if maxResults > 0 {
		req = req.MaxResults(maxResults)
	}
	if pageToken != "" {
		req = req.PageToken(pageToken)
	}
	
	// Execute request
	resp, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf("Gmail paginated messages list failed: %w", err)
	}
	
	var messages []EmailMessage
	for _, msg := range resp.Messages {
		// Rate limiting between requests
		time.Sleep(g.config.RateLimitDelay)
		
		fullMessage, err := g.GetEnhancedMessage(msg.Id)
		if err != nil {
			log.Printf("Failed to get enhanced message %s: %v", msg.Id, err)
			continue
		}
		
		messages = append(messages, *fullMessage)
	}
	
	return &EmailPage{
		Messages:      messages,
		NextPageToken: resp.NextPageToken,
		TotalSize:     len(messages),
	}, nil
}

// PerformRetroactiveScan scans all emails within the specified number of days
func (g *GmailClient) PerformRetroactiveScan(days int) ([]EmailMessage, error) {
	since := time.Now().AddDate(0, 0, -days)
	log.Printf("Starting retroactive scan for the last %d days (since %v)", days, since)
	
	messages, err := g.GetMessagesSince(since)
	if err != nil {
		return nil, fmt.Errorf("retroactive scan failed: %w", err)
	}
	
	log.Printf("Retroactive scan completed: found %d messages", len(messages))
	return messages, nil
}

// CompressEmailBody compresses email body text for efficient storage
func CompressEmailBody(text string) ([]byte, error) {
	if text == "" {
		return nil, nil
	}
	
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	
	if _, err := gz.Write([]byte(text)); err != nil {
		return nil, fmt.Errorf("failed to write to gzip: %w", err)
	}
	
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip: %w", err)
	}
	
	return buf.Bytes(), nil
}

// DecompressEmailBody decompresses compressed email body text
func DecompressEmailBody(compressed []byte) (string, error) {
	if len(compressed) == 0 {
		return "", nil
	}
	
	buf := bytes.NewReader(compressed)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()
	
	decompressed, err := io.ReadAll(gz)
	if err != nil {
		return "", fmt.Errorf("failed to read from gzip: %w", err)
	}
	
	return string(decompressed), nil
}