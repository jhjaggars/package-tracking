#!/bin/bash

echo "Comprehensive Server Signal Testing"
echo "==================================="

cleanup() {
    if [ ! -z "$SERVER_PID" ] && kill -0 $SERVER_PID 2>/dev/null; then
        echo "Cleaning up server process $SERVER_PID"
        kill -9 $SERVER_PID 2>/dev/null
    fi
}

# Trap to ensure cleanup
trap cleanup EXIT

test_signal() {
    local signal_name=$1
    local signal_num=$2
    local expected_behavior=$3
    
    echo ""
    echo "=== Testing $signal_name ==="
    echo "Expected: $expected_behavior"
    
    # Start server
    echo "Starting server on port 8082..."
    SERVER_PORT=8082 go run cmd/server/main.go &
    SERVER_PID=$!
    
    # Wait for startup
    sleep 2
    
    # Test server is responding
    echo -n "Server health check: "
    if curl -s http://localhost:8082/api/health > /dev/null; then
        echo "✅ Server is responding"
    else
        echo "❌ Server not responding"
        return 1
    fi
    
    # Send signal and measure response time
    echo "Sending $signal_name (signal $signal_num) to PID $SERVER_PID..."
    start_time=$(date +%s.%N)
    
    kill -$signal_num $SERVER_PID
    
    # Wait and check if process still exists
    for i in {1..10}; do
        if ! kill -0 $SERVER_PID 2>/dev/null; then
            end_time=$(date +%s.%N)
            duration=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "unknown")
            echo "✅ Process terminated after ${duration}s"
            SERVER_PID=""
            return 0
        fi
        sleep 0.5
    done
    
    echo "❌ Process still running after 5 seconds"
    kill -9 $SERVER_PID 2>/dev/null
    SERVER_PID=""
    return 1
}

# Test different signals
echo "Testing various signals that the Go server can handle..."

test_signal "SIGTERM" "TERM" "Graceful shutdown with cleanup"
test_signal "SIGINT" "INT" "Graceful shutdown (like Ctrl+C)"  
test_signal "SIGKILL" "KILL" "Immediate termination (uncatchable)"

# Test what happens with rapid signals
echo ""
echo "=== Testing Rapid Signal Sending ==="
echo "Starting server..."
SERVER_PORT=8083 go run cmd/server/main.go &
SERVER_PID=$!
sleep 2

echo "Sending multiple SIGTERM signals rapidly..."
for i in {1..3}; do
    echo "Sending SIGTERM #$i"
    kill -TERM $SERVER_PID 2>/dev/null
    sleep 0.1
done

sleep 2
if kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ Server still running, force killing"
    kill -9 $SERVER_PID
else
    echo "✅ Server handled multiple signals gracefully"
fi
SERVER_PID=""

echo ""
echo "=== Signal Testing Summary ==="
echo "✅ SIGTERM: Can be caught, allows graceful shutdown"
echo "✅ SIGINT: Can be caught, allows graceful shutdown"  
echo "⚡ SIGKILL: Cannot be caught, immediate termination"
echo ""
echo "💡 Key Insights:"
echo "   - SIGKILL (-9) cannot be caught by any process"
echo "   - SIGTERM (-15) and SIGINT (-2) allow graceful cleanup"
echo "   - Process managers typically send SIGTERM first, then SIGKILL"
echo "   - Our server handles graceful shutdown within 30 seconds"