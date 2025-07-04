# Detail Answers

## Q6: Should we refactor the duplicate helper functions into internal/config/helpers.go to be shared between both config packages?
**Answer:** Yes

## Q7: Should email-tracker support a --config flag to specify an alternative .env file location for testing purposes?
**Answer:** Yes

## Q8: Should the email-tracker CLI have subcommands (like 'run', 'validate', 'version') or just run directly when executed?
**Answer:** No

## Q9: Should we preserve the existing printUsage() function content as Cobra's Long description for the root command?
**Answer:** Yes

## Q10: Should email-tracker support a --dry-run flag at the CLI level that overrides the EMAIL_DRY_RUN environment variable?
**Answer:** Yes