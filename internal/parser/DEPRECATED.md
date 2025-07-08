# Deprecated Components in Parser Package

This document lists deprecated components that have been replaced by simplified alternatives as part of the email processing simplification project.

## Deprecated LLM Components for Tracking Extraction

### ❌ DEPRECATED: Complex LLM System for Tracking Extraction

The following components are **deprecated** and should not be used for new development:

- `LLMExtractor` interface in `llm.go`
- `LocalLLMExtractor` struct in `llm.go` 
- `NewLLMExtractor()` function in `llm.go`
- `LLMConfig` struct in `llm.go`
- Complex tracking extraction methods in `TrackingExtractor`

**Reason for Deprecation**: 
According to the simplified email processing requirements, LLM should only be used for description extraction, not tracking number extraction. Tracking numbers should be extracted using pattern matching only.

### ✅ RECOMMENDED: Simplified Alternatives

Use these simplified components instead:

**For Tracking Number Extraction (Pattern-based only):**
- `SimplifiedTrackingExtractor` in `tracking_extractor.go`
- `SimplifiedTrackingExtractorInterface`
- `NewSimplifiedTrackingExtractor()`

**For Description Extraction (LLM-focused):**
- `SimplifiedDescriptionExtractor` in `description_extractor.go`
- `SimplifiedDescriptionExtractorInterface`
- `NewSimplifiedDescriptionExtractor()`
- `SimplifiedLLMConfig` in `llm_description_client.go`
- `NewSimplifiedLLMClient()` in `llm_description_client.go`

**For Email Processing:**
- `SimplifiedEmailProcessor` in `../workers/email_processor_simplified.go`
- `NewSimplifiedEmailProcessor()`

## Migration Guide

### From Complex LLM Tracking Extraction to Simplified Pattern-based Extraction

**Old approach (deprecated):**
```go
// DON'T DO THIS - DEPRECATED
config := &LLMConfig{
    Provider: "ollama",
    Enabled: true,
}
llmExtractor := NewLLMExtractor(config)
results, err := llmExtractor.Extract(emailContent)
```

**New approach (recommended):**
```go
// DO THIS - Use pattern-based tracking extraction
trackingExtractor := NewSimplifiedTrackingExtractor()
trackingResults, err := trackingExtractor.ExtractTrackingNumbers(emailContent)

// And separate description extraction if needed
descriptionExtractor := NewSimplifiedDescriptionExtractor(llmClient, true)
description, err := descriptionExtractor.ExtractDescription(ctx, emailContent, trackingNumber)
```

### From Complex Email Processing to Simplified Email Processing

**Old approach (deprecated):**
```go
// DON'T DO THIS - DEPRECATED
// Multiple complex processors, relevance scoring, etc.
```

**New approach (recommended):**
```go
// DO THIS - Simple, focused algorithm
processor := NewSimplifiedEmailProcessor(
    emailClient,
    trackingExtractor,
    descriptionExtractor,
    shipmentCreator,
    stateManager,
    30, // days to scan
    false, // dry run
)
err := processor.ProcessEmails(ctx)
```

## Deprecated Configuration Components

### ❌ DEPRECATED: Complex Viper-based Multi-format Configuration

The following configuration components are **deprecated**:

- `viper_server.go` - Complex multi-format server configuration
- `viper_email.go` - Complex multi-format email configuration  
- `viper_cli.go` - Complex multi-format CLI configuration
- Multiple file format support (YAML, TOML, JSON)
- Complex search path resolution
- Dual environment variable binding (PKG_TRACKER_ and legacy formats)

**Reason for Deprecation**: 
The complex Viper-based configuration system adds unnecessary complexity for the simplified email processing workflow. A simple environment variable-based approach is sufficient and easier to understand.

### ✅ RECOMMENDED: Simplified Configuration

Use the simplified configuration approach instead:

- `SimplifiedConfig` in `config/simplified.go`
- `LoadSimplifiedConfig()` function
- Simple environment variable parsing
- Focused on essential settings only

**Migration Example:**

**Old approach (deprecated):**
```go
// DON'T DO THIS - DEPRECATED
v := viper.New()
v.AddConfigPath(".")
v.AddConfigPath("./config") 
v.AddConfigPath("$HOME/.package-tracker")
v.SetConfigName("config")
config, err := LoadServerConfigWithViper(v)
```

**New approach (recommended):**
```go
// DO THIS - Simple environment variable loading
config, err := LoadSimplifiedConfig()
```

## Timeline

- **Phase 1 (Completed)**: Simplified extractors and email processor implemented
- **Phase 2 (Completed)**: LLM integration refactored to focus only on description extraction
- **Phase 3 (Completed)**: Configuration simplified to remove complex multi-format support
- **Phase 4 (Future)**: Complete removal of deprecated components (after migration period)

## Support

The deprecated components will continue to work but are not recommended for new development. They may be removed in a future major version.

For questions about migration, refer to the simplified email processing plan in `email-simplification-plan.md`.