# Discovery Questions

## Q1: Should the interactive table remain compatible with existing --format and --quiet flags?
**Default if unknown:** Yes (maintains backward compatibility with existing scripts and workflows)

## Q2: Should the interactive table support keyboard navigation (arrow keys, vim-style keys)?
**Default if unknown:** Yes (standard expectation for terminal UI applications)

## Q3: Should the interactive table work alongside the existing JSON output format?
**Default if unknown:** No (interactive mode conflicts with structured output for scripts)

## Q4: Should the interactive table display all shipment data or focus on key fields like the current table?
**Default if unknown:** Key fields only (matches current behavior and improves readability)

## Q5: Should the interactive table support real-time updates or operate on a static snapshot?
**Default if unknown:** Static snapshot (simpler implementation and avoids API polling)