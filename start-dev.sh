#!/bin/bash

# Package Tracker Development Server Startup Script
# This script starts both the Go backend and React frontend servers in a tmux session

set -e

# Parse command line arguments
RESTART=false
SESSION_NAME=""

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --restart)
      RESTART=true
      shift
      ;;
    *)
      SESSION_NAME="$1"
      shift
      ;;
  esac
done

# Set default session name if not provided
SESSION_NAME="${SESSION_NAME:-package-tracker-dev}"

echo "🚀 Starting Package Tracker Development Environment..."
echo "📺 Session: $SESSION_NAME"
echo ""

# Check for .env file and suggest creating one
if [ ! -f ".env" ]; then
    echo "💡 Tip: You can create a .env file for custom configuration:"
    echo "   cp .env.example .env"
    echo ""
fi

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -d "web" ]; then
    echo "❌ Error: Please run this script from the package-tracking root directory"
    exit 1
fi

# Check if tmux is installed
if ! command -v tmux &> /dev/null; then
    echo "❌ Error: tmux is not installed or not in PATH"
    echo "💡 Install tmux with: sudo apt-get install tmux (Ubuntu/Debian) or brew install tmux (macOS)"
    exit 1
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed or not in PATH"
    exit 1
fi

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo "❌ Error: Node.js is not installed or not in PATH"
    exit 1
fi

# Check if npm is installed
if ! command -v npm &> /dev/null; then
    echo "❌ Error: npm is not installed or not in PATH"
    exit 1
fi

# Check if session already exists and handle restart
if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
    if [ "$RESTART" = true ]; then
        echo "🔄 Killing existing session '$SESSION_NAME' and restarting..."
        tmux kill-session -t "$SESSION_NAME"
    else
        echo "🔄 Tmux session '$SESSION_NAME' already exists"
        echo "💡 To attach: tmux attach -t $SESSION_NAME"
        echo "💡 To kill:   tmux kill-session -t $SESSION_NAME"
        echo "💡 To restart: $0 --restart"
        echo "💡 To list:   tmux list-sessions"
        exit 0
    fi
fi

echo "📱 Installing frontend dependencies..."
cd web
if [ ! -d "node_modules" ]; then
    npm install
fi
cd ..

echo ""
echo "🎉 Creating tmux session and starting servers..."
echo ""

# Create detached tmux session
tmux new-session -d -s "$SESSION_NAME"

# Rename first window to 'backend'
tmux rename-window -t "$SESSION_NAME:0" 'backend'

# Create second window for frontend
tmux new-window -t "$SESSION_NAME:1" -n 'frontend'

# Start backend server in first window
echo "🔧 Starting backend server on http://localhost:8080"
tmux send-keys -t "$SESSION_NAME:backend" 'go run cmd/server/main.go' C-m

# Start frontend server in second window
echo "🎨 Starting frontend server on http://localhost:5173"
tmux send-keys -t "$SESSION_NAME:frontend" 'cd web && npm run dev' C-m

# Give servers a moment to start
sleep 2

echo ""
echo "✨ Development environment is ready!"
echo ""
echo "📍 Backend API:  http://localhost:8080"
echo "🌐 Frontend UI:  http://localhost:5173"
echo ""
echo "🎯 Open http://localhost:5173 in your browser to see the delightful UI!"
echo ""
echo "💡 Tip: The frontend will auto-reload when you make changes"
echo ""
echo "📺 Tmux Session Management:"
echo "   Attach to session:    tmux attach -t $SESSION_NAME"
echo "   List all sessions:    tmux list-sessions"
echo "   Kill this session:    tmux kill-session -t $SESSION_NAME"
echo "   Restart this session: $0 --restart"
echo "   Detach from session:  Ctrl+b then d"
echo ""
echo "🎮 Inside tmux session:"
echo "   Switch to backend:    Ctrl+b then 0"
echo "   Switch to frontend:   Ctrl+b then 1"
echo "   Stop servers:         Ctrl+C in each window"
echo ""