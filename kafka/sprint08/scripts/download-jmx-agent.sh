#!/usr/bin/env bash
set -euo pipefail

VERSION="${JMX_EXPORTER_VERSION:-0.20.0}"
TARGET="${JMX_EXPORTER_JAR:-./configs/jmx/jmx_prometheus_javaagent.jar}"
URL="https://repo1.maven.org/maven2/io/prometheus/jmx/jmx_prometheus_javaagent/${VERSION}/jmx_prometheus_javaagent-${VERSION}.jar"

mkdir -p "$(dirname "${TARGET}")"
curl -L -o "${TARGET}" "${URL}"
echo "Downloaded ${URL} to ${TARGET}"
