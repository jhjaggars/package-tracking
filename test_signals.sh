#!/bin/bash

echo "Testing Server Signal Handling..."

# Function to test a specific signal
test_signal() {
    local signal_name=$1
    local signal_flag=$2
    
    echo ""
    echo "=== Testing $signal_name ==="
    
    # Start server in background
    echo "Starting server..."
    go run cmd/server/main.go &
    SERVER_PID=$!
    
    # Wait for server to start
    sleep 2
    
    # Test that server is responding
    echo "Testing server responsiveness..."
    HEALTH_RESPONSE=$(curl -s -w "%{http_code}" http://localhost:8080/api/health)
    echo "Health check: $HEALTH_RESPONSE"
    
    # Send the signal
    echo "Sending $signal_name to PID $SERVER_PID..."
    if [ "$signal_name" = "SIGKILL" ]; then
        kill -9 $SERVER_PID
    else
        kill $signal_flag $SERVER_PID
    fi
    
    # Wait a moment and check if process still exists
    sleep 1
    if kill -0 $SERVER_PID 2>/dev/null; then
        echo "Process still alive after $signal_name"
        sleep 5  # Wait for graceful shutdown
        if kill -0 $SERVER_PID 2>/dev/null; then
            echo "Process still alive after 5 seconds - force killing"
            kill -9 $SERVER_PID
        else
            echo "Process gracefully shut down"
        fi
    else
        echo "Process terminated immediately"
    fi
    
    echo "Test complete for $signal_name"
}

# Test different signals
test_signal "SIGTERM" "-TERM"
test_signal "SIGINT" "-INT"  
test_signal "SIGKILL" "-9"

echo ""
echo "Signal testing complete!"