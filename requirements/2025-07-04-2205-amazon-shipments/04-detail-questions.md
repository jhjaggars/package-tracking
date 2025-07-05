# Expert Detail Questions

Based on deep analysis of the codebase architecture, here are the specific implementation questions:

## Q6: Should Amazon shipments that delegate to third-party carriers create separate shipment records or update the original Amazon shipment?
**Default if unknown:** Update original Amazon shipment (maintains single source of truth while storing delegation information in new fields)

## Q7: Should the system attempt to scrape Amazon directly for AMZL tracking, or rely solely on email parsing until Amazon provides public APIs?
**Default if unknown:** Email parsing only (avoids legal/technical risks of scraping Amazon's authenticated pages)

## Q8: When Amazon emails contain both order numbers and third-party tracking numbers, should the system create one shipment or two?
**Default if unknown:** One shipment (Amazon shipment with delegated tracking fields populated for seamless user experience)

## Q9: Should Amazon order number validation follow the existing pattern of adding to ValidateTrackingNumber() method in the carrier client?
**Default if unknown:** Yes (maintains consistency with existing UPS/FedEx/DHL validation patterns in internal/carriers/)

## Q10: Should the email tracker daemon automatically detect Amazon emails using the existing Gmail search patterns, or require specific Amazon email configuration?
**Default if unknown:** Automatic detection (extends existing email processing patterns in internal/parser/patterns.go)

## Reasoning for Defaults:
- **Q6**: Single shipment record prevents data duplication while new fields track delegation
- **Q7**: Email parsing is safer, more reliable, and follows existing successful patterns
- **Q8**: One shipment provides cleaner UX; users think "I'm tracking my Amazon order"
- **Q9**: Consistency with existing carrier validation patterns in the factory
- **Q10**: Automatic detection leverages existing email processing infrastructure