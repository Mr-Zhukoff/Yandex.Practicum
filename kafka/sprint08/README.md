# Marketplace Analytics Platform

Implementation of the final Kafka project described in `docs/Task.md`.

The target architecture is documented in `docs/ImplementationPlan.md`.

## Current implementation slice

This first slice includes:

- Go module and shared event contracts.
- Sample product and forbidden-product data.
- Local Docker Compose baseline with:
  - Kafka single-broker KRaft mode,
  - PostgreSQL,
  - topic initialization.
- Go services:
  - `services/shop-api` — reads `data/products.json` and writes product events to Kafka.
  - `services/forbidden-cli` — writes forbidden-product state updates to Kafka.
  - `services/product-filter` — baseline stream filter that consumes raw products and writes allowed/rejected/DLQ topics.
  - `services/postgres-sink` — consumes allowed products and upserts them to PostgreSQL.
  - `services/client-api` — terminal search/recommend commands.
  - `services/hdfs-ingestor` — placeholder for the HDFS implementation stage.

> Note: the current `product-filter` is a baseline Kafka consumer/producer implementation. It will be migrated to Goka in the dedicated stream-processing stage.

## Start infrastructure

```bash
docker compose up -d kafka postgres kafka-init
```

## Run local pipeline

In separate terminals:

```bash
go run ./services/forbidden-cli add --product-id forbidden-001 --reason "Seed product used to verify filtering"
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

## Reset local environment

```bash
./scripts/reset.sh
```

## Validation

```bash
go test ./...
docker compose config
```
