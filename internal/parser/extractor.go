package parser

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"package-tracking/internal/carriers"
	"package-tracking/internal/email"
)

// TrackingExtractor handles extraction of tracking numbers from emails
type TrackingExtractor struct {
	carrierFactory *carriers.ClientFactory
	patterns       *PatternManager
	llmExtractor   LLMExtractor
	config         *ExtractorConfig
}

// ExtractorConfig configures the extraction behavior
type ExtractorConfig struct {
	EnableLLM           bool
	MinConfidence       float64
	MaxCandidates       int
	UseHybridValidation bool
	DebugMode           bool
}

// NewTrackingExtractor creates a new tracking number extractor
func NewTrackingExtractor(carrierFactory *carriers.ClientFactory, config *ExtractorConfig, llmConfig *LLMConfig) *TrackingExtractor {
	if config == nil {
		config = &ExtractorConfig{
			EnableLLM:           false,
			MinConfidence:       0.5,
			MaxCandidates:       10,
			UseHybridValidation: true,
			DebugMode:           false,
		}
	} else {
		// Fill in missing fields with defaults
		if config.MinConfidence == 0 {
			config.MinConfidence = 0.5
		}
		if config.MaxCandidates == 0 {
			config.MaxCandidates = 10
		}
		// Note: EnableLLM, UseHybridValidation, and DebugMode default to false which is correct
	}

	// Initialize LLM extractor based on configuration
	var llmExtractor LLMExtractor
	if config.EnableLLM && llmConfig != nil {
		// Use the LLM extractor factory to create appropriate extractor
		llmExtractor = NewLLMExtractor(llmConfig)
	} else {
		llmExtractor = NewNoOpLLMExtractor()
	}

	return &TrackingExtractor{
		carrierFactory: carrierFactory,
		patterns:       NewPatternManager(),
		llmExtractor:   llmExtractor,
		config:         config,
	}
}

// Extract extracts tracking numbers from email content
func (e *TrackingExtractor) Extract(content *email.EmailContent) ([]email.TrackingInfo, error) {
	startTime := time.Now()

	if e.config.DebugMode {
		log.Printf("Starting extraction for email from: %s, subject: %s", content.From, content.Subject)
	}

	// Stage 1: Preprocess email content
	preprocessed := e.preprocessContent(content)

	// Stage 2: Identify likely carriers
	carrierHints := e.identifyCarriers(preprocessed)

	// Stage 3: Extract candidates using regex patterns
	candidates := e.extractCandidates(preprocessed, carrierHints)

	// Stage 4: Filter obvious false positives before validation
	filtered := e.filterFalsePositives(candidates)

	// Stage 5: Validate candidates against carrier patterns
	validated := e.validateCandidates(filtered, preprocessed)

	// Stage 5: Use LLM if enabled and needed
	var llmResults []email.TrackingInfo
	if e.config.EnableLLM && e.shouldUseLLM(validated, content) {
		var err error
		llmResults, err = e.extractWithEnhancedLLM(content)
		if err != nil {
			log.Printf("Enhanced LLM extraction failed, falling back to basic LLM: %v", err)
			// Fallback to basic LLM extraction
			llmResults, err = e.llmExtractor.Extract(content)
			if err != nil {
				log.Printf("LLM extraction failed: %v", err)
			}
		}
	}

	// Stage 6: Merge and score results
	results := e.mergeResults(validated, llmResults)

	// Stage 7: Final filtering and sorting
	final := e.filterAndSort(results)

	processingTime := time.Since(startTime)
	if e.config.DebugMode {
		log.Printf("Extraction completed in %v, found %d tracking numbers", processingTime, len(final))
	}

	return final, nil
}

// preprocessContent cleans and normalizes email content
func (e *TrackingExtractor) preprocessContent(content *email.EmailContent) *email.EmailContent {
	processed := &email.EmailContent{
		PlainText: e.cleanText(content.PlainText),
		HTMLText:  content.HTMLText,
		Subject:   strings.TrimSpace(content.Subject),
		From:      strings.ToLower(strings.TrimSpace(content.From)),
		Headers:   content.Headers,
		MessageID: content.MessageID,
		ThreadID:  content.ThreadID,
		Date:      content.Date,
	}

	// If no plain text, convert HTML
	if processed.PlainText == "" && processed.HTMLText != "" {
		processed.PlainText = e.htmlToText(processed.HTMLText)
	}

	return processed
}

// cleanText normalizes text content
func (e *TrackingExtractor) cleanText(text string) string {
	if text == "" {
		return ""
	}

	// Remove excessive whitespace
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	// Remove common email artifacts
	text = strings.ReplaceAll(text, "\r\n", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")

	return strings.TrimSpace(text)
}

// htmlToText converts HTML to plain text (basic implementation)
func (e *TrackingExtractor) htmlToText(html string) string {
	// Remove script and style tags completely
	re := regexp.MustCompile(`(?i)<(script|style)[^>]*>.*?</(script|style)>`)
	html = re.ReplaceAllString(html, "")

	// Replace some HTML tags with spaces/newlines
	re = regexp.MustCompile(`(?i)</(div|p|br|tr)>`)
	html = re.ReplaceAllString(html, " ")

	// Remove all remaining HTML tags
	re = regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, " ")

	// Decode common HTML entities
	entities := map[string]string{
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": "\"",
		"&#39;":  "'",
		"&nbsp;": " ",
	}

	for entity, replacement := range entities {
		text = strings.ReplaceAll(text, entity, replacement)
	}

	// Normalize whitespace
	re = regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// identifyCarriers analyzes email to identify likely carriers
func (e *TrackingExtractor) identifyCarriers(content *email.EmailContent) []email.CarrierHint {
	var hints []email.CarrierHint

	// Analyze sender domain
	hints = append(hints, e.analyzeFromAddress(content.From)...)

	// Analyze subject line
	hints = append(hints, e.analyzeSubject(content.Subject)...)

	// Analyze content keywords
	hints = append(hints, e.analyzeContent(content.PlainText)...)

	// Sort by confidence
	sort.Slice(hints, func(i, j int) bool {
		return hints[i].Confidence > hints[j].Confidence
	})

	return hints
}

// analyzeFromAddress extracts carrier hints from sender email
func (e *TrackingExtractor) analyzeFromAddress(from string) []email.CarrierHint {
	var hints []email.CarrierHint
	from = strings.ToLower(from)

	carriers := map[string][]string{
		"ups":    {"ups.com", "quantum.ups.com", "pkginfo.ups.com"},
		"usps":   {"usps.com", "email.usps.com", "informeddelivery.usps.com"},
		"fedex":  {"fedex.com", "tracking.fedex.com", "shipment.fedex.com"},
		"dhl":    {"dhl.com", "noreply.dhl.com", "dhl.de"},
		"amazon": {"amazon.com", "shipment-tracking.amazon.com", "marketplace.amazon.com", "amazonlogistics.com"},
	}

	for carrier, domains := range carriers {
		for _, domain := range domains {
			if strings.Contains(from, domain) {
				hints = append(hints, email.CarrierHint{
					Carrier:    carrier,
					Confidence: 0.9,
					Source:     "sender",
					Reason:     fmt.Sprintf("From %s domain", domain),
				})
				break
			}
		}
	}

	// Check for vendor emails that often contain shipping info
	vendors := []string{"amazon.com", "shopify.com", "ebay.com", "etsy.com"}
	for _, vendor := range vendors {
		if strings.Contains(from, vendor) {
			hints = append(hints, email.CarrierHint{
				Carrier:    "unknown",
				Confidence: 0.6,
				Source:     "sender",
				Reason:     fmt.Sprintf("From vendor %s", vendor),
			})
		}
	}

	return hints
}

// analyzeSubject extracts carrier hints from email subject
func (e *TrackingExtractor) analyzeSubject(subject string) []email.CarrierHint {
	var hints []email.CarrierHint
	subject = strings.ToLower(subject)

	// Direct carrier mentions
	carriers := []string{"ups", "usps", "fedex", "dhl", "amazon"}
	for _, carrier := range carriers {
		if strings.Contains(subject, carrier) {
			hints = append(hints, email.CarrierHint{
				Carrier:    carrier,
				Confidence: 0.7,
				Source:     "subject",
				Reason:     fmt.Sprintf("Contains '%s'", carrier),
			})
		}
	}

	// Amazon-specific terms in subject
	amazonTerms := []string{"amazon logistics", "amzl", "order shipped", "order update"}
	for _, term := range amazonTerms {
		if strings.Contains(subject, term) {
			hints = append(hints, email.CarrierHint{
				Carrier:    "amazon",
				Confidence: 0.8,
				Source:     "subject",
				Reason:     fmt.Sprintf("Contains Amazon term '%s'", term),
			})
		}
	}

	// Generic shipping terms
	shippingTerms := []string{"tracking", "shipment", "package", "delivery", "shipped"}
	for _, term := range shippingTerms {
		if strings.Contains(subject, term) {
			hints = append(hints, email.CarrierHint{
				Carrier:    "unknown",
				Confidence: 0.4,
				Source:     "subject",
				Reason:     fmt.Sprintf("Contains '%s'", term),
			})
		}
	}

	return hints
}

// analyzeContent extracts carrier hints from email content
func (e *TrackingExtractor) analyzeContent(content string) []email.CarrierHint {
	var hints []email.CarrierHint
	content = strings.ToLower(content)

	// Count carrier mentions
	carrierCounts := make(map[string]int)
	carriers := []string{"ups", "usps", "fedex", "dhl", "amazon"}

	for _, carrier := range carriers {
		count := strings.Count(content, carrier)
		if count > 0 {
			carrierCounts[carrier] = count
		}
	}

	// Special handling for Amazon-specific terms
	amazonTerms := []string{"amazon logistics", "amzl", "order number", "amazon.com"}
	amazonCount := 0
	for _, term := range amazonTerms {
		amazonCount += strings.Count(content, term)
	}
	if amazonCount > 0 {
		if existing, ok := carrierCounts["amazon"]; ok {
			carrierCounts["amazon"] = existing + amazonCount
		} else {
			carrierCounts["amazon"] = amazonCount
		}
	}

	// Convert counts to hints
	for carrier, count := range carrierCounts {
		confidence := 0.5 + float64(count)*0.1
		if confidence > 0.8 {
			confidence = 0.8
		}

		hints = append(hints, email.CarrierHint{
			Carrier:    carrier,
			Confidence: confidence,
			Source:     "content",
			Reason:     fmt.Sprintf("Mentioned %d times", count),
		})
	}

	return hints
}

// extractCandidates finds potential tracking numbers using regex patterns
func (e *TrackingExtractor) extractCandidates(content *email.EmailContent, hints []email.CarrierHint) []email.TrackingCandidate {
	var candidates []email.TrackingCandidate

	// Extract candidates for each suggested carrier
	for _, hint := range hints {
		if hint.Carrier != "unknown" {
			candidates = append(candidates, e.patterns.ExtractForCarrier(content.PlainText, hint.Carrier)...)
		}
	}

	// Also run generic extraction patterns
	candidates = append(candidates, e.patterns.ExtractGeneric(content.PlainText)...)

	// Deduplicate candidates
	seen := make(map[string]bool)
	var unique []email.TrackingCandidate

	for _, candidate := range candidates {
		key := candidate.Text + ":" + candidate.Carrier
		if !seen[key] {
			seen[key] = true
			unique = append(unique, candidate)
		}
	}

	// Limit number of candidates
	if len(unique) > e.config.MaxCandidates {
		// Sort by confidence and take top candidates
		sort.Slice(unique, func(i, j int) bool {
			return unique[i].Confidence > unique[j].Confidence
		})
		unique = unique[:e.config.MaxCandidates]
	}

	return unique
}

// filterFalsePositives removes obvious false positives before carrier validation
func (e *TrackingExtractor) filterFalsePositives(candidates []email.TrackingCandidate) []email.TrackingCandidate {
	var filtered []email.TrackingCandidate

	for _, candidate := range candidates {
		if !e.isObviousFalsePositive(candidate.Text) {
			filtered = append(filtered, candidate)
		} else if e.config.DebugMode {
			log.Printf("Filtered false positive: %s", candidate.Text)
		}
	}

	return filtered
}

// validateCandidates validates candidates against carrier validation logic
func (e *TrackingExtractor) validateCandidates(candidates []email.TrackingCandidate, content *email.EmailContent) []email.TrackingInfo {
	var results []email.TrackingInfo

	for _, candidate := range candidates {
		// Determine carrier validation order based on candidate context and email hints
		carrierOrder := e.getCarrierValidationOrder(candidate, content)

		// Try validating against carriers in optimized order
		for _, carrierCode := range carrierOrder {
			// Clean up the tracking number
			cleanNumber := e.cleanTrackingNumber(candidate.Text)

			// Apply carrier-specific validation logic
			if e.validateTrackingNumberForCarrier(cleanNumber, carrierCode, candidate, content) {
				// Calculate final confidence score
				confidence := e.calculateConfidence(candidate, carrierCode)

				if confidence >= e.config.MinConfidence {
					result := email.TrackingInfo{
						Number:      cleanNumber,
						Carrier:     carrierCode,
						Confidence:  confidence,
						Source:      "regex",
						Context:     candidate.Context,
						ExtractedAt: time.Now(),
					}

					results = append(results, result)
					break // Found valid carrier for this candidate
				}
			}
		}
	}

	return results
}

// getCarrierValidationOrder determines the optimal order to validate carriers
// based on the candidate's context and email sender information
func (e *TrackingExtractor) getCarrierValidationOrder(candidate email.TrackingCandidate, content *email.EmailContent) []string {
	// Default order: more specific patterns first
	defaultOrder := []string{"ups", "usps", "fedex", "dhl", "amazon"}

	// If the candidate has a suggested carrier, try that first
	if candidate.Carrier != "" && candidate.Carrier != "unknown" {
		// Create order with suggested carrier first
		order := []string{candidate.Carrier}
		for _, carrier := range defaultOrder {
			if carrier != candidate.Carrier {
				order = append(order, carrier)
			}
		}
		return order
	}

	// For Amazon email context, use Amazon-optimized order
	if e.isAmazonEmailContext(content) {
		// For Amazon emails, try standard carriers first (most common delegation)
		// then Amazon internal codes as fallback
		return []string{"ups", "usps", "fedex", "dhl", "amazon"}
	}

	return defaultOrder
}

// validateTrackingNumberForCarrier applies carrier-specific validation with enhanced logic
func (e *TrackingExtractor) validateTrackingNumberForCarrier(trackingNumber, carrierCode string, candidate email.TrackingCandidate, content *email.EmailContent) bool {
	client, _, err := e.carrierFactory.CreateClient(carrierCode)
	if err != nil {
		return false
	}

	// Apply standard validation
	if client.ValidateTrackingNumber(trackingNumber) {
		return true
	}

	// Enhanced validation for Amazon emails with relaxed Amazon internal reference matching
	if carrierCode == "amazon" && e.isAmazonEmailContext(content) {
		// For Amazon emails, be more permissive with Amazon internal codes
		// but still require basic alphanumeric format validation
		return e.isLikelyAmazonInternalCode(trackingNumber)
	}

	return false
}

// isAmazonEmailContext checks if the email comes from an Amazon context
func (e *TrackingExtractor) isAmazonEmailContext(content *email.EmailContent) bool {
	// Check if the email content hints suggest Amazon
	fromLower := strings.ToLower(content.From)
	subjectLower := strings.ToLower(content.Subject)

	// Check for Amazon domains
	amazonDomains := []string{"amazon.com", "amazonlogistics.com", "marketplace.amazon.com", "shipment-tracking.amazon.com"}
	for _, domain := range amazonDomains {
		if strings.Contains(fromLower, domain) {
			return true
		}
	}

	// Check for Amazon-related terms in subject
	amazonTerms := []string{"amazon", "amazon logistics", "amzl"}
	for _, term := range amazonTerms {
		if strings.Contains(subjectLower, term) {
			return true
		}
	}

	return false
}

// isLikelyAmazonInternalCode performs relaxed validation for Amazon internal codes
func (e *TrackingExtractor) isLikelyAmazonInternalCode(trackingNumber string) bool {
	// More lenient validation for tracking numbers found in Amazon emails
	// that don't match standard Amazon formats but could be internal references

	// Basic length check (Amazon internal codes are usually 6-20 characters)
	if len(trackingNumber) < 6 || len(trackingNumber) > 20 {
		return false
	}

	// Normalize to uppercase for consistency
	normalizedNumber := strings.ToUpper(trackingNumber)

	// Must be alphanumeric
	if matched, _ := regexp.MatchString(`^[A-Z0-9]+$`, normalizedNumber); !matched {
		return false
	}

	// Must contain at least one letter (to distinguish from pure numbers)
	if matched, _ := regexp.MatchString(`[A-Z]`, normalizedNumber); !matched {
		return false
	}

	// Exclude obvious false positives (like years, common words, etc.)
	falsePositives := []string{
		`^(19|20)\d{2}$`,                                     // Years
		`^(mon|tue|wed|thu|fri|sat|sun)`,                     // Days
		`^(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)`, // Months
		`^(http|www|email|phone|address)`,                    // Common words
	}

	for _, pattern := range falsePositives {
		if matched, _ := regexp.MatchString(`(?i)`+pattern, trackingNumber); matched {
			return false
		}
	}

	return true
}

// cleanTrackingNumber normalizes tracking number format
func (e *TrackingExtractor) cleanTrackingNumber(number string) string {
	// Remove spaces and common formatting
	cleaned := strings.ReplaceAll(number, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "_", "")

	// Convert to uppercase for consistency
	cleaned = strings.ToUpper(cleaned)

	return cleaned
}

// calculateConfidence computes final confidence score
func (e *TrackingExtractor) calculateConfidence(candidate email.TrackingCandidate, carrierCode string) float64 {
	score := candidate.Confidence

	// Boost confidence if carrier matches candidate suggestion
	if candidate.Carrier == carrierCode {
		score += 0.2
	}

	// Boost for labeled context (e.g., "Tracking Number: 1Z...")
	if strings.Contains(strings.ToLower(candidate.Context), "tracking") {
		score += 0.1
	}

	// Boost for early position in email
	if candidate.Position < 1000 {
		score += 0.1
	}

	// Penalize obvious false positives that somehow got through
	text := strings.ToLower(candidate.Text)

	// Penalize pure alphabetic strings heavily
	if regexp.MustCompile(`^[a-z]+$`).MatchString(text) {
		score *= 0.1
	}

	// Penalize common words
	if e.isObviousFalsePositive(candidate.Text) {
		score *= 0.01
	}

	// Penalize if no digits for carriers that require them
	if carrierCode != "unknown" && !strings.ContainsAny(text, "0123456789") {
		score *= 0.1
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// shouldUseLLM determines if LLM extraction should be used
func (e *TrackingExtractor) shouldUseLLM(regexResults []email.TrackingInfo, content *email.EmailContent) bool {
	if !e.config.EnableLLM {
		return false
	}

	// Use LLM if no regex results
	if len(regexResults) == 0 {
		return true
	}

	// Use LLM if low confidence results
	maxConfidence := 0.0
	for _, result := range regexResults {
		if result.Confidence > maxConfidence {
			maxConfidence = result.Confidence
		}
	}

	if maxConfidence < 0.7 {
		return true
	}

	// Use LLM for complex email structure
	if e.isComplexEmail(content) {
		return true
	}

	// Use LLM for unknown senders
	if !e.isKnownCarrierSender(content.From) {
		return true
	}

	return false
}

// isComplexEmail checks if email has complex structure
func (e *TrackingExtractor) isComplexEmail(content *email.EmailContent) bool {
	// Check for lots of HTML
	if len(content.HTMLText) > len(content.PlainText)*2 {
		return true
	}

	// Check for table structures
	if strings.Contains(strings.ToLower(content.HTMLText), "<table") {
		return true
	}

	// Check for very long emails
	if len(content.PlainText) > 10000 {
		return true
	}

	return false
}

// isKnownCarrierSender checks if email is from known carrier domain
func (e *TrackingExtractor) isKnownCarrierSender(from string) bool {
	from = strings.ToLower(from)
	knownDomains := []string{
		"ups.com", "usps.com", "fedex.com", "dhl.com",
	}

	for _, domain := range knownDomains {
		if strings.Contains(from, domain) {
			return true
		}
	}

	return false
}

// extractWithEnhancedLLM performs enhanced LLM extraction with confidence-based fallback
func (e *TrackingExtractor) extractWithEnhancedLLM(content *email.EmailContent) ([]email.TrackingInfo, error) {
	// Try to use enhanced LLM extraction
	if localExtractor, ok := e.llmExtractor.(*LocalLLMExtractor); ok {
		// Use enhanced prompt
		prompt := localExtractor.buildEnhancedPrompt(content)
		response, err := localExtractor.callLLM(prompt)
		if err != nil {
			return nil, fmt.Errorf("enhanced LLM call failed: %w", err)
		}

		// Parse enhanced response
		results, err := localExtractor.parseEnhancedResponse(response)
		if err != nil {
			return nil, fmt.Errorf("enhanced response parsing failed: %w", err)
		}

		// Apply confidence-based filtering
		confidenceThreshold := 0.7 // Configurable threshold
		filtered := localExtractor.filterByConfidence(results, confidenceThreshold)

		// If we have high-confidence results, use them
		if len(filtered) > 0 {
			return filtered, nil
		}

		// If no high-confidence results, return all results for fallback processing
		return results, nil
	}

	// Fallback to standard extraction for non-local extractors
	return e.llmExtractor.Extract(content)
}

// mergeResults combines regex and LLM results with enhanced merchant/description handling
func (e *TrackingExtractor) mergeResults(regexResults, llmResults []email.TrackingInfo) []email.TrackingInfo {
	merged := make(map[string]*email.TrackingInfo)

	// Add regex results
	for _, result := range regexResults {
		key := result.Number + ":" + result.Carrier
		merged[key] = &result
	}

	// Add or enhance with LLM results
	for _, llmResult := range llmResults {
		key := llmResult.Number + ":" + llmResult.Carrier

		if existing, found := merged[key]; found {
			// Merge information, taking best confidence and most complete description
			if llmResult.Confidence > existing.Confidence {
				existing.Confidence = llmResult.Confidence
			}

			// Enhanced description merging with merchant information
			enhancedDesc := e.combineDescriptionAndMerchant(llmResult.Description, llmResult.Merchant)
			if enhancedDesc != "" && existing.Description == "" {
				existing.Description = enhancedDesc
			} else if enhancedDesc != "" && llmResult.Confidence > existing.Confidence {
				existing.Description = enhancedDesc
			}

			existing.Source = "hybrid"
		} else {
			// For new LLM results, combine description and merchant
			llmResult.Description = e.combineDescriptionAndMerchant(llmResult.Description, llmResult.Merchant)
			merged[key] = &llmResult
		}
	}

	// Convert back to slice
	var results []email.TrackingInfo
	for _, info := range merged {
		results = append(results, *info)
	}

	return results
}

// combineDescriptionAndMerchant formats description with merchant information
func (e *TrackingExtractor) combineDescriptionAndMerchant(description, merchant string) string {
	if description == "" && merchant == "" {
		return ""
	}

	if description == "" {
		return fmt.Sprintf("Package from %s", merchant)
	}

	if merchant == "" {
		return description
	}

	// Format as "Product description from Merchant"
	return fmt.Sprintf("%s from %s", description, merchant)
}

// filterAndSort applies final filtering and sorts results
func (e *TrackingExtractor) filterAndSort(results []email.TrackingInfo) []email.TrackingInfo {
	// Filter by minimum confidence
	var filtered []email.TrackingInfo
	for _, result := range results {
		if result.Confidence >= e.config.MinConfidence {
			filtered = append(filtered, result)
		}
	}

	// Sort by confidence descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Confidence > filtered[j].Confidence
	})

	return filtered
}

// isObviousFalsePositive checks if a candidate is obviously not a tracking number
func (e *TrackingExtractor) isObviousFalsePositive(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))

	// Reject common English words that might match DHL patterns
	commonWords := []string{
		"information", "confirmation", "notification", "description",
		"application", "registration", "verification", "installation",
		"transaction", "subscription", "publication", "organization",
		"communication", "documentation", "administration", "recommendation",
		"congratulations", "specifications", "instructions", "requirements",
		"acknowledgment", "acknowledgement", "establishment", "development",
		"announcement", "arrangement", "appointment", "agreement",
		"management", "department", "government", "environment",
		"improvement", "achievement", "measurement", "assessment",
		"assignment", "attachment", "statement", "treatment",
		"equipment", "requirement", "movement", "moment",
		"comment", "content", "present", "current", "account",
		"amount", "payment", "element", "segment", "document",
		"equipment", "instrument", "supplement", "complement",
	}

	for _, word := range commonWords {
		if text == word {
			return true
		}
	}

	// Reject if it's all letters (tracking numbers should have some digits)
	if regexp.MustCompile(`^[a-z]+$`).MatchString(text) {
		return true
	}

	// Reject very short candidates
	if len(text) < 8 {
		return true
	}

	// Reject if it contains common non-tracking words
	if strings.Contains(text, "email") || strings.Contains(text, "phone") ||
		strings.Contains(text, "address") || strings.Contains(text, "website") {
		return true
	}

	return false
}
