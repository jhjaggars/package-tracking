# Discovery Questions

## Q1: Should the tmux session be automatically attached when the script runs?
**Default if unknown:** Yes (developers typically want to see the running servers immediately)

## Q2: Should the script create separate tmux windows/panes for backend and frontend servers?
**Default if unknown:** Yes (separating concerns makes it easier to monitor and debug each service)

## Q3: Should the script preserve the existing cleanup behavior when Ctrl+C is pressed?
**Default if unknown:** Yes (graceful shutdown is important for development workflow)

## Q4: Should the script display the same informational messages about server URLs and tips?
**Default if unknown:** Yes (developers rely on these messages for quick reference)

## Q5: Should the tmux session have a specific name to make it easy to reconnect later?
**Default if unknown:** Yes (named sessions are easier to find and manage)