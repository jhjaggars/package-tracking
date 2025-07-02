# Initial Request

**Date:** 2025-06-30 11:00
**Request:** update the start-dev.sh script to use tmux to start the servers, it should check to see if a tmux session is already running an only create one if there isn't a running session

## Context
User wants to enhance the development workflow by using tmux to manage development servers. The script should be intelligent about session management to avoid creating duplicate sessions.