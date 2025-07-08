package parser

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// InputValidator provides comprehensive input validation for LLM processing
type InputValidator struct {
	sanitizer *ContentSanitizer
}

// NewInputValidator creates a new input validator
func NewInputValidator() *InputValidator {
	return &InputValidator{
		sanitizer: NewContentSanitizer(),
	}
}

// ValidationResult contains the results of input validation
type ValidationResult struct {
	IsValid        bool
	SanitizedEmail string
	SanitizedTrackingNumber string
	Errors         []string
	Warnings       []string
}

// ValidateEmailProcessingInput validates and sanitizes input for LLM processing
func (v *InputValidator) ValidateEmailProcessingInput(emailContent, trackingNumber string) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Validate email content
	if err := v.validateEmailContent(emailContent); err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.IsValid = false
	}

	// Validate tracking number
	if err := v.validateTrackingNumber(trackingNumber); err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.IsValid = false
	}

	// If basic validation fails, don't proceed with sanitization
	if !result.IsValid {
		return result
	}

	// Sanitize the inputs
	result.SanitizedEmail = v.sanitizer.SanitizeEmailContent(emailContent)
	result.SanitizedTrackingNumber = v.sanitizer.SanitizeTrackingNumber(trackingNumber)

	// Validate content safety after sanitization
	if !v.sanitizer.ValidateContentSafety(result.SanitizedEmail) {
		result.Errors = append(result.Errors, "email content failed safety validation")
		result.IsValid = false
	}

	// Check for significant content loss during sanitization
	if v.hasSignificantContentLoss(emailContent, result.SanitizedEmail) {
		result.Warnings = append(result.Warnings, "significant content was removed during sanitization")
	}

	return result
}

// validateEmailContent performs basic validation on email content
func (v *InputValidator) validateEmailContent(content string) error {
	if content == "" {
		return fmt.Errorf("email content cannot be empty")
	}

	// Check for valid UTF-8 encoding
	if !utf8.ValidString(content) {
		return fmt.Errorf("email content contains invalid UTF-8 characters")
	}

	// Check for reasonable length limits
	if len(content) > 50000 { // 50KB limit
		return fmt.Errorf("email content too large (maximum 50KB)")
	}

	// Check for minimum meaningful content
	if len(strings.TrimSpace(content)) < 10 {
		return fmt.Errorf("email content too short (minimum 10 characters)")
	}

	// Check for suspicious patterns that might indicate malformed input
	if v.hasSuspiciousPatterns(content) {
		return fmt.Errorf("email content contains suspicious patterns")
	}

	return nil
}

// validateTrackingNumber performs basic validation on tracking numbers
func (v *InputValidator) validateTrackingNumber(trackingNumber string) error {
	if trackingNumber == "" {
		return fmt.Errorf("tracking number cannot be empty")
	}

	// Check for valid UTF-8 encoding
	if !utf8.ValidString(trackingNumber) {
		return fmt.Errorf("tracking number contains invalid UTF-8 characters")
	}

	// Check length limits
	if len(trackingNumber) > 100 {
		return fmt.Errorf("tracking number too long (maximum 100 characters)")
	}

	if len(strings.TrimSpace(trackingNumber)) < 3 {
		return fmt.Errorf("tracking number too short (minimum 3 characters)")
	}

	// Check for basic alphanumeric pattern (lenient validation since we sanitize)
	// We allow special characters here because they'll be removed during sanitization
	if !v.isValidTrackingNumberFormat(trackingNumber) {
		return fmt.Errorf("tracking number contains invalid characters")
	}

	return nil
}

// hasSuspiciousPatterns checks for patterns that might indicate malicious input
func (v *InputValidator) hasSuspiciousPatterns(content string) bool {
	suspiciousPatterns := []*regexp.Regexp{
		// Excessive null bytes or control characters
		regexp.MustCompile(`\x00{5,}`),
		// Excessive repetition of special characters
		regexp.MustCompile(`[^\w\s]{50,}`),
		// Patterns that look like code injection attempts
		regexp.MustCompile(`(?i)<script[^>]*>[^<]*</script>`),
		regexp.MustCompile(`(?i)javascript:[^"'\s]+`),
		// Excessive unicode control characters (using hex notation)
		regexp.MustCompile(`[\x00-\x1F\x7F-\x9F]{10,}`),
		// Patterns that might be binary data
		regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F-\xFF]{20,}`),
	}

	for _, pattern := range suspiciousPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}

	return false
}

// isValidTrackingNumberFormat checks if tracking number has valid format
func (v *InputValidator) isValidTrackingNumberFormat(trackingNumber string) bool {
	// Allow alphanumeric characters and common separators, but exclude dangerous patterns
	// We're lenient here since we sanitize afterward
	
	// Check for extreme cases that might indicate malicious input
	if strings.Contains(trackingNumber, "\x00") {
		return false // Null bytes are never valid
	}
	
	// Must contain at least some alphanumeric characters
	alphanumericPattern := regexp.MustCompile(`[A-Za-z0-9]`)
	if !alphanumericPattern.MatchString(trackingNumber) {
		return false
	}
	
	// Otherwise, allow it (sanitization will clean it up)
	return true
}

// hasSignificantContentLoss checks if sanitization removed too much content
func (v *InputValidator) hasSignificantContentLoss(original, sanitized string) bool {
	if original == "" {
		return false
	}

	// Calculate the percentage of content retained
	retainedPercentage := float64(len(sanitized)) / float64(len(original))
	
	// Consider it significant loss if less than 50% of content remains
	return retainedPercentage < 0.5
}

// ValidateRequestSize validates the overall size of the request
func (v *InputValidator) ValidateRequestSize(emailContent, trackingNumber string) error {
	totalSize := len(emailContent) + len(trackingNumber)
	
	// Set a reasonable total request size limit
	maxSize := 100000 // 100KB
	
	if totalSize > maxSize {
		return fmt.Errorf("total request size too large: %d bytes (maximum %d bytes)", totalSize, maxSize)
	}
	
	return nil
}

// SanitizeForLogging safely sanitizes content for logging purposes
func (v *InputValidator) SanitizeForLogging(content string) string {
	if content == "" {
		return ""
	}

	// Truncate very long content for logging
	maxLogLength := 200
	if len(content) > maxLogLength {
		content = content[:maxLogLength] + "..."
	}

	// Remove potential sensitive data patterns
	securityUtils := NewSecurityUtils()
	content = securityUtils.RedactAPIKey(content)

	// Remove excessive whitespace for cleaner logs
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
	content = strings.TrimSpace(content)

	return content
}