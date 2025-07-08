package parser

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// SimplifiedLLMConfig holds configuration for description-focused LLM clients
type SimplifiedLLMConfig struct {
	Provider    string        `json:"provider"`    // "openai", "anthropic", "ollama", "disabled"
	Model       string        `json:"model"`       // Model name (e.g., "llama3.2", "gpt-4", "claude-3-sonnet")
	APIKey      string        `json:"api_key"`     // API key for cloud providers
	Endpoint    string        `json:"endpoint"`    // API endpoint for local LLMs like Ollama
	MaxTokens   int           `json:"max_tokens"`  // Maximum response tokens
	Temperature float64       `json:"temperature"` // Creativity vs consistency (0.0-1.0)
	Timeout     time.Duration `json:"timeout"`     // Request timeout
	RetryCount  int           `json:"retry_count"` // Number of retries for failed requests
	Enabled     bool          `json:"enabled"`     // Enable/disable LLM extraction
}

// DefaultSimplifiedLLMConfig returns a default configuration
func DefaultSimplifiedLLMConfig() *SimplifiedLLMConfig {
	return &SimplifiedLLMConfig{
		Provider:    "disabled",
		Model:       "",
		APIKey:      "",
		Endpoint:    "",
		MaxTokens:   1000,
		Temperature: 0.1, // Low temperature for consistent results
		Timeout:     120 * time.Second,
		RetryCount:  2,
		Enabled:     false,
	}
}

// NoOpLLMClient implements LLMClient but does no processing
type NoOpLLMClient struct{}

// NewNoOpLLMClient creates a no-operation LLM client
func NewNoOpLLMClient() LLMClient {
	return &NoOpLLMClient{}
}

// ExtractDescription returns empty description for no-op client
func (n *NoOpLLMClient) ExtractDescription(ctx context.Context, emailContent string, trackingNumber string) (DescriptionResult, error) {
	return DescriptionResult{
		Description: "",
		Merchant:    "",
		Confidence:  0.0,
	}, nil
}

// OllamaLLMClient implements LLMClient for local Ollama instances
type OllamaLLMClient struct {
	config       *SimplifiedLLMConfig
	httpClient   *http.Client
	sanitizer    *ContentSanitizer
	securityUtils *SecurityUtils
	validator    *InputValidator
	rateLimiter  *RateLimiter
}

// NewOllamaLLMClient creates a new Ollama LLM client
func NewOllamaLLMClient(config *SimplifiedLLMConfig) LLMClient {
	return &OllamaLLMClient{
		config:        config,
		httpClient:    createSecureHTTPClient(config),
		sanitizer:     NewContentSanitizer(),
		securityUtils: NewSecurityUtils(),
		validator:     NewInputValidator(),
		rateLimiter:   DefaultLLMRateLimiter(),
	}
}

// ExtractDescription extracts description and merchant info from email content
func (o *OllamaLLMClient) ExtractDescription(ctx context.Context, emailContent string, trackingNumber string) (DescriptionResult, error) {
	if !o.config.Enabled {
		return DescriptionResult{}, nil
	}

	// Validate and sanitize inputs
	validationResult := o.validator.ValidateEmailProcessingInput(emailContent, trackingNumber)
	if !validationResult.IsValid {
		return DescriptionResult{}, fmt.Errorf("input validation failed: %v", validationResult.Errors)
	}

	// Use sanitized inputs for processing
	sanitizedContent := validationResult.SanitizedEmail
	sanitizedTrackingNumber := validationResult.SanitizedTrackingNumber

	// Apply rate limiting to prevent service overwhelm
	if err := o.rateLimiter.Wait(ctx); err != nil {
		return DescriptionResult{}, o.securityUtils.SafeErrorMessage("rate limit exceeded", err)
	}

	// Build focused prompt for description extraction only
	prompt := o.buildDescriptionPrompt(sanitizedContent, sanitizedTrackingNumber)
	
	// Call Ollama API
	response, err := o.callOllama(ctx, prompt)
	if err != nil {
		return DescriptionResult{}, o.securityUtils.SafeErrorMessage("Ollama API call failed", err)
	}

	// Parse response
	result, err := o.parseDescriptionResponse(response)
	if err != nil {
		return DescriptionResult{}, o.securityUtils.SafeErrorMessage("failed to parse LLM response", err)
	}

	return result, nil
}

// buildDescriptionPrompt creates a focused prompt for description extraction only
func (o *OllamaLLMClient) buildDescriptionPrompt(emailContent string, trackingNumber string) string {
	// Note: emailContent and trackingNumber should already be sanitized by caller
	prompt := fmt.Sprintf(`Extract the product description and merchant information from this shipping email. Return ONLY a JSON response.

Email Content: %s

Tracking Number: %s

Task: Extract meaningful product description and merchant information for this specific tracking number.

Instructions:
1. Extract specific product names, models, colors, sizes, quantities when available
2. Identify the merchant/retailer from sender domain, subject line, or content
3. If multiple products, combine them in a readable format
4. If no specific products mentioned, use generic descriptions like "Package" or "Shipment"
5. Assign confidence score based on specificity: 0.9+ for detailed product info, 0.7-0.9 for good matches, 0.5-0.7 for generic descriptions

Examples:

Input: "Your Amazon order of Apple iPhone 15 Pro 256GB Space Black has shipped"
Output: {"description": "Apple iPhone 15 Pro 256GB Space Black", "merchant": "Amazon", "confidence": 0.95}

Input: "Your order containing Dell XPS 13 Laptop and Logitech MX Master 3 Mouse has shipped from TechStore"
Output: {"description": "Dell XPS 13 Laptop, Logitech MX Master 3 Mouse", "merchant": "TechStore", "confidence": 0.9}

Input: "Your package has shipped"
Output: {"description": "Package", "merchant": "", "confidence": 0.3}

Return JSON format:
{
  "description": "specific product description here",
  "merchant": "merchant/retailer name",
  "confidence": 0.95
}`, emailContent, trackingNumber)

	return prompt
}

// truncateContent limits content size for API efficiency
func (o *OllamaLLMClient) truncateContent(content string) string {
	maxLength := 1500 // Reasonable limit for description extraction
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength] + "..."
}

// callOllama makes the API call to the Ollama endpoint
func (o *OllamaLLMClient) callOllama(ctx context.Context, prompt string) (string, error) {
	// Prepare request body for Ollama API
	requestBody := map[string]interface{}{
		"model":       o.config.Model,
		"prompt":      prompt,
		"stream":      false,
		"temperature": o.config.Temperature,
		"max_tokens":  o.config.MaxTokens,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", o.securityUtils.SafeErrorMessage("failed to marshal request", err)
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "POST", o.config.Endpoint+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", o.securityUtils.SafeErrorMessage("failed to create request", err)
	}

	// Set security headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "package-tracker-llm-client/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "close") // Prevent connection reuse for security
	
	// Set authentication header if API key is provided
	if o.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.config.APIKey)
	}

	// Make the request
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", o.securityUtils.SafeErrorMessage("request failed", err)
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
		return "", o.securityUtils.SafeErrorMessage("failed to decode response", err)
	}

	return ollamaResp.Response, nil
}

// parseDescriptionResponse parses the LLM JSON response into DescriptionResult
func (o *OllamaLLMClient) parseDescriptionResponse(response string) (DescriptionResult, error) {
	// Clean up the response (remove any markdown formatting)
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
	}
	response = strings.TrimSpace(response)

	// Parse JSON response
	var parsed struct {
		Description string  `json:"description"`
		Merchant    string  `json:"merchant"`
		Confidence  float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return DescriptionResult{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return DescriptionResult{
		Description: strings.TrimSpace(parsed.Description),
		Merchant:    strings.TrimSpace(parsed.Merchant),
		Confidence:  parsed.Confidence,
	}, nil
}

// createSecureHTTPClient creates an HTTP client with security best practices
func createSecureHTTPClient(config *SimplifiedLLMConfig) *http.Client {
	// Configure TLS with security best practices
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12, // Minimum TLS 1.2
		InsecureSkipVerify: false,            // Always verify certificates
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		},
	}

	// Configure transport with security settings and timeouts
	transport := &http.Transport{
		// Connection timeouts
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second, // Connection timeout
			KeepAlive: 30 * time.Second,
		}).DialContext,
		
		// TLS configuration
		TLSClientConfig: tlsConfig,
		
		// Connection limits to prevent resource exhaustion
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
		MaxConnsPerHost:     5,
		
		// Timeouts
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		
		// Security settings
		DisableKeepAlives: false, // Enable keep-alives for performance
		DisableCompression: false, // Enable compression
	}

	return &http.Client{
		Transport: transport,
		Timeout:   config.Timeout, // Overall request timeout
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Limit redirects to prevent redirect loops and abuse
			if len(via) >= 3 {
				return fmt.Errorf("stopped after 3 redirects")
			}
			return nil
		},
	}
}

// NewSimplifiedLLMClient creates an appropriate LLM client based on configuration
func NewSimplifiedLLMClient(config *SimplifiedLLMConfig) LLMClient {
	if !config.Enabled {
		return NewNoOpLLMClient()
	}

	switch strings.ToLower(config.Provider) {
	case "ollama":
		return NewOllamaLLMClient(config)
	case "openai":
		// TODO: Implement OpenAI client for description extraction
		return NewNoOpLLMClient()
	case "anthropic":
		// TODO: Implement Anthropic client for description extraction
		return NewNoOpLLMClient()
	case "disabled":
		return NewNoOpLLMClient()
	default:
		return NewNoOpLLMClient()
	}
}