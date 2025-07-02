# Context Findings

## Current Implementation Analysis

### Existing start-dev.sh Structure
- **File:** `start-dev.sh` (103 lines)
- **Current behavior:** 
  - Validates environment (Go, Node.js, npm)
  - Builds Go backend to `bin/server`
  - Installs frontend dependencies if needed
  - Starts backend server in background (`./bin/server &`)
  - Starts frontend dev server in background (`npm run dev &` in web/)
  - Uses trap handlers for graceful cleanup on SIGINT/SIGTERM
  - Waits for both processes with `wait $BACKEND_PID $FRONTEND_PID`

### Key Components to Preserve
1. **Environment validation** (lines 42-57): Go, Node.js, npm checks
2. **Directory validation** (lines 19-22): Ensures script runs from project root
3. **Build process** (lines 59-67): Go build and npm install
4. **Informational output** (lines 8-101): Extensive user-friendly messages
5. **Cleanup function** (lines 25-36): Graceful process termination
6. **Signal handling** (line 39): trap cleanup SIGINT SIGTERM

### Related Scripts
- **start-prod.sh**: Single server for production build
- **test_server.sh**: Automated testing with process management

## Tmux Best Practices Research

### Session Management Pattern
```bash
SESSION="package-tracker-dev"
SESSIONEXISTS=$(tmux list-sessions | grep $SESSION)

if [ "$SESSIONEXISTS" = "" ]; then
    tmux new-session -d -s $SESSION
fi
```

### Window/Pane Structure
- Create detached session first
- Rename default window (0) rather than creating it
- Use descriptive window names
- Send commands with `tmux send-keys -t SESSION:WINDOW 'command' C-m`

### Error Handling
- Check for tmux installation
- Validate session creation success
- Provide cleanup for tmux sessions

## Technical Requirements

### Files to Modify
- **Primary:** `start-dev.sh` - Complete rewrite to use tmux
- **Potential:** Documentation updates for new tmux workflow

### Implementation Patterns
1. **Session check and creation** - Avoid duplicate sessions
2. **Window structure** - Separate windows for backend/frontend
3. **Command execution** - Use tmux send-keys for server startup
4. **Cleanup integration** - Maintain existing Ctrl+C behavior
5. **User arguments** - Accept optional session name parameter

### Integration Points
- Preserve all existing validation logic
- Maintain same build process flow
- Keep informational messages for user feedback
- Integrate with existing signal handling patterns

### Dependencies
- **Required:** tmux must be installed
- **Existing:** Go, Node.js, npm (already validated)
- **Unchanged:** All project dependencies remain the same