#!/bin/bash

# Test script to verify server functionality
echo "Testing Package Tracking Server..."

# Start server in background
go run cmd/server/main.go &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Test health endpoint
echo "Testing health endpoint..."
HEALTH_RESPONSE=$(curl -s http://localhost:8080/api/health)
echo "Health response: $HEALTH_RESPONSE"

# Test carriers endpoint
echo "Testing carriers endpoint..."
CARRIERS_RESPONSE=$(curl -s http://localhost:8080/api/carriers)
echo "Carriers response: $CARRIERS_RESPONSE"

# Test empty shipments list
echo "Testing shipments list..."
SHIPMENTS_RESPONSE=$(curl -s http://localhost:8080/api/shipments)
echo "Shipments response: $SHIPMENTS_RESPONSE"

# Test creating a shipment
echo "Testing shipment creation..."
CREATE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/shipments \
  -H "Content-Type: application/json" \
  -d '{"tracking_number":"1Z999AA1234567890","carrier":"ups","description":"Test Package"}')
echo "Create response: $CREATE_RESPONSE"

# Extract ID from response (basic parsing)
if [[ $CREATE_RESPONSE == *'"id":'* ]]; then
    SHIPMENT_ID=$(echo $CREATE_RESPONSE | grep -o '"id":[0-9]*' | grep -o '[0-9]*')
    echo "Created shipment with ID: $SHIPMENT_ID"
    
    # Test getting the shipment
    echo "Testing get shipment by ID..."
    GET_RESPONSE=$(curl -s http://localhost:8080/api/shipments/$SHIPMENT_ID)
    echo "Get response: $GET_RESPONSE"
    
    # Test getting shipment events
    echo "Testing get shipment events..."
    EVENTS_RESPONSE=$(curl -s http://localhost:8080/api/shipments/$SHIPMENT_ID/events)
    echo "Events response: $EVENTS_RESPONSE"
fi

# Clean up
echo "Stopping server gracefully..."
kill -TERM $SERVER_PID

# Wait for graceful shutdown (max 5 seconds)
for i in {1..5}; do
    if ! kill -0 $SERVER_PID 2>/dev/null; then
        echo "Server shut down gracefully"
        break
    fi
    echo "Waiting for server shutdown... ($i/5)"
    sleep 1
done

# Force kill if still running
if kill -0 $SERVER_PID 2>/dev/null; then
    echo "Server didn't shut down gracefully, force killing..."
    kill -9 $SERVER_PID
fi

echo "Test complete!"