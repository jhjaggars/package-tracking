# Detail Questions

## Q6: Should the new email scanning coexist with the current search-based system as a configurable hybrid mode?
**Default if unknown:** Yes (allows gradual migration and fallback capability)
**Technical Context:** The current search-based system at `internal/email/gmail.go` could run alongside time-based scanning, with configuration option `SCAN_MODE` to switch between "search", "time_based", or "hybrid" modes.

## Q7: Should email body storage use the existing SQLite database with compression, or implement a separate storage solution?
**Default if unknown:** Yes (use existing SQLite with compression to maintain architectural consistency)
**Technical Context:** Email bodies can be large. The current system uses SQLite exclusively. We could add a compressed TEXT column to store email bodies, or implement a separate storage layer.

## Q8: Should the email-to-shipment linking be automatic based on tracking numbers, or also allow manual linking through the UI?
**Default if unknown:** Yes (support both automatic and manual linking for flexibility)
**Technical Context:** The current system auto-creates shipments from tracking numbers. We could extend this to auto-link emails to existing shipments, plus add manual linking controls in the web interface.

## Q9: Should the system implement email thread conversation tracking to group related emails together?
**Default if unknown:** Yes (Gmail thread IDs provide natural conversation grouping)
**Technical Context:** Gmail API provides thread IDs that group related emails. We could create an `email_threads` table to store conversation data and display emails as threaded conversations.

## Q10: Should the configurable time period setting be stored in the existing Viper configuration system at `internal/config/viper_email.go`?
**Default if unknown:** Yes (maintains consistency with existing configuration patterns)
**Technical Context:** The current email config uses both legacy environment variables and modern Viper configuration. New time-based settings should follow the Viper pattern for consistency.

## Progress
- Total Questions: 5
- Answered: 0
- Status: Ready to begin asking expert questions