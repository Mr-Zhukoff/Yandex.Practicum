#!/bin/bash

echo "Starting Kafka Message Filter System..."
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "Error: Docker is not running. Please start Docker first."
    exit 1
fi

# Start infrastructure
echo "Step 1: Starting Kafka infrastructure..."
docker-compose up -d

echo "Waiting for Kafka to be ready (60 seconds)..."
sleep 60

# Create topics
echo ""
echo "Step 2: Creating Kafka topics..."
bash scripts/create-topics.sh

# Download Go dependencies
echo ""
echo "Step 3: Downloading Go dependencies..."
go mod download
go mod tidy

echo ""
echo "Setup complete!"
echo ""
echo "To start the system, run these commands in separate terminals:"
echo "  1. go run cmd/processor/main.go    # Start the processor"
echo "  2. go run cmd/consumer/main.go     # Start the consumer (optional)"
echo "  3. go run cmd/producer/main.go     # Send test data"
echo ""
echo "Kafka UI is available at: http://localhost:8080"
