#!/usr/bin/env bash
set -euo pipefail

docker compose up -d kafka postgres kafka-init

echo "Waiting for infrastructure..."
sleep 10

echo "Run the first pipeline manually:"
echo "  go run ./services/forbidden-cli add --product-id forbidden-001 --reason 'Seed product used to verify filtering'"
echo "  go run ./services/product-filter"
echo "  go run ./services/postgres-sink"
echo "  go run ./services/shop-api --file ./data/products.json"
echo "  go run ./services/client-api search --user-id user_001 --query 'умные часы'"
