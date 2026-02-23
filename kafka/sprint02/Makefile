# Kafka Message Filter System Makefile

.PHONY: help docker-up docker-down docker-test docker-logs docker-rebuild setup start-infra stop-infra create-topics processor consumer producer test clean

help:
	@echo "Kafka Message Filter System - Available commands:"
	@echo ""
	@echo "Docker Commands (Recommended):"
	@echo "  make docker-up        - Start all services in Docker containers"
	@echo "  make docker-down      - Stop all Docker containers"
	@echo "  make docker-test      - Run test producer in Docker"
	@echo "  make docker-logs      - Follow logs from all containers"
	@echo "  make docker-rebuild   - Rebuild and restart containers"
	@echo ""
	@echo "Local Development Commands:"
	@echo "  make setup            - Setup infrastructure and dependencies"
	@echo "  make start-infra      - Start Kafka infrastructure only"
	@echo "  make stop-infra       - Stop Kafka infrastructure"
	@echo "  make create-topics    - Create Kafka topics"
	@echo "  make processor        - Run message processor locally"
	@echo "  make consumer         - Run filtered messages consumer locally"
	@echo "  make producer         - Run test data producer locally"
	@echo "  make test             - Run complete test scenario locally"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean            - Clean all data and stop containers"
	@echo ""

# Docker commands
docker-up:
	@echo "Starting all services in Docker..."
	@docker-compose up -d --build
	@echo ""
	@echo "Services started successfully!"
	@echo "Kafka UI: http://localhost:8080"
	@echo ""
	@echo "View logs:"
	@echo "  docker-compose logs -f app-processor"
	@echo "  docker-compose logs -f app-consumer"
	@echo ""
	@echo "Run tests:"
	@echo "  make docker-test"

docker-down:
	@echo "Stopping all Docker containers..."
	@docker-compose down

docker-test:
	@echo "Running test producer..."
	@docker-compose --profile test up app-producer

docker-logs:
	@docker-compose logs -f

docker-rebuild:
	@echo "Rebuilding and restarting containers..."
	@docker-compose down
	@docker-compose up -d --build

# Local development commands
setup: start-infra
	@echo "Waiting for Kafka to be ready..."
	@sleep 60
	@echo "Downloading Go dependencies..."
	@go mod download
	@go mod tidy
	@$(MAKE) create-topics
	@echo ""
	@echo "Setup complete! Kafka UI available at http://localhost:8080"

start-infra:
	@echo "Starting Kafka infrastructure..."
	@docker-compose up -d kafka zookeeper kafka-ui
	@echo "Infrastructure started"

stop-infra:
	@echo "Stopping Kafka infrastructure..."
	@docker-compose down
	@echo "Infrastructure stopped"

create-topics:
	@echo "Creating Kafka topics..."
	@docker exec kafka kafka-topics --create --topic messages --bootstrap-server localhost:9093 --partitions 3 --replication-factor 1 --if-not-exists 2>/dev/null || true
	@docker exec kafka kafka-topics --create --topic filtered_messages --bootstrap-server localhost:9093 --partitions 3 --replication-factor 1 --if-not-exists 2>/dev/null || true
	@docker exec kafka kafka-topics --create --topic blocked_users --bootstrap-server localhost:9093 --partitions 3 --replication-factor 1 --if-not-exists 2>/dev/null || true
	@docker exec kafka kafka-topics --create --topic censor-actions --bootstrap-server localhost:9093 --partitions 1 --replication-factor 1 --if-not-exists 2>/dev/null || true
	@echo "Topics created"

processor:
	@echo "Starting message processor locally..."
	@go run cmd/processor/main.go

consumer:
	@echo "Starting filtered messages consumer locally..."
	@go run cmd/consumer/main.go

producer:
	@echo "Starting test data producer locally..."
	@go run cmd/producer/main.go

test: producer

clean:
	@echo "Cleaning up..."
	@docker-compose down -v
	@rm -rf block-manager/ censor-manager/ message-filter/ bin/
	@echo "Cleanup complete"
