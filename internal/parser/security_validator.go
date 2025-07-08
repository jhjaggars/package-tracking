package parser

import (
	"fmt"
	"net/url"
	"strings"
)

// SecurityValidator provides comprehensive security validation for the entire system
type SecurityValidator struct {
	inputValidator *InputValidator
	sanitizer      *ContentSanitizer
	securityUtils  *SecurityUtils
}

// NewSecurityValidator creates a new comprehensive security validator
func NewSecurityValidator() *SecurityValidator {
	return &SecurityValidator{
		inputValidator: NewInputValidator(),
		sanitizer:      NewContentSanitizer(),
		securityUtils:  NewSecurityUtils(),
	}
}

// SecurityReport contains the results of a comprehensive security validation
type SecurityReport struct {
	Passed   bool
	Issues   []SecurityIssue
	Warnings []SecurityWarning
}

// SecurityIssue represents a security issue that must be addressed
type SecurityIssue struct {
	Severity    string // "critical", "high", "medium", "low"
	Component   string // Component where issue was found
	Description string
	Mitigation  string
}

// SecurityWarning represents a security warning that should be noted
type SecurityWarning struct {
	Component   string
	Description string
	Suggestion  string
}

// ValidateSystemSecurity performs comprehensive security validation
func (sv *SecurityValidator) ValidateSystemSecurity(config *SimplifiedLLMConfig, emailContent, trackingNumber string) *SecurityReport {
	report := &SecurityReport{
		Passed:   true,
		Issues:   []SecurityIssue{},
		Warnings: []SecurityWarning{},
	}

	// Validate configuration security first (handles nil config)
	sv.validateConfigSecurity(config, report)

	// Only continue with other validations if config is not nil
	if config != nil {
		// Validate input security
		sv.validateInputSecurity(emailContent, trackingNumber, report)

		// Validate endpoint security
		if config.Endpoint != "" {
			sv.validateEndpointSecurity(config.Endpoint, report)
		}
	} else {
		// If config is nil, we can't validate inputs properly, so mark as failed
		report.Passed = false
	}

	return report
}

// validateConfigSecurity checks for configuration-related security issues
func (sv *SecurityValidator) validateConfigSecurity(config *SimplifiedLLMConfig, report *SecurityReport) {
	if config == nil {
		report.Issues = append(report.Issues, SecurityIssue{
			Severity:    "high",
			Component:   "configuration",
			Description: "LLM configuration is nil",
			Mitigation:  "Ensure proper configuration initialization",
		})
		report.Passed = false
		return
	}

	// Check for insecure API key patterns
	if config.APIKey != "" {
		if sv.hasWeakAPIKey(config.APIKey) {
			report.Issues = append(report.Issues, SecurityIssue{
				Severity:    "medium",
				Component:   "configuration",
				Description: "API key appears to be weak or test key",
				Mitigation:  "Use a strong, production API key",
			})
		}
	}

	// Check for insecure endpoint configurations
	if config.Endpoint != "" {
		if strings.HasPrefix(config.Endpoint, "http://") {
			report.Issues = append(report.Issues, SecurityIssue{
				Severity:    "medium",
				Component:   "configuration",
				Description: "LLM endpoint uses insecure HTTP protocol",
				Mitigation:  "Use HTTPS for all external communications",
			})
		}
	}

	// Check for overly permissive timeout settings
	if config.Timeout.Seconds() > 600 { // 10 minutes
		report.Warnings = append(report.Warnings, SecurityWarning{
			Component:   "configuration",
			Description: "LLM timeout is very long (>10 minutes)",
			Suggestion:  "Consider using shorter timeouts to prevent resource exhaustion",
		})
	}

	// Check for disabled security features in production
	if !config.Enabled {
		report.Warnings = append(report.Warnings, SecurityWarning{
			Component:   "configuration",
			Description: "LLM processing is disabled",
			Suggestion:  "Ensure this is intentional for your use case",
		})
	}
}

// validateInputSecurity checks for input-related security issues
func (sv *SecurityValidator) validateInputSecurity(emailContent, trackingNumber string, report *SecurityReport) {
	// Validate email content
	if err := sv.inputValidator.validateEmailContent(emailContent); err != nil {
		report.Issues = append(report.Issues, SecurityIssue{
			Severity:    "medium",
			Component:   "input_validation",
			Description: fmt.Sprintf("Email content validation failed: %v", err),
			Mitigation:  "Sanitize and validate all email input",
		})
		report.Passed = false
	}

	// Validate tracking number
	if err := sv.inputValidator.validateTrackingNumber(trackingNumber); err != nil {
		report.Issues = append(report.Issues, SecurityIssue{
			Severity:    "medium",
			Component:   "input_validation",
			Description: fmt.Sprintf("Tracking number validation failed: %v", err),
			Mitigation:  "Sanitize and validate all tracking number input",
		})
		report.Passed = false
	}

	// Check for potential injection attempts
	if !sv.sanitizer.ValidateContentSafety(emailContent) {
		report.Issues = append(report.Issues, SecurityIssue{
			Severity:    "high",
			Component:   "content_safety",
			Description: "Email content failed safety validation - potential injection attempt",
			Mitigation:  "Content sanitization and safety validation are required",
		})
		report.Passed = false
	}

	// Check request size
	if err := sv.inputValidator.ValidateRequestSize(emailContent, trackingNumber); err != nil {
		report.Issues = append(report.Issues, SecurityIssue{
			Severity:    "medium",
			Component:   "input_validation",
			Description: fmt.Sprintf("Request size validation failed: %v", err),
			Mitigation:  "Implement request size limits to prevent DoS attacks",
		})
		report.Passed = false
	}
}

// validateEndpointSecurity checks for endpoint-related security issues
func (sv *SecurityValidator) validateEndpointSecurity(endpoint string, report *SecurityReport) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		report.Issues = append(report.Issues, SecurityIssue{
			Severity:    "high",
			Component:   "endpoint_validation",
			Description: "Invalid endpoint URL format",
			Mitigation:  "Ensure endpoint URL is properly formatted",
		})
		report.Passed = false
		return
	}

	// Check for localhost in production
	if sv.isProductionEndpoint(parsedURL) {
		if parsedURL.Hostname() == "localhost" || parsedURL.Hostname() == "127.0.0.1" {
			report.Warnings = append(report.Warnings, SecurityWarning{
				Component:   "endpoint_security",
				Description: "Using localhost endpoint may not be suitable for production",
				Suggestion:  "Consider using a proper production endpoint",
			})
		}
	}

	// Check for non-standard ports
	if parsedURL.Port() != "" && parsedURL.Port() != "80" && parsedURL.Port() != "443" {
		report.Warnings = append(report.Warnings, SecurityWarning{
			Component:   "endpoint_security",
			Description: "Using non-standard port for LLM endpoint",
			Suggestion:  "Ensure firewall rules and network security are properly configured",
		})
	}
}

// HasWeakAPIKey checks if an API key appears to be weak or for testing (public for testing)
func (sv *SecurityValidator) HasWeakAPIKey(apiKey string) bool {
	return sv.hasWeakAPIKey(apiKey)
}

// hasWeakAPIKey checks if an API key appears to be weak or for testing
func (sv *SecurityValidator) hasWeakAPIKey(apiKey string) bool {
	weakPatterns := []string{
		"test",
		"demo",
		"example",
		"placeholder",
		"change-me",
		"12345",
		"abcde",
	}

	lowerKey := strings.ToLower(apiKey)
	for _, pattern := range weakPatterns {
		if strings.Contains(lowerKey, pattern) {
			return true
		}
	}

	// Check for very short keys (likely test keys)
	if len(apiKey) < 20 {
		return true
	}

	return false
}

// isProductionEndpoint determines if an endpoint is likely for production use
func (sv *SecurityValidator) isProductionEndpoint(parsedURL *url.URL) bool {
	// Simple heuristic: not localhost and using HTTPS
	return parsedURL.Hostname() != "localhost" &&
		parsedURL.Hostname() != "127.0.0.1" &&
		parsedURL.Scheme == "https"
}

// GenerateSecuritySummary creates a human-readable security summary
func (report *SecurityReport) GenerateSecuritySummary() string {
	var summary strings.Builder

	summary.WriteString("=== Security Validation Report ===\n")
	
	if report.Passed {
		summary.WriteString("Status: PASSED ✓\n")
	} else {
		summary.WriteString("Status: FAILED ✗\n")
	}

	if len(report.Issues) > 0 {
		summary.WriteString(fmt.Sprintf("\nSecurity Issues (%d):\n", len(report.Issues)))
		for i, issue := range report.Issues {
			summary.WriteString(fmt.Sprintf("  %d. [%s] %s: %s\n", 
				i+1, strings.ToUpper(issue.Severity), issue.Component, issue.Description))
			summary.WriteString(fmt.Sprintf("     Mitigation: %s\n", issue.Mitigation))
		}
	}

	if len(report.Warnings) > 0 {
		summary.WriteString(fmt.Sprintf("\nSecurity Warnings (%d):\n", len(report.Warnings)))
		for i, warning := range report.Warnings {
			summary.WriteString(fmt.Sprintf("  %d. %s: %s\n", 
				i+1, warning.Component, warning.Description))
			summary.WriteString(fmt.Sprintf("     Suggestion: %s\n", warning.Suggestion))
		}
	}

	if len(report.Issues) == 0 && len(report.Warnings) == 0 {
		summary.WriteString("\nNo security issues or warnings found.\n")
	}

	summary.WriteString("\n=== End Report ===")
	return summary.String()
}