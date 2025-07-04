# Expert Detail Questions - Email Tracker Enhancement

## Q6: Should the enhanced LLM prompt use few-shot examples with real email samples to improve merchant and description extraction accuracy?
**Default if unknown:** Yes (based on 2025 LLM best practices showing significant accuracy improvements with few-shot prompting)

## Q7: Should the system modify the Gmail search query to include order confirmation emails (from:amazon.com, from:shopify.com, etc.) alongside shipping notifications?
**Default if unknown:** Yes (order confirmations typically contain better product descriptions than shipping notifications)

## Q8: Should the merchant field be added to the existing shipment API payload, or should merchant information be embedded within the description field?
**Default if unknown:** Embedded in description (based on user preference from Q4: "merchant should be captured in the description")

## Q9: Should the system maintain backward compatibility by making the enhanced extraction optional via a configuration flag (e.g., ENABLE_ENHANCED_EXTRACTION=true)?
**Default if unknown:** Yes (allows gradual rollout and fallback to current behavior if issues arise)

## Q10: Should the system implement confidence-based fallback where low-confidence LLM extractions fall back to the current regex-only approach?
**Default if unknown:** Yes (ensures reliability while benefiting from LLM enhancement when confident)