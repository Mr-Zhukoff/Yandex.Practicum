#!/bin/bash

echo "Creating Kafka topics..."

docker exec -it kafka kafka-topics --create --topic messages \
  --bootstrap-server localhost:9093 \
  --partitions 3 \
  --replication-factor 1 \
  --if-not-exists

docker exec -it kafka kafka-topics --create --topic filtered_messages \
  --bootstrap-server localhost:9093 \
  --partitions 3 \
  --replication-factor 1 \
  --if-not-exists

docker exec -it kafka kafka-topics --create --topic blocked_users \
  --bootstrap-server localhost:9093 \
  --partitions 3 \
  --replication-factor 1 \
  --if-not-exists

docker exec -it kafka kafka-topics --create --topic censor-actions \
  --bootstrap-server localhost:9093 \
  --partitions 1 \
  --replication-factor 1 \
  --if-not-exists

echo ""
echo "Topics created successfully!"
echo ""
echo "Listing all topics:"
docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9093

