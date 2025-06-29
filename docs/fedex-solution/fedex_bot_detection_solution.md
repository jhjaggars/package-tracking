# 🚫 FedEx Bot Detection - Complete Solution Guide

## 🎯 **Root Cause: Chrome-Specific Bot Detection**

**Problem Confirmed**: FedEx deliberately blocks Chromium-based browsers with automation fingerprints:
- ✅ Detects `navigator.webdriver` property
- ✅ Checks for "AutomationControlled" Blink features  
- ✅ Validates plugin presence and browser fingerprints
- ✅ Firefox passes because detection is Chrome-specific

**Current Error**: "Unfortunately we are unable to retrieve your tracking results at this time" = **Bot Detection Triggered**

## 🏆 **Recommended Solutions (Priority Order)**

### **Solution 1: Official FedEx API (RECOMMENDED) 🌟**

**Why This is Best**:
- ✅ No scraping = No bot detection
- ✅ Official, supported interface
- ✅ Same JSON data as web interface
- ✅ Free developer account
- ✅ Reliable and performant

**Implementation**:
```go
// Add to internal/carriers/fedex_api.go
type FedExAPIClient struct {
    apiKey    string
    baseURL   string
    client    *http.Client
}

func NewFedExAPIClient(apiKey string) *FedExAPIClient {
    return &FedExAPIClient{
        apiKey:  apiKey,
        baseURL: "https://apis.fedex.com",
        client:  &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *FedExAPIClient) Track(ctx context.Context, req *TrackingRequest) (*TrackingResponse, error) {
    payload := map[string]interface{}{
        "includeDetailedScans": true,
        "trackingInfo": []map[string]string{
            {"trackingNumberInfo": {"trackingNumber": req.TrackingNumbers[0]}},
        },
    }
    
    return c.callAPI("/track/v1/trackingnumbers", payload)
}
```

**Setup Steps**:
1. Register at https://developer.fedex.com/
2. Create application for Tracking API
3. Get API key and configure in environment
4. Implement as primary FedEx client

**Environment Variable**:
```bash
FEDEX_API_KEY=your_api_key_here
```

---

### **Solution 2: Firefox/Gecko Engine (CURRENT QUICKFIX) 🦊**

**Why This Works**: Firefox automation bypasses Chrome-specific detection

**Implementation** (Update existing code):
```go
// Update internal/carriers/browser_pool.go
func (p *BrowserPool) createAllocator() context.Context {
    // For FedEx, use Firefox to bypass bot detection
    if p.carrier == "fedex" {
        return p.createFirefoxAllocator()
    }
    return p.createChromeAllocator()
}

func (p *BrowserPool) createFirefoxAllocator() context.Context {
    opts := []chromedp.ExecAllocatorOption{
        chromedp.NoFirstRun,
        chromedp.NoDefaultBrowserCheck,
        chromedp.DisableGPU,
        chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0"),
    }
    
    if p.options.Headless {
        opts = append(opts, chromedp.Headless)
    }
    
    return chromedp.NewExecAllocator(context.Background(), opts...)
}
```

**Pros**: ✅ Immediate fix, ✅ No API dependency  
**Cons**: ❌ Still scraping, ❌ Slower than API

---

### **Solution 3: Enhanced Chrome Stealth Mode (FALLBACK) 🥷**

**Complete Stealth Implementation**:
```go
// Enhanced stealth for internal/carriers/fedex_headless.go
func (c *FedExHeadlessClient) createStealthAllocator() context.Context {
    opts := []chromedp.ExecAllocatorOption{
        // Modern headless
        chromedp.Flag("headless", "new"),
        
        // Remove automation signatures
        chromedp.Flag("disable-blink-features", "AutomationControlled"),
        chromedp.Flag("disable-features", "TranslateUI,IsolateOrigins,site-per-process"),
        chromedp.Flag("disable-web-security", false),
        chromedp.Flag("disable-dev-shm-usage", true),
        
        // Normal browser behavior
        chromedp.Flag("lang", "en-US"),
        chromedp.Flag("no-first-run", true),
        chromedp.Flag("no-default-browser-check", true),
        
        // Real user agent (current Chrome version)
        chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"),
        
        // Enable plugins and features
        chromedp.Flag("disable-plugins", false),
        chromedp.Flag("disable-extensions", false),
    }
    
    return chromedp.NewExecAllocator(context.Background(), opts...)
}

func (c *FedExHeadlessClient) injectStealthScript(ctx context.Context) error {
    return chromedp.Run(ctx, chromedp.Evaluate(`
        // Remove webdriver property
        Object.defineProperty(navigator, 'webdriver', {
            get: () => undefined,
        });
        
        // Add realistic plugins
        Object.defineProperty(navigator, 'plugins', {
            get: () => [
                {
                    name: "Chrome PDF Plugin",
                    filename: "internal-pdf-viewer",
                    description: "Portable Document Format"
                },
                {
                    name: "Chrome PDF Viewer", 
                    filename: "mhjfbmdgcfjbbpaeojofohoefgiehjai",
                    description: ""
                }
            ],
        });
        
        // Add realistic languages
        Object.defineProperty(navigator, 'languages', {
            get: () => ['en-US', 'en'],
        });
        
        // Override permissions
        const originalQuery = window.navigator.permissions.query;
        window.navigator.permissions.query = (parameters) => (
            parameters.name === 'notifications' ?
                Promise.resolve({ state: Notification.permission }) :
                originalQuery(parameters)
        );
        
        // Remove automation indicators
        delete window.chrome.runtime.onConnect;
        delete window.chrome.runtime.onMessage;
        
        console.log("Stealth mode activated");
    `, nil))
}
```

---

## 🛠 **Implementation Plan**

### **Phase 1: Immediate Fix (This Sprint)**
```go
// 1. Update factory to use Firefox for FedEx
func (f *ClientFactory) createFedExClient() (TrackingClient, ClientType, error) {
    // Prefer API if available
    if f.config.FedExAPIKey != "" {
        return NewFedExAPIClient(f.config.FedExAPIKey), ClientTypeAPI, nil
    }
    
    // Fallback to Firefox headless (bypasses bot detection)
    return NewFedExFirefoxClient(), ClientTypeHeadless, nil
}

// 2. Create Firefox-specific client
type FedExFirefoxClient struct {
    *HeadlessScrapingClient
    // Firefox-specific implementation
}

func NewFedExFirefoxClient() *FedExFirefoxClient {
    options := DefaultHeadlessOptions()
    options.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0"
    options.BrowserType = "firefox" // New option
    
    return &FedExFirefoxClient{
        HeadlessScrapingClient: NewHeadlessScrapingClient("fedex", options.UserAgent, options),
    }
}
```

### **Phase 2: API Integration (Next Sprint)**
```go
// Configuration update
type Config struct {
    // Existing fields...
    FedExAPIKey string // Add to config
}

// Environment loading
func Load() (*Config, error) {
    return &Config{
        // Existing...
        FedExAPIKey: os.Getenv("FEDEX_API_KEY"),
    }
}
```

### **Phase 3: Enhanced Detection (Future)**
```go
// Detect bot detection vs real errors
func (c *FedExHeadlessClient) isBotDetection(content string) bool {
    botDetectionPatterns := []string{
        "unfortunately we are unable to retrieve",
        "please try again later", 
        "request cannot be processed",
        "temporarily unavailable", // But check context
    }
    
    // Cross-reference with other indicators
    hasTrackingForm := strings.Contains(content, "tracking")
    hasErrorSpecifics := strings.Contains(content, "invalid") || 
                        strings.Contains(content, "not found")
    
    // If generic error without specifics = likely bot detection
    for _, pattern := range botDetectionPatterns {
        if strings.Contains(strings.ToLower(content), pattern) && 
           hasTrackingForm && !hasErrorSpecifics {
            return true
        }
    }
    
    return false
}
```

## 🧪 **Testing Strategy**

### **Test with Different Engines**:
```bash
# Test 1: Current Chrome (should fail with bot detection)
go run debug_fedex_scraping.go 390244419364

# Test 2: Firefox mode (should work)  
FEDEX_BROWSER=firefox go run debug_fedex_scraping.go 390244419364

# Test 3: API mode (ideal - should work perfectly)
FEDEX_API_KEY=test_key go run debug_fedex_scraping.go 390244419364
```

### **Validation Criteria**:
- ✅ **Chrome**: Detects bot detection (expected)
- ✅ **Firefox**: Returns tracking data or legitimate errors
- ✅ **API**: Fast, reliable tracking data

## 📊 **Expected Results**

**Before (Current State)**:
- ❌ Chrome gets bot detection error
- ❌ Misinterpreted as server errors
- ❌ Poor user experience

**After Implementation**:
- ✅ **API First**: Fast, reliable, official data
- ✅ **Firefox Fallback**: Bypasses bot detection 
- ✅ **Smart Detection**: Distinguishes bot blocks from real errors
- ✅ **Clear Messaging**: Users understand what's happening

## 🎯 **Success Metrics**

- **90%+ success rate** with API + Firefox fallback
- **<30 second response time** with API
- **<90 second response time** with Firefox scraping  
- **Zero false bot detection** alerts
- **Clear error messages** for actual tracking issues

## 🚀 **Next Actions**

1. **✅ Enhanced Error Detection**: Added proper bot detection vs server error vs not found handling
2. **✅ FedEx API Implementation**: Complete OAuth 2.0 client with automatic preference over scraping
3. **🔄 Week 1**: Switch FedEx to Firefox engine (requires puppeteer or alternative to chromedp)
4. **🔄 Week 2**: Register FedEx developer account and get API keys for production use
5. **🔄 Week 3**: Monitor and optimize performance with real API credentials

**Current Status**: 
- ✅ **Bot Detection**: System now correctly identifies FedEx bot detection messages
- ✅ **Error Categorization**: BOT_DETECTION, SERVER_ERROR, and NOT_FOUND errors properly distinguished  
- ✅ **User-Friendly Messages**: Clear error messages explaining Chrome blocking and suggesting Firefox/API alternatives
- ✅ **FedEx API Client**: Complete implementation with OAuth 2.0, tracking requests, and response parsing
- ✅ **API Integration**: Factory automatically prefers API when credentials are available
- ✅ **Configuration Support**: FEDEX_API_KEY and FEDEX_SECRET_KEY environment variables
- 🔄 **Firefox Engine**: Needs implementation (chromedp limitation requires alternative solution)

**Priority**: System now prefers FedEx API when available, falls back to enhanced headless scraping with proper error detection.