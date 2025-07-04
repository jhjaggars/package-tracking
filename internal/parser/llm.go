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

// buildPrompt creates a prompt for tracking number extraction with merchant and description
func (l *LocalLLMExtractor) buildPrompt(content *email.EmailContent) string {
	prompt := fmt.Sprintf(`Extract shipping tracking numbers, product descriptions, and merchant information from this email. Return ONLY a JSON response.

Email From: %s
Subject: %s
Content: %s

EXAMPLES:
Email: "Your Apple iPhone 15 Pro 256GB Space Black has shipped from Amazon. Tracking: 1Z999AA1234567890"
Output: {
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

Email: "Order #12345 from Best Buy has shipped. Your Samsung Galaxy S24 Ultra tracking number is 9400123456789012345"
Output: {
  "tracking_numbers": [
    {
      "number": "9400123456789012345",
      "carrier": "usps",
      "confidence": 0.90,
      "description": "Samsung Galaxy S24 Ultra",
      "merchant": "Best Buy"
    }
  ]
}

Find tracking numbers for these carriers:
- UPS: Format like 1Z999AA1234567890 (starts with 1Z, 18 characters)  
- USPS: 20-22 digits, often starts with 94, 92, 93, 82
- FedEx: 12 digits or 15 digits starting with 96
- DHL: 10-11 digits

Extract:
1. Tracking numbers (highest priority)
2. Product descriptions (what was shipped)
3. Merchant/retailer information (who sent it)

Return JSON format:
{
  "tracking_numbers": [
    {
      "number": "tracking_number_here",
      "carrier": "ups|usps|fedex|dhl",
      "confidence": 0.95,
      "description": "product description here",
      "merchant": "merchant name here"
    }
  ]
}

If no tracking numbers found, return: {"tracking_numbers": []}`, 
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

// parseResponse parses the LLM JSON response into TrackingInfo
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
			})
		}
	}

	return results, nil
}

// NewLLMExtractor creates an appropriate LLM extractor based on configuration
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