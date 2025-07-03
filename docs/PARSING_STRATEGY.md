# Email Parsing Strategy for Tracking Numbers

## Overview
The parsing strategy combines three complementary approaches:
1. **Regex-based pattern matching** leveraging existing carrier validation patterns
2. **LLM-powered semantic analysis** for complex email formats and edge cases
3. **Hybrid validation** using both methods for maximum accuracy

This multi-method approach ensures high precision while handling the wide variety of email formats from carriers and vendors.

## Dual-Track Parsing Architecture

### Track 1: Pattern-Based Extraction (Fast Path)
Traditional regex-based parsing for well-structured emails with clear patterns.

### Track 2: LLM-Based Extraction (Smart Path)  
AI-powered parsing for complex, unstructured, or ambiguous email content.

### Track 3: Hybrid Validation
Cross-validation and confidence scoring using both methods.

## Multi-Stage Parsing Approach

### Stage 1: Email Content Preprocessing
```go
type EmailContent struct {
    PlainText string
    HTMLText  string
    Subject   string
    From      string
    Headers   map[string]string
}

func PreprocessEmail(msg *EmailMessage) *EmailContent {
    // 1. Extract both HTML and plain text content
    // 2. HTML-to-text conversion for HTML emails
    // 3. Remove common email signatures and footers
    // 4. Normalize whitespace and line endings
    // 5. Extract structured data from headers
}
```

**Key Operations**:
- **HTML Processing**: Convert HTML to plain text while preserving structure
- **Content Cleaning**: Remove boilerplate text, signatures, unsubscribe links
- **Text Normalization**: Standardize spacing, remove non-printable characters
- **Multi-part Handling**: Process both HTML and plain text versions

### Stage 2: Carrier Identification
```go
type CarrierHint struct {
    Carrier    string
    Confidence float64
    Source     string // "sender", "subject", "content"
}

func IdentifyCarrier(content *EmailContent) []CarrierHint {
    hints := []CarrierHint{}
    
    // Analyze sender domain
    if strings.Contains(content.From, "@ups.com") {
        hints = append(hints, CarrierHint{"ups", 0.9, "sender"})
    }
    
    // Analyze subject patterns
    if strings.Contains(content.Subject, "UPS") {
        hints = append(hints, CarrierHint{"ups", 0.7, "subject"})
    }
    
    // Analyze content keywords
    // ...
    
    return hints
}
```

**Identification Methods**:
1. **Sender Domain Analysis**:
   - `@ups.com`, `@usps.com`, `@fedex.com`, `@dhl.com`
   - Third-party senders: `@amazon.com`, `@shopify.com`
   - Confidence: 90% for direct carrier domains, 60% for vendors

2. **Subject Line Patterns**:
   - Keywords: "UPS", "FedEx", "USPS", "DHL", "tracking", "shipment"
   - Confidence: 70% for carrier names, 40% for generic terms

3. **Content Keywords**:
   - Carrier-specific terminology and branding
   - Service names: "Priority Mail", "Ground", "Express"
   - Confidence: 50-60% based on frequency and context

### Stage 3: Candidate Extraction
```go
type TrackingCandidate struct {
    Text       string
    Position   int
    Context    string
    Carrier    string
    Confidence float64
}

func ExtractCandidates(content *EmailContent, hints []CarrierHint) []TrackingCandidate {
    candidates := []TrackingCandidate{}
    
    // Apply carrier-specific extraction patterns
    for _, hint := range hints {
        switch hint.Carrier {
        case "ups":
            candidates = append(candidates, extractUPSCandidates(content)...)
        case "usps":
            candidates = append(candidates, extractUSPSCandidates(content)...)
        // ...
        }
    }
    
    // Apply generic number extraction as fallback
    candidates = append(candidates, extractGenericCandidates(content)...)
    
    return candidates
}
```

**Extraction Patterns**:

#### UPS Extraction
```go
func extractUPSCandidates(content *EmailContent) []TrackingCandidate {
    patterns := []struct{
        regex   *regexp.Regexp
        context string
    }{
        // Direct pattern matching
        {regexp.MustCompile(`\b1Z[A-Z0-9]{6}\d{2}\d{7}\b`), "direct"},
        
        // Context-aware patterns
        {regexp.MustCompile(`(?i)tracking\s*(?:number|#)?\s*:?\s*([1Z][A-Z0-9\s]{15,20})`), "labeled"},
        {regexp.MustCompile(`(?i)shipment\s*(?:id|number)?\s*:?\s*([1Z][A-Z0-9\s]{15,20})`), "labeled"},
        
        // Table/structured data patterns
        {regexp.MustCompile(`<td[^>]*>([1Z][A-Z0-9\s]{15,20})</td>`), "table"},
    }
    
    // Apply patterns and score candidates
}
```

#### USPS Extraction
```go
func extractUSPSCandidates(content *EmailContent) []TrackingCandidate {
    patterns := []struct{
        regex   *regexp.Regexp
        format  string
    }{
        // Priority Mail patterns
        {regexp.MustCompile(`\b94\d{20}\b`), "priority_mail"},
        {regexp.MustCompile(`\b93\d{20}\b`), "signature_confirmation"},
        {regexp.MustCompile(`\b92\d{20}\b`), "certified_mail"},
        {regexp.MustCompile(`\b91\d{20}\b`), "signature_confirmation"},
        
        // International patterns
        {regexp.MustCompile(`\b[A-Z]{2}\d{9}US\b`), "international"},
        {regexp.MustCompile(`\b(LC|LK|EA|CP|RA|RB|RC|RD)\d{9}US\b`), "international_specific"},
        
        // Certified mail
        {regexp.MustCompile(`\b7\d{19}\b`), "certified_mail"},
        
        // Context-aware extraction
        {regexp.MustCompile(`(?i)tracking\s*(?:number|#)?\s*:?\s*([94][0-9\s]{20,25})`), "labeled_priority"},
        {regexp.MustCompile(`(?i)tracking\s*(?:number|#)?\s*:?\s*([A-Z]{2}[0-9]{9}US)`), "labeled_intl"},
    }
}
```

#### FedEx Extraction
```go
func extractFedExCandidates(content *EmailContent) []TrackingCandidate {
    // FedEx uses pure numeric patterns with specific lengths
    patterns := []struct{
        regex  *regexp.Regexp
        length int
    }{
        {regexp.MustCompile(`\b\d{12}\b`), 12},        // Express
        {regexp.MustCompile(`\b\d{14}\b`), 14},        // Ground
        {regexp.MustCompile(`\b\d{15}\b`), 15},        // Ground
        {regexp.MustCompile(`\b\d{18}\b`), 18},        // Ground
        {regexp.MustCompile(`\b\d{20}\b`), 20},        // Ground
        {regexp.MustCompile(`\b\d{22}\b`), 22},        // Ground
    }
    
    // Context-aware patterns
    contextPatterns := []string{
        `(?i)tracking\s*(?:number|#)?\s*:?\s*(\d{12,22})`,
        `(?i)shipment\s*(?:id|number)?\s*:?\s*(\d{12,22})`,
        `(?i)fedex\s*(?:tracking)?\s*:?\s*(\d{12,22})`,
    }
}
```

#### DHL Extraction
```go
func extractDHLCandidates(content *EmailContent) []TrackingCandidate {
    // DHL uses 10-11 digit numbers (most ambiguous)
    patterns := []struct{
        regex   *regexp.Regexp
        context string
    }{
        // Direct patterns (lower confidence due to ambiguity)
        {regexp.MustCompile(`\b\d{10}\b`), "direct_10"},
        {regexp.MustCompile(`\b\d{11}\b`), "direct_11"},
        
        // Context-required patterns (higher confidence)
        {regexp.MustCompile(`(?i)dhl\s*(?:tracking)?\s*:?\s*(\d{10,11})`), "labeled_dhl"},
        {regexp.MustCompile(`(?i)tracking\s*(?:number|#)?\s*:?\s*(\d{10,11})`), "labeled_generic"},
    }
}
```

### Stage 4: Validation and Scoring
```go
func ValidateAndScore(candidates []TrackingCandidate) []TrackingInfo {
    validated := []TrackingInfo{}
    
    for _, candidate := range candidates {
        // Apply existing carrier validation
        for _, carrierCode := range []string{"ups", "usps", "fedex", "dhl"} {
            client := carriers.GetClient(carrierCode)
            if client.ValidateTrackingNumber(candidate.Text) {
                info := TrackingInfo{
                    Number:      candidate.Text,
                    Carrier:     carrierCode,
                    Confidence:  calculateConfidence(candidate),
                    Description: extractDescription(candidate),
                }
                validated = append(validated, info)
            }
        }
    }
    
    // Sort by confidence score
    sort.Slice(validated, func(i, j int) bool {
        return validated[i].Confidence > validated[j].Confidence
    })
    
    return validated
}
```

**Confidence Scoring Algorithm**:
```go
func calculateConfidence(candidate TrackingCandidate) float64 {
    score := 0.0
    
    // Base score from carrier hint
    score += candidate.CarrierHint * 0.4
    
    // Context scoring
    switch candidate.Context {
    case "labeled":           // "Tracking Number: 1Z..."
        score += 0.4
    case "table":            // Structured HTML table
        score += 0.3
    case "direct":           // Pattern match in text
        score += 0.2
    }
    
    // Position scoring (earlier in email = higher confidence)
    if candidate.Position < 1000 {  // First 1000 characters
        score += 0.1
    }
    
    // Length and format scoring
    if candidate.ValidatesCleanly {
        score += 0.1
    }
    
    return min(score, 1.0)
}
```

### Stage 5: Metadata Extraction
```go
type ShipmentMetadata struct {
    Description     string
    ExpectedDelivery *time.Time
    Service         string
    Recipient       string
    Vendor          string
}

func ExtractMetadata(content *EmailContent, tracking TrackingInfo) ShipmentMetadata {
    metadata := ShipmentMetadata{}
    
    // Extract description from email subject or content
    metadata.Description = extractDescription(content, tracking)
    
    // Look for delivery dates
    metadata.ExpectedDelivery = extractDeliveryDate(content)
    
    // Identify vendor/sender
    metadata.Vendor = extractVendor(content)
    
    return metadata
}
```

**Description Extraction Strategies**:
1. **Email Subject Analysis**: Remove carrier names, extract product/order info
2. **Structured Data**: Look for item names in tables or lists
3. **Order Numbers**: Extract and use order/reference numbers
4. **Vendor Information**: Use sender information for context

**Delivery Date Extraction**:
```go
func extractDeliveryDate(content *EmailContent) *time.Time {
    patterns := []*regexp.Regexp{
        regexp.MustCompile(`(?i)delivery\s*(?:date|by)?\s*:?\s*(\w+,?\s*\w+\s+\d{1,2},?\s*\d{4})`),
        regexp.MustCompile(`(?i)expected\s*(?:delivery)?\s*:?\s*(\d{1,2}\/\d{1,2}\/\d{4})`),
        regexp.MustCompile(`(?i)arrives?\s*(?:by)?\s*:?\s*(\w+\s+\d{1,2})`),
    }
    
    for _, pattern := range patterns {
        if matches := pattern.FindStringSubmatch(content.PlainText); len(matches) > 1 {
            if date, err := parseFlexibleDate(matches[1]); err == nil {
                return &date
            }
        }
    }
    return nil
}
```

## LLM-Based Parsing (Smart Path)

### Stage 6: LLM Content Analysis
```go
type LLMExtractor struct {
    client LLMClient
    config LLMConfig
}

type LLMParsingRequest struct {
    EmailContent string
    From         string
    Subject      string
    MaxTokens    int
    Temperature  float64
}

type LLMParsingResponse struct {
    TrackingNumbers []LLMTrackingInfo `json:"tracking_numbers"`
    Confidence      float64           `json:"confidence"`
    Reasoning       string            `json:"reasoning"`
}

type LLMTrackingInfo struct {
    Number      string  `json:"number"`
    Carrier     string  `json:"carrier"`
    Description string  `json:"description"`
    Confidence  float64 `json:"confidence"`
    Context     string  `json:"context"`
}
```

### LLM Prompt Engineering
```go
const TRACKING_EXTRACTION_PROMPT = `You are an expert at extracting shipping information from emails. Analyze the following email and extract any tracking numbers, carriers, and item descriptions.

Email From: {{.From}}
Email Subject: {{.Subject}}

Email Content:
{{.EmailContent}}

Please extract the following information and respond in JSON format:

{
  "tracking_numbers": [
    {
      "number": "the exact tracking number (clean, no spaces)",
      "carrier": "ups|usps|fedex|dhl|unknown",
      "description": "brief description of the item being shipped",
      "confidence": 0.95,
      "context": "where/how you found this in the email"
    }
  ],
  "confidence": 0.90,
  "reasoning": "brief explanation of your analysis"
}

Important guidelines:
1. Only extract legitimate tracking numbers - ignore order numbers, phone numbers, etc.
2. Tracking number formats:
   - UPS: starts with "1Z" followed by 16 alphanumeric characters
   - USPS: various formats (20-22 digits starting with 94/93/92/91, or letters ending in "US")
   - FedEx: 12, 14, 15, 18, 20, or 22 digits only
   - DHL: 10 or 11 digits only
3. For carrier, use the specific carrier code if identifiable, "unknown" if unclear
4. For description, extract item name/type from email content, not generic shipping terms
5. Set confidence based on how certain you are (0.0-1.0)
6. If no tracking numbers found, return empty array

Respond only with valid JSON.`

func (e *LLMExtractor) ExtractTracking(content *EmailContent) (*LLMParsingResponse, error) {
    prompt := renderTemplate(TRACKING_EXTRACTION_PROMPT, content)
    
    request := LLMParsingRequest{
        EmailContent: content.PlainText,
        From:         content.From,
        Subject:      content.Subject,
        MaxTokens:    1000,
        Temperature:  0.1, // Low temperature for consistent, factual extraction
    }
    
    response, err := e.client.Complete(prompt, request)
    if err != nil {
        return nil, fmt.Errorf("LLM extraction failed: %w", err)
    }
    
    var result LLMParsingResponse
    if err := json.Unmarshal([]byte(response), &result); err != nil {
        return nil, fmt.Errorf("failed to parse LLM response: %w", err)
    }
    
    // Validate extracted tracking numbers against carrier patterns
    for i := range result.TrackingNumbers {
        if !validateWithExistingCarriers(result.TrackingNumbers[i]) {
            result.TrackingNumbers[i].Confidence *= 0.5 // Reduce confidence for invalid format
        }
    }
    
    return &result, nil
}
```

### LLM Client Interface
```go
type LLMClient interface {
    Complete(prompt string, request LLMParsingRequest) (string, error)
    Health() error
}

// OpenAI GPT implementation
type OpenAIClient struct {
    apiKey    string
    model     string
    client    *http.Client
    baseURL   string
}

// Anthropic Claude implementation  
type AnthropicClient struct {
    apiKey  string
    model   string
    client  *http.Client
    baseURL string
}

// Local LLM implementation (Ollama, etc.)
type LocalLLMClient struct {
    endpoint string
    model    string
    client   *http.Client
}

func NewLLMClient(config LLMConfig) (LLMClient, error) {
    switch config.Provider {
    case "openai":
        return &OpenAIClient{
            apiKey:  config.APIKey,
            model:   config.Model,
            baseURL: "https://api.openai.com/v1",
            client:  &http.Client{Timeout: 30 * time.Second},
        }, nil
    case "anthropic":
        return &AnthropicClient{
            apiKey:  config.APIKey,
            model:   config.Model,
            baseURL: "https://api.anthropic.com",
            client:  &http.Client{Timeout: 30 * time.Second},
        }, nil
    case "local":
        return &LocalLLMClient{
            endpoint: config.Endpoint,
            model:    config.Model,
            client:   &http.Client{Timeout: 60 * time.Second},
        }, nil
    default:
        return nil, fmt.Errorf("unsupported LLM provider: %s", config.Provider)
    }
}
```

### Hybrid Parsing Strategy
```go
type HybridParser struct {
    regexExtractor *RegexExtractor
    llmExtractor   *LLMExtractor
    config         *ParsingConfig
}

func (p *HybridParser) ExtractTracking(content *EmailContent) ([]TrackingInfo, error) {
    var results []TrackingInfo
    
    // Strategy 1: Try regex-based extraction first (fast path)
    regexResults := p.regexExtractor.Extract(content)
    
    // Strategy 2: Use LLM for complex cases or validation
    shouldUseLLM := p.shouldUseLLM(regexResults, content)
    
    if shouldUseLLM {
        llmResponse, err := p.llmExtractor.ExtractTracking(content)
        if err != nil {
            log.Printf("LLM extraction failed, falling back to regex: %v", err)
            return regexResults, nil
        }
        
        // Merge and validate results
        results = p.mergeResults(regexResults, llmResponse)
    } else {
        results = regexResults
    }
    
    return p.validateAndScore(results), nil
}

func (p *HybridParser) shouldUseLLM(regexResults []TrackingInfo, content *EmailContent) bool {
    // Use LLM if:
    // 1. No regex results found
    if len(regexResults) == 0 {
        return true
    }
    
    // 2. Low confidence regex results
    maxConfidence := 0.0
    for _, result := range regexResults {
        if result.Confidence > maxConfidence {
            maxConfidence = result.Confidence
        }
    }
    if maxConfidence < 0.7 {
        return true
    }
    
    // 3. Complex email structure (lots of HTML, tables, etc.)
    if p.isComplexEmail(content) {
        return true
    }
    
    // 4. Unknown sender (not from known carrier domains)
    if !p.isKnownCarrierSender(content.From) {
        return true
    }
    
    return false
}

func (p *HybridParser) mergeResults(regexResults []TrackingInfo, llmResponse *LLMParsingResponse) []TrackingInfo {
    merged := make(map[string]*TrackingInfo)
    
    // Add regex results
    for _, result := range regexResults {
        key := result.Number + ":" + result.Carrier
        merged[key] = &result
    }
    
    // Add or enhance with LLM results
    for _, llmResult := range llmResponse.TrackingNumbers {
        trackingInfo := TrackingInfo{
            Number:      llmResult.Number,
            Carrier:     llmResult.Carrier,
            Description: llmResult.Description,
            Confidence:  llmResult.Confidence,
            Source:      "llm",
            Context:     llmResult.Context,
        }
        
        key := trackingInfo.Number + ":" + trackingInfo.Carrier
        
        if existing, found := merged[key]; found {
            // Merge information, taking best confidence and most complete description
            if trackingInfo.Confidence > existing.Confidence {
                existing.Confidence = trackingInfo.Confidence
            }
            if trackingInfo.Description != "" && existing.Description == "" {
                existing.Description = trackingInfo.Description
            }
            existing.Source = "hybrid"
        } else {
            merged[key] = &trackingInfo
        }
    }
    
    // Convert back to slice
    var results []TrackingInfo
    for _, info := range merged {
        results = append(results, *info)
    }
    
    return results
}
```

### LLM Configuration
```go
type LLMConfig struct {
    Provider    string        // "openai", "anthropic", "local"
    Model       string        // "gpt-4", "claude-3", "llama2", etc.
    APIKey      string        // API key for hosted services
    Endpoint    string        // For local LLMs
    MaxTokens   int           // Response length limit
    Temperature float64       // Creativity vs consistency (0.0-1.0)
    Timeout     time.Duration // Request timeout
    RetryCount  int           // Number of retries on failure
    Enabled     bool          // Enable/disable LLM parsing
}

// Environment variables for LLM configuration
LLM_PROVIDER=openai
LLM_MODEL=gpt-4
LLM_API_KEY=sk-...
LLM_ENABLED=true
LLM_MAX_TOKENS=1000
LLM_TEMPERATURE=0.1
LLM_TIMEOUT=30s
LLM_RETRY_COUNT=2

// For local LLMs
LLM_PROVIDER=local
LLM_ENDPOINT=http://localhost:11434/api/generate
LLM_MODEL=llama2
```

### Performance and Cost Optimization
```go
type LLMCacheManager struct {
    cache map[string]*LLMParsingResponse
    mutex sync.RWMutex
    ttl   time.Duration
}

func (c *LLMCacheManager) Get(content *EmailContent) *LLMParsingResponse {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    
    // Create cache key from email content hash
    key := hashEmailContent(content)
    
    if result, found := c.cache[key]; found {
        if time.Since(result.CachedAt) < c.ttl {
            return result
        }
        delete(c.cache, key) // Expired
    }
    
    return nil
}

func (c *LLMCacheManager) Set(content *EmailContent, response *LLMParsingResponse) {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    key := hashEmailContent(content)
    response.CachedAt = time.Now()
    c.cache[key] = response
}

// Cost optimization strategies:
// 1. Cache LLM responses by email content hash
// 2. Use regex first, LLM only when needed
// 3. Truncate very long emails before sending to LLM
// 4. Use cheaper models for simple extraction tasks
// 5. Batch multiple requests when possible
```

### Error Handling for LLM Integration
```go
func (e *LLMExtractor) ExtractWithFallback(content *EmailContent) ([]TrackingInfo, error) {
    // Try LLM extraction
    llmResponse, err := e.ExtractTracking(content)
    if err != nil {
        log.Printf("LLM extraction failed: %v, falling back to regex", err)
        
        // Fallback to regex-only parsing
        regexExtractor := NewRegexExtractor()
        return regexExtractor.Extract(content), nil
    }
    
    // Validate LLM response format
    if len(llmResponse.TrackingNumbers) == 0 {
        log.Printf("LLM returned no tracking numbers, trying regex fallback")
        regexExtractor := NewRegexExtractor()
        return regexExtractor.Extract(content), nil
    }
    
    // Convert LLM response to TrackingInfo format
    var results []TrackingInfo
    for _, llmTrack := range llmResponse.TrackingNumbers {
        results = append(results, TrackingInfo{
            Number:      llmTrack.Number,
            Carrier:     llmTrack.Carrier,
            Description: llmTrack.Description,
            Confidence:  llmTrack.Confidence,
            Source:      "llm",
        })
    }
    
    return results, nil
}
```

## Error Handling and Edge Cases

### False Positive Mitigation
1. **Cross-Validation**: Test extracted numbers against all carrier validators
2. **Context Requirements**: Require contextual clues for ambiguous patterns (DHL)
3. **Blacklisting**: Maintain list of common false positives (phone numbers, order IDs)
4. **Length Filtering**: Filter out suspiciously long/short number sequences

### Edge Case Handling
1. **Multiple Tracking Numbers**: Handle emails with multiple shipments
2. **Malformed Numbers**: Clean up spacing, formatting issues
3. **International Formats**: Handle country-specific variations
4. **Forwarded Emails**: Parse through email forwarding chains
5. **HTML Artifacts**: Handle HTML encoding, entities, broken formatting

### Performance Optimizations
1. **Early Termination**: Stop processing after finding high-confidence matches
2. **Regex Caching**: Compile and cache regex patterns
3. **Content Limits**: Process only first N characters for initial scanning
4. **Parallel Processing**: Process different carriers in parallel

## Testing Strategy

### Test Data Requirements
1. **Real Email Samples**: Collect emails from each carrier
2. **Edge Cases**: Malformed, forwarded, multi-part emails
3. **False Positives**: Emails with no tracking numbers
4. **Multiple Formats**: HTML, plain text, mixed content

### Validation Tests
```go
func TestTrackingExtractionAccuracy(t *testing.T) {
    testCases := []struct {
        emailFile       string
        expectedNumbers []string
        expectedCarrier string
        minConfidence   float64
    }{
        {"ups_standard.html", []string{"1Z999AA1234567890"}, "ups", 0.8},
        {"usps_priority.txt", []string{"9400111699000367046792"}, "usps", 0.8},
        {"fedex_ground.html", []string{"123456789012"}, "fedex", 0.7},
        {"amazon_multi.html", []string{"1Z999AA1234567890", "9400111699000367046792"}, "", 0.6},
    }
    
    for _, tc := range testCases {
        content := loadTestEmail(tc.emailFile)
        results := ExtractTrackingNumbers(content)
        
        assert.Len(t, results, len(tc.expectedNumbers))
        for i, result := range results {
            assert.Equal(t, tc.expectedNumbers[i], result.Number)
            assert.GreaterOrEqual(t, result.Confidence, tc.minConfidence)
        }
    }
}

func TestLLMExtractionAccuracy(t *testing.T) {
    if !isLLMAvailable() {
        t.Skip("LLM not available for testing")
    }
    
    testCases := []struct {
        emailFile         string
        expectedNumbers   []string
        expectedCarrier   string
        minConfidence     float64
        requiresLLM       bool
    }{
        // Complex emails that regex might miss
        {"shopify_embedded.html", []string{"1Z999AA1234567890"}, "ups", 0.8, true},
        {"forwarded_amazon.eml", []string{"9400111699000367046792"}, "usps", 0.7, true},
        {"multilingual_dhl.html", []string{"1234567890"}, "dhl", 0.6, true},
        
        // Edge cases
        {"broken_html.html", []string{"123456789012"}, "fedex", 0.5, true},
        {"image_only_email.html", []string{}, "", 0.0, true},
    }
    
    llmExtractor := NewLLMExtractor(testLLMConfig())
    
    for _, tc := range testCases {
        content := loadTestEmail(tc.emailFile)
        results, err := llmExtractor.ExtractWithFallback(content)
        
        assert.NoError(t, err)
        if len(tc.expectedNumbers) > 0 {
            assert.Len(t, results, len(tc.expectedNumbers))
            for i, result := range results {
                assert.Equal(t, tc.expectedNumbers[i], result.Number)
                assert.GreaterOrEqual(t, result.Confidence, tc.minConfidence)
            }
        } else {
            assert.Empty(t, results)
        }
    }
}

func TestHybridParsingComparison(t *testing.T) {
    testEmails := []string{
        "ups_standard.html",
        "complex_shopify.html", 
        "forwarded_multi.eml",
        "dhl_german.html",
    }
    
    regexParser := NewRegexExtractor()
    llmParser := NewLLMExtractor(testLLMConfig())
    hybridParser := NewHybridParser(regexParser, llmParser, defaultConfig())
    
    for _, emailFile := range testEmails {
        content := loadTestEmail(emailFile)
        
        regexResults := regexParser.Extract(content)
        llmResults, _ := llmParser.ExtractWithFallback(content)
        hybridResults, _ := hybridParser.ExtractTracking(content)
        
        // Log comparison for analysis
        t.Logf("Email: %s", emailFile)
        t.Logf("  Regex: %d results", len(regexResults))
        t.Logf("  LLM: %d results", len(llmResults))
        t.Logf("  Hybrid: %d results", len(hybridResults))
        
        // Hybrid should perform at least as well as the better of the two
        assert.GreaterOrEqual(t, len(hybridResults), 
            max(len(regexResults), len(llmResults)))
    }
}
```

## Advanced LLM Techniques

### Few-Shot Learning Examples
```go
const FEW_SHOT_EXAMPLES = `Here are examples of successful tracking number extraction:

Example 1:
Email: "Your UPS package 1Z999AA1234567890 containing iPhone 15 Pro will arrive tomorrow"
Output: {"tracking_numbers": [{"number": "1Z999AA1234567890", "carrier": "ups", "description": "iPhone 15 Pro", "confidence": 0.95}]}

Example 2: 
Email: "USPS tracking 9400111699000367046792 for your Amazon order of coffee beans"
Output: {"tracking_numbers": [{"number": "9400111699000367046792", "carrier": "usps", "description": "coffee beans", "confidence": 0.90}]}

Example 3:
Email: "Order #12345 shipped via FedEx 123456789012 - Running shoes size 10"
Output: {"tracking_numbers": [{"number": "123456789012", "carrier": "fedex", "description": "Running shoes size 10", "confidence": 0.85}]}

Now analyze this email:`

func (e *LLMExtractor) ExtractWithFewShot(content *EmailContent) (*LLMParsingResponse, error) {
    prompt := FEW_SHOT_EXAMPLES + "\n\n" + renderTemplate(TRACKING_EXTRACTION_PROMPT, content)
    // ... rest of extraction logic
}
```

### Chain-of-Thought Prompting
```go
const COT_PROMPT = `Let me analyze this email step by step:

1. First, I'll identify if this is a shipping-related email
2. Then look for tracking number patterns
3. Determine the carrier based on format and context
4. Extract item description from the email content
5. Assign confidence based on clarity of information

Email to analyze:
{{.EmailContent}}

Step 1 - Is this shipping-related?
[LLM reasoning...]

Step 2 - Looking for tracking patterns...
[LLM analysis...]

Final extraction:`

// Use for complex emails where reasoning is important
```

### Multi-Model Validation
```go
type MultiModelExtractor struct {
    models []LLMClient
    voter  ConsensusVoter
}

func (m *MultiModelExtractor) ExtractWithConsensus(content *EmailContent) (*LLMParsingResponse, error) {
    results := make([]*LLMParsingResponse, len(m.models))
    
    // Query multiple models
    for i, model := range m.models {
        result, err := model.ExtractTracking(content)
        if err != nil {
            continue // Skip failed models
        }
        results[i] = result
    }
    
    // Use consensus voting to determine final result
    return m.voter.Vote(results), nil
}
```

## Benefits of LLM Integration

### **1. Superior Accuracy for Complex Emails**
- **Unstructured content**: Handles emails without clear patterns
- **Context understanding**: Distinguishes tracking numbers from other numbers
- **Semantic analysis**: Understands meaning beyond pattern matching

### **2. Better Description Extraction**
- **Natural language processing**: Extracts meaningful item descriptions
- **Context awareness**: Links tracking numbers to specific items
- **Multi-language support**: Handles international emails

### **3. Adaptability**
- **New carrier formats**: Learns from examples without code changes
- **Vendor variations**: Adapts to new email formats automatically
- **Edge case handling**: Processes unusual email structures

### **4. Cost-Effective Implementation**
- **Intelligent fallback**: Uses LLM only when regex fails
- **Caching**: Avoids duplicate API calls
- **Local LLM option**: Reduces ongoing costs

### **5. Validation and Quality Assurance**
- **Cross-validation**: Checks LLM results against regex patterns
- **Confidence scoring**: Provides reliability metrics
- **Hybrid approach**: Combines best of both methods

This enhanced parsing strategy provides a robust, intelligent system that can handle the full spectrum of shipping email formats while maintaining performance and cost efficiency.