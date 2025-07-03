# Expert Detail Questions

## Q6: Should the email processor store state about processed emails to avoid duplicates?
**Default if unknown:** Yes (prevents re-processing emails and duplicate shipment creation attempts)

## Q7: Should we implement retry logic when the API server is temporarily unavailable?
**Default if unknown:** Yes (ensures reliability when the main server has downtime)

## Q8: Should the email processor support custom parsing rules for non-standard email formats?
**Default if unknown:** No (start with common carrier formats, add custom rules later if needed)

## Q9: Should the processor mark emails as read/processed after extracting tracking numbers?
**Default if unknown:** Yes (helps track which emails have been processed)

## Q10: Should we implement a dry-run mode that extracts tracking numbers without creating shipments?
**Default if unknown:** Yes (useful for testing email parsing rules without affecting the database)

---

*Note: These detailed questions focus on implementation specifics now that we understand the codebase architecture.*