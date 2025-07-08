package workers

import (
	"testing"

	"package-tracking/internal/email"
)

func TestRelevanceScorer_CalculateRelevanceScore(t *testing.T) {
	scorer := NewRelevanceScorer()

	tests := []struct {
		name     string
		message  *email.EmailMessage
		expected float64 // rough expected score range
		minScore float64
		maxScore float64
	}{
		{
			name: "High relevance - Amazon shipping email",
			message: &email.EmailMessage{
				From:    "auto-confirm@amazon.com",
				Subject: "Your package has shipped",
				Snippet: "Your order has been shipped via UPS tracking number 1Z999AA1234567890",
			},
			minScore: 0.7,
			maxScore: 1.0,
		},
		{
			name: "Medium relevance - Order confirmation",
			message: &email.EmailMessage{
				From:    "orders@example-store.com",
				Subject: "Order confirmation #12345",
				Snippet: "Thank you for your purchase. Your order will be processed shortly.",
			},
			minScore: 0.3,
			maxScore: 0.7,
		},
		{
			name: "Low relevance - Newsletter",
			message: &email.EmailMessage{
				From:    "newsletter@example.com",
				Subject: "Weekly deals and promotions",
				Snippet: "Check out our latest offers and discounts this week.",
			},
			minScore: 0.0,
			maxScore: 0.3,
		},
		{
			name: "High relevance - FedEx tracking update",
			message: &email.EmailMessage{
				From:    "noreply@fedex.com",
				Subject: "Your package is out for delivery",
				Snippet: "Tracking number 123456789012 is out for delivery today",
			},
			minScore: 0.8,
			maxScore: 1.0,
		},
		{
			name: "Zero relevance - Social media notification",
			message: &email.EmailMessage{
				From:    "notifications@social.com",
				Subject: "Someone liked your photo",
				Snippet: "John Doe liked your recent photo upload",
			},
			minScore: 0.0,
			maxScore: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scorer.CalculateRelevanceScore(tt.message)
			
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("CalculateRelevanceScore() = %v, want between %v and %v", 
					score, tt.minScore, tt.maxScore)
			}
			
			// Log the score breakdown for debugging
			breakdown := scorer.GetScoreBreakdown(tt.message)
			t.Logf("Score breakdown for %s: %+v", tt.name, breakdown)
		})
	}
}

func TestRelevanceScorer_SenderAnalysis(t *testing.T) {
	scorer := NewRelevanceScorer()

	tests := []struct {
		sender   string
		expected float64
		minScore float64
	}{
		{"auto-confirm@amazon.com", 0.8, 1.0},
		{"tracking@ups.com", 0.8, 1.0},
		{"noreply@fedex.com", 0.8, 1.0},
		{"orders@walmart.com", 0.5, 0.7},
		{"newsletter@randomstore.com", 0.0, 0.0},
		{"friend@gmail.com", 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.sender, func(t *testing.T) {
			score := scorer.scoreSender(tt.sender)
			
			if score < tt.minScore {
				t.Errorf("scoreSender(%s) = %v, want >= %v", tt.sender, score, tt.minScore)
			}
		})
	}
}

func TestRelevanceScorer_SubjectAnalysis(t *testing.T) {
	scorer := NewRelevanceScorer()

	tests := []struct {
		subject  string
		minScore float64
	}{
		{"Your package has shipped", 0.5},
		{"Tracking update for order #12345", 0.6},
		{"Out for delivery", 0.4},
		{"Order confirmation", 0.2},
		{"Weekly newsletter", 0.0},
		{"Happy birthday!", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			score := scorer.scoreSubject(tt.subject)
			
			if score < tt.minScore {
				t.Errorf("scoreSubject(%s) = %v, want >= %v", tt.subject, score, tt.minScore)
			}
		})
	}
}

func TestRelevanceScorer_IsRelevant(t *testing.T) {
	scorer := NewRelevanceScorer()

	highRelevanceEmail := &email.EmailMessage{
		From:    "shipping@amazon.com",
		Subject: "Your package has been delivered",
		Snippet: "Tracking number 1Z999AA1234567890 has been delivered to your address",
	}

	lowRelevanceEmail := &email.EmailMessage{
		From:    "marketing@randomcompany.com",
		Subject: "Check out our new products",
		Snippet: "We have exciting new products available in our store",
	}

	if !scorer.IsRelevant(highRelevanceEmail) {
		t.Error("Expected high relevance email to be relevant")
	}

	if scorer.IsRelevant(lowRelevanceEmail) {
		t.Error("Expected low relevance email to not be relevant")
	}
}

func TestRelevanceScorer_TrackingPatterns(t *testing.T) {
	scorer := NewRelevanceScorer()

	tests := []struct {
		name     string
		content  string
		minScore float64
	}{
		{
			name:     "UPS tracking number",
			content:  "Your package 1Z999AA1234567890 has shipped",
			minScore: 0.3,
		},
		{
			name:     "USPS tracking number",
			content:  "Track your package: 94001234567890123456",
			minScore: 0.3,
		},
		{
			name:     "FedEx tracking number",
			content:  "Tracking: 123456789012",
			minScore: 0.3,
		},
		{
			name:     "Amazon order number",
			content:  "Order 123-4567890-1234567 has been shipped",
			minScore: 0.3,
		},
		{
			name:     "No tracking patterns",
			content:  "Thank you for signing up for our newsletter",
			minScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scorer.scoreTrackingPatterns(tt.content)
			
			if score < tt.minScore {
				t.Errorf("scoreTrackingPatterns(%s) = %v, want >= %v", 
					tt.content, score, tt.minScore)
			}
		})
	}
}

func TestRelevanceScorer_GetScoreBreakdown(t *testing.T) {
	scorer := NewRelevanceScorer()

	email := &email.EmailMessage{
		From:    "shipping@amazon.com",
		Subject: "Your package has shipped",
		Snippet: "UPS tracking number 1Z999AA1234567890",
	}

	breakdown := scorer.GetScoreBreakdown(email)

	expectedFields := []string{
		"sender_score",
		"subject_score", 
		"content_score",
		"carrier_score",
		"tracking_score",
		"total_score",
	}

	for _, field := range expectedFields {
		if _, exists := breakdown[field]; !exists {
			t.Errorf("Expected breakdown to contain field: %s", field)
		}
	}

	// Total score should be reasonable
	if breakdown["total_score"] < 0.0 || breakdown["total_score"] > 1.0 {
		t.Errorf("Total score %v should be between 0.0 and 1.0", breakdown["total_score"])
	}
}

func BenchmarkRelevanceScorer_CalculateRelevanceScore(b *testing.B) {
	scorer := NewRelevanceScorer()
	
	email := &email.EmailMessage{
		From:    "shipping@amazon.com",
		Subject: "Your package has shipped via UPS",
		Snippet: "Your order has been shipped. Tracking number: 1Z999AA1234567890",
		PlainText: "Dear customer, your order #123-4567890-1234567 has been shipped via UPS. " +
			"You can track your package using tracking number 1Z999AA1234567890. " +
			"Estimated delivery date is tomorrow.",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scorer.CalculateRelevanceScore(email)
	}
}