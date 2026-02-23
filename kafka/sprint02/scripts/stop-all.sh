#!/bin/bash

echo "Stopping Kafka Message Filter System..."

# Stop Docker containers
docker-compose down

echo "System stopped successfully!"
echo ""
echo "To remove all data (including Kafka topics), run:"
echo "  docker-compose down -v"
