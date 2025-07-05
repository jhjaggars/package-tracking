# Expert Detail Answers

## Q6: Should Amazon shipments that delegate to third-party carriers create separate shipment records or update the original Amazon shipment?
**Answer:** Update original Amazon shipment

## Q7: Should the system attempt to scrape Amazon directly for AMZL tracking, or rely solely on email parsing until Amazon provides public APIs?
**Answer:** Email parsing only

## Q8: When Amazon emails contain both order numbers and third-party tracking numbers, should the system create one shipment or two?
**Answer:** One shipment

## Q9: Should Amazon order number validation follow the existing pattern of adding to ValidateTrackingNumber() method in the carrier client?
**Answer:** Yes

## Q10: Should the email tracker daemon automatically detect Amazon emails using the existing Gmail search patterns, or require specific Amazon email configuration?
**Answer:** Automatic detection

## Implementation Summary
Based on these answers, the Amazon integration will:
- Update original Amazon shipments with delegation information rather than creating separate records
- Use email parsing exclusively for AMZL tracking (no direct Amazon scraping)
- Create single shipment records that represent Amazon orders with delegation fields
- Implement standard carrier validation patterns in AmazonClient.ValidateTrackingNumber()
- Automatically detect Amazon emails using existing pattern system infrastructure