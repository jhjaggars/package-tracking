#!/bin/bash

# Package Tracker Production Preview Script
# This builds the frontend and serves it alongside the backend

set -e

echo "🏗️  Building Package Tracker for Production Preview..."
echo ""

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -d "web" ]; then
    echo "❌ Error: Please run this script from the package-tracking root directory"
    exit 1
fi

# Function to cleanup processes on exit
cleanup() {
    echo ""
    echo "🛑 Shutting down server..."
    if [ ! -z "$BACKEND_PID" ]; then
        kill $BACKEND_PID 2>/dev/null || true
    fi
    echo "✅ Cleanup complete"
    exit 0
}

# Set up signal handlers
trap cleanup SIGINT SIGTERM

# Build the Go backend
echo "📦 Building Go backend..."
go build -o bin/server cmd/server/main.go

# Build the frontend
echo "🎨 Building frontend for production..."
cd web
npm run build
cd ..

# Start the backend server (it will serve the built frontend)
echo ""
echo "🚀 Starting production server..."
echo ""
echo "🌐 Application will be available at: http://localhost:8080"
echo ""
echo "💡 Press Ctrl+C to stop the server"
echo ""

./bin/server &
BACKEND_PID=$!

# Wait for the process
wait $BACKEND_PID