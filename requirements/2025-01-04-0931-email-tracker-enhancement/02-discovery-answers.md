# Discovery Answers - Email Tracker Enhancement

## Q1: Should the email-tracker continue to use Gmail's search API but modify the query to target the last 30 days of unread emails?
**Answer:** Yes

## Q2: Should the system extract product/item descriptions from order confirmation emails (Amazon, merchants) in addition to shipping notifications?
**Answer:** Yes

## Q3: Should the description extraction prioritize structured data from HTML emails over plain text parsing?
**Answer:** No - The extraction should be performed by an LLM primarily. Let's defer other types of parsing for another change.

## Q4: Should the shipper extraction distinguish between the shipping carrier (UPS, FedEx) and the merchant/retailer (Amazon, Best Buy)?
**Answer:** No - The shipper should be stored in the carrier field and the merchant should be captured in the description.

## Q5: Should the system use the existing LLM integration to enhance description and shipper extraction for complex emails?
**Answer:** Yes