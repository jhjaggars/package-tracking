package carriers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

// SimpleBrowserPool implements a basic browser pool for managing chrome instances
type SimpleBrowserPool struct {
	config      *BrowserPoolConfig
	options     *HeadlessOptions
	instances   []*BrowserInstance
	mu          sync.RWMutex
	closed      bool
	cleanupDone chan struct{}
}

// ValidateChromeAvailable checks if Chrome/Chromium is available and working
func ValidateChromeAvailable() error {
	// Create a test allocator to check Chrome availability
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(),
		chromedp.Headless,
		chromedp.NoSandbox,
		chromedp.DisableGPU,
	)
	defer allocCancel()

	// Create browser context with short timeout
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Test Chrome availability with a simple operation
	testCtx, testCancel := context.WithTimeout(ctx, 10*time.Second)
	defer testCancel()

	err := chromedp.Run(testCtx, chromedp.Navigate("about:blank"))
	if err != nil {
		return fmt.Errorf("Chrome/Chromium not available or not working: %w", err)
	}

	return nil
}

// NewBrowserPool creates a new browser pool with the given configuration
func NewBrowserPool(config *BrowserPoolConfig, options *HeadlessOptions) *SimpleBrowserPool {
	if config == nil {
		config = DefaultBrowserPoolConfig()
	}
	if options == nil {
		options = DefaultHeadlessOptions()
	}

	pool := &SimpleBrowserPool{
		config:      config,
		options:     options,
		instances:   make([]*BrowserInstance, 0, config.MaxBrowsers),
		cleanupDone: make(chan struct{}),
	}

	// Start cleanup goroutine
	go pool.cleanupLoop()

	return pool
}

// Get retrieves an available browser instance from the pool
func (p *SimpleBrowserPool) Get(ctx context.Context) (*BrowserInstance, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, fmt.Errorf("browser pool is closed")
	}

	// Look for an idle instance
	for _, instance := range p.instances {
		if !instance.inUse {
			instance.inUse = true
			instance.lastUsed = time.Now()
			return instance, nil
		}
	}

	// No idle instances, create a new one if under limit
	if len(p.instances) < p.config.MaxBrowsers {
		instance, err := p.createInstance(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create browser instance: %w", err)
		}
		
		instance.inUse = true
		instance.lastUsed = time.Now()
		p.instances = append(p.instances, instance)
		return instance, nil
	}

	return nil, fmt.Errorf("browser pool exhausted: %d instances in use", len(p.instances))
}

// Put returns a browser instance to the pool
func (p *SimpleBrowserPool) Put(instance *BrowserInstance) error {
	if instance == nil {
		return fmt.Errorf("cannot return nil instance to pool")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		// Pool is closed, clean up the instance
		p.cleanupInstance(instance)
		return nil
	}

	// Mark as not in use
	instance.inUse = false
	instance.lastUsed = time.Now()

	return nil
}

// Close shuts down all browser instances in the pool
func (p *SimpleBrowserPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	// Clean up all instances
	for _, instance := range p.instances {
		p.cleanupInstance(instance)
	}
	p.instances = nil

	// Signal cleanup goroutine to stop
	close(p.cleanupDone)

	return nil
}

// Stats returns current pool statistics
func (p *SimpleBrowserPool) Stats() BrowserPoolStats {
	p.mu.RLock()
	
	active := 0
	idle := 0
	total := len(p.instances)

	// Make a copy of the slice to avoid potential race conditions
	instances := make([]*BrowserInstance, total)
	copy(instances, p.instances)
	
	p.mu.RUnlock()

	// Count active/idle without holding the lock
	for _, instance := range instances {
		if instance.inUse {
			active++
		} else {
			idle++
		}
	}

	return BrowserPoolStats{
		Active: active,
		Idle:   idle,
		Total:  total, // Use the snapshot total
	}
}

// createInstance creates a new browser instance
func (p *SimpleBrowserPool) createInstance(ctx context.Context) (*BrowserInstance, error) {
	// Create allocator context
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), p.buildAllocatorOptions()...)

	// Create browser context
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)

	// Test the browser by running a simple task
	err := chromedp.Run(browserCtx, chromedp.Navigate("about:blank"))
	if err != nil {
		browserCancel()
		allocCancel()
		return nil, fmt.Errorf("failed to initialize browser: %w", err)
	}

	return &BrowserInstance{
		ctx:         browserCtx,
		cancel:      browserCancel,
		allocCancel: allocCancel,
		allocator:   allocCtx,
		lastUsed:    time.Now(),
		inUse:       false,
	}, nil
}

// cleanupInstance properly shuts down a browser instance
func (p *SimpleBrowserPool) cleanupInstance(instance *BrowserInstance) {
	if instance.cancel != nil {
		instance.cancel()
	}
	if instance.allocCancel != nil {
		instance.allocCancel()
	}
}

// cleanupLoop periodically removes idle instances that have exceeded the idle timeout
func (p *SimpleBrowserPool) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanupIdleInstances()
		case <-p.cleanupDone:
			return
		}
	}
}

// cleanupIdleInstances removes idle instances that have exceeded the timeout
func (p *SimpleBrowserPool) cleanupIdleInstances() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	now := time.Now()
	idleCount := 0
	activeInstances := make([]*BrowserInstance, 0, len(p.instances))

	for _, instance := range p.instances {
		if instance.inUse {
			activeInstances = append(activeInstances, instance)
		} else {
			idleCount++
			// Keep instance if it's within idle timeout and we're under max idle limit
			if now.Sub(instance.lastUsed) < p.config.IdleTimeout && idleCount <= p.config.MaxIdleBrowsers {
				activeInstances = append(activeInstances, instance)
			} else {
				// Clean up expired idle instance
				p.cleanupInstance(instance)
			}
		}
	}

	p.instances = activeInstances
}

// buildAllocatorOptions builds Chrome allocator options based on configuration
func (p *SimpleBrowserPool) buildAllocatorOptions() []chromedp.ExecAllocatorOption {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.UserAgent(p.options.UserAgent),
		chromedp.WindowSize(int(p.options.ViewportWidth), int(p.options.ViewportHeight)),
		chromedp.NoSandbox, // Often needed in containerized environments
		chromedp.DisableGPU,
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
	}

	// Add headless option
	if p.options.Headless {
		opts = append(opts, chromedp.Headless)
	}

	// Disable images for performance
	if p.options.DisableImages {
		opts = append(opts, chromedp.Flag("blink-settings", "imagesEnabled=false"))
	}

	// Add debug options if enabled
	if p.options.DebugMode {
		opts = append(opts, chromedp.Flag("enable-logging", true))
		opts = append(opts, chromedp.Flag("log-level", "0"))
	}

	// Performance optimizations
	opts = append(opts,
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
	)

	return opts
}

// ExecuteWithBrowser executes a function with a browser instance from the pool
func (p *SimpleBrowserPool) ExecuteWithBrowser(ctx context.Context, fn func(context.Context) error) error {
	instance, err := p.Get(ctx)
	if err != nil {
		return err
	}
	defer p.Put(instance)

	// Create a timeout context that respects both parent and pool timeouts
	var opCtx context.Context
	var cancel context.CancelFunc
	
	// Use the shorter of parent context deadline or pool timeout
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		remaining := time.Until(deadline)
		if remaining < p.options.Timeout {
			opCtx, cancel = context.WithTimeout(instance.ctx, remaining)
		} else {
			opCtx, cancel = context.WithTimeout(instance.ctx, p.options.Timeout)
		}
	} else {
		opCtx, cancel = context.WithTimeout(instance.ctx, p.options.Timeout)
	}
	defer cancel()

	return fn(opCtx)
}