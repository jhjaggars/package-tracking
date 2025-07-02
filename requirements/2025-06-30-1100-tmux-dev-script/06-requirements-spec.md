# Requirements Specification: Tmux Development Script

## Problem Statement

The current `start-dev.sh` script blocks the terminal session by running backend and frontend servers in the foreground with signal handling. This prevents developers from using the same terminal for other tasks while the development servers are running. The solution is to migrate to tmux session management, allowing detached server processes while maintaining the same development workflow.

## Solution Overview

Rewrite `start-dev.sh` to create a detached tmux session with separate windows for backend and frontend servers. The script will perform all setup tasks, launch the servers in tmux, display connection information, and immediately return control to the shell.

## Functional Requirements

### FR1: Command Line Interface
- Accept optional session name argument: `./start-dev.sh [session-name]`
- Default session name: `package-tracker-dev`
- Script exits immediately after setup completion

### FR2: Session Management
- Check for existing tmux sessions with the same name
- Gracefully handle existing sessions (reuse or inform user)
- Create detached tmux session with specified name
- Create separate tmux windows for backend and frontend services

### FR3: Environment Validation
- Validate tmux installation before proceeding
- Preserve existing validation: Go, Node.js, npm, directory checks
- Maintain same error messaging patterns

### FR4: Development Workflow
- Preserve exact build process: Go build, npm install
- Start backend server in dedicated tmux window
- Start frontend dev server in dedicated tmux window
- Maintain all existing informational output

### FR5: User Guidance
- Display same server URLs and development tips
- Show tmux session attachment instructions
- Provide session management commands (list, attach, kill)

## Technical Requirements

### TR1: File Modifications
**Primary Target:** `start-dev.sh`
- Complete rewrite using tmux commands
- Maintain existing script structure and validation logic
- Remove blocking `wait` commands and signal handlers

### TR2: Tmux Session Structure
```
Session: [user-specified-name]
├── Window 0: "backend" - Go server
└── Window 1: "frontend" - npm dev server
```

### TR3: Implementation Pattern
```bash
# Session creation pattern
SESSION_NAME="${1:-package-tracker-dev}"
if ! tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
    tmux new-session -d -s "$SESSION_NAME"
fi

# Window setup pattern
tmux rename-window -t "$SESSION_NAME:0" 'backend'
tmux new-window -t "$SESSION_NAME:1" -n 'frontend'

# Command execution pattern
tmux send-keys -t "$SESSION_NAME:backend" './bin/server' C-m
tmux send-keys -t "$SESSION_NAME:frontend" 'cd web && npm run dev' C-m
```

### TR4: Error Handling
- Validate tmux installation with clear error messages
- Check session creation success
- Provide fallback messaging for tmux command failures
- Maintain existing Go/Node.js/npm validation patterns

### TR5: Output Requirements
- Preserve all existing informational messages
- Add tmux-specific guidance:
  - Session name confirmation
  - Attach command: `tmux attach -t [session-name]`
  - List sessions: `tmux list-sessions`
  - Kill session: `tmux kill-session -t [session-name]`

## Implementation Hints

### Preserve from Current Implementation
- Lines 8-22: Directory and environment validation
- Lines 42-57: Go/Node.js/npm installation checks
- Lines 59-67: Build process (go build, npm install)
- Lines 70-101: Informational output structure

### New Implementation Sections
1. **Tmux validation** (after line 57)
2. **Session argument handling** (after line 7)
3. **Session existence check** (before session creation)
4. **Tmux session creation and window setup** (replace lines 74-103)
5. **User instruction output** (replace wait section)

### Command Reference
```bash
# Check tmux installation
command -v tmux &> /dev/null

# Check session exists
tmux has-session -t "$SESSION_NAME" 2>/dev/null

# Create detached session
tmux new-session -d -s "$SESSION_NAME"

# Window management
tmux rename-window -t "$SESSION_NAME:0" 'backend'
tmux new-window -t "$SESSION_NAME:1" -n 'frontend'

# Send commands
tmux send-keys -t "$SESSION_NAME:backend" 'command' C-m
tmux send-keys -t "$SESSION_NAME:frontend" 'command' C-m
```

## Acceptance Criteria

### AC1: Script Execution
- [ ] Script accepts optional session name argument
- [ ] Script validates all dependencies (tmux, Go, Node.js, npm)
- [ ] Script performs build steps (go build, npm install)
- [ ] Script exits immediately after setup completion

### AC2: Tmux Session Management
- [ ] Creates detached tmux session with specified name
- [ ] Handles existing sessions gracefully
- [ ] Creates separate windows for backend and frontend
- [ ] Starts both servers in their respective windows

### AC3: User Experience
- [ ] Displays all existing informational messages
- [ ] Shows tmux session attachment instructions
- [ ] Provides session management commands
- [ ] Maintains same error handling quality

### AC4: Development Workflow
- [ ] Backend server accessible at http://localhost:8080
- [ ] Frontend server accessible at http://localhost:5173
- [ ] Hot reload functionality preserved
- [ ] Development environment matches current behavior

### AC5: Session Reusability
- [ ] Can attach to running session: `tmux attach -t [session-name]`
- [ ] Can list active sessions: `tmux list-sessions`
- [ ] Can stop servers: `tmux kill-session -t [session-name]`
- [ ] Multiple sessions can coexist with different names

## Assumptions

1. **Tmux Installation**: Users have tmux installed or can install it
2. **Session Reuse Strategy**: Existing sessions are preserved and script informs user
3. **Working Directory**: All tmux commands execute from project root
4. **Window Naming**: "backend" and "frontend" are descriptive enough for developer workflow
5. **Command Timing**: No additional delays needed between tmux commands
6. **Error Recovery**: Standard tmux error messages are sufficient for debugging

## Documentation Updates

- Update README.md development section with tmux workflow
- Add tmux session management commands to CLAUDE.md
- Include session attachment examples in development documentation