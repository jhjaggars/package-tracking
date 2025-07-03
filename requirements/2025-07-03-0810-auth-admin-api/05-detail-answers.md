# Detail Answers

**Date:** 2025-07-03

## Q1: Should the API key be passed in the Authorization header using Bearer token format?
**Answer:** Yes

## Q2: Should we generate a secure random API key automatically if ADMIN_API_KEY is not provided?
**Answer:** No

## Q3: Should authentication failures include the specific reason in the response body?
**Answer:** No

## Q4: Should the API key be redacted when logging configuration at startup?
**Answer:** Yes

## Q5: Should we allow multiple comma-separated API keys in ADMIN_API_KEY for key rotation?
**Answer:** No

## Summary
- Use standard "Authorization: Bearer <api-key>" format
- Require explicit API key configuration (no auto-generation)
- Return generic "Unauthorized" without specific reasons
- Redact API key in logs for security
- Single API key only (no multi-key support)