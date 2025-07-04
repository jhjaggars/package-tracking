package parser

import (
	"testing"
	"time"

	"package-tracking/internal/carriers"
	"package-tracking/internal/email"
)

func TestSimpleUPSExtraction(t *testing.T) {
	// Test just UPS extraction
	carrierFactory := carriers.NewClientFactory()
	config := &ExtractorConfig{
		EnableLLM:     false,
		MinConfidence: 0.5,
		DebugMode:     true,
	}
	
	extractor := NewTrackingExtractor(carrierFactory, config, &LLMConfig{Enabled: false})
	
	content := &email.EmailContent{
		PlainText: "Your package with tracking number 1Z999AA1234567890 has been shipped.",
		From:      "noreply@ups.com",
		Subject:   "UPS Shipment Notification",
		MessageID: "test-simple",
		Date:      time.Now(),
	}
	
	results, err := extractor.Extract(content)
	if err != nil {
		t.Fatalf("Extraction failed: %v", err)
	}
	
	t.Logf("Found %d results", len(results))
	for i, result := range results {
		t.Logf("Result %d: %s (%s) confidence=%f source=%s", 
			i+1, result.Number, result.Carrier, result.Confidence, result.Source)
	}
	
	// Just verify we found at least one valid result
	if len(results) == 0 {
		t.Error("No tracking numbers found")
	}
	
	// Check if we found the UPS number
	found := false
	for _, result := range results {
		if result.Number == "1Z999AA1234567890" && result.Carrier == "ups" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Expected UPS tracking number not found")
	}
}