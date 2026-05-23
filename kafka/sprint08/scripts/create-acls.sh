#!/usr/bin/env bash
set -euo pipefail

BOOTSTRAP_SERVER="${1:-localhost:9092}"
COMMAND_CONFIG="${COMMAND_CONFIG:-./configs/kafka/certs/admin-ssl.properties}"

acl() {
  kafka-acls.sh \
    --bootstrap-server "${BOOTSTRAP_SERVER}" \
    --command-config "${COMMAND_CONFIG}" \
    "$@"
}

add_topic_acl() {
  local principal="$1"
  local operation="$2"
  local topic="$3"
  acl --add --allow-principal "${principal}" --operation "${operation}" --topic "${topic}"
}

add_group_acl() {
  local principal="$1"
  local group="$2"
  acl --add --allow-principal "${principal}" --operation Read --group "${group}"
}

add_cluster_acl() {
  local principal="$1"
  local operation="$2"
  acl --add --allow-principal "${principal}" --operation "${operation}" --cluster
}

SHOP="User:CN=shop-api"
CLIENT="User:CN=client-api"
FILTER="User:CN=product-filter"
POSTGRES="User:CN=postgres-sink"
MIRROR="User:CN=mirror-maker"

add_topic_acl "${SHOP}" Write "shop.products.raw"
add_topic_acl "${SHOP}" Describe "shop.products.raw"

add_topic_acl "${CLIENT}" Write "client.search.requests"
add_topic_acl "${CLIENT}" Write "client.recommendation.requests"
add_topic_acl "${CLIENT}" Read "analytics.recommendations"
add_topic_acl "${CLIENT}" Describe "analytics.recommendations"
add_group_acl "${CLIENT}" "client-api"

add_topic_acl "${FILTER}" Read "shop.products.raw"
add_topic_acl "${FILTER}" Describe "shop.products.raw"
add_topic_acl "${FILTER}" Read "forbidden.products.state"
add_topic_acl "${FILTER}" Describe "forbidden.products.state"
add_topic_acl "${FILTER}" Write "shop.products.allowed"
add_topic_acl "${FILTER}" Write "shop.products.rejected"
add_topic_acl "${FILTER}" Write "shop.products.dlq"
add_topic_acl "${FILTER}" Describe "shop.products.allowed"
add_topic_acl "${FILTER}" Describe "shop.products.rejected"
add_topic_acl "${FILTER}" Describe "shop.products.dlq"
add_group_acl "${FILTER}" "product-filter"

add_topic_acl "${POSTGRES}" Read "shop.products.allowed"
add_topic_acl "${POSTGRES}" Describe "shop.products.allowed"
add_group_acl "${POSTGRES}" "postgres-sink"

# The local demo uses the admin certificate for forbidden-list management.
add_topic_acl "User:CN=admin" Write "forbidden.products.state"
add_topic_acl "User:CN=admin" Read "forbidden.products.state"
add_topic_acl "User:CN=admin" Describe "forbidden.products.state"

# MirrorMaker 2 needs broad enough permissions on both clusters because it
# creates/uses internal heartbeat/checkpoint/offset-sync topics and mirrors a
# selected set of business topics. In production these permissions should be
# narrowed to exact topic and group patterns.
add_topic_acl "${MIRROR}" All "*"
add_cluster_acl "${MIRROR}" Create
add_group_acl "${MIRROR}" "*"

echo "ACLs created"
