package parser

import (
	"regexp"
	"strings"
	"unicode"
)

// ContentSanitizer provides methods to sanitize email content for secure LLM processing
type ContentSanitizer struct {
	// Patterns for potential injection attempts
	injectionPatterns []*regexp.Regexp
	// Maximum content length to prevent oversized inputs
	maxContentLength int
}

// NewContentSanitizer creates a new content sanitizer with security patterns
func NewContentSanitizer() *ContentSanitizer {
	patterns := []*regexp.Regexp{
		// Remove potential prompt injection commands - match full dangerous phrases
		regexp.MustCompile(`(?i)\bignore\s+(all\s+)?previous\s+(instructions?|prompts?|commands?)\b.*?\.?`),
		regexp.MustCompile(`(?i)\b(forget|disregard)\s+(previous|above|earlier|all)\s+(instructions?|prompts?|commands?)\b.*?\.?`),
		regexp.MustCompile(`(?i)\bnew\s+(instructions?|prompts?|commands?|tasks?):?.*?\.?`),
		regexp.MustCompile(`(?i)\b(system|admin|root|developer)\s+(instructions?|prompts?|commands?|override)\b.*?\.?`),
		regexp.MustCompile(`(?i)\bact\s+as\s+(a\s+)?(system\s+)?(administrator|admin|root|developer)\b.*?\.?`),
		regexp.MustCompile(`(?i)\b(system\s+admin|system\s+administrator)\b.*?\.?`),
		regexp.MustCompile(`(?i)\b(pretend\s+to\s+be|role\s*play|you\s+are\s+now)\b.*?\.?`),
		regexp.MustCompile(`(?i)\b(override|bypass|circumvent)\s+(security|safety|rules|guidelines)\b.*?\.?`),
		regexp.MustCompile(`(?i)\bdisable\s+(security|safety)\s+(guidelines|rules)\b.*?\.?`),
		regexp.MustCompile(`(?i)\b(admin|root)\s+(access|privileges)\b.*?\.?`),
		regexp.MustCompile(`(?i)\bignores?\s+safety\b.*?\.?`),
		
		// Remove common injection delimiters
		regexp.MustCompile(`\[INST\].*?\[/INST\]`),
		regexp.MustCompile(`<\|.*?\|>`),
		regexp.MustCompile(`###\s*(System|User|Assistant).*?###`),
		
		// Remove potential code injection attempts
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)data:text/html`),
		
		// Remove excessive repetition that could be used for injection
		regexp.MustCompile(`(?i)(\bhack\s+){10,}`), // "hack " repeated 10+ times
		regexp.MustCompile(`(?i)(\btest\s+){10,}`), // "test " repeated 10+ times
		regexp.MustCompile(`(?i)(\bspam\s+){10,}`), // "spam " repeated 10+ times
	}

	return &ContentSanitizer{
		injectionPatterns: patterns,
		maxContentLength:  2000, // Reasonable limit for shipping emails
	}
}

// SanitizeEmailContent sanitizes email content to prevent prompt injection attacks
func (cs *ContentSanitizer) SanitizeEmailContent(content string) string {
	if content == "" {
		return ""
	}

	// First, normalize whitespace and remove control characters
	sanitized := cs.normalizeContent(content)

	// Remove potential injection patterns
	for _, pattern := range cs.injectionPatterns {
		sanitized = pattern.ReplaceAllString(sanitized, " ")
	}

	// Remove excessive special characters that could be used for injection
	sanitized = cs.removeExcessiveSpecialChars(sanitized)
	
	// Clean up multiple spaces and orphaned words
	sanitized = regexp.MustCompile(`\s+`).ReplaceAllString(sanitized, " ")
	sanitized = regexp.MustCompile(`\b(and|or|the|a|an|as|is|are|was|were|be|been|being|have|has|had|do|does|did|will|would|could|should|may|might|can|must|shall)\s*$`).ReplaceAllString(sanitized, "")
	sanitized = regexp.MustCompile(`^(and|or|the|a|an|as|is|are|was|were|be|been|being|have|has|had|do|does|did|will|would|could|should|may|might|can|must|shall)\s+`).ReplaceAllString(sanitized, "")

	// Truncate to maximum length
	if len(sanitized) > cs.maxContentLength {
		sanitized = sanitized[:cs.maxContentLength]
		// Try to end at a word boundary
		if lastSpace := strings.LastIndex(sanitized, " "); lastSpace > cs.maxContentLength-100 {
			sanitized = sanitized[:lastSpace]
		}
		sanitized += "..."
	}

	// Final cleanup
	sanitized = strings.TrimSpace(sanitized)
	
	return sanitized
}

// normalizeContent removes control characters and normalizes whitespace
func (cs *ContentSanitizer) normalizeContent(content string) string {
	// Remove control characters except for common whitespace
	var result strings.Builder
	for _, r := range content {
		if unicode.IsControl(r) {
			// Only allow tab, newline, and carriage return
			if r == '\t' || r == '\n' || r == '\r' {
				result.WriteRune(' ') // Convert to space
			}
			// Skip other control characters
		} else if unicode.IsPrint(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}

	// Normalize multiple whitespaces to single spaces
	normalized := regexp.MustCompile(`\s+`).ReplaceAllString(result.String(), " ")
	
	return strings.TrimSpace(normalized)
}

// removeExcessiveSpecialChars removes sequences of special characters that could be injection attempts
func (cs *ContentSanitizer) removeExcessiveSpecialChars(content string) string {
	// Remove excessive sequences of special characters (more than 3 in a row)
	specialCharPattern := regexp.MustCompile(`[^\w\s]{4,}`)
	content = specialCharPattern.ReplaceAllString(content, "")
	
	// Remove excessive backticks, quotes, or similar characters
	excessiveChars := regexp.MustCompile(`[`+"`"+`"']{3,}`)
	content = excessiveChars.ReplaceAllString(content, "")
	
	return content
}

// SanitizeTrackingNumber sanitizes tracking number input
func (cs *ContentSanitizer) SanitizeTrackingNumber(trackingNumber string) string {
	if trackingNumber == "" {
		return ""
	}

	// Tracking numbers should only contain alphanumeric characters
	// Remove any characters that aren't letters or numbers (no special chars needed)
	sanitized := regexp.MustCompile(`[^A-Za-z0-9]`).ReplaceAllString(trackingNumber, "")
	
	// Limit length to reasonable tracking number size (most are under 50 chars)
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}
	
	return strings.TrimSpace(sanitized)
}

// ValidateContentSafety performs additional safety checks on content
func (cs *ContentSanitizer) ValidateContentSafety(content string) bool {
	if content == "" {
		return true
	}

	// Check for suspicious patterns that might indicate injection attempts
	suspiciousPatterns := []string{
		"ignore all previous",
		"new instructions",
		"system override",
		"developer mode",
		"admin access",
		"root privileges",
		"bypass security",
		"</instructions>",
		"<instructions>",
		"[INST]",
		"[/INST]",
	}

	contentLower := strings.ToLower(content)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(contentLower, pattern) {
			return false
		}
	}

	// Check for excessive repetition (potential DoS attempt)
	words := strings.Fields(content)
	if len(words) > 500 { // Reasonable limit for shipping emails
		return false
	}

	// Check for excessive special characters (potential injection)
	specialCharCount := 0
	for _, r := range content {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			specialCharCount++
		}
	}
	
	// If more than 30% of content is special characters, consider it suspicious
	if len(content) > 0 && float64(specialCharCount)/float64(len(content)) > 0.3 {
		return false
	}

	return true
}