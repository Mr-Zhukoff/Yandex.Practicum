#!/usr/bin/env bash
set -euo pipefail

BOOTSTRAP_SERVER="${1:-localhost:9092}"

create_topic() {
  local name="$1"
  local cleanup_policy="$2"

  kafka-topics.sh \
    --bootstrap-server "${BOOTSTRAP_SERVER}" \
    --create \
    --if-not-exists \
    --topic "${name}" \
    --partitions 3 \
    --replication-factor 1 \
    --config "cleanup.policy=${cleanup_policy}"
}

create_topic "shop.products.raw" "delete"
create_topic "shop.products.allowed" "delete"
create_topic "shop.products.rejected" "delete"
create_topic "shop.products.dlq" "delete"
create_topic "client.search.requests" "delete"
create_topic "client.recommendation.requests" "delete"
create_topic "analytics.recommendations" "compact"
create_topic "forbidden.products.commands" "delete"
create_topic "forbidden.products.state" "compact"

kafka-topics.sh --bootstrap-server "${BOOTSTRAP_SERVER}" --list
