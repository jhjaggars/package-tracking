package parser

import (
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

// TODO: Implement actual LLM extractors:
// - OpenAIExtractor
// - AnthropicExtractor  
// - LocalLLMExtractor
// 
// These will be implemented in a future phase based on the 
// detailed LLM integration specification in PARSING_STRATEGY.md