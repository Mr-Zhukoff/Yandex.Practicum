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
  - HDFS namenode/datanode plus local `data-lake/` mirror for debugging,
  - Spark master/worker and Spark recommendation job,
  - Prometheus, Grafana, Alertmanager and Kafka JMX metrics,
  - topic initialization.
- Go services:
  - `services/shop-api` — reads `data/products.json` and writes product events to Kafka.
  - `services/forbidden-cli` — writes and lists forbidden-product state updates in Kafka.
  - `services/product-filter` — Goka stream processor that consumes raw products and writes allowed/rejected/DLQ topics.
  - `services/postgres-sink` — consumes allowed products and upserts them to PostgreSQL.
  - `services/client-api` — terminal search/recommend commands.
  - `services/hdfs-ingestor` — consumes mirrored analytics topics from the secondary Kafka cluster and writes JSON datasets to HDFS via WebHDFS plus a local JSONL mirror in `data-lake/`.

The Spark analytics job reads allowed product events from HDFS, calculates simple category recommendations and writes results back to the secondary Kafka topic `analytics.recommendations`.

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

Start HDFS, Spark and the analytics ingestion worker explicitly:

```bash
docker compose --profile analytics up -d --build namenode datanode spark-master spark-worker hdfs-ingestor
```

It reads from the secondary Kafka cluster:

- `shop.products.allowed`
- `client.search.requests`
- `client.recommendation.requests`

It writes datasets to HDFS under:

```text
hdfs://namenode:8020/marketplace-analytics/products_allowed/YYYY-MM-DD/*.json
hdfs://namenode:8020/marketplace-analytics/search_requests/YYYY-MM-DD/*.json
hdfs://namenode:8020/marketplace-analytics/recommendation_requests/YYYY-MM-DD/*.json
```

and mirrors them locally for debugging:

```text
data-lake/products_allowed/YYYY-MM-DD.jsonl
data-lake/search_requests/YYYY-MM-DD.jsonl
data-lake/recommendation_requests/YYYY-MM-DD.jsonl
```

Run the Spark recommendation job after product data has been mirrored and ingested:

```bash
docker compose --profile analytics --profile analytics-jobs run --rm spark-recommendations
```

Spark writes recommendation JSON to HDFS under:

```text
hdfs://namenode:8020/marketplace-analytics/spark_recommendations
```

and publishes calculated recommendations to Kafka:

```text
analytics.recommendations
```

HDFS UI is available at http://localhost:9870. Spark UI is available at http://localhost:8080.

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

## Monitoring

Download the JMX Prometheus Java agent if it is missing:

```bash
./scripts/download-jmx-agent.sh
```

Start monitoring:

```bash
docker compose --profile monitoring up -d
```

Endpoints:

- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (`admin` / `admin`)
- Alertmanager: http://localhost:9093

Grafana provisions the `Marketplace Kafka Overview` dashboard automatically. Prometheus scrapes Kafka JMX exporter endpoints and Kafka exporter metrics for both clusters. Alertmanager receives alerts for broker scrape failures and under-replicated partitions.

## Reset local environment

```bash
./scripts/reset.sh
```

## Validation

```bash
go test ./...
docker compose config
```

Check analytics output:

```bash
ls data-lake
docker compose exec namenode hdfs dfs -ls -R /marketplace-analytics
docker compose exec kafka2-1 bash -lc 'unset KAFKA_OPTS; /opt/bitnami/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server kafka2-1:29092,kafka2-2:29092,kafka2-3:29092 \
  --consumer.config /opt/bitnami/kafka/config/certs/admin-ssl.properties \
  --topic analytics.recommendations \
  --from-beginning \
  --timeout-ms 5000'
```
