package parser

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLLMClient for testing
type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) ExtractDescription(ctx context.Context, emailContent string, trackingNumber string) (DescriptionResult, error) {
	args := m.Called(ctx, emailContent, trackingNumber)
	return args.Get(0).(DescriptionResult), args.Error(1)
}

// Test for creating a new simplified description extractor
func TestNewSimplifiedDescriptionExtractor(t *testing.T) {
	mockLLM := &MockLLMClient{}
	
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "Enabled description extractor",
			enabled: true,
		},
		{
			name:    "Disabled description extractor",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewSimplifiedDescriptionExtractor(mockLLM, tt.enabled)
			
			assert.NotNil(t, extractor)
			assert.IsType(t, &SimplifiedDescriptionExtractor{}, extractor)
			assert.Equal(t, tt.enabled, extractor.IsEnabled())
		})
	}
}

// Test for extracting description when enabled
func TestSimplifiedDescriptionExtractor_ExtractDescription_Enabled(t *testing.T) {
	mockLLM := &MockLLMClient{}
	extractor := NewSimplifiedDescriptionExtractor(mockLLM, true)

	tests := []struct {
		name          string
		emailContent  string
		trackingNumber string
		llmResult     DescriptionResult
		llmError      error
		expected      string
		expectError   bool
	}{
		{
			name:           "Successful extraction",
			emailContent:   "Your order has shipped. Item: Apple iPhone 15 Pro",
			trackingNumber: "1Z999AA1234567890",
			llmResult: DescriptionResult{
				Description: "Apple iPhone 15 Pro",
				Merchant:    "Apple Store",
				Confidence:  0.95,
			},
			llmError:    nil,
			expected:    "Apple iPhone 15 Pro",
			expectError: false,
		},
		{
			name:           "Extraction with merchant information",
			emailContent:   "Amazon order shipped. Your MacBook Pro is on the way.",
			trackingNumber: "TBA123456789012",
			llmResult: DescriptionResult{
				Description: "MacBook Pro",
				Merchant:    "Amazon",
				Confidence:  0.88,
			},
			llmError:    nil,
			expected:    "MacBook Pro",
			expectError: false,
		},
		{
			name:           "Low confidence extraction",
			emailContent:   "Your package has shipped.",
			trackingNumber: "9400111899562537624840",
			llmResult: DescriptionResult{
				Description: "Package",
				Merchant:    "Unknown",
				Confidence:  0.3,
			},
			llmError:    nil,
			expected:    "Package",
			expectError: false,
		},
		{
			name:           "LLM error",
			emailContent:   "Your order has shipped.",
			trackingNumber: "1Z999AA1234567890",
			llmResult:      DescriptionResult{},
			llmError:       errors.New("LLM service unavailable"),
			expected:       "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			
			mockLLM.On("ExtractDescription", ctx, tt.emailContent, tt.trackingNumber).
				Return(tt.llmResult, tt.llmError).Once()

			result, err := extractor.ExtractDescription(ctx, tt.emailContent, tt.trackingNumber)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expected, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			mockLLM.AssertExpectations(t)
		})
	}
}

// Test for extracting description when disabled
func TestSimplifiedDescriptionExtractor_ExtractDescription_Disabled(t *testing.T) {
	mockLLM := &MockLLMClient{}
	extractor := NewSimplifiedDescriptionExtractor(mockLLM, false)

	ctx := context.Background()
	emailContent := "Your order has shipped. Item: Apple iPhone 15 Pro"
	trackingNumber := "1Z999AA1234567890"

	result, err := extractor.ExtractDescription(ctx, emailContent, trackingNumber)

	assert.NoError(t, err)
	assert.Equal(t, "", result) // Should return empty string when disabled

	// Verify LLM was not called
	mockLLM.AssertNotCalled(t, "ExtractDescription")
}

// Test for different email content types
func TestSimplifiedDescriptionExtractor_DifferentEmailTypes(t *testing.T) {
	mockLLM := &MockLLMClient{}
	extractor := NewSimplifiedDescriptionExtractor(mockLLM, true)

	tests := []struct {
		name          string
		emailContent  string
		trackingNumber string
		expectedDesc  string
	}{
		{
			name: "Amazon shipping notification",
			emailContent: `
				From: shipment-tracking@amazon.com
				Subject: Your package has shipped
				
				Your order #123-4567890-1234567 has shipped and is on the way.
				
				Items in this shipment:
				- Apple AirPods Pro (2nd Generation)
				- Delivery date: Tomorrow by 10 PM
				
				Track your package: TBA123456789012
			`,
			trackingNumber: "TBA123456789012",
			expectedDesc:   "Apple AirPods Pro (2nd Generation)",
		},
		{
			name: "UPS shipping notification",
			emailContent: `
				From: noreply@ups.com
				Subject: UPS Delivery Notice
				
				Your package from Best Buy is scheduled for delivery.
				
				Package details:
				- Samsung 65" QLED Smart TV
				- Tracking: 1Z999AA1234567890
				- Delivery: Friday, December 15
			`,
			trackingNumber: "1Z999AA1234567890",
			expectedDesc:   "Samsung 65\" QLED Smart TV",
		},
		{
			name: "FedEx shipping notification",
			emailContent: `
				From: tracking@fedex.com
				Subject: FedEx Package Delivery
				
				Your FedEx package from Nike is in transit.
				
				Contents: Air Jordan 1 High OG - Size 10
				Tracking number: 123456789012
				Expected delivery: Monday
			`,
			trackingNumber: "123456789012",
			expectedDesc:   "Air Jordan 1 High OG - Size 10",
		},
		{
			name: "USPS shipping notification",
			emailContent: `
				From: USPSTrackingUpdates@email.usps.com
				Subject: Expected Delivery Update
				
				Your package from Etsy seller is being delivered.
				
				Item: Handmade Wooden Chess Set
				Tracking: 9400111899562537624840
				Status: Out for delivery
			`,
			trackingNumber: "9400111899562537624840",
			expectedDesc:   "Handmade Wooden Chess Set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			
			mockLLM.On("ExtractDescription", ctx, tt.emailContent, tt.trackingNumber).
				Return(DescriptionResult{
					Description: tt.expectedDesc,
					Merchant:    "Test Merchant",
					Confidence:  0.9,
				}, nil).Once()

			result, err := extractor.ExtractDescription(ctx, tt.emailContent, tt.trackingNumber)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedDesc, result)

			mockLLM.AssertExpectations(t)
		})
	}
}

// Test for graceful handling of empty content
func TestSimplifiedDescriptionExtractor_EmptyContent(t *testing.T) {
	mockLLM := &MockLLMClient{}
	extractor := NewSimplifiedDescriptionExtractor(mockLLM, true)

	tests := []struct {
		name          string
		emailContent  string
		trackingNumber string
		expected      string
	}{
		{
			name:           "Empty email content",
			emailContent:   "",
			trackingNumber: "1Z999AA1234567890",
			expected:       "",
		},
		{
			name:           "Empty tracking number",
			emailContent:   "Your order has shipped.",
			trackingNumber: "",
			expected:       "",
		},
		{
			name:           "Both empty",
			emailContent:   "",
			trackingNumber: "",
			expected:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			
			if tt.emailContent != "" && tt.trackingNumber != "" {
				mockLLM.On("ExtractDescription", ctx, tt.emailContent, tt.trackingNumber).
					Return(DescriptionResult{
						Description: "",
						Merchant:    "",
						Confidence:  0.0,
					}, nil).Once()
			}

			result, err := extractor.ExtractDescription(ctx, tt.emailContent, tt.trackingNumber)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test for timeout handling
func TestSimplifiedDescriptionExtractor_Timeout(t *testing.T) {
	mockLLM := &MockLLMClient{}
	extractor := NewSimplifiedDescriptionExtractor(mockLLM, true)

	ctx := context.Background()
	emailContent := "Your order has shipped."
	trackingNumber := "1Z999AA1234567890"

	mockLLM.On("ExtractDescription", ctx, emailContent, trackingNumber).
		Return(DescriptionResult{}, errors.New("context deadline exceeded")).Once()

	result, err := extractor.ExtractDescription(ctx, emailContent, trackingNumber)

	assert.Error(t, err)
	assert.Equal(t, "", result)
	assert.Contains(t, err.Error(), "context deadline exceeded")

	mockLLM.AssertExpectations(t)
}

// Test for fallback behavior
func TestSimplifiedDescriptionExtractor_FallbackBehavior(t *testing.T) {
	mockLLM := &MockLLMClient{}
	extractor := NewSimplifiedDescriptionExtractor(mockLLM, true)

	ctx := context.Background()
	emailContent := "Your order has shipped."
	trackingNumber := "1Z999AA1234567890"

	// Test various error conditions
	errorTests := []struct {
		name        string
		llmError    error
		expected    string
		expectError bool
	}{
		{
			name:        "Network error",
			llmError:    errors.New("network error: connection refused"),
			expected:    "",
			expectError: true,
		},
		{
			name:        "API rate limit",
			llmError:    errors.New("rate limit exceeded"),
			expected:    "",
			expectError: true,
		},
		{
			name:        "Invalid API key",
			llmError:    errors.New("invalid API key"),
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM.On("ExtractDescription", ctx, emailContent, trackingNumber).
				Return(DescriptionResult{}, tt.llmError).Once()

			result, err := extractor.ExtractDescription(ctx, emailContent, trackingNumber)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expected, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			mockLLM.AssertExpectations(t)
		})
	}
}

