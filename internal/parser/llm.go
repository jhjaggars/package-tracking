package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"package-tracking/internal/email"
)

// LLMExtractor defines the interface for LLM-based tracking number extraction
//
// DEPRECATED: This interface is deprecated and should not be used for new development.
// Use SimplifiedTrackingExtractorInterface for pattern-based tracking extraction and
// SimplifiedDescriptionExtractorInterface for LLM-based description extraction instead.
// See DEPRECATED.md for migration guide.
type LLMExtractor interface {
	// Extract tracking numbers using LLM analysis
	Extract(content *email.EmailContent) ([]email.TrackingInfo, error)
	
	// HealthCheck verifies LLM service is available
	HealthCheck() error
	
	// IsEnabled returns whether LLM extraction is enabled
	IsEnabled() bool
}

// NoOpLLMExtractor is a no-operation implementation
type NoOpLLMExtractor struct{}

// NewNoOpLLMExtractor creates a no-op LLM extractor
func NewNoOpLLMExtractor() LLMExtractor {
	return &NoOpLLMExtractor{}
}

// Extract returns empty results
func (n *NoOpLLMExtractor) Extract(content *email.EmailContent) ([]email.TrackingInfo, error) {
	return []email.TrackingInfo{}, nil
}

// HealthCheck always returns nil
func (n *NoOpLLMExtractor) HealthCheck() error {
	return nil
}

// IsEnabled returns false
func (n *NoOpLLMExtractor) IsEnabled() bool {
	return false
}

// LLMConfig holds configuration for LLM extractors
//
// DEPRECATED: This type is deprecated. Use SimplifiedLLMConfig instead.
// See DEPRECATED.md for migration guide.
type LLMConfig struct {
	Provider    string
	Model       string
	APIKey      string
	Endpoint    string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
	RetryCount  int
	Enabled     bool
}

// LocalLLMExtractor implements LLM extraction using local endpoints (e.g., Ollama)
//
// DEPRECATED: This type is deprecated. Use OllamaLLMClient instead.
// See DEPRECATED.md for migration guide.
type LocalLLMExtractor struct {
	config     *LLMConfig
	httpClient *http.Client
}

// NewLocalLLMExtractor creates a new local LLM extractor
func NewLocalLLMExtractor(config *LLMConfig) *LocalLLMExtractor {
	return &LocalLLMExtractor{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Extract uses the local LLM to extract tracking numbers
func (l *LocalLLMExtractor) Extract(content *email.EmailContent) ([]email.TrackingInfo, error) {
	if !l.config.Enabled {
		return []email.TrackingInfo{}, nil
	}

	// Prepare the prompt for tracking number extraction
	prompt := l.buildPrompt(content)
	
	// Call the local LLM API
	response, err := l.callLLM(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM API call failed: %w", err)
	}

	// Parse the response
	trackingInfo, err := l.parseResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return trackingInfo, nil
}

// HealthCheck verifies the LLM service is available
func (l *LocalLLMExtractor) HealthCheck() error {
	if !l.config.Enabled {
		return nil
	}
	
	// Simple health check - try a minimal request
	testPrompt := "Test health check. Respond with: OK"
	_, err := l.callLLM(testPrompt)
	return err
}

// IsEnabled returns whether LLM extraction is enabled
func (l *LocalLLMExtractor) IsEnabled() bool {
	return l.config.Enabled
}

// buildPrompt creates a prompt for tracking number extraction (legacy method)
func (l *LocalLLMExtractor) buildPrompt(content *email.EmailContent) string {
	prompt := fmt.Sprintf(`Extract shipping tracking numbers from this email. Return ONLY a JSON response.

Email From: %s
Subject: %s
Content: %s

Find tracking numbers for these carriers:
- UPS: Format like 1Z999AA1234567890 (starts with 1Z, 18 characters)  
- USPS: 20-22 digits, often starts with 94, 92, 93, 82
- FedEx: 12 digits or 15 digits starting with 96
- DHL: 10-11 digits
- Amazon Logistics: Format like TBA123456789000 (starts with TBA, 15 characters)
- Amazon Order: Format like 123-4567890-1234567 (3-7-7 digit pattern with dashes)

Return JSON format:
{
  "tracking_numbers": [
    {
      "number": "tracking_number_here",
      "carrier": "ups|usps|fedex|dhl|amazon",
      "confidence": 0.95
    }
  ]
}

If no tracking numbers found, return: {"tracking_numbers": []}`, 
		content.From, content.Subject, l.truncateContent(content.PlainText))
		
	return prompt
}

// buildEnhancedPrompt creates an enhanced prompt for tracking number, merchant, and description extraction
func (l *LocalLLMExtractor) buildEnhancedPrompt(content *email.EmailContent) string {
	prompt := fmt.Sprintf(`Extract shipping tracking numbers, product descriptions, and merchant information from this email. Return ONLY a JSON response.

Email From: %s
Subject: %s
Content: %s

Task: Find tracking numbers and extract meaningful product descriptions and merchant information.

Tracking number formats:
- UPS: Format like 1Z999AA1234567890 (starts with 1Z, 18 characters)  
- USPS: 20-22 digits, often starts with 94, 92, 93, 82
- FedEx: 12 digits or 15 digits starting with 96
- DHL: 10-11 digits
- Amazon Logistics: Format like TBA123456789000 (starts with TBA, 15 characters)
- Amazon Order: Format like 123-4567890-1234567 (3-7-7 digit pattern with dashes)

For each tracking number found:
1. Extract the tracking number and identify the carrier
2. Extract product description from the email content (what was purchased)
3. Extract merchant/retailer information (who sold it)
4. Assign confidence score (0.0-1.0)

Example 1:
From: noreply@amazon.com
Subject: Your Amazon order has shipped
Content: Your order of Apple iPhone 15 Pro 256GB Space Black has been shipped via UPS. Tracking number: 1Z999AA1234567890

Expected output:
{
  "tracking_numbers": [
    {
      "number": "1Z999AA1234567890",
      "carrier": "ups",
      "confidence": 0.95,
      "description": "Apple iPhone 15 Pro 256GB Space Black",
      "merchant": "Amazon"
    }
  ]
}

Example 2:
From: orders@shopify.com
Subject: Your TechStore order is on its way
Content: Your order containing Dell XPS 13 Laptop and Logitech MX Master 3 Mouse has been shipped via FedEx. Tracking: 961234567890. From TechStore.

Expected output:
{
  "tracking_numbers": [
    {
      "number": "961234567890",
      "carrier": "fedex",
      "confidence": 0.9,
      "description": "Dell XPS 13 Laptop, Logitech MX Master 3 Mouse",
      "merchant": "TechStore"
    }
  ]
}

Example 3:
From: support@bestbuy.com
Subject: Order Confirmation - Nike Air Max 270
Content: Thank you for your order! Your Nike Air Max 270 sneakers in size 10 have been shipped via USPS. Tracking number: 9405511206213414325732.

Expected output:
{
  "tracking_numbers": [
    {
      "number": "9405511206213414325732",
      "carrier": "usps",
      "confidence": 0.92,
      "description": "Nike Air Max 270 sneakers size 10",
      "merchant": "Best Buy"
    }
  ]
}

Example 4 (Amazon Logistics):
From: shipment-tracking@amazon.com
Subject: Your package has been shipped
Content: Your Amazon order #123-4567890-1234567 containing Echo Dot (5th Gen) Smart Speaker has been shipped via Amazon Logistics. Track your package: TBA123456789000

Expected output:
{
  "tracking_numbers": [
    {
      "number": "TBA123456789000",
      "carrier": "amazon",
      "confidence": 0.95,
      "description": "Echo Dot (5th Gen) Smart Speaker",
      "merchant": "Amazon"
    }
  ]
}

Example 5 (Amazon Order Number):
From: auto-confirm@amazon.com
Subject: Your Amazon.com order of Fire TV Stick has shipped
Content: Hello, your order 111-2233445-6677889 of Amazon Fire TV Stick 4K Max with Alexa Voice Remote has been shipped. You can track your order using this number.

Expected output:
{
  "tracking_numbers": [
    {
      "number": "111-2233445-6677889",
      "carrier": "amazon",
      "confidence": 0.90,
      "description": "Amazon Fire TV Stick 4K Max with Alexa Voice Remote",
      "merchant": "Amazon"
    }
  ]
}

Instructions:
- Extract specific product names, models, colors, sizes when available
- Identify merchant from sender domain, subject line, or content
- Use confidence scores: 0.9+ for clear matches, 0.7-0.9 for good matches, 0.5-0.7 for uncertain matches
- If no tracking numbers found, return: {"tracking_numbers": []}
- If tracking number found but no product/merchant info, use generic descriptions

Return JSON format:
{
  "tracking_numbers": [
    {
      "number": "tracking_number_here",
      "carrier": "ups|usps|fedex|dhl|amazon",
      "confidence": 0.95,
      "description": "specific product description",
      "merchant": "merchant/retailer name"
    }
  ]
}`, 
		content.From, content.Subject, l.truncateContent(content.PlainText))
		
	return prompt
}

// truncateContent limits content size for API efficiency
func (l *LocalLLMExtractor) truncateContent(content string) string {
	maxLength := 2000 // Reasonable limit for tracking extraction
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength] + "..."
}

// callLLM makes the API call to the local LLM endpoint
func (l *LocalLLMExtractor) callLLM(prompt string) (string, error) {
	// Prepare request body for Ollama-style API
	requestBody := map[string]interface{}{
		"model":       l.config.Model,
		"prompt":      prompt,
		"stream":      false,
		"temperature": l.config.Temperature,
		"max_tokens":  l.config.MaxTokens,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", l.config.Endpoint+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if l.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+l.config.APIKey)
	}

	// Make the request
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse Ollama response
	var ollamaResp struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return ollamaResp.Response, nil
}

// parseResponse parses the LLM JSON response into TrackingInfo (legacy method)
func (l *LocalLLMExtractor) parseResponse(response string) ([]email.TrackingInfo, error) {
	// Clean up the response (remove any markdown formatting)
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
	}
	response = strings.TrimSpace(response)

	// Parse JSON response
	var parsed struct {
		TrackingNumbers []struct {
			Number     string  `json:"number"`
			Carrier    string  `json:"carrier"`
			Confidence float64 `json:"confidence"`
		} `json:"tracking_numbers"`
	}

	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert to TrackingInfo
	var results []email.TrackingInfo
	for _, item := range parsed.TrackingNumbers {
		if item.Number != "" && item.Carrier != "" {
			results = append(results, email.TrackingInfo{
				Number:     item.Number,
				Carrier:    strings.ToLower(item.Carrier),
				Confidence: item.Confidence,
				Source:     "llm",
			})
		}
	}

	return results, nil
}

// parseEnhancedResponse parses the enhanced LLM JSON response into TrackingInfo with merchant and description
func (l *LocalLLMExtractor) parseEnhancedResponse(response string) ([]email.TrackingInfo, error) {
	// Clean up the response (remove any markdown formatting)
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
	}
	response = strings.TrimSpace(response)

	// Parse JSON response with enhanced fields
	var parsed struct {
		TrackingNumbers []struct {
			Number      string  `json:"number"`
			Carrier     string  `json:"carrier"`
			Confidence  float64 `json:"confidence"`
			Description string  `json:"description"`
			Merchant    string  `json:"merchant"`
		} `json:"tracking_numbers"`
	}

	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert to TrackingInfo
	var results []email.TrackingInfo
	for _, item := range parsed.TrackingNumbers {
		if item.Number != "" && item.Carrier != "" {
			results = append(results, email.TrackingInfo{
				Number:      item.Number,
				Carrier:     strings.ToLower(item.Carrier),
				Description: item.Description,
				Merchant:    item.Merchant,
				Confidence:  item.Confidence,
				Source:      "llm",
				ExtractedAt: time.Now(),
			})
		}
	}

	return results, nil
}

// filterByConfidence filters tracking results based on confidence threshold
func (l *LocalLLMExtractor) filterByConfidence(results []email.TrackingInfo, minConfidence float64) []email.TrackingInfo {
	var filtered []email.TrackingInfo
	for _, result := range results {
		if result.Confidence >= minConfidence {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// NewLLMExtractor creates an appropriate LLM extractor based on configuration
//
// DEPRECATED: This function is deprecated. Use NewSimplifiedLLMClient instead.
// See DEPRECATED.md for migration guide.
func NewLLMExtractor(config *LLMConfig) LLMExtractor {
	if !config.Enabled {
		return NewNoOpLLMExtractor()
	}

	switch strings.ToLower(config.Provider) {
	case "local":
		return NewLocalLLMExtractor(config)
	case "openai":
		// TODO: Implement OpenAI extractor
		return NewNoOpLLMExtractor()
	case "anthropic":
		// TODO: Implement Anthropic extractor  
		return NewNoOpLLMExtractor()
	default:
		return NewNoOpLLMExtractor()
	}
}