# Detail Questions

## Q1: Should the interactive mode be triggered by default when no format flags are specified and stdout is a TTY?
**Default if unknown:** Yes (provides natural user experience while preserving script compatibility)

## Q2: Should the interactive table reuse the existing StyleConfig and color detection logic from internal/cli/output.go?
**Default if unknown:** Yes (maintains visual consistency and respects user environment settings)

## Q3: Should operations like refresh and update show confirmation dialogs or execute immediately when selected?
**Default if unknown:** No confirmation for refresh/update, Yes for delete (matches current CLI behavior where only destructive actions need confirmation)

## Q4: Should the interactive table support multi-select operations or only single-row operations?
**Default if unknown:** Single-row operations only (simpler implementation and matches current CLI patterns)

## Q5: Should field configuration be done through CLI flags (like --fields) or through a config file?
**Default if unknown:** CLI flags (--fields=id,tracking,status,description) (consistent with existing CLI flag patterns and easier for users)