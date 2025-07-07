# Detail Answers

## Q6: Should the new email scanning coexist with the current search-based system as a configurable hybrid mode?
**Answer:** No, remove the search mode

## Q7: Should email body storage use the existing SQLite database with compression, or implement a separate storage solution?
**Answer:** Yes

## Q8: Should the email-to-shipment linking be automatic based on tracking numbers, or also allow manual linking through the UI?
**Answer:** Yes

## Q9: Should the system implement email thread conversation tracking to group related emails together?
**Answer:** Yes

## Q10: Should the configurable time period setting be stored in the existing Viper configuration system at `internal/config/viper_email.go`?
**Answer:** Yes

## Summary
All expert questions have been answered. The system will:
- Replace search-based scanning entirely with time-based scanning
- Use existing SQLite database with compression for email body storage
- Support both automatic and manual email-to-shipment linking
- Implement email thread conversation tracking using Gmail thread IDs
- Store new configuration options in the existing Viper configuration system