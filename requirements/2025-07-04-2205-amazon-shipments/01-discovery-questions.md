# Discovery Questions

Based on the analysis of the package tracking system, here are the key discovery questions to understand how Amazon shipments should be integrated:

## Q1: Will users primarily track Amazon shipments using Amazon order numbers rather than carrier tracking numbers?
**Default if unknown:** Yes (Amazon emails typically show order numbers prominently before revealing carrier tracking numbers)

## Q2: Should the system extract the actual carrier tracking numbers from Amazon emails and route them to existing carrier implementations?
**Default if unknown:** Yes (leverages existing UPS, USPS, FedEx, DHL integrations rather than scraping Amazon directly)

## Q3: Do users need to track Amazon shipments that use Amazon's own logistics network (not UPS/USPS/FedEx/DHL)?
**Default if unknown:** Yes (Amazon Logistics is increasingly common and users would expect this to work)

## Q4: Will users authenticate with their Amazon accounts to access tracking information?
**Default if unknown:** No (authentication adds complexity and the email-based approach is more practical)

## Q5: Should Amazon shipments be treated as a separate carrier type or as a wrapper around existing carriers?
**Default if unknown:** Separate carrier type (provides cleaner user experience and allows for Amazon-specific features)

## Reasoning for Defaults:
- **Q1**: Amazon's customer-facing communications emphasize order numbers
- **Q2**: Reusing existing carrier integrations is more maintainable and reliable
- **Q3**: Amazon Logistics is a major delivery method that users expect to track
- **Q4**: Email parsing is already implemented and avoids Amazon's anti-bot measures
- **Q5**: Separate carrier type provides cleaner UX and allows Amazon-specific handling