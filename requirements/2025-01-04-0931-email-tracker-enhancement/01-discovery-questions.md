# Discovery Questions - Email Tracker Enhancement

## Q1: Should the email-tracker continue to use Gmail's search API but modify the query to target the last 30 days of unread emails?
**Default if unknown:** Yes (maintains API efficiency while achieving 30-day unread scanning)

## Q2: Should the system extract product/item descriptions from order confirmation emails (Amazon, merchants) in addition to shipping notifications?
**Default if unknown:** Yes (order emails often contain better product descriptions than shipping notifications)

## Q3: Should the description extraction prioritize structured data from HTML emails over plain text parsing?
**Default if unknown:** Yes (HTML emails from merchants typically have more structured product information)

## Q4: Should the shipper extraction distinguish between the shipping carrier (UPS, FedEx) and the merchant/retailer (Amazon, Best Buy)?
**Default if unknown:** Yes (users typically want to know both who sold it and who's delivering it)

## Q5: Should the system use the existing LLM integration to enhance description and shipper extraction for complex emails?
**Default if unknown:** Yes (LLM can help extract meaningful descriptions from unstructured email content)