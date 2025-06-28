package carriers

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"
)

// WaitStrategy defines different strategies for waiting for content to load
type WaitStrategy int

const (
	// WaitForSelector waits for a specific CSS selector to appear
	WaitForSelector WaitStrategy = iota
	// WaitForNetworkIdle waits for network activity to cease
	WaitForNetworkIdle
	// WaitForTimeout waits for a fixed duration
	WaitForTimeout
	// WaitForCustom uses custom logic defined by the implementation
	WaitForCustom
)

// HeadlessOptions contains configuration for headless browser operations
type HeadlessOptions struct {
	// Headless controls whether to run browser in headless mode
	Headless bool
	// Timeout for browser operations
	Timeout time.Duration
	// WaitStrategy defines how to wait for content
	WaitStrategy WaitStrategy
	// DisableImages optimizes performance by not loading images
	DisableImages bool
	// UserAgent to use for requests
	UserAgent string
	// ViewportWidth sets browser viewport width
	ViewportWidth int64
	// ViewportHeight sets browser viewport height
	ViewportHeight int64
	// DebugMode enables additional logging
	DebugMode bool
}

// DefaultHeadlessOptions returns sensible defaults for headless browsing
func DefaultHeadlessOptions() *HeadlessOptions {
	return &HeadlessOptions{
		Headless:       true,
		Timeout:        30 * time.Second,
		WaitStrategy:   WaitForSelector,
		DisableImages:  true,
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		DebugMode:      false,
	}
}

// ContentExtractor defines how to extract specific content from a page
type ContentExtractor struct {
	// Name identifies this extractor
	Name string
	// Selector is the CSS selector to find elements
	Selector string
	// Attribute specifies which attribute to extract (empty for text content)
	Attribute string
	// Multiple indicates whether to extract all matches or just the first
	Multiple bool
	// Required indicates whether this content must be found
	Required bool
}

// BrowserPoolConfig contains configuration for browser pool management
type BrowserPoolConfig struct {
	// MaxBrowsers limits the number of concurrent browser instances
	MaxBrowsers int
	// IdleTimeout defines how long to keep idle browsers alive
	IdleTimeout time.Duration
	// MaxIdleBrowsers limits the number of idle browsers to keep
	MaxIdleBrowsers int
}

// DefaultBrowserPoolConfig returns sensible defaults for browser pool
func DefaultBrowserPoolConfig() *BrowserPoolConfig {
	return &BrowserPoolConfig{
		MaxBrowsers:     5,
		IdleTimeout:     5 * time.Minute,
		MaxIdleBrowsers: 2,
	}
}

// BrowserInstance represents a managed browser instance
type BrowserInstance struct {
	ctx       context.Context
	cancel    context.CancelFunc
	lastUsed  time.Time
	inUse     bool
	allocator context.Context
}

// HeadlessBrowserClient extends the base Client interface with headless-specific capabilities
type HeadlessBrowserClient interface {
	Client

	// NavigateAndExtract navigates to a URL and extracts content using selectors
	NavigateAndExtract(ctx context.Context, url string, extractors []ContentExtractor) (map[string]interface{}, error)

	// WaitForContent waits for specific content to appear on the page
	WaitForContent(ctx context.Context, selector string, timeout time.Duration) error

	// ExecuteScript runs JavaScript in the browser context
	ExecuteScript(ctx context.Context, script string, result interface{}) error

	// Screenshot captures a screenshot (useful for debugging)
	Screenshot(ctx context.Context) ([]byte, error)

	// GetPageSource returns the current page's HTML source
	GetPageSource(ctx context.Context) (string, error)

	// SetOptions updates the headless browser options
	SetOptions(options *HeadlessOptions)

	// Close cleanly shuts down the browser instance
	Close() error
}

// BrowserPool manages a pool of browser instances for efficient reuse
type BrowserPool interface {
	// Get retrieves an available browser instance
	Get(ctx context.Context) (*BrowserInstance, error)

	// Put returns a browser instance to the pool
	Put(instance *BrowserInstance) error

	// Close shuts down all browser instances in the pool
	Close() error

	// Stats returns current pool statistics
	Stats() BrowserPoolStats
}

// BrowserPoolStats provides information about browser pool usage
type BrowserPoolStats struct {
	Active int `json:"active"`
	Idle   int `json:"idle"`
	Total  int `json:"total"`
}

// HeadlessCarrierError extends CarrierError with browser-specific error information
type HeadlessCarrierError struct {
	*CarrierError
	BrowserError  string `json:"browser_error,omitempty"`
	Screenshot    []byte `json:"screenshot,omitempty"`
	PageSource    string `json:"page_source,omitempty"`
	JavaScriptLog string `json:"javascript_log,omitempty"`
}

func (e *HeadlessCarrierError) Error() string {
	if e.BrowserError != "" {
		return e.CarrierError.Error() + " (browser: " + e.BrowserError + ")"
	}
	return e.CarrierError.Error()
}

// ChromeDPActions contains common chromedp actions for carrier tracking
type ChromeDPActions struct{}

// WaitForTrackingData returns chromedp actions to wait for tracking information to load
func (a *ChromeDPActions) WaitForTrackingData(selectors []string, timeout time.Duration) chromedp.Tasks {
	var tasks chromedp.Tasks

	// Wait for any of the tracking selectors to appear
	for _, selector := range selectors {
		tasks = append(tasks, chromedp.WaitVisible(selector, chromedp.ByQuery))
	}

	return tasks
}

// ExtractTrackingEvents returns chromedp actions to extract tracking events
func (a *ChromeDPActions) ExtractTrackingEvents(extractors []ContentExtractor) chromedp.Tasks {
	var tasks chromedp.Tasks
	var results = make(map[string]interface{})

	for _, extractor := range extractors {
		if extractor.Multiple {
			var elements []string
			if extractor.Attribute == "" {
				tasks = append(tasks, chromedp.Evaluate(`
					Array.from(document.querySelectorAll('`+extractor.Selector+`')).map(el => el.textContent.trim())
				`, &elements))
			} else {
				tasks = append(tasks, chromedp.Evaluate(`
					Array.from(document.querySelectorAll('`+extractor.Selector+`')).map(el => el.getAttribute('`+extractor.Attribute+`'))
				`, &elements))
			}
			results[extractor.Name] = elements
		} else {
			var element string
			if extractor.Attribute == "" {
				tasks = append(tasks, chromedp.Text(extractor.Selector, &element, chromedp.ByQuery))
			} else {
				tasks = append(tasks, chromedp.AttributeValue(extractor.Selector, extractor.Attribute, &element, nil, chromedp.ByQuery))
			}
			results[extractor.Name] = element
		}
	}

	return tasks
}