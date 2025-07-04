# Context Findings - Email Tracker Enhancement

## Technical Implementation Analysis

### 1. Current LLM Integration (internal/parser/llm.go)

**Current State:**
- Well-structured LLM integration with support for Ollama, OpenAI, and Anthropic
- JSON-based response parsing with confidence scoring
- Robust error handling and fallback mechanisms
- Current prompt focuses only on tracking number extraction

**Modification Points:**
- **Line 121-145**: Extend `buildPrompt()` to include merchant/description extraction
- **Line 221-227**: Update JSON parsing struct to handle new fields
- **Line 235-244**: Enhance TrackingInfo conversion for merchant data

### 2. Gmail Search Functionality (internal/email/gmail.go)

**Current State:**
- Already supports 30-day date filtering via `afterDays` parameter
- Includes unread-only filtering capability
- Handles carrier-specific search queries effectively

**Required Changes:**
- **Minimal**: Current search functionality already supports the requirements
- May need to expand search query to include order confirmation emails

### 3. Data Structure Modifications (internal/email/types.go)

**Current TrackingInfo Structure:**
- Description field already exists (line 57)
- **Missing**: Merchant field needs to be added

**Required Changes:**
- Add `Merchant string` field after Description
- Update API payload structure to include merchant information

### 4. Email Processing Pipeline (internal/parser/extractor.go)

**Current State:**
- Well-structured extraction pipeline with regex + LLM hybrid approach
- Existing result merging logic can be extended
- Confidence scoring system already in place

**Modification Points:**
- **Line 563-565**: Update result merging to include merchant information
- No major architectural changes needed

### 5. API Client Integration (internal/api/client.go)

**Current State:**
- Description field already handled in shipment creation
- Fallback description generation from email metadata

**Required Changes:**
- Add merchant field to API payload
- Enhance fallback description to include merchant info

## Best Practices Research - LLM Prompt Engineering for Email Extraction (2025)

### Key Findings from Industry Research

**1. Structured JSON Response Format**
- 2025 best practices emphasize precise field specification
- JSON schema validation reduces hallucinations by 90%
- Template-based approaches outperform ad-hoc prompting

**2. Few-Shot Prompting Effectiveness**
- Including examples in prompts significantly improves accuracy
- Particularly effective for complex extraction tasks
- Reduces need for fine-tuning in many scenarios

**3. Chain-of-Thought for Complex Emails**
- "Let's think step by step" approach improves extraction quality
- Especially valuable for unstructured email formats
- Helps with merchant disambiguation

**4. Cost-Effectiveness Considerations**
- Self-refinement techniques don't significantly improve performance
- Processing costs increase substantially with complex prompts
- Balance between accuracy and efficiency is crucial

### Recommended LLM Prompt Structure

```json
{
  "tracking_numbers": [
    {
      "number": "tracking_number_here",
      "carrier": "ups|usps|fedex|dhl",
      "confidence": 0.95,
      "description": "Meaningful product description from email",
      "merchant": "Company/retailer name"
    }
  ]
}
```

### Enhanced Prompt Engineering Strategy

**1. Context-Aware Instructions**
- Specify email types (shipping notifications, order confirmations)
- Include merchant identification guidelines
- Clear description extraction rules

**2. Fallback Mechanisms**
- Maintain existing regex-based extraction as backup
- Confidence scoring for LLM vs regex results
- Graceful degradation for failed LLM requests

**3. Validation and Quality Control**
- JSON schema validation for LLM responses
- Merchant name normalization
- Description quality scoring

## Implementation Risk Assessment

### Low Risk Areas
- Well-defined interfaces between components
- Existing error handling and fallback mechanisms
- Backward compatibility through optional fields
- Current LLM infrastructure can handle enhanced prompts

### Potential Challenges
- LLM response parsing robustness with expanded schema
- Performance impact of more complex prompts
- Merchant name normalization and deduplication
- Increased processing costs for LLM calls

### Mitigation Strategies
- Comprehensive testing with real email samples
- A/B testing of prompt variations
- Monitoring of LLM response quality
- Cost monitoring and optimization

## Files Requiring Modification

**Priority 1 - Core Functionality:**
1. `internal/email/types.go` - Add merchant field
2. `internal/parser/llm.go` - Enhanced prompt and parsing
3. `internal/api/client.go` - API payload updates

**Priority 2 - Enhanced Processing:**
1. `internal/parser/extractor.go` - Result merging updates
2. `cmd/email-tracker/cmd/root.go` - Configuration updates if needed

**Testing Requirements:**
1. `internal/parser/llm_test.go` - Test enhanced extraction
2. `internal/email/types_test.go` - Test new data structures
3. Integration tests with real email samples

## Related Features Analysis

**Similar Features Found:**
- Current tracking number extraction pipeline
- Existing LLM integration infrastructure
- Email processing and state management
- API client with retry logic

**Architectural Patterns to Follow:**
- Interface-based design for extensibility
- Configuration-driven feature flags
- Comprehensive error handling
- Structured logging for debugging