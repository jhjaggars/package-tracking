package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// SecurityUtils provides utility functions for handling sensitive data securely
type SecurityUtils struct{}

// RedactAPIKey redacts API keys from strings for safe logging
func (s *SecurityUtils) RedactAPIKey(content string) string {
	if content == "" {
		return content
	}

	// Pattern to match common API key formats
	patterns := []*regexp.Regexp{
		// Bearer tokens
		regexp.MustCompile(`(?i)bearer\s+([a-zA-Z0-9_\-\.]{8,})`),
		// API keys in various formats
		regexp.MustCompile(`(?i)(api[_\-]?key[^a-zA-Z0-9]*)([a-zA-Z0-9_\-\.]{8,})`),
		// Authorization headers
		regexp.MustCompile(`(?i)(authorization[^a-zA-Z0-9]*)([a-zA-Z0-9_\-\.]{8,})`),
		// Generic long alphanumeric strings that might be keys
		regexp.MustCompile(`\b([a-zA-Z0-9_\-\.]{32,})\b`),
	}

	redacted := content
	for _, pattern := range patterns {
		redacted = pattern.ReplaceAllStringFunc(redacted, func(match string) string {
			// Extract the prefix and the potential key
			submatches := pattern.FindStringSubmatch(match)
			if len(submatches) >= 2 {
				prefix := ""
				key := submatches[len(submatches)-1] // Get the last capture group (the key)
				
				// If there are multiple capture groups, the first is likely the prefix
				if len(submatches) > 2 {
					prefix = submatches[1]
				}
				
				return prefix + s.maskKey(key)
			}
			return s.maskKey(match)
		})
	}

	return redacted
}

// maskKey masks an API key showing only first and last few characters
func (s *SecurityUtils) maskKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	
	// Show first 4 and last 4 characters with asterisks in between
	visible := 4
	if len(key) < 12 {
		visible = 2
	}
	
	return key[:visible] + strings.Repeat("*", len(key)-2*visible) + key[len(key)-visible:]
}

// RedactConfig redacts sensitive fields from configuration for safe logging
func (s *SecurityUtils) RedactConfig(config *SimplifiedLLMConfig) *SimplifiedLLMConfig {
	if config == nil {
		return nil
	}

	// Create a copy to avoid modifying the original
	redacted := *config
	
	// Redact the API key
	if redacted.APIKey != "" {
		redacted.APIKey = s.maskKey(redacted.APIKey)
	}
	
	return &redacted
}

// SafeErrorMessage creates a safe error message that doesn't expose sensitive data
func (s *SecurityUtils) SafeErrorMessage(operation string, err error) error {
	if err == nil {
		return nil
	}

	// Redact any potential API keys from the error message
	safeMessage := s.RedactAPIKey(err.Error())
	
	return fmt.Errorf("%s: %s", operation, safeMessage)
}

// NewSecurityUtils creates a new SecurityUtils instance
func NewSecurityUtils() *SecurityUtils {
	return &SecurityUtils{}
}