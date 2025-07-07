# Detail Answers

## Q1: Should validation use the same 5-minute cache TTL as refresh, or a longer period since tracking data is more stable?
**Answer:** Yes, use the same 5-minute TTL

## Q2: Should validation failures be stored in the same `refresh_cache` table or a separate `validation_cache` table?
**Answer:** Same table - the act of validating a number is performing a refresh before including it in the list of valid numbers

## Q3: Should the validation service integrate with the existing `carriers.Factory` at line 57 in `internal/carriers/factory.go`?
**Answer:** Yes

## Q4: Should validation bypass rate limiting when processing emails in batch mode (like the `--dry-run` flag)?
**Answer:** No

## Q5: Should failed validation attempts count against the same tracking fields as refresh (like `last_manual_refresh` in the database)?
**Answer:** Yes, validation is performing a refresh. The behavior we want is to mark an incoming number as invalid if it fails to refresh initially