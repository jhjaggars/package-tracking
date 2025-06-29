#!/bin/bash

# Package Tracker Development Server Startup Script
# This script starts both the Go backend and React frontend servers

set -e

echo "🚀 Starting Package Tracker Development Environment..."
echo ""

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -d "web" ]; then
    echo "❌ Error: Please run this script from the package-tracking root directory"
    exit 1
fi

# Function to cleanup processes on exit
cleanup() {
    echo ""
    echo "🛑 Shutting down servers..."
    if [ ! -z "$BACKEND_PID" ]; then
        kill $BACKEND_PID 2>/dev/null || true
    fi
    if [ ! -z "$FRONTEND_PID" ]; then
        kill $FRONTEND_PID 2>/dev/null || true
    fi
    echo "✅ Cleanup complete"
    exit 0
}

# Set up signal handlers
trap cleanup SIGINT SIGTERM

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

echo "📦 Building Go backend..."
go build -o bin/server cmd/server/main.go

echo "📱 Installing frontend dependencies..."
cd web
if [ ! -d "node_modules" ]; then
    npm install
fi

cd ..

echo ""
echo "🎉 Starting servers..."
echo ""

# Start the Go backend server in the background
echo "🔧 Starting backend server on http://localhost:8080"
./bin/server &
BACKEND_PID=$!

# Give the backend a moment to start
sleep 2

# Start the frontend development server in the background
echo "🎨 Starting frontend server on http://localhost:5173"
cd web
npm run dev &
FRONTEND_PID=$!

cd ..

echo ""
echo "✨ Development environment is ready!"
echo ""
echo "📍 Backend API:  http://localhost:8080"
echo "🌐 Frontend UI:  http://localhost:5173"
echo ""
echo "🎯 Open http://localhost:5173 in your browser to see the delightful UI!"
echo ""
echo "💡 Tip: The frontend will auto-reload when you make changes"
echo "💡 Tip: Press Ctrl+C to stop both servers"
echo ""

# Wait for both processes
wait $BACKEND_PID $FRONTEND_PID