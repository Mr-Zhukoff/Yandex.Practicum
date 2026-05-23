#!/usr/bin/env bash
set -euo pipefail

# ── colours ──────────────────────────────────────────────────
BLD="\033[1m"; GRN="\033[32m"; YLW="\033[33m"; RED="\033[31m"
CYN="\033[36m";  RST="\033[0m"

banner() { printf "\n${BLD}${CYN}[%s]${RST} ${BLD}%s${RST}\n" "$1" "$2"; }
ok()    { printf "  ${GRN}✓${RST} %s\n" "$1"; }
warn()  { printf "  ${YLW}!${RST} %s\n" "$1"; }
err()   { printf "  ${RED}✗${RST} %s\n" "$1"; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(dirname "$SCRIPT_DIR")"
cd "$ROOT"

# ══════════════════════════════════════════════════════════════
# 1. Pre-flight
# ══════════════════════════════════════════════════════════════
banner "1/6" "Pre-flight checks"

for tool in docker; do
  if command -v "$tool" &>/dev/null; then
    ok "$tool found"
  else
    err "$tool not found — please install Docker"
    exit 1
  fi
done

if ! docker compose version &>/dev/null; then
  err "docker compose not available"
  exit 1
fi
ok "docker compose available"

# ══════════════════════════════════════════════════════════════
# 2. Artifacts — TLS certs + JMX agent
# ══════════════════════════════════════════════════════════════
banner "2/6" "Artifacts"

if [[ ! -f configs/kafka/certs/ca.crt ]]; then
  ok "generating TLS certificates..."
  ./scripts/generate-certs.sh
else
  ok "TLS certificates already present"
fi

if [[ ! -f configs/jmx/jmx_prometheus_javaagent.jar ]]; then
  ok "downloading JMX Prometheus agent..."
  ./scripts/download-jmx-agent.sh
else
  ok "JMX agent already present"
fi

# ══════════════════════════════════════════════════════════════
# 3. Infrastructure + long-running services
# ══════════════════════════════════════════════════════════════
banner "3/6" "Starting infrastructure and services"

docker compose \
  --profile app \
  --profile analytics \
  up -d --build \
  --remove-orphans

ok "containers launched"

# ── healthcheck helpers ──────────────────────────────────────
health_ok() {
  local container="$1"
  local state
  state=$(docker inspect -f '{{.State.Health.Status}}' "$container" 2>/dev/null || echo "missing")
  [[ "$state" == "healthy" ]]
}

wait_for() {
  local container="$1" timeout="${2:-90s}" label="${3:-$1}"
  local start end elapsed
  start=$(date +%s)
  end=$(( start + ${timeout%s} ))

  printf "  waiting for %-35s" "${label}..."
  while (( $(date +%s) < end )); do
    if health_ok "$container"; then
      elapsed=$(($(date +%s) - start))
      printf " ${GRN}healthy${RST} (${elapsed}s)\n"
      return 0
    fi
    sleep 2
  done
  printf " ${RED}timeout${RST}\n"
  return 1
}

wait_for "marketplace-kafka-1"    "120s"
wait_for "marketplace-postgres"   "60s"
wait_for "marketplace-hdfs-namenode" "60s" "HDFS namenode"
ok "core infrastructure healthy"

# For app services (no healthcheck defined in compose), give them a moment
sleep 5
ok "app services started"

# ══════════════════════════════════════════════════════════════
# 4. One-shot jobs — seed data
# ══════════════════════════════════════════════════════════════
banner "4/6" "Seeding data"

run_job() {
  local jname="$1"; shift
  printf "  running %-30s ..." "${jname}"
  if docker compose --profile jobs run --rm "$jname" "$@" >/dev/null 2>&1; then
    printf " ${GRN}done${RST}\n"
  else
    printf " ${RED}failed${RST}\n"
    return 1
  fi
}

run_job forbidden-cli
run_job shop-api
run_job client-api

# ══════════════════════════════════════════════════════════════
# 5. Wait for MirrorMaker replication then run Spark
# ══════════════════════════════════════════════════════════════
banner "5/6" "MirrorMaker replication + Spark analytics"

printf "  waiting for MirrorMaker to replicate topics..."
sleep 20
printf " ${GRN}done${RST}\n"

printf "  running spark-recommendations..."
if docker compose \
     --profile analytics \
     --profile analytics-jobs \
     run --rm spark-recommendations >/dev/null 2>&1; then
  printf " ${GRN}done${RST}\n"
else
  warn "spark-recommendations exited with an error — check container logs"
fi

# ══════════════════════════════════════════════════════════════
# 6. Monitoring + verification
# ══════════════════════════════════════════════════════════════
banner "6/6" "Monitoring"

docker compose --profile monitoring up -d
ok "Prometheus, Grafana, Alertmanager started"

sleep 5

# ── Verification output ─────────────────────────────────────
echo ""
echo  "═══════════════════════════════════════════════════════════"
echo  "  ${BLD}Demo pipeline complete. Verify results:${RST}"
echo  "═══════════════════════════════════════════════════════════"
echo  ""
echo  "  ${BLD}─ Endpoints ────────────────────────────────────${RST}"
echo  "  Grafana:       http://localhost:3000  (admin / admin)"
echo  "  Prometheus:    http://localhost:9090"
echo  "  Alertmanager:  http://localhost:9093"
echo  "  HDFS UI:       http://localhost:9870"
echo  "  Spark UI:      http://localhost:8080"
echo  ""
echo  "  ${BLD}─ Local data lake ────────────────────────────${RST}"
echo  "  ls -R data-lake/"
echo  ""
echo  "  ${BLD}─ HDFS datasets ──────────────────────────────${RST}"
echo  "  docker compose exec namenode hdfs dfs -ls -R /marketplace-analytics"
echo  ""
echo  "  ${BLD}─ Kafka recommendations topic ────────────────${RST}"
echo  "  docker compose exec kafka2-1 bash -lc 'unset KAFKA_OPTS; /opt/bitnami/kafka/bin/kafka-console-consumer.sh \\"
echo  "    --bootstrap-server kafka2-1:29092,kafka2-2:29092,kafka2-3:29092 \\"
echo  "    --consumer.config /opt/bitnami/kafka/config/certs/admin-ssl.properties \\"
echo  "    --topic analytics.recommendations --from-beginning --timeout-ms 5000'"
echo  ""
echo  "  ${BLD}─ PostgreSQL ─────────────────────────────────${RST}"
echo  "  docker compose exec postgres psql -U marketplace -d marketplace \\"
echo  "    -c \"SELECT product_id, name, category, price_amount FROM products ORDER BY name;\""
echo  ""
echo  "  ${BLD}─ Stop everything ────────────────────────────${RST}"
echo  "  ./scripts/reset.sh"
echo  ""
