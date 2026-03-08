# Kafka Sprint 04 Playground

This project provides a local data platform based on Docker Compose with:

- 3-node Kafka cluster (`kafka-0`, `kafka-1`, `kafka-2`)
- Zookeeper
- Kafka UI
- Schema Registry
- PostgreSQL (source database for CDC)
- Kafka Connect (custom image + plugin mount)

Main compose file: [`docker-compose.yaml`](docker-compose.yaml)

## What this project does

It starts a complete local event-streaming environment for development and learning:

- run a multi-broker Kafka cluster,
- inspect brokers/topics/messages in Kafka UI,
- register and resolve schemas in Schema Registry,
- run Kafka Connect connectors,
- capture changes from PostgreSQL and stream them into Kafka.

## Prerequisites

- Docker Desktop (with Docker Compose)

## How to run

From the project root (where [`docker-compose.yaml`](docker-compose.yaml) is located):

```bash
docker compose up -d --build
```

Check container status:

```bash
docker compose ps
```

View logs:

```bash
docker compose logs -f
```

Stop everything:

```bash
docker compose down
```

Stop and remove volumes (clean reset):

```bash
docker compose down -v
```

Create topic `mytopics` directly via Docker Compose:

```bash
docker compose exec kafka-0 kafka-topics.sh --bootstrap-server kafka-0:9092 --create --if-not-exists --topic mytopics --partitions 3 --replication-factor 3
```

List topics:

```bash
docker compose exec kafka-0 kafka-topics.sh --bootstrap-server kafka-0:9092 --list
```

## Service endpoints

- Kafka UI: <http://127.0.0.1:8080>
- Schema Registry: <http://127.0.0.1:8081>
- Kafka Connect REST API: <http://127.0.0.1:8083>
- PostgreSQL: `127.0.0.1:5432`

## Notes

- Kafka UI and Schema Registry are configured with all three brokers for bootstrap/failover.
- Kafka Connect image is defined in [`kafka-connect/Dockerfile`](kafka-connect/Dockerfile) and uses mounted plugins from [`confluent-hub-components/`](confluent-hub-components/).
