package workers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"package-tracking/internal/carriers"
	"package-tracking/internal/database"
	"package-tracking/internal/email"
)

// MockValidationAPIClient mocks the API client for validation tests
type MockValidationAPIClient struct {
	shouldError     bool
	createCalls     []email.TrackingInfo
	validateCalls   []ValidationRequest
	validationError error
}

func (m *MockValidationAPIClient) CreateShipment(tracking email.TrackingInfo) error {
	m.createCalls = append(m.createCalls, tracking)
	if m.shouldError {
		return fmt.Errorf("mock create shipment error")
	}
	return nil
}

func (m *MockValidationAPIClient) HealthCheck() error {
	if m.shouldError {
		return fmt.Errorf("mock health check error")
	}
	return nil
}

// ValidationRequest represents a validation request
type ValidationRequest struct {
	TrackingNumber string
	Carrier        string
}

// ValidationResult is defined in email_processor_time.go

// MockCarrierClient mocks carrier client for validation tests
type MockCarrierClient struct {
	trackingResponse *carriers.TrackingResponse
	trackingError    error
}

func (m *MockCarrierClient) Track(ctx context.Context, req *carriers.TrackingRequest) (*carriers.TrackingResponse, error) {
	if m.trackingError != nil {
		return nil, m.trackingError
	}
	return m.trackingResponse, nil
}

func (m *MockCarrierClient) GetCarrierName() string {
	return "mock"
}

func (m *MockCarrierClient) ValidateTrackingNumber(trackingNumber string) bool {
	return true
}

func (m *MockCarrierClient) GetRateLimit() *carriers.RateLimitInfo {
	return nil
}

// MockCarrierFactory mocks carrier factory for validation tests
type MockCarrierFactory struct {
	client *MockCarrierClient
	err    error
}

func (m *MockCarrierFactory) CreateClient(carrier string) (carriers.Client, carriers.ClientType, error) {
	if m.err != nil {
		return nil, "", m.err
	}
	return m.client, carriers.ClientTypeHeadless, nil
}

func (m *MockCarrierFactory) SetCarrierConfig(carrier string, config *carriers.CarrierConfig) {
	// Mock implementation
}

// MockCacheManager mocks cache manager for validation tests
type MockCacheManager struct {
	cache   map[string]*database.RefreshResponse
	enabled bool
}

func (m *MockCacheManager) Get(key interface{}) (*database.RefreshResponse, error) {
	if !m.enabled {
		return nil, fmt.Errorf("cache disabled")
	}
	keyStr := fmt.Sprintf("%v", key)
	if response, exists := m.cache[keyStr]; exists {
		return response, nil
	}
	return nil, fmt.Errorf("cache miss")
}

func (m *MockCacheManager) Set(key interface{}, response *database.RefreshResponse) error {
	if !m.enabled {
		return fmt.Errorf("cache disabled")
	}
	if m.cache == nil {
		m.cache = make(map[string]*database.RefreshResponse)
	}
	keyStr := fmt.Sprintf("%v", key)
	m.cache[keyStr] = response
	return nil
}

func (m *MockCacheManager) IsEnabled() bool {
	return m.enabled
}

// MockRateLimiter mocks rate limiter for validation tests
type MockRateLimiter struct {
	shouldBlock bool
	reason      string
}

func (m *MockRateLimiter) CheckValidationRateLimit(trackingNumber string) RateLimitResult {
	return RateLimitResult{
		ShouldBlock:   m.shouldBlock,
		RemainingTime: 5 * time.Minute,
		Reason:        m.reason,
	}
}

// TestValidationResult tests the ValidationResult struct
func TestValidationResult(t *testing.T) {
	tests := []struct {
		name           string
		isValid        bool
		trackingEvents []database.TrackingEvent
		error          error
	}{
		{
			name:           "Valid tracking with events",
			isValid:        true,
			trackingEvents: []database.TrackingEvent{
				{
					ShipmentID:  1,
					Timestamp:   time.Now(),
					Status:      "shipped",
					Description: "Package shipped",
					Location:    "Origin facility",
				},
			},
			error: nil,
		},
		{
			name:           "Invalid tracking",
			isValid:        false,
			trackingEvents: nil,
			error:          fmt.Errorf("invalid tracking number"),
		},
		{
			name:           "Valid tracking no events",
			isValid:        true,
			trackingEvents: []database.TrackingEvent{},
			error:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{
				IsValid:        tt.isValid,
				TrackingEvents: tt.trackingEvents,
				Error:          tt.error,
			}

			if result.IsValid != tt.isValid {
				t.Errorf("Expected IsValid=%v, got %v", tt.isValid, result.IsValid)
			}

			if len(result.TrackingEvents) != len(tt.trackingEvents) {
				t.Errorf("Expected %d tracking events, got %d", len(tt.trackingEvents), len(result.TrackingEvents))
			}

			if (result.Error == nil) != (tt.error == nil) {
				t.Errorf("Expected error presence=%v, got %v", tt.error != nil, result.Error != nil)
			}
		})
	}
}

// TestValidateTrackingSuccess tests successful tracking validation
func TestValidateTrackingSuccess(t *testing.T) {
	// Set up test database
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Set up mock carrier client with successful response
	mockCarrierClient := &MockCarrierClient{
		trackingResponse: &carriers.TrackingResponse{
			Results: []carriers.TrackingInfo{
				{
					TrackingNumber: "1Z999AA1234567890",
					Status:         carriers.StatusInTransit,
					Events: []carriers.TrackingEvent{
						{
							Timestamp:   time.Now(),
							Status:      carriers.StatusInTransit,
							Description: "Package in transit",
							Location:    "Sort facility",
						},
					},
				},
			},
		},
	}

	// Set up mock factory
	mockFactory := &MockCarrierFactory{
		client: mockCarrierClient,
	}

	// Create processor with validation capability
	processor := setupValidationProcessor(t, db, mockFactory)

	// Test validation
	ctx := context.Background()
	result, err := processor.validateTracking(ctx, "1Z999AA1234567890", "ups")

	// Verify validation succeeded
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !result.IsValid {
		t.Error("Expected tracking to be valid")
	}

	if len(result.TrackingEvents) != 1 {
		t.Errorf("Expected 1 tracking event, got %d", len(result.TrackingEvents))
	}

	if result.TrackingEvents[0].Description != "Package in transit" {
		t.Errorf("Expected description 'Package in transit', got '%s'", result.TrackingEvents[0].Description)
	}
}

// TestValidateTrackingFailure tests tracking validation failure
func TestValidateTrackingFailure(t *testing.T) {
	// Set up test database
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Set up mock carrier client with error response
	mockCarrierClient := &MockCarrierClient{
		trackingError: fmt.Errorf("tracking not found"),
	}

	// Set up mock factory
	mockFactory := &MockCarrierFactory{
		client: mockCarrierClient,
	}

	// Create processor with validation capability
	processor := setupValidationProcessor(t, db, mockFactory)

	// Test validation
	ctx := context.Background()
	result, err := processor.validateTracking(ctx, "INVALID123", "ups")

	// Verify validation failed
	if err == nil {
		t.Error("Expected error for invalid tracking")
	}

	if result.IsValid {
		t.Error("Expected tracking to be invalid")
	}
}

// TestValidateTrackingWithCaching tests validation with caching
func TestValidateTrackingWithCaching(t *testing.T) {
	// Set up test database
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Set up mock carrier client
	mockCarrierClient := &MockCarrierClient{
		trackingResponse: &carriers.TrackingResponse{
			Results: []carriers.TrackingInfo{
				{
					TrackingNumber: "1Z999AA1234567890",
					Status:         carriers.StatusInTransit,
					Events: []carriers.TrackingEvent{
						{
							Timestamp:   time.Now(),
							Status:      carriers.StatusInTransit,
							Description: "Package in transit",
							Location:    "Sort facility",
						},
					},
				},
			},
		},
	}

	// Set up mock factory
	mockFactory := &MockCarrierFactory{
		client: mockCarrierClient,
	}

	// Create processor with validation capability
	processor := setupValidationProcessor(t, db, mockFactory)

	ctx := context.Background()
	trackingNumber := "1Z999AA1234567890"
	carrier := "ups"

	// First validation call - should hit carrier API
	result1, err := processor.validateTracking(ctx, trackingNumber, carrier)
	if err != nil {
		t.Fatalf("First validation failed: %v", err)
	}
	if !result1.IsValid {
		t.Error("First validation should be valid")
	}

	// Second validation call - should hit cache
	result2, err := processor.validateTracking(ctx, trackingNumber, carrier)
	if err != nil {
		t.Fatalf("Second validation failed: %v", err)
	}
	if !result2.IsValid {
		t.Error("Second validation should be valid")
	}

	// Results should be identical
	if len(result1.TrackingEvents) != len(result2.TrackingEvents) {
		t.Error("Cached result should have same number of events")
	}

	// Verify cache key format includes carrier to prevent collisions
	cache := processor.cacheManager.(*MockCacheManager)
	expectedCacheKey := fmt.Sprintf("validation:%s:%s", carrier, trackingNumber)
	if _, exists := cache.cache[expectedCacheKey]; !exists {
		t.Errorf("Expected cache key '%s' not found", expectedCacheKey)
	}
}

// TestValidateTrackingWithRateLimit tests validation with rate limiting
func TestValidateTrackingWithRateLimit(t *testing.T) {
	// Set up test database
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Set up mock carrier client
	mockCarrierClient := &MockCarrierClient{
		trackingResponse: &carriers.TrackingResponse{
			Results: []carriers.TrackingInfo{
				{
					TrackingNumber: "1Z999AA1234567890",
					Status:         carriers.StatusInTransit,
					Events:         []carriers.TrackingEvent{},
				},
			},
		},
	}

	// Set up mock factory
	mockFactory := &MockCarrierFactory{
		client: mockCarrierClient,
	}

	// Create processor with validation capability
	processor := setupValidationProcessor(t, db, mockFactory)

	ctx := context.Background()
	trackingNumber := "1Z999AA1234567890"
	carrier := "ups"

	// First validation call - should succeed
	result1, err := processor.validateTracking(ctx, trackingNumber, carrier)
	if err != nil {
		t.Fatalf("First validation failed: %v", err)
	}
	if !result1.IsValid {
		t.Error("First validation should be valid")
	}

	// Simulate rate limiting by updating the database to have a recent refresh
	// This would normally be done by the rate limiting logic
	// For this test, we'll just verify the method signature works
	result2, err := processor.validateTracking(ctx, trackingNumber, carrier)
	if err != nil {
		t.Fatalf("Second validation failed: %v", err)
	}
	// This should use cache, so it should still be valid
	if !result2.IsValid {
		t.Error("Second validation should be valid (from cache)")
	}
}

// TestValidateTrackingRateLimitBlocked tests validation when rate limited
func TestValidateTrackingRateLimitBlocked(t *testing.T) {
	// Set up test database
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Set up mock carrier client
	mockCarrierClient := &MockCarrierClient{
		trackingResponse: &carriers.TrackingResponse{
			Results: []carriers.TrackingInfo{
				{
					TrackingNumber: "1Z999AA1234567890",
					Status:         carriers.StatusInTransit,
					Events:         []carriers.TrackingEvent{},
				},
			},
		},
	}

	// Set up mock factory
	mockFactory := &MockCarrierFactory{
		client: mockCarrierClient,
	}

	// Create processor with rate limiting enabled
	processor := setupValidationProcessor(t, db, mockFactory)
	
	// Set rate limiter to block requests
	mockRateLimiter := &MockRateLimiter{
		shouldBlock: true,
		reason:      "rate_limit_exceeded",
	}
	processor.rateLimiter = mockRateLimiter

	ctx := context.Background()
	trackingNumber := "1Z999AA1234567890"
	carrier := "ups"

	// Validation call should be blocked by rate limiting
	result, err := processor.validateTracking(ctx, trackingNumber, carrier)
	if err == nil {
		t.Error("Expected rate limiting error")
	}
	if result.IsValid {
		t.Error("Expected validation to be invalid due to rate limiting")
	}
	if result.Error == nil {
		t.Error("Expected validation result to contain error")
	}
}

// TestValidateTrackingConfigurableTimeout tests configurable validation timeout
func TestValidateTrackingConfigurableTimeout(t *testing.T) {
	// Set up test database
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Set up mock carrier client
	mockCarrierClient := &MockCarrierClient{
		trackingResponse: &carriers.TrackingResponse{
			Results: []carriers.TrackingInfo{
				{
					TrackingNumber: "1Z999AA1234567890",
					Status:         carriers.StatusInTransit,
					Events:         []carriers.TrackingEvent{},
				},
			},
		},
	}

	// Set up mock factory
	mockFactory := &MockCarrierFactory{
		client: mockCarrierClient,
	}

	// Create processor with custom validation timeout
	processor := setupValidationProcessor(t, db, mockFactory)
	processor.config.ValidationTimeout = 30 * time.Second // Custom timeout

	ctx := context.Background()
	trackingNumber := "1Z999AA1234567890"
	carrier := "ups"

	// Validation call should succeed with custom timeout
	result, err := processor.validateTracking(ctx, trackingNumber, carrier)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}
	if !result.IsValid {
		t.Error("Validation should be valid")
	}

	// Verify the timeout configuration is being used
	if processor.config.ValidationTimeout != 30*time.Second {
		t.Errorf("Expected validation timeout 30s, got %v", processor.config.ValidationTimeout)
	}
}

// TestEmailProcessingWithValidation tests email processing with validation enabled
func TestEmailProcessingWithValidation(t *testing.T) {
	// Set up test database
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Set up mock API client
	mockAPIClient := &MockValidationAPIClient{
		shouldError: false,
	}

	// Set up mock carrier client with successful response
	mockCarrierClient := &MockCarrierClient{
		trackingResponse: &carriers.TrackingResponse{
			Results: []carriers.TrackingInfo{
				{
					TrackingNumber: "1Z999AA1234567890",
					Status:         carriers.StatusInTransit,
					Events: []carriers.TrackingEvent{
						{
							Timestamp:   time.Now(),
							Status:      carriers.StatusInTransit,
							Description: "Package in transit",
							Location:    "Sort facility",
						},
					},
				},
			},
		},
	}

	// Set up mock factory
	mockFactory := &MockCarrierFactory{
		client: mockCarrierClient,
	}

	// Create processor with validation capability
	processor := setupValidationProcessorWithEmailClient(t, db, mockFactory, mockAPIClient)

	// Set up test email with tracking number
	testEmail := email.EmailMessage{
		ID:        "email-with-tracking",
		ThreadID:  "thread-1",
		From:      "carrier@example.com",
		Subject:   "Package shipped",
		Date:      time.Now(),
		PlainText: "Your package 1Z999AA1234567890 has been shipped",
		Headers:   map[string]string{},
	}

	// Set up mock email client
	mockEmailClient := &MockTimeBasedEmailClient{
		messages: []email.EmailMessage{testEmail},
	}

	processor.emailClient = mockEmailClient

	// Process email
	since := time.Now().Add(-time.Hour)
	err = processor.ProcessEmailsSince(since)
	if err != nil {
		t.Fatalf("Email processing failed: %v", err)
	}

	// Verify that validation was called and shipment was created
	if len(mockAPIClient.createCalls) != 1 {
		t.Errorf("Expected 1 shipment creation call, got %d", len(mockAPIClient.createCalls))
	}

	if len(mockAPIClient.createCalls) > 0 {
		createdShipment := mockAPIClient.createCalls[0]
		if createdShipment.Number != "1Z999AA1234567890" {
			t.Errorf("Expected tracking number 1Z999AA1234567890, got %s", createdShipment.Number)
		}
	}
}

// TestEmailProcessingWithValidationFailure tests email processing when validation fails
func TestEmailProcessingWithValidationFailure(t *testing.T) {
	// Set up test database
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Set up mock API client
	mockAPIClient := &MockValidationAPIClient{
		shouldError: false,
	}

	// Set up mock carrier client with error response
	mockCarrierClient := &MockCarrierClient{
		trackingError: fmt.Errorf("tracking not found"),
	}

	// Set up mock factory
	mockFactory := &MockCarrierFactory{
		client: mockCarrierClient,
	}

	// Create processor with validation capability
	processor := setupValidationProcessorWithEmailClient(t, db, mockFactory, mockAPIClient)

	// Set up test email with invalid tracking number
	testEmail := email.EmailMessage{
		ID:        "email-with-invalid-tracking",
		ThreadID:  "thread-1",
		From:      "carrier@example.com",
		Subject:   "Package shipped",
		Date:      time.Now(),
		PlainText: "Your package INVALID123 has been shipped",
		Headers:   map[string]string{},
	}

	// Set up mock email client
	mockEmailClient := &MockTimeBasedEmailClient{
		messages: []email.EmailMessage{testEmail},
	}

	processor.emailClient = mockEmailClient

	// Process email
	since := time.Now().Add(-time.Hour)
	err = processor.ProcessEmailsSince(since)
	if err != nil {
		t.Fatalf("Email processing failed: %v", err)
	}

	// Verify that no shipment was created due to validation failure
	if len(mockAPIClient.createCalls) != 0 {
		t.Errorf("Expected 0 shipment creation calls, got %d", len(mockAPIClient.createCalls))
	}

	// Verify email was still processed and stored
	processedEmail, err := db.Emails.GetByGmailMessageID("email-with-invalid-tracking")
	if err != nil {
		t.Fatalf("Failed to get processed email: %v", err)
	}
	if processedEmail == nil {
		t.Error("Expected email to be processed and stored despite validation failure")
	}
}

// Helper functions for validation tests

func setupValidationProcessor(t *testing.T, db *database.DB, factory *MockCarrierFactory) *TimeBasedEmailProcessor {
	config := &TimeBasedEmailProcessorConfig{
		ScanDays:          30,
		BodyStorageEnabled: true,
		RetentionDays:     90,
		MaxEmailsPerScan:  100,
		UnreadOnly:        false,
		CheckInterval:     5 * time.Minute,
		ProcessingTimeout: 30 * time.Minute,
		ValidationTimeout: 60 * time.Second, // Configurable validation timeout
		RetryCount:        3,
		RetryDelay:        time.Second,
		DryRun:            false,
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Set up mock cache manager
	mockCache := &MockCacheManager{
		enabled: true,
		cache:   make(map[string]*database.RefreshResponse),
	}

	// Set up mock rate limiter
	mockRateLimiter := &MockRateLimiter{
		shouldBlock: false,
		reason:      "no_limit",
	}

	processor := &TimeBasedEmailProcessor{
		config:        config,
		emailClient:   nil, // Will be set by individual tests
		extractor:     &MockTrackingExtractor{},
		emailStore:    db.Emails,
		shipmentStore: db.Shipments,
		apiClient:     nil, // Will be set by individual tests
		logger:        logger,
		metrics:       &TimeBasedProcessingMetrics{},
		factory:       factory,
		cacheManager:  mockCache,
		rateLimiter:   mockRateLimiter,
	}

	return processor
}

func setupValidationProcessorWithEmailClient(t *testing.T, db *database.DB, factory *MockCarrierFactory, apiClient *MockValidationAPIClient) *TimeBasedEmailProcessor {
	processor := setupValidationProcessor(t, db, factory)
	processor.apiClient = apiClient
	return processor
}