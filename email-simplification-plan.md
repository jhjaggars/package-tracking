# Email Processing System Simplification Plan

## Executive Summary

The current email processing system has grown to over 15,600 lines of code across 34 Go files with three different processing approaches and excessive complexity. This plan outlines a test-driven approach to simplify the system to focus on core functionality:

1. **Read last 30 days of unprocessed emails** (user-configurable)
2. **Extract tracking numbers using pattern matching** (regex-based)
3. **Validate tracking numbers** against carrier formats
4. **Use LLM to extract item descriptions** from email content
5. **Create shipments** with tracking numbers and descriptions

## Current System Analysis

### Complexity Assessment

**Current Codebase Size:** ~15,600 lines across 34 Go files

**Largest Files:**
- `internal/parser/extractor.go` - 969 lines (complex multi-stage extraction)
- `internal/email/gmail.go` - 714 lines (Gmail API integration)
- `internal/workers/email_processor_time.go` - 716 lines (time-based processing)
- `internal/workers/email_processor_twophase.go` - 528 lines (two-phase processing)
- `internal/parser/llm.go` - 461 lines (LLM integration)
- `internal/email/state.go` - 444 lines (SQLite state management)

### Major Complexity Issues

1. **Multiple Overlapping Implementations:**
   - Three different email processors with similar but incompatible interfaces
   - `email_processor_deprecated.go` (421 lines)
   - `email_processor_time.go` (716 lines) 
   - `email_processor_twophase.go` (528 lines)

2. **Oversized Core Components:**
   - `extractor.go` handles too many responsibilities (969 lines)
   - Complex 7-stage processing pipeline
   - Multiple extraction strategies with fallbacks

3. **Configuration Overload:**
   - Two parallel configuration systems (legacy .env + Viper)
   - 40+ configuration parameters across 6 major sections
   - Complex precedence rules

4. **Testing Complexity:**
   - Tests spread across multiple approaches
   - Some test files exceed 1,283 lines
   - Complex mock implementations

## Simplified Target Architecture

### Core Algorithm
```
1. Scan Gmail for unprocessed emails (last 30 days, configurable)
2. Extract tracking number candidates using regex patterns
3. Validate candidates against known carrier formats
4. For valid tracking numbers:
   - Use LLM to extract item description from email content
   - Create shipment with tracking number and description
5. Mark email as processed
```

### Key Simplifications

1. **Single Email Processor:** Replace three processors with one configurable implementation
2. **Regex-First Extraction:** Use pattern matching for tracking numbers, LLM only for descriptions
3. **Streamlined Configuration:** Single config system with sensible defaults
4. **Focused LLM Usage:** LLM extracts descriptions only, not tracking numbers
5. **Simplified State Management:** Basic processed email tracking

## Implementation Plan

### Phase 1: Create Foundation and Tests

**Goal:** Establish simplified interfaces and test-driven development approach

**Tasks:**
1. Create `email-simplification-plan.md` (this document)
2. Write tests for simplified email processor interface
3. Write tests for simplified tracking extraction
4. Write tests for LLM-based description extraction
5. Commit initial test structure

**Files to Create:**
- `internal/workers/email_processor_simplified_test.go`
- `internal/parser/tracking_extractor_test.go`
- `internal/parser/description_extractor_test.go`

### Phase 2: Remove Complex Components

**Goal:** Eliminate unnecessary complexity and duplicate implementations

**Tasks:**
1. Delete `internal/workers/email_processor_twophase.go`
2. Delete `internal/workers/email_processor_deprecated.go`
3. Delete `internal/workers/email_relevance.go`
4. Delete `internal/services/description_enhancer.go`
5. Delete `cmd/cli/cmd/enhance_descriptions.go`
6. Remove associated test files

**Files to Delete:**
- `internal/workers/email_processor_twophase.go` (528 lines)
- `internal/workers/email_processor_deprecated.go` (421 lines)
- `internal/workers/email_relevance.go` (258 lines)
- `internal/services/description_enhancer.go` (417 lines)
- `cmd/cli/cmd/enhance_descriptions.go`
- Associated test files

### Phase 3: Simplify Core Parser

**Goal:** Reduce `extractor.go` from 969 lines to ~300 lines with focused functionality

**Current Problems:**
- 7-stage processing pipeline
- Complex confidence scoring
- Multiple extraction strategies
- Overengineered carrier detection

**Simplified Approach:**
1. **Tracking Extraction:** Pure regex pattern matching
2. **Description Extraction:** LLM-based with simplified prompts
3. **Validation:** Basic carrier format validation
4. **Results:** Simple success/failure with extracted data

**Tasks:**
1. Write tests for simplified tracking extractor
2. Implement `TrackingExtractor` interface
3. Write tests for simplified description extractor
4. Implement `DescriptionExtractor` interface
5. Refactor `extractor.go` to use new interfaces
6. Remove complex multi-stage processing

**New File Structure:**
```
internal/parser/
├── tracking_extractor.go      (~150 lines)
├── description_extractor.go   (~100 lines)
├── extractor.go              (~200 lines, simplified)
└── patterns.go               (keep essential patterns)
```

### Phase 4: Consolidate Email Processing

**Goal:** Single, configurable email processor replacing three implementations

**Current Problems:**
- Three different processors with incompatible interfaces
- Redundant code across processors
- Inconsistent error handling

**Simplified Approach:**
1. **Single Processor:** `EmailProcessor` with configurable behavior
2. **Simple Workflow:** Scan → Extract → Validate → Create → Mark Processed
3. **Unified Error Handling:** Consistent error patterns
4. **Basic State Management:** Track processed emails only

**Tasks:**
1. Write tests for unified email processor
2. Implement simplified `EmailProcessor`
3. Remove complex validation integration
4. Simplify metrics to basic counters
5. Update email processor to use new parser interfaces

**Target Structure:**
```
internal/workers/
├── email_processor.go          (~300 lines)
├── email_processor_test.go     (~200 lines)
└── email_types.go             (~100 lines)
```

### Phase 5: Streamline Configuration

**Goal:** Single configuration system with sensible defaults

**Current Problems:**
- Two parallel configuration systems
- 40+ configuration parameters
- Complex precedence rules

**Simplified Approach:**
1. **Single Config System:** Use Viper only, remove legacy .env
2. **Fewer Options:** Essential configuration only
3. **Sensible Defaults:** Minimize required configuration
4. **Clear Precedence:** Environment variables > Config file > Defaults

**Essential Configuration:**
```yaml
email:
  client_id: ""
  client_secret: ""  
  refresh_token: ""
  days_to_scan: 30
  check_interval: "5m"
  
processing:
  dry_run: false
  api_url: "http://localhost:8080"
  
llm:
  enabled: true
  provider: "openai"
  model: "gpt-4"
  api_key: ""
```

**Tasks:**
1. Write tests for simplified configuration
2. Remove legacy .env support
3. Streamline configuration structure
4. Update CLI to use simplified config
5. Update documentation

### Phase 6: Simplify LLM Integration

**Goal:** Focus LLM on description extraction only

**Current Problems:**
- LLM used for tracking extraction (unnecessary)
- Complex prompts with excessive examples
- Multiple response formats and fallbacks

**Simplified Approach:**
1. **Single Purpose:** Description extraction only
2. **Simple Prompts:** Focus on item description extraction
3. **Basic Validation:** Simple response parsing
4. **Error Handling:** Graceful fallback to empty description

**Tasks:**
1. Write tests for description-only LLM extraction
2. Simplify LLM prompts for description extraction
3. Remove tracking number extraction from LLM
4. Update LLM integration to focus on descriptions
5. Remove complex confidence scoring

### Phase 7: Update Tests and Documentation

**Goal:** Comprehensive test coverage for simplified system

**Test Strategy:**
1. **Test-Driven Development:** Write tests first for new components
2. **Focused Tests:** Each test file covers single responsibility
3. **Integration Tests:** End-to-end email processing tests
4. **Mock Simplification:** Minimal mocking for external dependencies

**Tasks:**
1. Write comprehensive test suite for simplified components
2. Remove complex test scenarios for deleted components
3. Update integration tests for simplified workflow
4. Update documentation and README
5. Update CLAUDE.md with simplified architecture

## Expected Outcomes

### Quantitative Improvements

- **Codebase Reduction:** ~40-50% reduction (15,600 → ~8,000 lines)
- **File Count:** Reduce from 34 to ~20 Go files
- **Configuration Options:** Reduce from 40+ to ~15 essential options
- **Test Complexity:** Focused test suites vs. mega-test files

### Qualitative Improvements

- **Maintainability:** Single processor vs. three implementations
- **Clarity:** Simple workflow vs. complex multi-stage processing
- **Performance:** Regex-based extraction vs. LLM for tracking numbers
- **Reliability:** Focused LLM usage vs. complex fallback mechanisms

### Preserved Functionality

- Gmail integration with OAuth2
- Tracking number extraction for all carriers
- Description extraction using LLM
- Shipment creation via REST API
- Email state management
- Configurable scanning intervals

## Risk Mitigation

### Testing Strategy

1. **Test-Driven Development:** Write tests first for all new components
2. **Integration Testing:** Verify end-to-end workflow
3. **Regression Testing:** Ensure no functionality loss
4. **Performance Testing:** Validate extraction accuracy

### Rollback Plan

1. **Feature Branch:** All work done in `feature/simplify-email-processing`
2. **Incremental Commits:** Each phase committed separately
3. **Backup:** Current system preserved in git history
4. **Staged Rollout:** Testing before merging to main

### Validation Criteria

1. **Functional:** Email processing works end-to-end
2. **Performance:** Extraction accuracy maintained or improved
3. **Maintainability:** Code is easier to understand and modify
4. **Configuration:** System is easier to configure and deploy

## Timeline

**Phase 1:** Foundation and Tests (Day 1)
**Phase 2:** Remove Complex Components (Day 1-2)
**Phase 3:** Simplify Core Parser (Day 2-3)
**Phase 4:** Consolidate Email Processing (Day 3-4)
**Phase 5:** Streamline Configuration (Day 4-5)
**Phase 6:** Simplify LLM Integration (Day 5)
**Phase 7:** Update Tests and Documentation (Day 5-6)

**Total Estimated Time:** 5-6 days

## Success Metrics

1. **Code Quality:** Reduced complexity while maintaining functionality
2. **Test Coverage:** Comprehensive coverage of simplified components
3. **Performance:** Maintained or improved extraction accuracy
4. **Maintainability:** Easier to understand and modify codebase
5. **Documentation:** Clear documentation of simplified architecture

---

*This plan follows test-driven development principles and focuses on maintaining core functionality while dramatically reducing system complexity.*