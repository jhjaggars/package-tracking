package carriers

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// FedExHeadlessClient implements headless browser tracking for FedEx
type FedExHeadlessClient struct {
	*HeadlessScrapingClient
	baseURL string
}

// NewFedExHeadlessClient creates a new FedEx headless client
func NewFedExHeadlessClient() *FedExHeadlessClient {
	options := DefaultHeadlessOptions()
	options.WaitStrategy = WaitForSelector
	options.Timeout = 90 * time.Second // Extended timeout for Angular SPA loading
	
	// Use Firefox user agent since FedEx blocks Chrome
	options.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0"
	
	// Enable maximum stealth mode for FedEx bot detection
	options.StealthMode = true
	options.EnablePlugins = true
	options.LoadImages = true
	options.EnableWebGL = true

	headlessClient := NewHeadlessScrapingClient("fedex", options.UserAgent, options)

	return &FedExHeadlessClient{
		HeadlessScrapingClient: headlessClient,
		baseURL:                "https://www.fedex.com",
	}
}

// ValidateTrackingNumber validates FedEx tracking number formats
func (c *FedExHeadlessClient) ValidateTrackingNumber(trackingNumber string) bool {
	if trackingNumber == "" {
		return false
	}
	
	// Remove spaces and keep only digits
	cleaned := strings.ReplaceAll(trackingNumber, " ", "")
	
	// Check if it's all digits
	if matched, _ := regexp.MatchString(`^\d+$`, cleaned); !matched {
		return false
	}
	
	// FedEx tracking number lengths: 12, 14, 15, 16, 18, 20, 22
	validLengths := []int{12, 14, 15, 16, 18, 20, 22}
	
	for _, length := range validLengths {
		if len(cleaned) == length {
			return true
		}
	}
	
	return false
}

// Track retrieves tracking information for the given tracking numbers
func (c *FedExHeadlessClient) Track(ctx context.Context, req *TrackingRequest) (*TrackingResponse, error) {
	if len(req.TrackingNumbers) == 0 {
		return nil, fmt.Errorf("no tracking numbers provided")
	}
	
	var results []TrackingInfo
	var errors []CarrierError
	
	// FedEx tracking website handles one tracking number per request
	for _, trackingNumber := range req.TrackingNumbers {
		result, err := c.trackSingle(ctx, trackingNumber)
		if err != nil {
			// Handle different error types consistently
			switch e := err.(type) {
			case *CarrierError:
				errors = append(errors, *e)
				// For rate limits, return immediately
				if e.RateLimit {
					return nil, err
				}
			case *HeadlessCarrierError:
				errors = append(errors, *e.CarrierError)
				// For rate limits in headless errors, also return immediately
				if e.CarrierError.RateLimit {
					return nil, err
				}
			default:
				// Convert unknown errors to CarrierError for consistency
				carrierErr := &CarrierError{
					Carrier:   "fedex",
					Code:      "UNKNOWN_ERROR",
					Message:   err.Error(),
					Retryable: false,
					RateLimit: false,
				}
				errors = append(errors, *carrierErr)
			}
		} else if result != nil {
			results = append(results, *result)
		}
	}
	
	return &TrackingResponse{
		Results:   results,
		Errors:    errors,
		RateLimit: c.GetRateLimit(),
	}, nil
}

func (c *FedExHeadlessClient) trackSingle(ctx context.Context, trackingNumber string) (*TrackingInfo, error) {
	// Build tracking URL - Updated format for modern FedEx
	trackURL := fmt.Sprintf("%s/wtrk/track/?tracknumbers=%s", c.baseURL, url.QueryEscape(trackingNumber))
	
	// Define selectors for FedEx tracking elements (Angular SPA)
	trackingSelectors := []string{
		"[data-test-id='tracking-details']",           // Primary tracking container
		".tracking-details",                           // Alternative container
		"[data-automation-id='trackingEvents']",       // Events container
		".tracking-events",                            // Alternative events
		".timeline-container",                         // Timeline view
		"[role='main'] .tracking",                     // Main tracking section
		".shipment-progress",                          // Progress indicator
		"app-tracking-timeline",                       // Angular component
		"app-root",                                    // Angular root component
		"[ng-if*='tracking']",                         // Angular conditional elements
		".shipment-details",                           // Shipment container
		"#tracking-results",                           // Results container
	}
	
	// First navigate and wait for Angular to initialize
	err := c.NavigateAndWaitForAngular(ctx, trackURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Angular SPA: %w", err)
	}
	
	// Then wait for tracking data to load
	pageSource, err := c.NavigateAndWaitForTrackingData(ctx, trackURL, trackingSelectors)
	if err != nil {
		return nil, err
	}
	
	// Check for bot detection first (most common issue)
	if c.isBotDetection(pageSource) {
		return nil, &CarrierError{
			Carrier:   "fedex",
			Code:      "BOT_DETECTION",
			Message:   "FedEx detected automated access. Try using Firefox browser or FedEx official API for " + trackingNumber,
			Retryable: true,
			RateLimit: false,
		}
	}
	
	// Check for server errors
	if c.isServerError(pageSource) {
		return nil, &CarrierError{
			Carrier:   "fedex",
			Code:      "SERVER_ERROR", 
			Message:   "FedEx systems temporarily unavailable for " + trackingNumber,
			Retryable: true,
			RateLimit: false,
		}
	}
	
	// Check for legitimate "not found" errors
	if c.isTrackingNotFound(pageSource) {
		return nil, &CarrierError{
			Carrier:   "fedex",
			Code:      "NOT_FOUND",
			Message:   "Tracking information not found for " + trackingNumber,
			Retryable: false,
			RateLimit: false,
		}
	}
	
	// Parse tracking information from the rendered HTML
	trackingInfo := c.parseFedExTrackingInfo(pageSource, trackingNumber)
	
	// If no events were found, try to extract using headless-specific methods
	if len(trackingInfo.Events) == 0 {
		events, err := c.extractEventsWithHeadless(ctx, trackingNumber)
		if err == nil && len(events) > 0 {
			trackingInfo.Events = events
		}
	}
	
	// If still no events, return error
	if len(trackingInfo.Events) == 0 {
		return nil, &CarrierError{
			Carrier:   "fedex",
			Code:      "NO_EVENTS",
			Message:   "No tracking events found for " + trackingNumber,
			Retryable: true,
			RateLimit: false,
		}
	}
	
	return &trackingInfo, nil
}

// extractEventsWithHeadless uses headless browser capabilities to extract events
func (c *FedExHeadlessClient) extractEventsWithHeadless(ctx context.Context, trackingNumber string) ([]TrackingEvent, error) {
	extractors := []ContentExtractor{
		{
			Name:     "event_dates",
			Selector: "[data-test-id='event-date'], .event-date, .timeline-date",
			Multiple: true,
			Required: false,
		},
		{
			Name:     "event_times",
			Selector: "[data-test-id='event-time'], .event-time, .timeline-time",
			Multiple: true,
			Required: false,
		},
		{
			Name:     "event_statuses",
			Selector: "[data-test-id='event-status'], .event-status, .timeline-status",
			Multiple: true,
			Required: false,
		},
		{
			Name:     "event_locations",
			Selector: "[data-test-id='event-location'], .event-location, .timeline-location",
			Multiple: true,
			Required: false,
		},
		{
			Name:     "event_descriptions",
			Selector: "[data-test-id='event-description'], .event-description, .timeline-description",
			Multiple: true,
			Required: false,
		},
		{
			Name:     "event_containers",
			Selector: "[data-test-id='tracking-event'], .tracking-event, .timeline-event",
			Multiple: true,
			Required: false,
		},
	}
	
	trackURL := fmt.Sprintf("%s/wtrk/track/?tracknumbers=%s", c.baseURL, url.QueryEscape(trackingNumber))
	
	results, err := c.NavigateAndExtract(ctx, trackURL, extractors)
	if err != nil {
		return nil, err
	}
	
	return c.parseExtractedEvents(results), nil
}

// parseExtractedEvents converts extracted data into TrackingEvent structs
func (c *FedExHeadlessClient) parseExtractedEvents(results map[string]interface{}) []TrackingEvent {
	var events []TrackingEvent
	
	// Try to get parallel arrays of event data
	dates, _ := results["event_dates"].([]string)
	times, _ := results["event_times"].([]string)
	statuses, _ := results["event_statuses"].([]string)
	locations, _ := results["event_locations"].([]string)
	descriptions, _ := results["event_descriptions"].([]string)
	
	// Find the maximum length to iterate over
	maxLen := len(dates)
	if len(times) > maxLen {
		maxLen = len(times)
	}
	if len(statuses) > maxLen {
		maxLen = len(statuses)
	}
	if len(locations) > maxLen {
		maxLen = len(locations)
	}
	if len(descriptions) > maxLen {
		maxLen = len(descriptions)
	}
	
	// Create events from parallel arrays
	for i := 0; i < maxLen; i++ {
		var date, timeStr, status, location, description string
		
		if i < len(dates) {
			date = dates[i]
		}
		if i < len(times) {
			timeStr = times[i]
		}
		if i < len(statuses) {
			status = statuses[i]
		}
		if i < len(locations) {
			location = locations[i]
		}
		if i < len(descriptions) {
			description = descriptions[i]
		}
		
		// Skip if we don't have meaningful data
		if date == "" && timeStr == "" && status == "" && description == "" {
			continue
		}
		
		event := c.createTrackingEvent(date, timeStr, status, location, description)
		events = append(events, event)
	}
	
	return events
}

// createTrackingEvent creates a tracking event from extracted data
func (c *FedExHeadlessClient) createTrackingEvent(date, timeStr, status, location, description string) TrackingEvent {
	// Clean up extracted text
	date = c.cleanHTML(date)
	timeStr = c.cleanHTML(timeStr)
	status = c.cleanHTML(status)
	location = c.cleanHTML(location)
	description = c.cleanHTML(description)
	
	// Parse timestamp
	var parsedTime time.Time
	if date != "" && timeStr != "" {
		dateTimeStr := date + " " + timeStr
		parsedTime, _ = c.parseDateTime(dateTimeStr)
	} else if date != "" {
		parsedTime, _ = c.parseDateTime(date)
	} else {
		parsedTime = time.Now()
	}
	
	// Map status
	mappedStatus := c.mapScrapedStatus(status + " " + description)
	
	// Use description if available, otherwise use status
	eventDescription := description
	if eventDescription == "" {
		eventDescription = status
	}
	
	return TrackingEvent{
		Timestamp:   parsedTime,
		Status:      mappedStatus,
		Location:    location,
		Description: eventDescription,
	}
}

// isTrackingNotFound checks for legitimate "not found" errors in the rendered page
func (c *FedExHeadlessClient) isTrackingNotFound(html string) bool {
	// Check for legitimate "not found" patterns (actual tracking errors)
	notFoundPatterns := []string{
		"Tracking number not found",
		"cannot locate",
		"shipment details for this tracking number",
		"check the number and try again",
		"invalid tracking number",
		"no results found",
	}
	
	lowerHTML := strings.ToLower(html)
	for _, pattern := range notFoundPatterns {
		if strings.Contains(lowerHTML, strings.ToLower(pattern)) {
			return true
		}
	}
	
	return false
}

// isBotDetection checks for FedEx bot detection messages
func (c *FedExHeadlessClient) isBotDetection(html string) bool {
	// FedEx bot detection typically shows generic "unable to retrieve" messages
	botDetectionPatterns := []string{
		"unfortunately we are unable to retrieve your tracking results at this time",
		"please try again later",
		"unable to retrieve your tracking results",
	}
	
	lowerHTML := strings.ToLower(html)
	for _, pattern := range botDetectionPatterns {
		if strings.Contains(lowerHTML, pattern) {
			return true
		}
	}
	
	return false
}

// isServerError checks for legitimate FedEx server errors
func (c *FedExHeadlessClient) isServerError(html string) bool {
	// Server errors that are not bot detection
	serverErrorPatterns := []string{
		"service temporarily unavailable",
		"system is currently unavailable", 
		"maintenance mode",
		"temporarily down for maintenance",
	}
	
	lowerHTML := strings.ToLower(html)
	for _, pattern := range serverErrorPatterns {
		if strings.Contains(lowerHTML, pattern) {
			return true
		}
	}
	
	return false
}

// parseFedExTrackingInfo parses tracking info from rendered HTML (fallback method)
func (c *FedExHeadlessClient) parseFedExTrackingInfo(html, trackingNumber string) TrackingInfo {
	info := TrackingInfo{
		TrackingNumber: trackingNumber,
		Carrier:        "fedex",
		Events:         []TrackingEvent{},
		LastUpdated:    time.Now(),
		Status:         StatusUnknown,
	}
	
	// Extract events from tracking information using existing regex patterns
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

// extractTrackingEvents extracts events using the base scraping patterns
func (c *FedExHeadlessClient) extractTrackingEvents(html string) []TrackingEvent {
	// Use the existing pattern-based extraction as fallback
	var events []TrackingEvent
	
	// Updated patterns for modern FedEx with data attributes
	patterns := []string{
		// Pattern 1: Modern FedEx with data attributes
		`(?s)<div[^>]*data-test-id="tracking-event"[^>]*>.*?<div[^>]*data-test-id="event-date"[^>]*>([^<]+)</div>.*?<div[^>]*data-test-id="event-time"[^>]*>([^<]+)</div>.*?<div[^>]*data-test-id="event-status"[^>]*>([^<]+)</div>.*?<div[^>]*data-test-id="event-location"[^>]*>([^<]+)</div>.*?<div[^>]*data-test-id="event-description"[^>]*>([^<]+)</div>.*?</div>`,
		
		// Pattern 2: Angular components
		`(?s)<app-tracking-event[^>]*>.*?<span[^>]*class="[^"]*date[^"]*"[^>]*>([^<]+)</span>.*?<span[^>]*class="[^"]*time[^"]*"[^>]*>([^<]+)</span>.*?<span[^>]*class="[^"]*status[^"]*"[^>]*>([^<]+)</span>.*?<span[^>]*class="[^"]*location[^"]*"[^>]*>([^<]+)</span>.*?<span[^>]*class="[^"]*description[^"]*"[^>]*>([^<]+)</span>.*?</app-tracking-event>`,
		
		// Pattern 3: Legacy patterns (from original implementation)
		`(?s)<div[^>]*class="[^"]*tracking-event[^"]*"[^>]*>.*?<div[^>]*class="[^"]*event-date[^"]*"[^>]*>([^<]+)</div>.*?<div[^>]*class="[^"]*event-time[^"]*"[^>]*>([^<]+)</div>.*?<div[^>]*class="[^"]*event-status[^"]*"[^>]*>([^<]+)</div>.*?<div[^>]*class="[^"]*event-location[^"]*"[^>]*>([^<]+)</div>.*?<div[^>]*class="[^"]*event-description[^"]*"[^>]*>([^<]+)</div>.*?</div>`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		
		for _, match := range matches {
			if len(match) >= 6 {
				// date, time, status, location, description
				event := c.createTrackingEvent(match[1], match[2], match[3], match[4], match[5])
				events = append(events, event)
			}
		}
		
		// If we found events with this pattern, use them
		if len(events) > 0 {
			break
		}
	}
	
	return events
}

// NavigateAndWaitForAngular navigates to FedEx page and waits for Angular SPA to initialize
func (c *FedExHeadlessClient) NavigateAndWaitForAngular(ctx context.Context, url string) error {
	return c.browserPool.ExecuteWithBrowser(ctx, func(browserCtx context.Context) error {
		// Step 1: Navigate to the page
		err := chromedp.Run(browserCtx, chromedp.Navigate(url))
		if err != nil {
			return fmt.Errorf("navigation failed: %w", err)
		}

		// Step 2: Wait for basic DOM to load
		err = chromedp.Run(browserCtx, chromedp.WaitReady("body", chromedp.ByQuery))
		if err != nil {
			return fmt.Errorf("body not ready: %w", err)
		}

		// Step 3: Execute enhanced stealth script to bypass FedEx bot detection
		err = chromedp.Run(browserCtx, chromedp.Evaluate(`
			// Remove webdriver property (primary bot detection)
			Object.defineProperty(navigator, 'webdriver', {
				get: () => undefined,
			});
			
			// Hide automation indicators
			if (window.chrome) {
				window.chrome.runtime = {};
				// Remove automation-specific chrome properties
				delete window.chrome.loadTimes;
				delete window.chrome.csi;
			}
			
			// Add realistic plugins to mimic real Firefox
			Object.defineProperty(navigator, 'plugins', {
				get: () => [
					{
						name: "PDF.js",
						description: "Portable Document Format",
						filename: "internal-pdf-viewer"
					},
					{
						name: "OpenH264 Video Codec provided by Cisco Systems, Inc.",
						description: "OpenH264 Video Codec provided by Cisco Systems, Inc.",
						filename: "gmpopenh264"
					}
				],
			});
			
			// Realistic mimeTypes
			Object.defineProperty(navigator, 'mimeTypes', {
				get: () => [
					{
						type: "application/pdf",
						description: "Portable Document Format",
						suffixes: "pdf"
					}
				],
			});
			
			// Override languages to match Firefox
			Object.defineProperty(navigator, 'languages', {
				get: () => ['en-US', 'en'],
			});
			
			// Override permissions API 
			if (navigator.permissions && navigator.permissions.query) {
				const originalQuery = navigator.permissions.query;
				navigator.permissions.query = (parameters) => (
					parameters.name === 'notifications' ?
						Promise.resolve({ state: Notification.permission }) :
						originalQuery(parameters)
				);
			}
			
			// Hide automation-controlled flag
			if (navigator.connection) {
				Object.defineProperty(navigator.connection, 'rtt', {
					get: () => 50 + Math.floor(Math.random() * 50),
				});
			}
			
			// Console log to verify stealth mode
			console.log("Enhanced FedEx stealth mode activated");
		`, nil))
		if err != nil {
			// Non-critical error, continue
			fmt.Printf("Warning: stealth script execution failed: %v\n", err)
		}

		// Step 4: Wait for Angular to initialize with timeout
		err = chromedp.Run(browserCtx, chromedp.Poll(`
			// Check if Angular is ready
			if (window.angular) {
				try {
					var rootElement = document.querySelector('app-root') || document.body;
					var scope = window.angular.element(rootElement).scope();
					return scope && document.readyState === 'complete';
				} catch (e) {
					return false;
				}
			}
			// Fallback: check if DOM is ready and page seems loaded
			return document.readyState === 'complete' && 
				   document.querySelector('app-root') !== null;
		`, nil, chromedp.WithPollingTimeout(30*time.Second)))
		
		if err != nil {
			// Try alternative Angular detection
			err = chromedp.Run(browserCtx, chromedp.WaitVisible("app-root", chromedp.ByQuery))
			if err != nil {
				return fmt.Errorf("Angular app not ready: %w", err)
			}
		}

		// Step 5: Wait for network to be idle (AJAX calls complete)
		err = chromedp.Run(browserCtx, chromedp.Sleep(3*time.Second))
		if err != nil {
			return fmt.Errorf("network idle wait failed: %w", err)
		}

		// Step 6: Simulate human behavior
		err = chromedp.Run(browserCtx, 
			// Random scroll to trigger lazy loading
			chromedp.Evaluate(`
				window.scrollTo({
					top: Math.floor(Math.random() * 300),
					behavior: 'smooth'
				});
			`, nil),
			chromedp.Sleep(1*time.Second),
			
			// Scroll back to top
			chromedp.Evaluate(`
				window.scrollTo({
					top: 0,
					behavior: 'smooth'
				});
			`, nil),
			chromedp.Sleep(1*time.Second),
		)
		
		if err != nil {
			// Non-critical, continue
			fmt.Printf("Warning: human behavior simulation failed: %v\n", err)
		}

		return nil
	})
}