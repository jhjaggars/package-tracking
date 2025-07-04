# Expert Detail Answers - Email Tracker Enhancement

## Q6: Should the enhanced LLM prompt use few-shot examples with real email samples to improve merchant and description extraction accuracy?
**Answer:** Yes

## Q7: Should the system modify the Gmail search query to include order confirmation emails (from:amazon.com, from:shopify.com, etc.) alongside shipping notifications?
**Answer:** No - The search should be used to fetch emails in the date range, filtering from and subject will miss messages

## Q8: Should the merchant field be added to the existing shipment API payload, or should merchant information be embedded within the description field?
**Answer:** Embed into description

## Q9: Should the system maintain backward compatibility by making the enhanced extraction optional via a configuration flag (e.g., ENABLE_ENHANCED_EXTRACTION=true)?
**Answer:** No, let's just change the behavior

## Q10: Should the system implement confidence-based fallback where low-confidence LLM extractions fall back to the current regex-only approach?
**Answer:** Yes, extracting the tracking number is more important and we would rather have a number and no description than nothing at all