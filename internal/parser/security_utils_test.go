package parser

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurityUtils_RedactAPIKey(t *testing.T) {
	utils := NewSecurityUtils()

	tests := []struct {
		name     string
		input    string
		expected func(string) bool // Function to validate the result
	}{
		{
			name:  "Bearer token",
			input: "Authorization: Bearer sk-1234567890abcdef1234567890abcdef",
			expected: func(result string) bool {
				return !containsStr(result, "sk-1234567890abcdef1234567890abcdef") &&
					containsStr(result, "sk-1****") &&
					containsStr(result, "****cdef")
			},
		},
		{
			name:  "API key in text",
			input: "API key is abc123def456ghi789jkl012mno345pqr678",
			expected: func(result string) bool {
				return !containsStr(result, "abc123def456ghi789jkl012mno345pqr678") &&
					containsStr(result, "abc1****")
			},
		},
		{
			name:  "Authorization header",
			input: "Authorization: sk-proj-abcdefghijklmnopqrstuvwxyz123456789",
			expected: func(result string) bool {
				return !containsStr(result, "sk-proj-abcdefghijklmnopqrstuvwxyz123456789") &&
					containsStr(result, "sk-p****")
			},
		},
		{
			name:  "Short string (no redaction)",
			input: "short",
			expected: func(result string) bool {
				return result == "short"
			},
		},
		{
			name:  "Empty string",
			input: "",
			expected: func(result string) bool {
				return result == ""
			},
		},
		{
			name:  "Normal email content",
			input: "Your package from Amazon has shipped with tracking 1Z999AA1234567890",
			expected: func(result string) bool {
				return containsStr(result, "Your package from Amazon")
			},
		},
		{
			name:  "Long alphanumeric string (potential key)",
			input: "Error with key: abcdefghijklmnopqrstuvwxyz1234567890abcdefghij in request",
			expected: func(result string) bool {
				return !containsStr(result, "abcdefghijklmnopqrstuvwxyz1234567890abcdefghij") &&
					containsStr(result, "abcd****")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.RedactAPIKey(tt.input)
			assert.True(t, tt.expected(result), "Result: %s", result)
		})
	}
}

func TestSecurityUtils_maskKey(t *testing.T) {
	utils := NewSecurityUtils()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Long key",
			input:    "sk-1234567890abcdef1234567890abcdef",
			expected: "sk-1***************************cdef",
		},
		{
			name:     "Medium key",
			input:    "abc123def456",
			expected: "abc1****f456",
		},
		{
			name:     "Short key",
			input:    "shortkey",
			expected: "********",
		},
		{
			name:     "Very short key",
			input:    "abc",
			expected: "***",
		},
		{
			name:     "Empty key",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.maskKey(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSecurityUtils_RedactConfig(t *testing.T) {
	utils := NewSecurityUtils()

	t.Run("Config with API key", func(t *testing.T) {
		original := &SimplifiedLLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
			APIKey:   "sk-1234567890abcdef1234567890abcdef",
			Endpoint: "https://api.openai.com",
		}

		redacted := utils.RedactConfig(original)

		// Original should be unchanged
		assert.Equal(t, "sk-1234567890abcdef1234567890abcdef", original.APIKey)

		// Redacted should have masked key
		assert.Equal(t, "sk-1***************************cdef", redacted.APIKey)
		assert.Equal(t, "openai", redacted.Provider)
		assert.Equal(t, "gpt-4", redacted.Model)
		assert.Equal(t, "https://api.openai.com", redacted.Endpoint)
	})

	t.Run("Config without API key", func(t *testing.T) {
		original := &SimplifiedLLMConfig{
			Provider: "ollama",
			Model:    "llama2",
			APIKey:   "",
			Endpoint: "http://localhost:11434",
		}

		redacted := utils.RedactConfig(original)

		assert.Equal(t, "", redacted.APIKey)
		assert.Equal(t, "ollama", redacted.Provider)
	})

	t.Run("Nil config", func(t *testing.T) {
		redacted := utils.RedactConfig(nil)
		assert.Nil(t, redacted)
	})
}

func TestSecurityUtils_SafeErrorMessage(t *testing.T) {
	utils := NewSecurityUtils()

	tests := []struct {
		name      string
		operation string
		err       error
		expected  func(string) bool
	}{
		{
			name:      "Error with API key",
			operation: "LLM request",
			err:       errors.New("authentication failed with key sk-1234567890abcdef1234567890abcdef"),
			expected: func(result string) bool {
				return containsStr(result, "LLM request") &&
					!containsStr(result, "sk-1234567890abcdef1234567890abcdef") &&
					containsStr(result, "authentication failed")
			},
		},
		{
			name:      "Normal error",
			operation: "database query",
			err:       errors.New("connection timeout"),
			expected: func(result string) bool {
				return containsStr(result, "database query: connection timeout")
			},
		},
		{
			name:      "Nil error",
			operation: "test operation",
			err:       nil,
			expected: func(result string) bool {
				return result == ""
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.SafeErrorMessage(tt.operation, tt.err)
			if result == nil {
				assert.True(t, tt.expected(""))
			} else {
				assert.True(t, tt.expected(result.Error()), "Result: %s", result.Error())
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsStr(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && 
		   strings.Contains(s, substr)
}