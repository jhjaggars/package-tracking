package carriers

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// USPSHeadlessClient implements headless browser tracking for USPS
type USPSHeadlessClient struct {
	*HeadlessScrapingClient
	baseURL string
}

// NewUSPSHeadlessClient creates a new USPS headless client
func NewUSPSHeadlessClient() *USPSHeadlessClient {
	options := DefaultHeadlessOptions()
	options.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	options.WaitStrategy = WaitForTimeout // USPS loads content asynchronously
	options.Timeout = 60 * time.Second    // USPS can be slow
	options.StealthMode = true             // Enable stealth mode for bot detection avoidance
	options.SimulateHumanBehavior = true   // Add human-like delays

	headlessClient := NewHeadlessScrapingClient("usps", options.UserAgent, options)

	return &USPSHeadlessClient{
		HeadlessScrapingClient: headlessClient,
		baseURL:                "https://tools.usps.com",
	}
}

// ValidateTrackingNumber validates USPS tracking number formats
func (c *USPSHeadlessClient) ValidateTrackingNumber(trackingNumber string) bool {
	if trackingNumber == "" {
		return false
	}

	// Remove spaces and keep only digits
	cleaned := strings.ReplaceAll(trackingNumber, " ", "")

	// Check if it's all digits
	if matched, _ := regexp.MatchString(`^\d+$`, cleaned); !matched {
		return false
	}

	// USPS tracking number patterns and lengths
	validPatterns := []struct {
		length int
		prefix string
	}{
		{22, "94"}, // Priority Mail Express, Priority Mail
		{22, "95"}, // Priority Mail
		{22, "93"}, // Certified Mail, Collect on Delivery, Global Express Guaranteed
		{22, "92"}, // Certified Mail
		{22, "91"}, // Signature Confirmation
		{20, "94"}, // Some Priority Mail variants
		{20, "95"}, // Some Priority Mail variants
		{20, "93"}, // Some Certified Mail variants
		{13, ""},   // Some tracking numbers
		{11, "82"}, // Global Express Guaranteed
		{11, ""},   // Some tracking numbers
	}

	for _, pattern := range validPatterns {
		if len(cleaned) == pattern.length {
			if pattern.prefix == "" || strings.HasPrefix(cleaned, pattern.prefix) {
				return true
			}
		}
	}

	return false
}

// Track retrieves tracking information for the given tracking numbers
func (c *USPSHeadlessClient) Track(ctx context.Context, req *TrackingRequest) (*TrackingResponse, error) {
	if len(req.TrackingNumbers) == 0 {
		return nil, fmt.Errorf("no tracking numbers provided")
	}

	var results []TrackingInfo
	var errors []CarrierError

	// USPS tracking website handles one tracking number per request
	for _, trackingNumber := range req.TrackingNumbers {
		result, err := c.trackSingle(ctx, trackingNumber)
		if err != nil {
			if carrierErr, ok := err.(*CarrierError); ok {
				errors = append(errors, *carrierErr)
				// For rate limits, return immediately
				if carrierErr.RateLimit {
					return nil, err
				}
			} else {
				return nil, err
			}
		} else if result != nil {
			results = append(results, *result)
		}
	}

	return &TrackingResponse{
		Results:   results,
		Errors:    errors,
		RateLimit: c.rateLimit,
	}, nil
}

func (c *USPSHeadlessClient) trackSingle(ctx context.Context, trackingNumber string) (*TrackingInfo, error) {
	// Build tracking URL using modern USPS format
	trackURL := fmt.Sprintf("%s/go/TrackConfirmAction?qtc_tLabels1=%s", c.baseURL, trackingNumber)

	// Use a custom approach for USPS since their JavaScript SPA loads dynamically
	pageSource, err := c.loadUSPSPageWithRetry(ctx, trackURL)
	if err != nil {
		return nil, &CarrierError{
			Carrier:   "usps",
			Code:      "NAVIGATION_ERROR",
			Message:   fmt.Sprintf("Failed to load tracking page for %s: %v", trackingNumber, err),
			Retryable: true,
			RateLimit: false,
		}
	}

	// Check for "not found" or error messages
	if c.isTrackingNotFound(pageSource) {
		return nil, &CarrierError{
			Carrier:   "usps",
			Code:      "NOT_FOUND",
			Message:   "Tracking information not found for " + trackingNumber,
			Retryable: false,
			RateLimit: false,
		}
	}

	// Parse tracking information
	trackingInfo := c.parseUSPSTrackingInfo(pageSource, trackingNumber)

	// If no events were found, it might be an error
	if len(trackingInfo.Events) == 0 {
		return nil, &CarrierError{
			Carrier:   "usps",
			Code:      "NO_EVENTS",
			Message:   "No tracking events found for " + trackingNumber,
			Retryable: true,
			RateLimit: false,
		}
	}

	return &trackingInfo, nil
}

// loadUSPSPageWithRetry loads the USPS page and waits for content with multiple strategies
func (c *USPSHeadlessClient) loadUSPSPageWithRetry(ctx context.Context, url string) (string, error) {
	var pageSource string

	err := c.browserPool.ExecuteWithBrowser(ctx, func(browserCtx context.Context) error {
		// Navigate to the URL
		err := chromedp.Run(browserCtx, chromedp.Navigate(url))
		if err != nil {
			return fmt.Errorf("failed to navigate to %s: %w", url, err)
		}

		// Wait for the page to initially load
		err = chromedp.Run(browserCtx, chromedp.WaitReady("body"))
		if err != nil {
			return fmt.Errorf("failed to wait for body: %w", err)
		}

		// USPS loads content dynamically, so wait a bit for JavaScript to execute
		err = chromedp.Run(browserCtx, chromedp.Sleep(10*time.Second))
		if err != nil {
			return fmt.Errorf("failed to wait: %w", err)
		}

		// Try to execute JavaScript to trigger any remaining loading
		var jsResult string
		script := `
			// Wait for any pending AJAX requests to complete
			var checkInterval = setInterval(function() {
				if (document.readyState === 'complete') {
					clearInterval(checkInterval);
				}
			}, 100);
			
			// Return page title to verify JavaScript execution
			document.title || 'loaded';
		`
		err = chromedp.Run(browserCtx, chromedp.Evaluate(script, &jsResult))
		if err != nil {
			// JavaScript execution failed, but continue anyway
		}

		// Wait a bit more for dynamic content
		err = chromedp.Run(browserCtx, chromedp.Sleep(5*time.Second))
		if err != nil {
			return fmt.Errorf("failed to wait: %w", err)
		}

		// Get the final page source
		return chromedp.Run(browserCtx, chromedp.OuterHTML("html", &pageSource))
	})

	if err != nil {
		return "", c.wrapError(err, "failed to load USPS page")
	}

	return pageSource, nil
}

func (c *USPSHeadlessClient) isTrackingNotFound(html string) bool {
	// Check for various "not found" patterns in USPS HTML
	notFoundPatterns := []string{
		"Status Not Available",
		"could not locate",
		"tracking information for your request",
		"verify your tracking number",
		"No record of that item",
		"not found",
		"invalid tracking number",
		"We could not locate the tracking information",
		"Check that you entered the number correctly",
	}

	lowerHTML := strings.ToLower(html)
	for _, pattern := range notFoundPatterns {
		if strings.Contains(lowerHTML, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

func (c *USPSHeadlessClient) parseUSPSTrackingInfo(html, trackingNumber string) TrackingInfo {
	info := TrackingInfo{
		TrackingNumber: trackingNumber,
		Carrier:        "usps",
		Events:         []TrackingEvent{},
		LastUpdated:    time.Now(),
		Status:         StatusUnknown,
	}

	// Extract events from tracking history
	events := c.extractTrackingEvents(html)
	info.Events = events

	// Sort events by timestamp (newest first)
	for i := 0; i < len(info.Events)-1; i++ {
		for j := i + 1; j < len(info.Events); j++ {
			if info.Events[i].Timestamp.Before(info.Events[j].Timestamp) {
				info.Events[i], info.Events[j] = info.Events[j], info.Events[i]
			}
		}
	}

	// Set current status from most recent event
	if len(info.Events) > 0 {
		info.Status = info.Events[0].Status

		// Set delivery time if delivered
		if info.Status == StatusDelivered {
			info.ActualDelivery = &info.Events[0].Timestamp
		}
	}

	return info
}

func (c *USPSHeadlessClient) extractTrackingEvents(html string) []TrackingEvent {
	var events []TrackingEvent

	// Modern USPS SPA uses JSON data embedded in the page or loads via AJAX
	// Try multiple extraction strategies for different USPS page layouts

	// Strategy 1: Look for JSON data in script tags
	jsonEvents := c.extractEventsFromJSON(html)
	if len(jsonEvents) > 0 {
		events = append(events, jsonEvents...)
	}

	// Strategy 2: Extract from structured HTML elements
	if len(events) == 0 {
		htmlEvents := c.extractEventsFromHTML(html)
		events = append(events, htmlEvents...)
	}

	// Strategy 3: Extract from table rows (legacy format)
	if len(events) == 0 {
		tableEvents := c.extractEventsFromTable(html)
		events = append(events, tableEvents...)
	}

	// Strategy 4: Fallback to simple text patterns
	if len(events) == 0 {
		simpleEvents := c.extractSimpleEvents(html)
		events = append(events, simpleEvents...)
	}

	return events
}

func (c *USPSHeadlessClient) extractEventsFromJSON(html string) []TrackingEvent {
	var events []TrackingEvent

	// Look for JSON data patterns in script tags or data attributes
	jsonPatterns := []string{
		`window\.trackingData\s*=\s*(\{[^;]+\});`,
		`data-tracking-info="([^"]+)"`,
		`"trackingEvents":\s*(\[[^\]]+\])`,
		`"history":\s*(\[[^\]]+\])`,
	}

	for _, pattern := range jsonPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			// This would require JSON parsing, which we'll implement if needed
			// For now, continue to HTML extraction
			break
		}
	}

	return events
}

func (c *USPSHeadlessClient) extractEventsFromHTML(html string) []TrackingEvent {
	var events []TrackingEvent

	// Modern USPS uses .tb-step divs for tracking events
	// Extract tb-step tracking events (current USPS format)
	// Note: tb-date contains both date and time on separate lines
	tbStepPattern := `(?s)<div[^>]*class="[^"]*tb-step[^"]*"[^>]*>.*?<p[^>]*class="[^"]*tb-status-detail[^"]*"[^>]*>([^<]+)</p>.*?<p[^>]*class="[^"]*tb-location[^"]*"[^>]*>([^<]+)</p>.*?<p[^>]*class="[^"]*tb-date[^"]*"[^>]*>(.*?)</p>.*?</div>`
	re := regexp.MustCompile(tbStepPattern)
	matches := re.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			status := c.cleanHTML(match[1])
			location := c.cleanHTML(match[2])
			dateStr := c.cleanHTML(match[3])
			
			// Parse the date - USPS uses format like "July 2, 2025,"
			timestamp, err := c.parseUSPSDateTime(dateStr)
			if err != nil {
				// If date parsing fails, use current time
				timestamp = time.Now()
			}
			
			event := TrackingEvent{
				Timestamp:   timestamp,
				Status:      c.mapScrapedStatus(status),
				Location:    location,
				Description: status,
			}
			events = append(events, event)
		}
	}

	// If no tb-step events found, try other patterns
	if len(events) == 0 {
		patterns := []string{
			// Pattern 1: tracking-summary with nested elements
			`(?s)<div[^>]*class="[^"]*tracking-summary[^"]*"[^>]*>.*?<div[^>]*class="[^"]*timestamp[^"]*"[^>]*>([^<]+)</div>.*?<div[^>]*class="[^"]*status[^"]*"[^>]*>([^<]+)</div>.*?<div[^>]*class="[^"]*location[^"]*"[^>]*>([^<]+)</div>.*?</div>`,

			// Pattern 2: tracking-event divs
			`(?s)<div[^>]*class="[^"]*tracking-event[^"]*"[^>]*>.*?<span[^>]*class="[^"]*date[^"]*"[^>]*>([^<]+)</span>.*?<span[^>]*class="[^"]*time[^"]*"[^>]*>([^<]+)</span>.*?<div[^>]*class="[^"]*status[^"]*"[^>]*>([^<]+)</div>.*?<div[^>]*class="[^"]*location[^"]*"[^>]*>([^<]+)</div>.*?</div>`,

			// Pattern 3: delivery-status section
			`(?s)<div[^>]*class="[^"]*delivery-status[^"]*"[^>]*>.*?<p[^>]*>([^<]+)</p>.*?</div>`,

			// Pattern 4: tracking-progress-bar events
			`(?s)<li[^>]*class="[^"]*progress-event[^"]*"[^>]*>.*?<span[^>]*class="[^"]*date[^"]*"[^>]*>([^<]+)</span>.*?<span[^>]*class="[^"]*description[^"]*"[^>]*>([^<]+)</span>.*?</li>`,
		}

		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindAllStringSubmatch(html, -1)

			for _, match := range matches {
				if len(match) >= 4 {
					// Parse multi-field event
					event := c.parseUSPSEvent(match[1], match[2], match[3], match[4])
					events = append(events, event)
				} else if len(match) >= 2 {
					// Parse simple status event
					event := c.parseUSPSSummary(match[1])
					events = append(events, event)
				}
			}

			// If we found events with this pattern, use them
			if len(events) > 0 {
				break
			}
		}
	}

	return events
}

func (c *USPSHeadlessClient) extractEventsFromTable(html string) []TrackingEvent {
	var events []TrackingEvent

	// Extract from table format (older USPS format)
	tablePattern := `(?s)<tr[^>]*>.*?<td[^>]*>([^<]+)</td>.*?<td[^>]*>([^<]+)</td>.*?<td[^>]*>([^<]+)</td>.*?<td[^>]*>([^<]+)</td>.*?</tr>`
	re := regexp.MustCompile(tablePattern)
	matches := re.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) >= 5 {
			event := c.parseUSPSEvent(match[1], match[2], match[3], match[4])
			events = append(events, event)
		}
	}

	return events
}

func (c *USPSHeadlessClient) extractSimpleEvents(html string) []TrackingEvent {
	var events []TrackingEvent

	// Look for any mentions of delivery status in the HTML
	deliveryPatterns := []string{
		`(?i)delivered.*?(\w+ \d+, \d+).*?(\d+:\d+ [ap]m).*?([A-Z ]+\d{5})`,
		`(?i)out for delivery.*?(\w+ \d+, \d+).*?(\d+:\d+ [ap]m).*?([A-Z ]+\d{5})`,
		`(?i)in transit.*?(\w+ \d+, \d+).*?(\d+:\d+ [ap]m).*?([A-Z ]+\d{5})`,
		`(?i)package received.*?(\w+ \d+, \d+).*?(\d+:\d+ [ap]m).*?([A-Z ]+\d{5})`,
	}

	for _, pattern := range deliveryPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)

		for _, match := range matches {
			if len(match) >= 4 {
				dateTimeStr := match[1] + " at " + match[2]
				timestamp, _ := c.parseDateTime(dateTimeStr)

				status := StatusUnknown
				matchText := strings.ToLower(match[0])
				if strings.Contains(matchText, "delivered") {
					status = StatusDelivered
				} else if strings.Contains(matchText, "out for delivery") {
					status = StatusOutForDelivery
				} else if strings.Contains(matchText, "in transit") {
					status = StatusInTransit
				} else if strings.Contains(matchText, "package received") {
					status = StatusPreShip
				}

				event := TrackingEvent{
					Timestamp:   timestamp,
					Status:      status,
					Location:    strings.TrimSpace(match[3]),
					Description: c.cleanHTML(match[0]),
				}

				events = append(events, event)
			}
		}

		if len(events) > 0 {
			break
		}
	}

	return events
}

func (c *USPSHeadlessClient) parseUSPSEvent(timestamp, status, location, description string) TrackingEvent {
	// Clean up extracted text
	timestamp = c.cleanHTML(timestamp)
	status = c.cleanHTML(status)
	location = c.cleanHTML(location)
	description = c.cleanHTML(description)

	// Parse timestamp
	parsedTime, _ := c.parseDateTime(timestamp)

	// Map status
	mappedStatus := c.mapScrapedStatus(status + " " + description)

	return TrackingEvent{
		Timestamp:   parsedTime,
		Status:      mappedStatus,
		Location:    location,
		Description: description,
	}
}

func (c *USPSHeadlessClient) parseUSPSSummary(summaryText string) TrackingEvent {
	summaryText = c.cleanHTML(summaryText)

	// Extract timestamp and location from summary text
	// Example: "Your item was delivered at 2:15 pm on May 15, 2023 in ATLANTA GA 30309."
	timePattern := `(\w+ \d+, \d+) at (\d+:\d+ [ap]m)`
	locationPattern := `in ([A-Z ]+\d{5})`

	var timestamp time.Time
	var location string

	if timeRe := regexp.MustCompile(timePattern); timeRe.MatchString(summaryText) {
		timeMatches := timeRe.FindStringSubmatch(summaryText)
		if len(timeMatches) > 2 {
			dateTimeStr := timeMatches[1] + " at " + timeMatches[2]
			timestamp, _ = c.parseDateTime(dateTimeStr)
		}
	}

	if locRe := regexp.MustCompile(locationPattern); locRe.MatchString(summaryText) {
		locMatches := locRe.FindStringSubmatch(summaryText)
		if len(locMatches) > 1 {
			location = strings.TrimSpace(locMatches[1])
		}
	}

	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	return TrackingEvent{
		Timestamp:   timestamp,
		Status:      c.mapScrapedStatus(summaryText),
		Location:    location,
		Description: summaryText,
	}
}

// parseUSPSDateTime parses USPS-specific date formats
func (c *USPSHeadlessClient) parseUSPSDateTime(dateStr string) (time.Time, error) {
	// Clean up the date string and handle multi-line format
	dateStr = strings.TrimSpace(dateStr)
	dateStr = regexp.MustCompile(`\s+`).ReplaceAllString(dateStr, " ")
	
	// Remove HTML entities and extra whitespace
	dateStr = strings.ReplaceAll(dateStr, "&nbsp;", " ")
	dateStr = strings.TrimSpace(dateStr)
	
	// Handle the USPS format: "July 2, 2025, 12:35 am"
	// First try parsing the full string with various time formats
	fullLayouts := []string{
		"January 2, 2006, 3:04 pm",
		"January 2, 2006, 3:04 am", 
		"January 2, 2006, 15:04",
		"Jan 2, 2006, 3:04 pm",
		"Jan 2, 2006, 3:04 am",
		"January 2, 2006 3:04 pm",
		"January 2, 2006 3:04 am", 
		"Jan 2, 2006 3:04 pm",
		"Jan 2, 2006 3:04 am",
	}
	
	for _, layout := range fullLayouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t, nil
		}
	}
	
	// If full parsing fails, try to extract date and time separately
	if strings.Contains(dateStr, ",") {
		// Look for pattern like "July 2, 2025, 12:35 am"
		// Split carefully to handle "Month Day, Year, Time"
		commaIndex := strings.LastIndex(dateStr, ",")
		if commaIndex > 0 && commaIndex < len(dateStr)-1 {
			datePart := strings.TrimSpace(dateStr[:commaIndex])    // "July 2, 2025"
			timePart := strings.TrimSpace(dateStr[commaIndex+1:]) // "12:35 am"
			
			if timePart != "" {
				dateTimeStr := datePart + " " + timePart // "July 2, 2025 12:35 am"
				
				timeLayouts := []string{
					"January 2, 2006 3:04 pm",
					"January 2, 2006 3:04 am", 
					"January 2, 2006 15:04",
					"Jan 2, 2006 3:04 pm",
					"Jan 2, 2006 3:04 am",
				}
				
				for _, layout := range timeLayouts {
					if t, err := time.Parse(layout, dateTimeStr); err == nil {
						return t, nil
					}
				}
			}
			
			// Try parsing just the date part
			dateLayouts := []string{
				"January 2, 2006",
				"Jan 2, 2006",
			}
			
			for _, layout := range dateLayouts {
				if t, err := time.Parse(layout, datePart); err == nil {
					return t, nil
				}
			}
		}
	}
	
	// Fallback to other common formats
	layouts := []string{
		"January 2, 2006",     // July 2, 2025
		"Jan 2, 2006",         // Jul 2, 2025  
		"01/02/2006",          // 07/02/2025
		"2006-01-02",          // 2025-07-02
		"January 2, 2006 at 3:04 PM",  // Full format with time
		"Jan 2, 2006 at 3:04 PM",      // Short format with time
		"01/02/2006 3:04 PM",          // Numeric with time
		"01/02/2006 15:04",            // 24-hour format
	}
	
	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t, nil
		}
	}
	
	return time.Now(), fmt.Errorf("unable to parse USPS date: %s", dateStr)
}