package workers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func (m *MockSimplifiedTrackingExtractor) ExtractTrackingNumbers(content string) ([]TrackingCandidate, error) {
	args := m.Called(content)
	return args.Get(0).([]TrackingCandidate), args.Error(1)
}

// MockSimplifiedDescriptionExtractor represents a simplified description extractor
type MockSimplifiedDescriptionExtractor struct {
	mock.Mock
}

func (m *MockSimplifiedDescriptionExtractor) ExtractDescription(content string, trackingNumber string) (string, error) {
	args := m.Called(content, trackingNumber)
	return args.String(0), args.Error(1)
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

// Test types based on simplified architecture
type EmailMessage struct {
	ID      string
	From    string
	Subject string
	Body    string
	Date    time.Time
}

type TrackingCandidate struct {
	Number  string
	Carrier string
	Valid   bool
}

type ShipmentRequest struct {
	TrackingNumber string
	Carrier        string
	Description    string
}

// SimplifiedEmailProcessor represents the simplified email processor
type SimplifiedEmailProcessor struct {
	emailClient          EmailClient
	trackingExtractor    SimplifiedTrackingExtractor
	descriptionExtractor SimplifiedDescriptionExtractor
	shipmentCreator      ShipmentCreator
	stateManager         EmailStateManager
	daysToScan           int
	dryRun               bool
}

// Interfaces for the simplified architecture
type EmailClient interface {
	SearchEmails(ctx context.Context, query string, since time.Time) ([]EmailMessage, error)
	GetMessage(ctx context.Context, messageID string) (*EmailMessage, error)
	HealthCheck(ctx context.Context) error
	Close() error
}

type SimplifiedTrackingExtractor interface {
	ExtractTrackingNumbers(content string) ([]TrackingCandidate, error)
}

type SimplifiedDescriptionExtractor interface {
	ExtractDescription(content string, trackingNumber string) (string, error)
}

type ShipmentCreator interface {
	CreateShipment(ctx context.Context, req ShipmentRequest) error
}

type EmailStateManager interface {
	IsProcessed(messageID string) (bool, error)
	MarkProcessed(messageID string) error
}

// Test for simplified email processor initialization
func TestSimplifiedEmailProcessor_New(t *testing.T) {
	emailClient := &MockEmailClient{}
	trackingExtractor := &MockSimplifiedTrackingExtractor{}
	descriptionExtractor := &MockSimplifiedDescriptionExtractor{}
	shipmentCreator := &MockShipmentCreator{}
	stateManager := &MockEmailStateManager{}

	processor := &SimplifiedEmailProcessor{
		emailClient:          emailClient,
		trackingExtractor:    trackingExtractor,
		descriptionExtractor: descriptionExtractor,
		shipmentCreator:      shipmentCreator,
		stateManager:         stateManager,
		daysToScan:           30,
		dryRun:               false,
	}

	assert.NotNil(t, processor)
	assert.Equal(t, 30, processor.daysToScan)
	assert.False(t, processor.dryRun)
}

// Test for processing emails with tracking numbers
func TestSimplifiedEmailProcessor_ProcessEmails_WithTrackingNumbers(t *testing.T) {
	// Setup mocks
	emailClient := &MockEmailClient{}
	trackingExtractor := &MockSimplifiedTrackingExtractor{}
	descriptionExtractor := &MockSimplifiedDescriptionExtractor{}
	shipmentCreator := &MockShipmentCreator{}
	stateManager := &MockEmailStateManager{}

	processor := &SimplifiedEmailProcessor{
		emailClient:          emailClient,
		trackingExtractor:    trackingExtractor,
		descriptionExtractor: descriptionExtractor,
		shipmentCreator:      shipmentCreator,
		stateManager:         stateManager,
		daysToScan:           30,
		dryRun:               false,
	}

	// Mock email message
	emailMsg := EmailMessage{
		ID:      "msg123",
		From:    "shipping@ups.com",
		Subject: "Your package has shipped",
		Body:    "Your package with tracking number 1Z999AA1234567890 has shipped.",
		Date:    time.Now(),
	}

	// Mock tracking candidate
	trackingCandidate := TrackingCandidate{
		Number:  "1Z999AA1234567890",
		Carrier: "ups",
		Valid:   true,
	}

	// Setup expectations
	ctx := context.Background()

	emailClient.On("SearchEmails", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).
		Return([]EmailMessage{emailMsg}, nil)

	stateManager.On("IsProcessed", "msg123").Return(false, nil)

	trackingExtractor.On("ExtractTrackingNumbers", emailMsg.Body).
		Return([]TrackingCandidate{trackingCandidate}, nil)

	descriptionExtractor.On("ExtractDescription", emailMsg.Body, "1Z999AA1234567890").
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

	processor := &SimplifiedEmailProcessor{
		emailClient:          emailClient,
		trackingExtractor:    trackingExtractor,
		descriptionExtractor: descriptionExtractor,
		shipmentCreator:      shipmentCreator,
		stateManager:         stateManager,
		daysToScan:           30,
		dryRun:               false,
	}

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

	trackingExtractor.On("ExtractTrackingNumbers", emailMsg.Body).
		Return([]TrackingCandidate{}, nil)

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

	processor := &SimplifiedEmailProcessor{
		emailClient:          emailClient,
		trackingExtractor:    trackingExtractor,
		descriptionExtractor: descriptionExtractor,
		shipmentCreator:      shipmentCreator,
		stateManager:         stateManager,
		daysToScan:           30,
		dryRun:               true, // Dry run mode
	}

	// Mock email message
	emailMsg := EmailMessage{
		ID:      "msg123",
		From:    "shipping@ups.com",
		Subject: "Your package has shipped",
		Body:    "Your package with tracking number 1Z999AA1234567890 has shipped.",
		Date:    time.Now(),
	}

	// Mock tracking candidate
	trackingCandidate := TrackingCandidate{
		Number:  "1Z999AA1234567890",
		Carrier: "ups",
		Valid:   true,
	}

	// Setup expectations
	ctx := context.Background()

	emailClient.On("SearchEmails", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).
		Return([]EmailMessage{emailMsg}, nil)

	stateManager.On("IsProcessed", "msg123").Return(false, nil)

	trackingExtractor.On("ExtractTrackingNumbers", emailMsg.Body).
		Return([]TrackingCandidate{trackingCandidate}, nil)

	descriptionExtractor.On("ExtractDescription", emailMsg.Body, "1Z999AA1234567890").
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

	processor := &SimplifiedEmailProcessor{
		emailClient:          emailClient,
		trackingExtractor:    trackingExtractor,
		descriptionExtractor: descriptionExtractor,
		shipmentCreator:      shipmentCreator,
		stateManager:         stateManager,
		daysToScan:           30,
		dryRun:               false,
	}

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

// Placeholder for the actual ProcessEmails method that will be implemented
func (p *SimplifiedEmailProcessor) ProcessEmails(ctx context.Context) error {
	// This method will be implemented after the tests are written
	// For now, return nil to make tests compile
	return nil
}