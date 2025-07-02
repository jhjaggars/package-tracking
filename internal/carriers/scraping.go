package carriers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ScrapingClient provides common functionality for web scraping-based tracking
type ScrapingClient struct {
	carrier    string
	userAgent  string
	client     *http.Client
	rateLimit  *RateLimitInfo
}

// NewScrapingClient creates a new base scraping client
func NewScrapingClient(carrier, userAgent string) *ScrapingClient {
	return &ScrapingClient{
		carrier:   carrier,
		userAgent: userAgent,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimit: &RateLimitInfo{
			Limit:     10, // Conservative rate limit for web scraping
			Remaining: 10,
			ResetTime: time.Now().Add(time.Minute),
		},
	}
}

// GetCarrierName returns the carrier name
func (c *ScrapingClient) GetCarrierName() string {
	return c.carrier
}

// GetRateLimit returns current rate limit information
func (c *ScrapingClient) GetRateLimit() *RateLimitInfo {
	return c.rateLimit
}

// fetchPage fetches a web page with proper headers and rate limiting
func (c *ScrapingClient) fetchPage(ctx context.Context, url string) (string, error) {
	// Check rate limit
	if c.rateLimit.Remaining <= 0 && time.Now().Before(c.rateLimit.ResetTime) {
		return "", &CarrierError{
			Carrier:   c.carrier,
			Code:      "RATE_LIMIT",
			Message:   "Rate limit exceeded for web scraping",
			Retryable: true,
			RateLimit: true,
		}
	}
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers to mimic a real browser
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	// Note: Removed Accept-Encoding to let Go HTTP client handle compression automatically
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	
	// Make request
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()
	
	// Update rate limit
	c.rateLimit.Remaining--
	if c.rateLimit.Remaining <= 0 {
		c.rateLimit.ResetTime = time.Now().Add(time.Minute)
	}
	
	// Check for errors
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return "", &CarrierError{
				Carrier:   c.carrier,
				Code:      "RATE_LIMIT",
				Message:   "Rate limited by carrier website",
				Retryable: true,
				RateLimit: true,
			}
		}
		return "", fmt.Errorf("HTTP error %d", resp.StatusCode)
	}
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	
	return string(body), nil
}

// extractText extracts text from HTML using regex patterns
func (c *ScrapingClient) extractText(html, pattern string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractAllText extracts all matches from HTML using regex patterns
func (c *ScrapingClient) extractAllText(html, pattern string) []string {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(html, -1)
	var results []string
	for _, match := range matches {
		if len(match) > 1 {
			results = append(results, strings.TrimSpace(match[1]))
		}
	}
	return results
}

// cleanHTML removes HTML tags and cleans up text
func (c *ScrapingClient) cleanHTML(text string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text = re.ReplaceAllString(text, "")
	
	// Clean up whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	
	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	
	return text
}

// parseDateTime attempts to parse various date/time formats commonly used by carriers
func (c *ScrapingClient) parseDateTime(dateStr string) (time.Time, error) {
	// Clean up the date string
	dateStr = strings.TrimSpace(dateStr)
	dateStr = regexp.MustCompile(`\s+`).ReplaceAllString(dateStr, " ")
	
	// Common date formats used by carrier websites
	layouts := []string{
		"January 2, 2006 at 3:04 PM",
		"January 2, 2006 3:04 PM",
		"Jan 2, 2006 3:04 PM",
		"01/02/2006 3:04 PM",
		"01/02/2006 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"Monday, January 2, 2006",
		"January 2, 2006",
		"01/02/2006",
		"2006-01-02",
	}
	
	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t, nil
		}
	}
	
	return time.Now(), fmt.Errorf("unable to parse date: %s", dateStr)
}

// mapScrapedStatus maps scraped status text to our standardized status
func (c *ScrapingClient) mapScrapedStatus(statusText string) TrackingStatus {
	status := strings.ToLower(statusText)
	
	switch {
	case strings.Contains(status, "delivered"):
		return StatusDelivered
	case strings.Contains(status, "out for delivery"), strings.Contains(status, "on vehicle"), 
		 strings.Contains(status, "on fedex vehicle"), strings.Contains(status, "vehicle for delivery"):
		return StatusOutForDelivery
	case strings.Contains(status, "in transit"), strings.Contains(status, "en route"), 
		 strings.Contains(status, "departed"), strings.Contains(status, "arrived"),
		 strings.Contains(status, "at local fedex facility"), strings.Contains(status, "at facility"):
		return StatusInTransit
	case strings.Contains(status, "picked up"), strings.Contains(status, "acceptance"), 
		 strings.Contains(status, "electronic"), strings.Contains(status, "pre-shipment"):
		return StatusPreShip
	case strings.Contains(status, "exception"), strings.Contains(status, "delay"), 
		 strings.Contains(status, "held"), strings.Contains(status, "customs"):
		return StatusException
	case strings.Contains(status, "returned"), strings.Contains(status, "return"):
		return StatusReturned
	default:
		return StatusUnknown
	}
}