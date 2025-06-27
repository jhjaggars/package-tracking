#!/bin/bash

echo "Demonstrating Server Signal Handling with Logs"
echo "=============================================="

# Test graceful shutdown
echo ""
echo "=== 1. Graceful Shutdown (SIGTERM) ==="
echo "Starting server..."

SERVER_PORT=8084 go run cmd/server/main.go &
SERVER_PID=$!

sleep 2
echo "Server started, sending graceful shutdown signal..."
echo "Command: kill -TERM $SERVER_PID"

kill -TERM $SERVER_PID
wait $SERVER_PID

echo "^ Notice the graceful shutdown logs above"

# Test immediate termination  
echo ""
echo "=== 2. Immediate Termination (SIGKILL) ==="
echo "Starting server..."

SERVER_PORT=8085 go run cmd/server/main.go &
SERVER_PID=$!

sleep 2
echo "Server started, sending immediate kill signal..."
echo "Command: kill -9 $SERVER_PID"

kill -9 $SERVER_PID

echo "^ Notice NO graceful shutdown logs - process terminated immediately"

echo ""
echo "=== Summary ==="
echo "✅ SIGTERM: Server logs graceful shutdown process"
echo "⚡ SIGKILL: No logs - immediate termination by OS"
echo ""
echo "Key difference: SIGKILL bypasses ALL application code"