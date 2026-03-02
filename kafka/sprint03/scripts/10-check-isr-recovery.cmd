@echo off
setlocal

echo [STEP 7.4] Checking ISR recovery after kafka-1 restart...
echo [INFO] Wait ~15-60 seconds for replicas to catch up, then run this script.

docker compose exec -T kafka-2 kafka-topics --bootstrap-server kafka-2:29092 --describe --topic balanced_topic
if errorlevel 1 (
  echo [ERROR] Failed to describe topic for ISR recovery check.
  exit /b 1
)

echo [INFO] Under-replicated partitions (should be empty when fully recovered):
docker compose exec -T kafka-2 kafka-topics --bootstrap-server kafka-2:29092 --describe --under-replicated-partitions --topic balanced_topic

echo [OK] ISR recovery check completed.
endlocal

