# Discovery Answers

## Q1: Should the interactive table remain compatible with existing --format and --quiet flags?
**Answer:** Yes

## Q2: Should the interactive table support keyboard navigation (arrow keys, vim-style keys)?
**Answer:** Yes

## Q3: Should the interactive table work alongside the existing JSON output format?
**Answer:** No - using format or quiet flags implies that it is being used in a script or pipeline

## Q4: Should the interactive table display all shipment data or focus on key fields like the current table?
**Answer:** Key fields by default, we should enable the user to configure the fields that are displayed

## Q5: Should the interactive table support real-time updates or operate on a static snapshot?
**Answer:** Static snapshot