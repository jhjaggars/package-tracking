package parser

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSimplifiedLLMConfig(t *testing.T) {
	config := DefaultSimplifiedLLMConfig()
	
	assert.NotNil(t, config)
	assert.Equal(t, "disabled", config.Provider)
	assert.Equal(t, "", config.Model)
	assert.Equal(t, false, config.Enabled)
	assert.Equal(t, 0.1, config.Temperature)
	assert.Equal(t, 1000, config.MaxTokens)
	assert.Equal(t, 120*time.Second, config.Timeout)
	assert.Equal(t, 2, config.RetryCount)
}

func TestNoOpLLMClient(t *testing.T) {
	client := NewNoOpLLMClient()
	assert.NotNil(t, client)
	
	ctx := context.Background()
	result, err := client.ExtractDescription(ctx, "test email content", "1Z999AA1234567890")
	
	assert.NoError(t, err)
	assert.Equal(t, "", result.Description)
	assert.Equal(t, "", result.Merchant)
	assert.Equal(t, 0.0, result.Confidence)
}

func TestNewSimplifiedLLMClient_Disabled(t *testing.T) {
	config := &SimplifiedLLMConfig{
		Provider: "disabled",
		Enabled:  false,
	}
	
	client := NewSimplifiedLLMClient(config)
	assert.IsType(t, &NoOpLLMClient{}, client)
}

func TestNewSimplifiedLLMClient_EnabledButUnsupportedProvider(t *testing.T) {
	config := &SimplifiedLLMConfig{
		Provider: "unsupported",
		Enabled:  true,
	}
	
	client := NewSimplifiedLLMClient(config)
	assert.IsType(t, &NoOpLLMClient{}, client)
}

func TestNewSimplifiedLLMClient_Ollama(t *testing.T) {
	config := &SimplifiedLLMConfig{
		Provider: "ollama",
		Enabled:  true,
		Endpoint: "http://localhost:11434",
		Model:    "llama3.2",
	}
	
	client := NewSimplifiedLLMClient(config)
	assert.IsType(t, &OllamaLLMClient{}, client)
}

func TestOllamaLLMClient_DisabledConfig(t *testing.T) {
	config := &SimplifiedLLMConfig{
		Provider: "ollama",
		Enabled:  false, // Disabled
		Endpoint: "http://localhost:11434",
		Model:    "llama3.2",
	}
	
	client := NewOllamaLLMClient(config)
	ctx := context.Background()
	
	result, err := client.ExtractDescription(ctx, "test content", "1Z999AA1234567890")
	
	assert.NoError(t, err)
	assert.Equal(t, "", result.Description)
	assert.Equal(t, "", result.Merchant)
	assert.Equal(t, 0.0, result.Confidence)
}

func TestOllamaLLMClient_BuildDescriptionPrompt(t *testing.T) {
	config := &SimplifiedLLMConfig{
		Provider: "ollama",
		Enabled:  true,
		Endpoint: "http://localhost:11434",
		Model:    "llama3.2",
	}
	
	client := NewOllamaLLMClient(config).(*OllamaLLMClient)
	
	emailContent := "Your Amazon order of Apple iPhone 15 Pro 256GB Space Black has shipped"
	trackingNumber := "1Z999AA1234567890"
	
	prompt := client.buildDescriptionPrompt(emailContent, trackingNumber)
	
	assert.Contains(t, prompt, "Extract the product description and merchant information")
	assert.Contains(t, prompt, emailContent)
	assert.Contains(t, prompt, trackingNumber)
	assert.Contains(t, prompt, "Apple iPhone 15 Pro 256GB Space Black")
	assert.Contains(t, prompt, "Amazon")
}

func TestOllamaLLMClient_TruncateContent(t *testing.T) {
	config := &SimplifiedLLMConfig{
		Provider: "ollama",
		Enabled:  true,
	}
	
	client := NewOllamaLLMClient(config).(*OllamaLLMClient)
	
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "Short content",
			content:  "Short email content",
			expected: "Short email content",
		},
		{
			name:     "Long content gets truncated",
			content:  strings.Repeat("a", 2000),
			expected: strings.Repeat("a", 1500) + "...",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.truncateContent(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOllamaLLMClient_ParseDescriptionResponse(t *testing.T) {
	config := &SimplifiedLLMConfig{
		Provider: "ollama",
		Enabled:  true,
	}
	
	client := NewOllamaLLMClient(config).(*OllamaLLMClient)
	
	tests := []struct {
		name     string
		response string
		expected DescriptionResult
		hasError bool
	}{
		{
			name:     "Valid JSON response",
			response: `{"description": "Apple iPhone 15 Pro", "merchant": "Amazon", "confidence": 0.95}`,
			expected: DescriptionResult{
				Description: "Apple iPhone 15 Pro",
				Merchant:    "Amazon",
				Confidence:  0.95,
			},
			hasError: false,
		},
		{
			name:     "JSON with markdown formatting",
			response: "```json\n{\"description\": \"MacBook Pro\", \"merchant\": \"Apple Store\", \"confidence\": 0.9}\n```",
			expected: DescriptionResult{
				Description: "MacBook Pro",
				Merchant:    "Apple Store",
				Confidence:  0.9,
			},
			hasError: false,
		},
		{
			name:     "Empty description",
			response: `{"description": "", "merchant": "", "confidence": 0.0}`,
			expected: DescriptionResult{
				Description: "",
				Merchant:    "",
				Confidence:  0.0,
			},
			hasError: false,
		},
		{
			name:     "Invalid JSON",
			response: `{invalid json}`,
			expected: DescriptionResult{},
			hasError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.parseDescriptionResponse(tt.response)
			
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}