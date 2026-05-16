# Marketplace Analytics Platform

Implementation of the final Kafka project described in `docs/Task.md`.

The target architecture is documented in `docs/ImplementationPlan.md`.

## Current implementation slice

The current slice includes:

- Go module and shared event contracts.
- Sample product and forbidden-product data.
- Local Docker Compose baseline with:
  - three-broker Kafka KRaft cluster,
  - secondary three-broker Kafka KRaft cluster for analytics,
  - MirrorMaker 2 topic replication from primary to secondary,
  - Kafka TLS/mTLS and ACL configuration,
  - replicated topics with `min.insync.replicas=2`,
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

Generate local development certificates first:

```bash
./scripts/generate-certs.sh
```

```bash
docker compose up -d kafka1 kafka2 kafka3 kafka2-1 kafka2-2 kafka2-3 postgres kafka-init kafka2-init kafka-acls kafka2-acls mirror-maker
```

Kafka is exposed in two ways:

- from the host: `localhost:9092`
- from Docker containers: `kafka1:29092,kafka2:29092,kafka3:29092`

Host broker ports:

Primary cluster:

- broker 1: `localhost:9092`
- broker 2: `localhost:9094`
- broker 3: `localhost:9096`

Secondary analytics cluster:

- broker 1: `localhost:9192`
- broker 2: `localhost:9194`
- broker 3: `localhost:9196`

MirrorMaker 2 replicates these primary topics to the secondary cluster:

- `shop.products.allowed`
- `client.search.requests`
- `client.recommendation.requests`
- `analytics.recommendations`

## Run local pipeline

In separate terminals:

```bash
go run ./services/forbidden-cli add \
  --brokers localhost:9092,localhost:9094,localhost:9096 \
  --tls \
  --tls-ca-cert ./configs/kafka/certs/ca.crt \
  --tls-client-cert ./configs/kafka/certs/admin.crt \
  --tls-client-key ./configs/kafka/certs/admin.key \
  --tls-server-name localhost \
  --product-id forbidden-001 \
  --reason "Seed product used to verify filtering"
```

List active forbidden products:

```bash
go run ./services/forbidden-cli list \
  --brokers localhost:9092,localhost:9094,localhost:9096 \
  --tls \
  --tls-ca-cert ./configs/kafka/certs/ca.crt \
  --tls-client-cert ./configs/kafka/certs/admin.crt \
  --tls-client-key ./configs/kafka/certs/admin.key \
  --tls-server-name localhost
```

```bash
go run ./services/product-filter \
  --brokers localhost:9092,localhost:9094,localhost:9096 \
  --tls \
  --tls-ca-cert ./configs/kafka/certs/ca.crt \
  --tls-client-cert ./configs/kafka/certs/product-filter.crt \
  --tls-client-key ./configs/kafka/certs/product-filter.key \
  --tls-server-name localhost
```

```bash
go run ./services/postgres-sink \
  --brokers localhost:9092,localhost:9094,localhost:9096 \
  --tls \
  --tls-ca-cert ./configs/kafka/certs/ca.crt \
  --tls-client-cert ./configs/kafka/certs/postgres-sink.crt \
  --tls-client-key ./configs/kafka/certs/postgres-sink.key \
  --tls-server-name localhost
```

Then send products:

```bash
go run ./services/shop-api \
  --brokers localhost:9092,localhost:9094,localhost:9096 \
  --tls \
  --tls-ca-cert ./configs/kafka/certs/ca.crt \
  --tls-client-cert ./configs/kafka/certs/shop-api.crt \
  --tls-client-key ./configs/kafka/certs/shop-api.key \
  --tls-server-name localhost \
  --file ./data/products.json
```

Search products:

```bash
go run ./services/client-api search \
  --brokers localhost:9092,localhost:9094,localhost:9096 \
  --tls \
  --tls-ca-cert ./configs/kafka/certs/ca.crt \
  --tls-client-cert ./configs/kafka/certs/client-api.crt \
  --tls-client-key ./configs/kafka/certs/client-api.key \
  --tls-server-name localhost \
  --user-id user_001 \
  --query "умные часы"
```

Request recommendations:

```bash
go run ./services/client-api recommend \
  --brokers localhost:9092,localhost:9094,localhost:9096 \
  --tls \
  --tls-ca-cert ./configs/kafka/certs/ca.crt \
  --tls-client-cert ./configs/kafka/certs/client-api.crt \
  --tls-client-key ./configs/kafka/certs/client-api.key \
  --tls-server-name localhost \
  --user-id user_001 \
  --category "Электроника"
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
