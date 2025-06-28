package carriers

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

// HeadlessScrapingClient provides headless browser functionality for web scraping
type HeadlessScrapingClient struct {
	*ScrapingClient
	browserPool *SimpleBrowserPool
	options     *HeadlessOptions
	actions     *ChromeDPActions
}

// NewHeadlessScrapingClient creates a new headless scraping client
func NewHeadlessScrapingClient(carrier, userAgent string, options *HeadlessOptions) *HeadlessScrapingClient {
	if options == nil {
		options = DefaultHeadlessOptions()
	}
	if userAgent != "" {
		options.UserAgent = userAgent
	}

	scrapingClient := NewScrapingClient(carrier, options.UserAgent)
	browserPool := NewBrowserPool(DefaultBrowserPoolConfig(), options)

	return &HeadlessScrapingClient{
		ScrapingClient: scrapingClient,
		browserPool:    browserPool,
		options:        options,
		actions:        &ChromeDPActions{},
	}
}

// NavigateAndExtract navigates to a URL and extracts content using selectors
func (h *HeadlessScrapingClient) NavigateAndExtract(ctx context.Context, url string, extractors []ContentExtractor) (map[string]interface{}, error) {
	results := make(map[string]interface{})

	err := h.browserPool.ExecuteWithBrowser(ctx, func(browserCtx context.Context) error {
		// Navigate to the URL
		err := chromedp.Run(browserCtx, chromedp.Navigate(url))
		if err != nil {
			return fmt.Errorf("failed to navigate to %s: %w", url, err)
		}

		// Wait for content based on strategy
		err = h.waitForContent(browserCtx, extractors)
		if err != nil {
			return fmt.Errorf("failed to wait for content: %w", err)
		}

		// Extract content using the extractors
		for _, extractor := range extractors {
			value, err := h.extractContent(browserCtx, extractor)
			if err != nil {
				if extractor.Required {
					return fmt.Errorf("failed to extract required content %s: %w", extractor.Name, err)
				}
				// Log warning for optional extractors but continue
				continue
			}
			results[extractor.Name] = value
		}

		return nil
	})

	if err != nil {
		return nil, h.wrapError(err, "navigate and extract failed")
	}

	return results, nil
}

// WaitForContent waits for specific content to appear on the page
func (h *HeadlessScrapingClient) WaitForContent(ctx context.Context, selector string, timeout time.Duration) error {
	return h.browserPool.ExecuteWithBrowser(ctx, func(browserCtx context.Context) error {
		timeoutCtx, cancel := context.WithTimeout(browserCtx, timeout)
		defer cancel()

		return chromedp.Run(timeoutCtx, chromedp.WaitVisible(selector, chromedp.ByQuery))
	})
}

// ExecuteScript runs JavaScript in the browser context
func (h *HeadlessScrapingClient) ExecuteScript(ctx context.Context, script string, result interface{}) error {
	return h.browserPool.ExecuteWithBrowser(ctx, func(browserCtx context.Context) error {
		return chromedp.Run(browserCtx, chromedp.Evaluate(script, result))
	})
}

// Screenshot captures a screenshot (useful for debugging)
func (h *HeadlessScrapingClient) Screenshot(ctx context.Context) ([]byte, error) {
	var buf []byte
	err := h.browserPool.ExecuteWithBrowser(ctx, func(browserCtx context.Context) error {
		return chromedp.Run(browserCtx, chromedp.FullScreenshot(&buf, 90))
	})
	if err != nil {
		return nil, h.wrapError(err, "screenshot failed")
	}
	return buf, nil
}

// GetPageSource returns the current page's HTML source
func (h *HeadlessScrapingClient) GetPageSource(ctx context.Context) (string, error) {
	var html string
	err := h.browserPool.ExecuteWithBrowser(ctx, func(browserCtx context.Context) error {
		return chromedp.Run(browserCtx, chromedp.OuterHTML("html", &html))
	})
	if err != nil {
		return "", h.wrapError(err, "get page source failed")
	}
	return html, nil
}

// SetOptions updates the headless browser options
func (h *HeadlessScrapingClient) SetOptions(options *HeadlessOptions) {
	h.options = options
	// Note: Existing browser instances won't be affected, only new ones
}

// Close cleanly shuts down the browser pool
func (h *HeadlessScrapingClient) Close() error {
	return h.browserPool.Close()
}

// waitForContent implements different waiting strategies
func (h *HeadlessScrapingClient) waitForContent(ctx context.Context, extractors []ContentExtractor) error {
	switch h.options.WaitStrategy {
	case WaitForSelector:
		// Wait for the first required selector to appear
		for _, extractor := range extractors {
			if extractor.Required {
				return chromedp.Run(ctx, chromedp.WaitVisible(extractor.Selector, chromedp.ByQuery))
			}
		}
		// If no required extractors, wait for the first one
		if len(extractors) > 0 {
			return chromedp.Run(ctx, chromedp.WaitVisible(extractors[0].Selector, chromedp.ByQuery))
		}

	case WaitForNetworkIdle:
		// Wait for network activity to settle
		return chromedp.Run(ctx, chromedp.Sleep(2*time.Second)) // Simple implementation

	case WaitForTimeout:
		// Wait for a fixed duration
		return chromedp.Run(ctx, chromedp.Sleep(3*time.Second))

	case WaitForCustom:
		// Custom logic would be implemented by specific carrier clients
		return nil
	}

	return nil
}

// extractContent extracts content based on the extractor configuration
func (h *HeadlessScrapingClient) extractContent(ctx context.Context, extractor ContentExtractor) (interface{}, error) {
	if extractor.Multiple {
		if extractor.Attribute == "" {
			// Extract text from multiple elements
			var texts []string
			script := fmt.Sprintf(`
				Array.from(document.querySelectorAll('%s')).map(el => el.textContent.trim())
			`, extractor.Selector)
			err := chromedp.Run(ctx, chromedp.Evaluate(script, &texts))
			return texts, err
		} else {
			// Extract attribute from multiple elements
			var attributes []string
			script := fmt.Sprintf(`
				Array.from(document.querySelectorAll('%s')).map(el => el.getAttribute('%s'))
			`, extractor.Selector, extractor.Attribute)
			err := chromedp.Run(ctx, chromedp.Evaluate(script, &attributes))
			return attributes, err
		}
	} else {
		if extractor.Attribute == "" {
			// Extract text from single element
			var text string
			err := chromedp.Run(ctx, chromedp.Text(extractor.Selector, &text, chromedp.ByQuery))
			return text, err
		} else {
			// Extract attribute from single element
			var attribute string
			err := chromedp.Run(ctx, chromedp.AttributeValue(extractor.Selector, extractor.Attribute, &attribute, nil, chromedp.ByQuery))
			return attribute, err
		}
	}
}

// wrapError creates a HeadlessCarrierError with additional context
func (h *HeadlessScrapingClient) wrapError(err error, message string) error {
	carrierErr := &CarrierError{
		Carrier:   h.GetCarrierName(),
		Code:      "HEADLESS_ERROR",
		Message:   message,
		Retryable: true,
		RateLimit: false,
	}

	headlessErr := &HeadlessCarrierError{
		CarrierError: carrierErr,
		BrowserError: err.Error(),
	}

	// Try to capture additional debug info if in debug mode
	if h.options.DebugMode {
		// Capture debug artifacts with size limits
		if h.options.MaxDebugArtifactSize > 0 {
			// Note: In a real implementation, you might want to capture
			// page source and screenshot here for debugging
			// For now, we just ensure the error is properly truncated
			headlessErr.TruncateDebugArtifacts(h.options.MaxDebugArtifactSize)
		}
	}

	return headlessErr
}

// NavigateAndWaitForTrackingData is a convenience method for carrier tracking pages
func (h *HeadlessScrapingClient) NavigateAndWaitForTrackingData(ctx context.Context, url string, trackingSelectors []string) (string, error) {
	var pageSource string

	err := h.browserPool.ExecuteWithBrowser(ctx, func(browserCtx context.Context) error {
		// Navigate to the URL
		err := chromedp.Run(browserCtx, chromedp.Navigate(url))
		if err != nil {
			return fmt.Errorf("failed to navigate to %s: %w", url, err)
		}

		// Wait for any of the tracking selectors to appear
		var waitTasks chromedp.Tasks
		for _, selector := range trackingSelectors {
			waitTasks = append(waitTasks, chromedp.WaitVisible(selector, chromedp.ByQuery))
		}

		// Use a timeout context for waiting
		waitCtx, cancel := context.WithTimeout(browserCtx, h.options.Timeout)
		defer cancel()

		// Try to wait for any of the selectors (first one that appears wins)
		for _, task := range waitTasks {
			err := chromedp.Run(waitCtx, task)
			if err == nil {
				break // Found at least one selector
			}
		}

		// Get the page source after content has loaded
		return chromedp.Run(browserCtx, chromedp.OuterHTML("html", &pageSource))
	})

	if err != nil {
		return "", h.wrapError(err, "navigate and wait for tracking data failed")
	}

	return pageSource, nil
}