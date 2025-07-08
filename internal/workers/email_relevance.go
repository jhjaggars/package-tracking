package workers

import (
	"regexp"
	"strings"

	"package-tracking/internal/email"
)

// RelevanceScorer calculates shipping relevance scores for emails
type RelevanceScorer struct {
	// Compiled regex patterns for performance
	shippingCarriers    *regexp.Regexp
	trackingPatterns    *regexp.Regexp
	shippingKeywords    *regexp.Regexp
	commercialKeywords  *regexp.Regexp
	shippingVerbs       *regexp.Regexp
	deliveryKeywords    *regexp.Regexp
}

// NewRelevanceScorer creates a new relevance scorer with pre-compiled patterns
func NewRelevanceScorer() *RelevanceScorer {
	return &RelevanceScorer{
		shippingCarriers: regexp.MustCompile(`(?i)\b(ups|fedex|usps|dhl|amazon|postal|express|shipment|delivery|tracking)\b`),
		trackingPatterns: regexp.MustCompile(`(?i)\b(track|tracking|shipment|order|package|delivery|shipped|dispatched)\b`),
		shippingKeywords: regexp.MustCompile(`(?i)\b(shipping|shipment|package|parcel|delivery|order|confirmation|receipt|invoice)\b`),
		commercialKeywords: regexp.MustCompile(`(?i)\b(order|purchase|payment|receipt|confirmation|invoice|billing)\b`),
		shippingVerbs: regexp.MustCompile(`(?i)\b(shipped|dispatched|delivered|tracking|en route|in transit|out for delivery)\b`),
		deliveryKeywords: regexp.MustCompile(`(?i)\b(delivered|delivery|arrival|received|pickup|collection)\b`),
	}
}

// CalculateRelevanceScore calculates a 0.0-1.0 relevance score for shipping emails
func (r *RelevanceScorer) CalculateRelevanceScore(msg *email.EmailMessage) float64 {
	score := 0.0
	
	// Combine all text content for analysis
	textContent := strings.Join([]string{
		msg.Subject,
		msg.From,
		msg.Snippet,
		// Note: PlainText and HTMLText might be empty for metadata-only messages
		msg.PlainText,
		msg.HTMLText,
	}, " ")
	
	textContent = strings.ToLower(textContent)
	
	// 1. Sender analysis (30% weight)
	score += r.scoreSender(msg.From) * 0.3
	
	// 2. Subject analysis (25% weight)
	score += r.scoreSubject(msg.Subject) * 0.25
	
	// 3. Content analysis (20% weight)
	score += r.scoreContent(textContent) * 0.2
	
	// 4. Carrier mention analysis (15% weight)
	score += r.scoreCarrierMentions(textContent) * 0.15
	
	// 5. Tracking pattern analysis (10% weight)
	score += r.scoreTrackingPatterns(textContent) * 0.1
	
	// Ensure score is between 0.0 and 1.0
	if score > 1.0 {
		score = 1.0
	} else if score < 0.0 {
		score = 0.0
	}
	
	return score
}

// scoreSender analyzes the sender for shipping relevance
func (r *RelevanceScorer) scoreSender(from string) float64 {
	from = strings.ToLower(from)
	
	// High confidence senders
	highConfidenceSenders := []string{
		"amazon", "ups", "fedex", "usps", "dhl", "shopify", "tracking",
		"shipping", "delivery", "order", "noreply", "auto-reply",
	}
	
	for _, sender := range highConfidenceSenders {
		if strings.Contains(from, sender) {
			return 1.0
		}
	}
	
	// Medium confidence senders (e-commerce platforms)
	mediumConfidenceSenders := []string{
		"ebay", "etsy", "walmart", "target", "bestbuy", "homedepot",
		"lowes", "costco", "samsclub", "macys", "nordstrom",
	}
	
	for _, sender := range mediumConfidenceSenders {
		if strings.Contains(from, sender) {
			return 0.7
		}
	}
	
	// Check for generic shipping patterns
	if r.shippingCarriers.MatchString(from) {
		return 0.8
	}
	
	return 0.0
}

// scoreSubject analyzes the subject line for shipping relevance
func (r *RelevanceScorer) scoreSubject(subject string) float64 {
	subject = strings.ToLower(subject)
	score := 0.0
	
	// Direct shipping indicators
	directIndicators := []string{
		"shipped", "tracking", "delivery", "delivered", "out for delivery",
		"in transit", "package", "order confirmation", "shipment",
	}
	
	for _, indicator := range directIndicators {
		if strings.Contains(subject, indicator) {
			score += 0.3
		}
	}
	
	// Commercial transaction indicators
	commercialIndicators := []string{
		"order", "purchase", "receipt", "confirmation", "invoice",
	}
	
	for _, indicator := range commercialIndicators {
		if strings.Contains(subject, indicator) {
			score += 0.2
		}
	}
	
	// Carrier mentions
	if r.shippingCarriers.MatchString(subject) {
		score += 0.4
	}
	
	// Shipping verbs
	if r.shippingVerbs.MatchString(subject) {
		score += 0.3
	}
	
	return score
}

// scoreContent analyzes the email content for shipping relevance
func (r *RelevanceScorer) scoreContent(content string) float64 {
	score := 0.0
	
	// Count shipping keyword occurrences
	shippingMatches := r.shippingKeywords.FindAllString(content, -1)
	score += float64(len(shippingMatches)) * 0.1
	
	// Count commercial keyword occurrences
	commercialMatches := r.commercialKeywords.FindAllString(content, -1)
	score += float64(len(commercialMatches)) * 0.05
	
	// Delivery status indicators
	deliveryMatches := r.deliveryKeywords.FindAllString(content, -1)
	score += float64(len(deliveryMatches)) * 0.15
	
	return score
}

// scoreCarrierMentions analyzes carrier mentions in content
func (r *RelevanceScorer) scoreCarrierMentions(content string) float64 {
	carrierMatches := r.shippingCarriers.FindAllString(content, -1)
	
	// Each carrier mention adds to the score
	score := float64(len(carrierMatches)) * 0.2
	
	// Bonus for multiple different carriers mentioned
	uniqueCarriers := make(map[string]bool)
	for _, match := range carrierMatches {
		uniqueCarriers[strings.ToLower(match)] = true
	}
	
	if len(uniqueCarriers) > 1 {
		score += 0.2
	}
	
	return score
}

// scoreTrackingPatterns analyzes tracking-specific patterns
func (r *RelevanceScorer) scoreTrackingPatterns(content string) float64 {
	trackingMatches := r.trackingPatterns.FindAllString(content, -1)
	
	// Base score from tracking keywords
	score := float64(len(trackingMatches)) * 0.15
	
	// Look for tracking number patterns (basic heuristics)
	trackingNumberPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\b1Z[A-Z0-9]{16}\b`),     // UPS
		regexp.MustCompile(`\b94\d{20}\b`),           // USPS
		regexp.MustCompile(`\b\d{12,14}\b`),          // FedEx
		regexp.MustCompile(`\b\d{10,11}\b`),          // DHL
		regexp.MustCompile(`\bTBA\d{12}\b`),          // Amazon Logistics
		regexp.MustCompile(`\b\d{3}-\d{7}-\d{7}\b`),  // Amazon Order
	}
	
	for _, pattern := range trackingNumberPatterns {
		if pattern.MatchString(content) {
			score += 0.3
			break // Only count one pattern match to avoid double scoring
		}
	}
	
	return score
}

// GetRelevanceThreshold returns the recommended threshold for considering emails relevant
func (r *RelevanceScorer) GetRelevanceThreshold() float64 {
	return 0.3 // Emails with score >= 0.3 are considered potentially relevant
}

// GetHighConfidenceThreshold returns the threshold for high-confidence shipping emails
func (r *RelevanceScorer) GetHighConfidenceThreshold() float64 {
	return 0.7 // Emails with score >= 0.7 are very likely shipping-related
}

// IsRelevant checks if an email meets the relevance threshold
func (r *RelevanceScorer) IsRelevant(msg *email.EmailMessage) bool {
	score := r.CalculateRelevanceScore(msg)
	return score >= r.GetRelevanceThreshold()
}

// IsHighConfidence checks if an email meets the high confidence threshold
func (r *RelevanceScorer) IsHighConfidence(msg *email.EmailMessage) bool {
	score := r.CalculateRelevanceScore(msg)
	return score >= r.GetHighConfidenceThreshold()
}

// GetScoreBreakdown returns a detailed breakdown of the relevance score calculation
func (r *RelevanceScorer) GetScoreBreakdown(msg *email.EmailMessage) map[string]float64 {
	textContent := strings.Join([]string{
		msg.Subject,
		msg.From,
		msg.Snippet,
		msg.PlainText,
		msg.HTMLText,
	}, " ")
	
	textContent = strings.ToLower(textContent)
	
	return map[string]float64{
		"sender_score":   r.scoreSender(msg.From),
		"subject_score":  r.scoreSubject(msg.Subject),
		"content_score":  r.scoreContent(textContent),
		"carrier_score":  r.scoreCarrierMentions(textContent),
		"tracking_score": r.scoreTrackingPatterns(textContent),
		"total_score":    r.CalculateRelevanceScore(msg),
	}
}