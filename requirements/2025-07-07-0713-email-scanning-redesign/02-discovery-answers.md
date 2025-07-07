# Discovery Answers

## Q1: Should the new system continue to use the existing Gmail API for email access?
**Answer:** Yes

## Q2: Should the system store complete email bodies including HTML content and attachments?
**Answer:** Keep the text content, or html if that is the only body content. Do not keep attachments. The body content should be storable in the database so that it can be shown to the user via the ui/cli.

## Q3: Should the configurable time period (30 days) be retroactive when first implemented?
**Answer:** Yes

## Q4: Should the system continue to create shipments automatically when tracking numbers are found?
**Answer:** Yes

## Q5: Should the email chain review feature be accessible through the existing web interface?
**Answer:** Yes

## Summary
All discovery questions have been answered. The system will:
- Continue using Gmail API
- Store text/HTML email bodies (no attachments) in database
- Retroactively scan historical emails on first implementation
- Maintain automatic shipment creation
- Provide email chain review through the web interface