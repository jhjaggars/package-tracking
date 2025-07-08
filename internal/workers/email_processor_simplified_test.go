package workers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"package-tracking/internal/parser"
)

// MockEmailClient represents a simplified email client interface
type MockEmailClient struct {
	mock.Mock
}

func (m *MockEmailClient) SearchEmails(ctx context.Context, query string, since time.Time) ([]EmailMessage, error) {
	args := m.Called(ctx, query, since)
	return args.Get(0).([]EmailMessage), args.Error(1)
}

func (m *MockEmailClient) GetMessage(ctx context.Context, messageID string) (*EmailMessage, error) {
	args := m.Called(ctx, messageID)
	return args.Get(0).(*EmailMessage), args.Error(1)
}

func (m *MockEmailClient) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEmailClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockSimplifiedTrackingExtractor represents a simplified tracking number extractor
type MockSimplifiedTrackingExtractor struct {
	mock.Mock
}

func (m *MockSimplifiedTrackingExtractor) ExtractTrackingNumbers(content string) ([]parser.TrackingResult, error) {
	args := m.Called(content)
	return args.Get(0).([]parser.TrackingResult), args.Error(1)
}

// MockSimplifiedDescriptionExtractor represents a simplified description extractor
type MockSimplifiedDescriptionExtractor struct {
	mock.Mock
}

func (m *MockSimplifiedDescriptionExtractor) ExtractDescription(ctx context.Context, content string, trackingNumber string) (string, error) {
	args := m.Called(ctx, content, trackingNumber)
	return args.String(0), args.Error(1)
}

func (m *MockSimplifiedDescriptionExtractor) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

// MockShipmentCreator represents a simplified shipment creation interface
type MockShipmentCreator struct {
	mock.Mock
}

func (m *MockShipmentCreator) CreateShipment(ctx context.Context, req ShipmentRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

// MockEmailStateManager represents a simplified email state management interface
type MockEmailStateManager struct {
	mock.Mock
}

func (m *MockEmailStateManager) IsProcessed(messageID string) (bool, error) {
	args := m.Called(messageID)
	return args.Bool(0), args.Error(1)
}

func (m *MockEmailStateManager) MarkProcessed(messageID string) error {
	args := m.Called(messageID)
	return args.Error(0)
}

// Test interfaces for mocking
type SimplifiedTrackingExtractor interface {
	ExtractTrackingNumbers(content string) ([]parser.TrackingResult, error)
}

type SimplifiedDescriptionExtractor interface {
	ExtractDescription(ctx context.Context, content string, trackingNumber string) (string, error)
}

// Test for simplified email processor initialization
func TestSimplifiedEmailProcessor_New(t *testing.T) {
	emailClient := &MockEmailClient{}
	trackingExtractor := &MockSimplifiedTrackingExtractor{}
	descriptionExtractor := &MockSimplifiedDescriptionExtractor{}
	shipmentCreator := &MockShipmentCreator{}
	stateManager := &MockEmailStateManager{}

	processor := NewSimplifiedEmailProcessor(
		emailClient,
		trackingExtractor,
		descriptionExtractor,
		shipmentCreator,
		stateManager,
		30,
		false,
	)

	assert.NotNil(t, processor)
}

// Test for processing emails with tracking numbers
func TestSimplifiedEmailProcessor_ProcessEmails_WithTrackingNumbers(t *testing.T) {
	// Setup mocks
	emailClient := &MockEmailClient{}
	trackingExtractor := &MockSimplifiedTrackingExtractor{}
	descriptionExtractor := &MockSimplifiedDescriptionExtractor{}
	shipmentCreator := &MockShipmentCreator{}
	stateManager := &MockEmailStateManager{}

	processor := NewSimplifiedEmailProcessor(
		emailClient,
		trackingExtractor,
		descriptionExtractor,
		shipmentCreator,
		stateManager,
		30,
		false,
	)

	// Mock email message
	emailMsg := EmailMessage{
		ID:      "msg123",
		From:    "shipping@ups.com",
		Subject: "Your package has shipped",
		Body:    "Your package with tracking number 1Z999AA1234567890 has shipped.",
		Date:    time.Now(),
	}

	// Mock tracking result
	trackingResult := parser.TrackingResult{
		Number:  "1Z999AA1234567890",
		Carrier: "ups",
		Valid:   true,
	}

	// Setup expectations
	ctx := context.Background()

	emailClient.On("SearchEmails", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).
		Return([]EmailMessage{emailMsg}, nil)

	stateManager.On("IsProcessed", "msg123").Return(false, nil)

	trackingExtractor.On("ExtractTrackingNumbers", emailMsg.Subject+" "+emailMsg.Body).
		Return([]parser.TrackingResult{trackingResult}, nil)

	descriptionExtractor.On("ExtractDescription", ctx, emailMsg.Subject+" "+emailMsg.Body, "1Z999AA1234567890").
		Return("Test package description", nil)

	shipmentCreator.On("CreateShipment", ctx, ShipmentRequest{
		TrackingNumber: "1Z999AA1234567890",
		Carrier:        "ups",
		Description:    "Test package description",
	}).Return(nil)

	stateManager.On("MarkProcessed", "msg123").Return(nil)

	// Execute
	err := processor.ProcessEmails(ctx)

	// Verify
	assert.NoError(t, err)
	emailClient.AssertExpectations(t)
	trackingExtractor.AssertExpectations(t)
	descriptionExtractor.AssertExpectations(t)
	shipmentCreator.AssertExpectations(t)
	stateManager.AssertExpectations(t)
}

// Test for processing emails without tracking numbers
func TestSimplifiedEmailProcessor_ProcessEmails_NoTrackingNumbers(t *testing.T) {
	// Setup mocks
	emailClient := &MockEmailClient{}
	trackingExtractor := &MockSimplifiedTrackingExtractor{}
	descriptionExtractor := &MockSimplifiedDescriptionExtractor{}
	shipmentCreator := &MockShipmentCreator{}
	stateManager := &MockEmailStateManager{}

	processor := NewSimplifiedEmailProcessor(
		emailClient,
		trackingExtractor,
		descriptionExtractor,
		shipmentCreator,
		stateManager,
		30,
		false,
	)

	// Mock email message
	emailMsg := EmailMessage{
		ID:      "msg123",
		From:    "noreply@example.com",
		Subject: "Regular email",
		Body:    "This is a regular email with no tracking numbers.",
		Date:    time.Now(),
	}

	// Setup expectations
	ctx := context.Background()

	emailClient.On("SearchEmails", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).
		Return([]EmailMessage{emailMsg}, nil)

	stateManager.On("IsProcessed", "msg123").Return(false, nil)

	trackingExtractor.On("ExtractTrackingNumbers", emailMsg.Subject+" "+emailMsg.Body).
		Return([]parser.TrackingResult{}, nil)

	stateManager.On("MarkProcessed", "msg123").Return(nil)

	// Execute
	err := processor.ProcessEmails(ctx)

	// Verify
	assert.NoError(t, err)
	emailClient.AssertExpectations(t)
	trackingExtractor.AssertExpectations(t)
	stateManager.AssertExpectations(t)
	
	// Verify that description extractor and shipment creator were not called
	descriptionExtractor.AssertNotCalled(t, "ExtractDescription")
	shipmentCreator.AssertNotCalled(t, "CreateShipment")
}

// Test for dry run mode
func TestSimplifiedEmailProcessor_ProcessEmails_DryRun(t *testing.T) {
	// Setup mocks
	emailClient := &MockEmailClient{}
	trackingExtractor := &MockSimplifiedTrackingExtractor{}
	descriptionExtractor := &MockSimplifiedDescriptionExtractor{}
	shipmentCreator := &MockShipmentCreator{}
	stateManager := &MockEmailStateManager{}

	processor := NewSimplifiedEmailProcessor(
		emailClient,
		trackingExtractor,
		descriptionExtractor,
		shipmentCreator,
		stateManager,
		30,
		true, // Dry run mode
	)

	// Mock email message
	emailMsg := EmailMessage{
		ID:      "msg123",
		From:    "shipping@ups.com",
		Subject: "Your package has shipped",
		Body:    "Your package with tracking number 1Z999AA1234567890 has shipped.",
		Date:    time.Now(),
	}

	// Mock tracking result
	trackingResult := parser.TrackingResult{
		Number:  "1Z999AA1234567890",
		Carrier: "ups",
		Valid:   true,
	}

	// Setup expectations
	ctx := context.Background()

	emailClient.On("SearchEmails", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).
		Return([]EmailMessage{emailMsg}, nil)

	stateManager.On("IsProcessed", "msg123").Return(false, nil)

	trackingExtractor.On("ExtractTrackingNumbers", emailMsg.Subject+" "+emailMsg.Body).
		Return([]parser.TrackingResult{trackingResult}, nil)

	descriptionExtractor.On("ExtractDescription", ctx, emailMsg.Subject+" "+emailMsg.Body, "1Z999AA1234567890").
		Return("Test package description", nil)

	stateManager.On("MarkProcessed", "msg123").Return(nil)

	// Execute
	err := processor.ProcessEmails(ctx)

	// Verify
	assert.NoError(t, err)
	emailClient.AssertExpectations(t)
	trackingExtractor.AssertExpectations(t)
	descriptionExtractor.AssertExpectations(t)
	stateManager.AssertExpectations(t)
	
	// Verify that shipment creator was not called in dry run mode
	shipmentCreator.AssertNotCalled(t, "CreateShipment")
}

// Test for processing already processed emails
func TestSimplifiedEmailProcessor_ProcessEmails_AlreadyProcessed(t *testing.T) {
	// Setup mocks
	emailClient := &MockEmailClient{}
	trackingExtractor := &MockSimplifiedTrackingExtractor{}
	descriptionExtractor := &MockSimplifiedDescriptionExtractor{}
	shipmentCreator := &MockShipmentCreator{}
	stateManager := &MockEmailStateManager{}

	processor := NewSimplifiedEmailProcessor(
		emailClient,
		trackingExtractor,
		descriptionExtractor,
		shipmentCreator,
		stateManager,
		30,
		false,
	)

	// Mock email message
	emailMsg := EmailMessage{
		ID:      "msg123",
		From:    "shipping@ups.com",
		Subject: "Your package has shipped",
		Body:    "Your package with tracking number 1Z999AA1234567890 has shipped.",
		Date:    time.Now(),
	}

	// Setup expectations
	ctx := context.Background()

	emailClient.On("SearchEmails", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).
		Return([]EmailMessage{emailMsg}, nil)

	stateManager.On("IsProcessed", "msg123").Return(true, nil) // Already processed

	// Execute
	err := processor.ProcessEmails(ctx)

	// Verify
	assert.NoError(t, err)
	emailClient.AssertExpectations(t)
	stateManager.AssertExpectations(t)
	
	// Verify that other components were not called for already processed emails
	trackingExtractor.AssertNotCalled(t, "ExtractTrackingNumbers")
	descriptionExtractor.AssertNotCalled(t, "ExtractDescription")
	shipmentCreator.AssertNotCalled(t, "CreateShipment")
}

