package workers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
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
	simulateDelay    time.Duration
}

func (m *MockCarrierClient) Track(ctx context.Context, req *carriers.TrackingRequest) (*carriers.TrackingResponse, error) {
	// Simulate delay if configured
	if m.simulateDelay > 0 {
		select {
		case <-time.After(m.simulateDelay):
			// Continue execution after delay
		case <-ctx.Done():
			// Context was cancelled during delay
			return nil, ctx.Err()
		}
	}
	
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
	mu      sync.RWMutex // Add mutex for thread safety
}

func (m *MockCacheManager) Get(key interface{}) (*database.RefreshResponse, error) {
	if !m.enabled {
		return nil, fmt.Errorf("cache disabled")
	}
	keyStr := fmt.Sprintf("%v", key)
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if response, exists := m.cache[keyStr]; exists {
		return response, nil
	}
	return nil, fmt.Errorf("cache miss")
}

func (m *MockCacheManager) Set(key interface{}, response *database.RefreshResponse) error {
	if !m.enabled {
		return fmt.Errorf("cache disabled")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
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

// TestValidateTrackingTimeout tests validation with context timeout
func TestValidateTrackingTimeout(t *testing.T) {
	// Set up test database
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Set up mock carrier client that will simulate slow response (longer than timeout)
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
		simulateDelay: 100 * time.Millisecond, // Simulate delay longer than timeout
	}

	// Set up mock factory
	mockFactory := &MockCarrierFactory{
		client: mockCarrierClient,
	}

	// Create processor with very short validation timeout
	processor := setupValidationProcessor(t, db, mockFactory)
	processor.config.ValidationTimeout = 10 * time.Millisecond // Shorter than simulated delay

	ctx := context.Background()
	trackingNumber := "1Z999AA1234567890"
	carrier := "ups"

	// Validation call should timeout
	result, err := processor.validateTracking(ctx, trackingNumber, carrier)
	if err == nil {
		t.Error("Expected timeout error")
	}
	if result.IsValid {
		t.Error("Expected validation to be invalid due to timeout")
	}

	// Test with cancelled context
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result2, err2 := processor.validateTracking(cancelCtx, trackingNumber, carrier)
	if err2 == nil {
		t.Error("Expected context cancelled error")
	}
	if result2.IsValid {
		t.Error("Expected validation to be invalid due to cancelled context")
	}
}

// TestValidateTrackingLargeEventList tests validation with large number of events
func TestValidateTrackingLargeEventList(t *testing.T) {
	// Set up test database
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Generate large event list (1000 events)
	largeEventList := make([]carriers.TrackingEvent, 1000)
	for i := 0; i < 1000; i++ {
		largeEventList[i] = carriers.TrackingEvent{
			Timestamp:   time.Now().Add(-time.Duration(i) * time.Hour),
			Status:      carriers.StatusInTransit,
			Description: fmt.Sprintf("Event %d - Package in transit", i),
			Location:    fmt.Sprintf("Location %d", i),
			Details:     fmt.Sprintf("Additional details for event %d", i),
		}
	}

	// Set up mock carrier client with large event response
	mockCarrierClient := &MockCarrierClient{
		trackingResponse: &carriers.TrackingResponse{
			Results: []carriers.TrackingInfo{
				{
					TrackingNumber: "1Z999AA1234567890",
					Status:         carriers.StatusInTransit,
					Events:         largeEventList,
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

	// Validation should handle large event list efficiently
	result, err := processor.validateTracking(ctx, trackingNumber, carrier)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}
	if !result.IsValid {
		t.Error("Validation should be valid")
	}

	// Verify all events were processed
	if len(result.TrackingEvents) != 1000 {
		t.Errorf("Expected 1000 events, got %d", len(result.TrackingEvents))
	}

	// Verify events contain combined description with details
	for i, event := range result.TrackingEvents {
		expectedDesc := fmt.Sprintf("Event %d - Package in transit - Additional details for event %d", i, i)
		if event.Description != expectedDesc {
			t.Errorf("Event %d: expected description '%s', got '%s'", i, expectedDesc, event.Description)
			break // Only check first mismatch to avoid spam
		}
	}
}

// TestValidateTrackingCacheKeyCollision tests cache key collision scenarios
func TestValidateTrackingCacheKeyCollision(t *testing.T) {
	// Set up test database
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Set up mock carrier clients with different responses for same tracking number
	upsResponse := &carriers.TrackingResponse{
		Results: []carriers.TrackingInfo{
			{
				TrackingNumber: "123456789",
				Status:         carriers.StatusInTransit,
				Events: []carriers.TrackingEvent{
					{
						Timestamp:   time.Now(),
						Status:      carriers.StatusInTransit,
						Description: "UPS Package in transit",
						Location:    "UPS Sort facility",
					},
				},
			},
		},
	}

	fedexResponse := &carriers.TrackingResponse{
		Results: []carriers.TrackingInfo{
			{
				TrackingNumber: "123456789",
				Status:         carriers.StatusDelivered,
				Events: []carriers.TrackingEvent{
					{
						Timestamp:   time.Now(),
						Status:      carriers.StatusDelivered,
						Description: "FedEx Package delivered",
						Location:    "Customer address",
					},
				},
			},
		},
	}

	// Set up different mock clients for different carriers
	upsMockClient := &MockCarrierClient{trackingResponse: upsResponse}
	fedexMockClient := &MockCarrierClient{trackingResponse: fedexResponse}

	// Create processor with validation capability
	processor := setupValidationProcessor(t, db, nil)

	ctx := context.Background()
	trackingNumber := "123456789" // Same tracking number for both carriers

	// Test UPS validation
	processor.factory = &MockCarrierFactory{client: upsMockClient}
	upsResult, err := processor.validateTracking(ctx, trackingNumber, "ups")
	if err != nil {
		t.Fatalf("UPS validation failed: %v", err)
	}
	if !upsResult.IsValid {
		t.Error("UPS validation should be valid")
	}

	// Test FedEx validation with same tracking number
	processor.factory = &MockCarrierFactory{client: fedexMockClient}
	fedexResult, err := processor.validateTracking(ctx, trackingNumber, "fedex")
	if err != nil {
		t.Fatalf("FedEx validation failed: %v", err)
	}
	if !fedexResult.IsValid {
		t.Error("FedEx validation should be valid")
	}

	// Verify different cache keys were used and different results cached
	cache := processor.cacheManager.(*MockCacheManager)
	
	upsKey := fmt.Sprintf("validation:ups:%s", trackingNumber)
	fedexKey := fmt.Sprintf("validation:fedex:%s", trackingNumber)

	cache.mu.RLock()
	upsCache, upsExists := cache.cache[upsKey]
	fedexCache, fedexExists := cache.cache[fedexKey]
	cache.mu.RUnlock()

	if !upsExists {
		t.Error("UPS cache entry should exist")
	}
	if !fedexExists {
		t.Error("FedEx cache entry should exist")
	}

	// Verify cached results are different
	if upsExists && fedexExists {
		if len(upsCache.Events) != 1 || len(fedexCache.Events) != 1 {
			t.Error("Each carrier should have cached exactly 1 event")
		}
		if upsCache.Events[0].Description == fedexCache.Events[0].Description {
			t.Error("Cached events should be different for different carriers")
		}
	}
}

// TestValidateTrackingConcurrentRequests tests concurrent validation requests
func TestValidateTrackingConcurrentRequests(t *testing.T) {
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

	// Run concurrent validation requests
	numGoroutines := 10
	results := make(chan *ValidationResult, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			result, err := processor.validateTracking(ctx, fmt.Sprintf("%s-%d", trackingNumber, id), carrier)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}

	// Collect results
	successCount := 0
	errorCount := 0

	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			if result.IsValid {
				successCount++
			}
		case <-errors:
			errorCount++
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent validation results")
		}
	}

	// Verify all requests completed successfully
	if successCount != numGoroutines {
		t.Errorf("Expected %d successful validations, got %d (errors: %d)", numGoroutines, successCount, errorCount)
	}

	// Verify cache handled concurrent access correctly
	cache := processor.cacheManager.(*MockCacheManager)
	cache.mu.RLock()
	cacheSize := len(cache.cache)
	cache.mu.RUnlock()
	
	if cacheSize != numGoroutines {
		t.Errorf("Expected %d cache entries, got %d", numGoroutines, cacheSize)
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

// TestEmailShipmentLinking tests the email-shipment linking functionality
func TestEmailShipmentLinking(t *testing.T) {
	// Set up test database
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Set up mock API client that tracks shipment creation
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

	// Create test email entry
	emailEntry := &database.EmailBodyEntry{
		GmailMessageID:    "email-link-test",
		GmailThreadID:     "thread-1",
		From:              "carrier@example.com",
		Subject:           "Package shipped",
		Date:              time.Now(),
		BodyText:          "Your package 1Z999AA1234567890 has been shipped",
		InternalTimestamp: time.Now(),
		ScanMethod:        "time-based",
		ProcessedAt:       time.Now(),
		Status:            "processing",
	}

	// Store email in database first
	err = db.Emails.CreateOrUpdate(emailEntry)
	if err != nil {
		t.Fatalf("Failed to create email: %v", err)
	}

	// Create tracking info
	trackingInfo := []email.TrackingInfo{
		{
			Number:  "1Z999AA1234567890",
			Carrier: "ups",
			Source:  "test",
			Context: "email body",
		},
	}

	// Test email-shipment linking
	err = processor.createShipmentsAndLinks(trackingInfo, emailEntry)
	if err != nil {
		t.Fatalf("Failed to create shipments and links: %v", err)
	}

	// Verify shipment was created
	if len(mockAPIClient.createCalls) != 1 {
		t.Errorf("Expected 1 shipment creation call, got %d", len(mockAPIClient.createCalls))
	}

	if len(mockAPIClient.createCalls) > 0 {
		createdShipment := mockAPIClient.createCalls[0]
		if createdShipment.Number != "1Z999AA1234567890" {
			t.Errorf("Expected tracking number 1Z999AA1234567890, got %s", createdShipment.Number)
		}
		if createdShipment.Carrier != "ups" {
			t.Errorf("Expected carrier ups, got %s", createdShipment.Carrier)
		}
	}

	// Test linking with multiple tracking numbers
	multipleTrackingInfo := []email.TrackingInfo{
		{
			Number:  "1Z999AA1234567890",
			Carrier: "ups",
			Source:  "test",
			Context: "email body",
		},
		{
			Number:  "9999999999999999999999",
			Carrier: "fedex",
			Source:  "test",
			Context: "email body",
		},
	}

	// Reset API client call tracking
	mockAPIClient.createCalls = []email.TrackingInfo{}

	// Create another email entry for multiple tracking test
	emailEntry2 := &database.EmailBodyEntry{
		GmailMessageID:    "email-link-test-2",
		GmailThreadID:     "thread-2",
		From:              "carrier@example.com",
		Subject:           "Multiple packages shipped",
		Date:              time.Now(),
		BodyText:          "Your packages 1Z999AA1234567890 and 9999999999999999999999 have been shipped",
		InternalTimestamp: time.Now(),
		ScanMethod:        "time-based",
		ProcessedAt:       time.Now(),
		Status:            "processing",
	}

	err = db.Emails.CreateOrUpdate(emailEntry2)
	if err != nil {
		t.Fatalf("Failed to create second email: %v", err)
	}

	// Test with multiple tracking numbers
	err = processor.createShipmentsAndLinks(multipleTrackingInfo, emailEntry2)
	if err != nil {
		t.Fatalf("Failed to create shipments and links for multiple tracking: %v", err)
	}

	// Verify both shipments were created
	if len(mockAPIClient.createCalls) != 2 {
		t.Errorf("Expected 2 shipment creation calls, got %d", len(mockAPIClient.createCalls))
	}
}

// TestEmailShipmentLinkingWithDryRun tests email-shipment linking in dry run mode
func TestEmailShipmentLinkingWithDryRun(t *testing.T) {
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

	// Create processor with dry run enabled
	processor := setupValidationProcessorWithEmailClient(t, db, mockFactory, mockAPIClient)
	processor.config.DryRun = true

	// Create test email entry
	emailEntry := &database.EmailBodyEntry{
		GmailMessageID:    "email-dry-run-test",
		GmailThreadID:     "thread-1",
		From:              "carrier@example.com",
		Subject:           "Package shipped",
		Date:              time.Now(),
		BodyText:          "Your package 1Z999AA1234567890 has been shipped",
		InternalTimestamp: time.Now(),
		ScanMethod:        "time-based",
		ProcessedAt:       time.Now(),
		Status:            "processing",
	}

	// Create tracking info
	trackingInfo := []email.TrackingInfo{
		{
			Number:  "1Z999AA1234567890",
			Carrier: "ups",
			Source:  "test",
			Context: "email body",
		},
	}

	// Test email-shipment linking in dry run mode
	err = processor.createShipmentsAndLinks(trackingInfo, emailEntry)
	if err != nil {
		t.Fatalf("Failed to create shipments and links in dry run: %v", err)
	}

	// Verify no shipment was actually created in dry run mode
	if len(mockAPIClient.createCalls) != 0 {
		t.Errorf("Expected 0 shipment creation calls in dry run mode, got %d", len(mockAPIClient.createCalls))
	}
}

func setupValidationProcessorWithEmailClient(t *testing.T, db *database.DB, factory *MockCarrierFactory, apiClient *MockValidationAPIClient) *TimeBasedEmailProcessor {
	processor := setupValidationProcessor(t, db, factory)
	processor.apiClient = apiClient
	return processor
}