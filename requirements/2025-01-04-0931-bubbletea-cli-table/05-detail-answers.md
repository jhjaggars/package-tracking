# Detail Answers

## Q1: Should the interactive mode be triggered by default when no format flags are specified and stdout is a TTY?
**Answer:** Yes

## Q2: Should the interactive table reuse the existing StyleConfig and color detection logic from internal/cli/output.go?
**Answer:** Yes

## Q3: Should operations like refresh and update show confirmation dialogs or execute immediately when selected?
**Answer:** Use the same pattern (no confirmation for refresh/update, Yes for delete)

## Q4: Should the interactive table support multi-select operations or only single-row operations?
**Answer:** Single row operations

## Q5: Should field configuration be done through CLI flags (like --fields) or through a config file?
**Answer:** CLI flags