# Marketplace Analytics Platform

Implementation of the final Kafka project described in `docs/Task.md`.

The target architecture is documented in `docs/ImplementationPlan.md`.

## Current implementation slice

The current slice includes:

- Go module and shared event contracts.
- Sample product and forbidden-product data.
- Local Docker Compose baseline with:
  - Kafka single-broker KRaft mode,
  - PostgreSQL,
  - topic initialization.
- Go services:
  - `services/shop-api` — reads `data/products.json` and writes product events to Kafka.
  - `services/forbidden-cli` — writes and lists forbidden-product state updates in Kafka.
  - `services/product-filter` — Goka stream processor that consumes raw products and writes allowed/rejected/DLQ topics.
  - `services/postgres-sink` — consumes allowed products and upserts them to PostgreSQL.
  - `services/client-api` — terminal search/recommend commands.
  - `services/hdfs-ingestor` — placeholder for the HDFS implementation stage.

## Start infrastructure

```bash
docker compose up -d kafka postgres kafka-init
```

Kafka is exposed in two ways:

- from the host: `localhost:9092`
- from Docker containers: `kafka:29092`

## Run local pipeline

In separate terminals:

```bash
go run ./services/forbidden-cli add --product-id forbidden-001 --reason "Seed product used to verify filtering"
```

List active forbidden products:

```bash
go run ./services/forbidden-cli list
```

```bash
go run ./services/product-filter
```

```bash
go run ./services/postgres-sink
```

Then send products:

```bash
go run ./services/shop-api --file ./data/products.json
```

Search products:

```bash
go run ./services/client-api search --user-id user_001 --query "умные часы"
```

Request recommendations:

```bash
go run ./services/client-api recommend --user-id user_001 --category "Электроника"
```

## Run with Docker Compose profiles

Start long-running application services:

```bash
docker compose --profile app up -d --build
```

Seed the forbidden list through a one-shot Compose job:

```bash
docker compose --profile jobs run --rm forbidden-cli
```

Send sample products through a one-shot Compose job:

```bash
docker compose --profile jobs run --rm shop-api
```

Run a sample client search through a one-shot Compose job:

```bash
docker compose --profile jobs run --rm client-api
```

## Reset local environment

```bash
./scripts/reset.sh
```

## Validation

```bash
go test ./...
docker compose config
```
