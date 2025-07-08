package parser

import (
	"context"
)

// SimplifiedDescriptionExtractor represents the simplified description extractor
type SimplifiedDescriptionExtractor struct {
	llmClient LLMClient
	enabled   bool
}

// DescriptionResult represents the result of description extraction
type DescriptionResult struct {
	Description string
	Merchant    string
	Confidence  float64
}

// LLMClient interface for description extraction
type LLMClient interface {
	ExtractDescription(ctx context.Context, emailContent string, trackingNumber string) (DescriptionResult, error)
}

// SimplifiedDescriptionExtractorInterface defines the interface for description extraction
type SimplifiedDescriptionExtractorInterface interface {
	ExtractDescription(ctx context.Context, emailContent string, trackingNumber string) (string, error)
	IsEnabled() bool
}

// NewSimplifiedDescriptionExtractor creates a new simplified description extractor
func NewSimplifiedDescriptionExtractor(llmClient LLMClient, enabled bool) SimplifiedDescriptionExtractorInterface {
	return &SimplifiedDescriptionExtractor{
		llmClient: llmClient,
		enabled:   enabled,
	}
}

// ExtractDescription extracts description from email content using LLM
func (s *SimplifiedDescriptionExtractor) ExtractDescription(ctx context.Context, emailContent string, trackingNumber string) (string, error) {
	// If disabled, return empty string
	if !s.enabled {
		return "", nil
	}
	
	// If no email content or tracking number, return empty
	if emailContent == "" || trackingNumber == "" {
		return "", nil
	}
	
	// Use LLM client to extract description
	result, err := s.llmClient.ExtractDescription(ctx, emailContent, trackingNumber)
	if err != nil {
		return "", err
	}
	
	return result.Description, nil
}

// IsEnabled returns whether the description extractor is enabled
func (s *SimplifiedDescriptionExtractor) IsEnabled() bool {
	return s.enabled
}