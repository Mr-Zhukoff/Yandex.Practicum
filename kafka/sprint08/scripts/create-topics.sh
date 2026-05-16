#!/usr/bin/env bash
set -euo pipefail

BOOTSTRAP_SERVER="${1:-localhost:9092}"
REPLICATION_FACTOR="${REPLICATION_FACTOR:-3}"
MIN_INSYNC_REPLICAS="${MIN_INSYNC_REPLICAS:-2}"
PARTITIONS="${PARTITIONS:-3}"
COMMAND_CONFIG="${COMMAND_CONFIG:-}"
COMMAND_CONFIG_ARGS=()
if [[ -n "${COMMAND_CONFIG}" ]]; then
  COMMAND_CONFIG_ARGS=(--command-config "${COMMAND_CONFIG}")
fi

create_topic() {
  local name="$1"
  local cleanup_policy="$2"

  kafka-topics.sh \
    --bootstrap-server "${BOOTSTRAP_SERVER}" \
    "${COMMAND_CONFIG_ARGS[@]}" \
    --create \
    --if-not-exists \
    --topic "${name}" \
    --partitions "${PARTITIONS}" \
    --replication-factor "${REPLICATION_FACTOR}" \
    --config "cleanup.policy=${cleanup_policy}" \
    --config "min.insync.replicas=${MIN_INSYNC_REPLICAS}"
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

kafka-topics.sh --bootstrap-server "${BOOTSTRAP_SERVER}" "${COMMAND_CONFIG_ARGS[@]}" --list
