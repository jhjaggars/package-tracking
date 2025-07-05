# Discovery Answers

## Q1: Will users primarily track Amazon shipments using Amazon order numbers rather than carrier tracking numbers?
**Answer:** Yes

## Q2: Should the system extract the actual carrier tracking numbers from Amazon emails and route them to existing carrier implementations?
**Answer:** Yes

## Q3: Do users need to track Amazon shipments that use Amazon's own logistics network (not UPS/USPS/FedEx/DHL)?
**Answer:** Yes

## Q4: Will users authenticate with their Amazon accounts to access tracking information?
**Answer:** No

## Q5: Should Amazon shipments be treated as a separate carrier type or as a wrapper around existing carriers?
**Answer:** Yes (separate carrier type)

## Summary of Approach
Based on these answers, the Amazon integration should:
- Use Amazon order numbers as primary identifiers
- Parse Amazon emails to extract carrier tracking numbers
- Support both Amazon Logistics and third-party carrier shipments
- Rely on email parsing rather than account authentication
- Present Amazon as a separate carrier type to users while internally delegating to existing carriers when possible