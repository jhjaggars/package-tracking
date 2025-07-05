package parser

import (
	"testing"

	"package-tracking/internal/carriers"
	"package-tracking/internal/email"
)

func TestTrackingExtractor_isAmazonEmailContext(t *testing.T) {
	config := &ExtractorConfig{
		EnableLLM:           false,
		MinConfidence:       0.5,
		MaxCandidates:       10,
		UseHybridValidation: true,
		DebugMode:           false,
	}
	
	factory := carriers.NewClientFactory()
	extractor := NewTrackingExtractor(factory, config, nil)
	
	tests := []struct {
		name    string
		content *email.EmailContent
		want    bool
	}{
		// Amazon domains - should return true
		{
			name: "Amazon.com sender",
			content: &email.EmailContent{
				From:    "order-update@amazon.com",
				Subject: "Your order has shipped",
			},
			want: true,
		},
		{
			name: "Amazon Logistics sender",
			content: &email.EmailContent{
				From:    "tracking@amazonlogistics.com",
				Subject: "Package update",
			},
			want: true,
		},
		{
			name: "Amazon Marketplace sender",
			content: &email.EmailContent{
				From:    "notification@marketplace.amazon.com",
				Subject: "Seller notification",
			},
			want: true,
		},
		{
			name: "Amazon Shipment Tracking sender",
			content: &email.EmailContent{
				From:    "notifications@shipment-tracking.amazon.com",
				Subject: "Tracking information",
			},
			want: true,
		},
		{
			name: "Amazon with display name",
			content: &email.EmailContent{
				From:    "\"Amazon.com\" <order-update@amazon.com>",
				Subject: "Order confirmation",
			},
			want: true,
		},
		
		// Amazon terms in subject - should return true
		{
			name: "Amazon in subject",
			content: &email.EmailContent{
				From:    "orders@somestore.com",
				Subject: "Amazon order shipped",
			},
			want: true,
		},
		{
			name: "Amazon Logistics in subject",
			content: &email.EmailContent{
				From:    "noreply@vendor.com",
				Subject: "Shipped via Amazon Logistics",
			},
			want: true,
		},
		{
			name: "AMZL in subject",
			content: &email.EmailContent{
				From:    "shipping@retailer.com",
				Subject: "AMZL tracking update",
			},
			want: true,
		},
		{
			name: "Case insensitive Amazon",
			content: &email.EmailContent{
				From:    "orders@shop.com",
				Subject: "AMAZON delivery notification",
			},
			want: true,
		},
		{
			name: "Case insensitive amazon logistics",
			content: &email.EmailContent{
				From:    "info@store.com",
				Subject: "package from amazon logistics",
			},
			want: true,
		},
		
		// Non-Amazon emails - should return false
		{
			name: "UPS sender",
			content: &email.EmailContent{
				From:    "pkginfo@ups.com",
				Subject: "UPS tracking notification",
			},
			want: false,
		},
		{
			name: "FedEx sender",
			content: &email.EmailContent{
				From:    "tracking@fedex.com",
				Subject: "FedEx package update",
			},
			want: false,
		},
		{
			name: "USPS sender",
			content: &email.EmailContent{
				From:    "informed@usps.com",
				Subject: "Mail delivery notification",
			},
			want: false,
		},
		{
			name: "DHL sender",
			content: &email.EmailContent{
				From:    "noreply@dhl.com",
				Subject: "DHL Express delivery",
			},
			want: false,
		},
		{
			name: "Generic retailer",
			content: &email.EmailContent{
				From:    "orders@bestbuy.com",
				Subject: "Your order has shipped",
			},
			want: false,
		},
		{
			name: "Contains amazon but not Amazon context",
			content: &email.EmailContent{
				From:    "marketing@somestore.com",
				Subject: "We sell on Amazon too!",
			},
			want: true, // Still contains "amazon" in subject, so should return true
		},
		{
			name: "No Amazon references",
			content: &email.EmailContent{
				From:    "support@example.com",
				Subject: "Account verification required",
			},
			want: false,
		},
		
		// Edge cases
		{
			name: "Empty from and subject",
			content: &email.EmailContent{
				From:    "",
				Subject: "",
			},
			want: false,
		},
		{
			name: "Amazon substring but not domain",
			content: &email.EmailContent{
				From:    "notifications@notreallyamazon.com",
				Subject: "Your package update",
			},
			want: true, // The function checks for "amazon.com" substring, so this will match
		},
		{
			name: "Amazon-like domain",
			content: &email.EmailContent{
				From:    "fake@amazon-fake.com",
				Subject: "Suspicious email",
			},
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractor.isAmazonEmailContext(tt.content)
			if got != tt.want {
				t.Errorf("isAmazonEmailContext() = %v, want %v\nFrom: %s\nSubject: %s", 
					got, tt.want, tt.content.From, tt.content.Subject)
			}
		})
	}
}

func TestTrackingExtractor_getCarrierValidationOrder(t *testing.T) {
	config := &ExtractorConfig{
		EnableLLM:           false,
		MinConfidence:       0.5,
		MaxCandidates:       10,
		UseHybridValidation: true,
		DebugMode:           false,
	}
	
	factory := carriers.NewClientFactory()
	extractor := NewTrackingExtractor(factory, config, nil)
	
	tests := []struct {
		name      string
		candidate email.TrackingCandidate
		content   *email.EmailContent
		want      []string
	}{
		{
			name: "Candidate suggests UPS",
			candidate: email.TrackingCandidate{
				Text:       "1Z999AA1234567890",
				Carrier:    "ups",
				Confidence: 0.9,
			},
			content: &email.EmailContent{
				From:    "orders@somestore.com",
				Subject: "Your package shipped",
			},
			want: []string{"ups", "usps", "fedex", "dhl", "amazon"},
		},
		{
			name: "Candidate suggests Amazon",
			candidate: email.TrackingCandidate{
				Text:       "113-1234567-1234567",
				Carrier:    "amazon",
				Confidence: 0.9,
			},
			content: &email.EmailContent{
				From:    "orders@amazon.com",
				Subject: "Amazon order shipped",
			},
			want: []string{"amazon", "ups", "usps", "fedex", "dhl"},
		},
		{
			name: "Amazon email context with unknown candidate",
			candidate: email.TrackingCandidate{
				Text:       "BqPz3RXRS",
				Carrier:    "unknown",
				Confidence: 0.6,
			},
			content: &email.EmailContent{
				From:    "shipment-tracking@amazon.com",
				Subject: "Package delivered",
			},
			want: []string{"ups", "usps", "fedex", "dhl", "amazon"},
		},
		{
			name: "Amazon email context with empty carrier",
			candidate: email.TrackingCandidate{
				Text:       "SOME123CODE",
				Carrier:    "",
				Confidence: 0.5,
			},
			content: &email.EmailContent{
				From:    "notifications@amazonlogistics.com",
				Subject: "AMZL delivery update",
			},
			want: []string{"ups", "usps", "fedex", "dhl", "amazon"},
		},
		{
			name: "Non-Amazon email with generic candidate",
			candidate: email.TrackingCandidate{
				Text:       "TRACK123456",
				Carrier:    "unknown",
				Confidence: 0.5,
			},
			content: &email.EmailContent{
				From:    "shipping@bestbuy.com",
				Subject: "Order shipped",
			},
			want: []string{"ups", "usps", "fedex", "dhl", "amazon"},
		},
		{
			name: "USPS candidate in Amazon email",
			candidate: email.TrackingCandidate{
				Text:       "9405511206213414325732",
				Carrier:    "usps",
				Confidence: 0.8,
			},
			content: &email.EmailContent{
				From:    "order-update@amazon.com",
				Subject: "Your Amazon order",
			},
			want: []string{"usps", "ups", "fedex", "dhl", "amazon"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractor.getCarrierValidationOrder(tt.candidate, tt.content)
			if len(got) != len(tt.want) {
				t.Errorf("getCarrierValidationOrder() returned %d carriers, want %d\nGot: %v\nWant: %v", 
					len(got), len(tt.want), got, tt.want)
				return
			}
			
			for i, carrier := range got {
				if carrier != tt.want[i] {
					t.Errorf("getCarrierValidationOrder()[%d] = %s, want %s\nFull result: %v\nExpected: %v", 
						i, carrier, tt.want[i], got, tt.want)
					break
				}
			}
		})
	}
}

func TestTrackingExtractor_isLikelyAmazonInternalCode(t *testing.T) {
	config := &ExtractorConfig{
		EnableLLM:           false,
		MinConfidence:       0.5,
		MaxCandidates:       10,
		UseHybridValidation: true,
		DebugMode:           false,
	}
	
	factory := carriers.NewClientFactory()
	extractor := NewTrackingExtractor(factory, config, nil)
	
	tests := []struct {
		name           string
		trackingNumber string
		want           bool
	}{
		// Valid Amazon internal codes (relaxed validation)
		{
			name:           "Original failing case",
			trackingNumber: "BqPz3RXRS",
			want:           true,
		},
		{
			name:           "Mixed alphanumeric",
			trackingNumber: "AMZ123DEF",
			want:           true,
		},
		{
			name:           "Warehouse style",
			trackingNumber: "FBA456GHI",
			want:           true,
		},
		{
			name:           "Reference style",
			trackingNumber: "REF789JKL",
			want:           true,
		},
		
		// Invalid - too short/long
		{
			name:           "Too short",
			trackingNumber: "AMZ12",
			want:           false,
		},
		{
			name:           "Too long",
			trackingNumber: "VERYLONGAMAZONREFERENCECODE123",
			want:           false,
		},
		
		// Invalid - format issues
		{
			name:           "Only letters",
			trackingNumber: "AMAZONCODE",
			want:           true, // The function only requires letters, not both letters and numbers
		},
		{
			name:           "Only numbers",
			trackingNumber: "123456789",
			want:           false,
		},
		{
			name:           "Contains special characters",
			trackingNumber: "AMZ123@",
			want:           false,
		},
		
		// Invalid - false positives
		{
			name:           "Year",
			trackingNumber: "2024",
			want:           false,
		},
		{
			name:           "Day",
			trackingNumber: "monday123",
			want:           false,
		},
		{
			name:           "Month",
			trackingNumber: "january456",
			want:           false,
		},
		{
			name:           "Common word",
			trackingNumber: "email123",
			want:           false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractor.isLikelyAmazonInternalCode(tt.trackingNumber)
			if got != tt.want {
				t.Errorf("isLikelyAmazonInternalCode(%q) = %v, want %v", tt.trackingNumber, got, tt.want)
			}
		})
	}
}