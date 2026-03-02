@echo off
setlocal

echo [STEP 7.2] Checking topic state while kafka-1 is down...
echo [INFO] Full metadata for balanced_topic:
docker compose exec -T kafka-2 kafka-topics --bootstrap-server kafka-2:29092 --describe --topic balanced_topic
if errorlevel 1 (
  echo [ERROR] Failed to describe topic during broker failure.
  exit /b 1
)

echo [INFO] Under-replicated partitions (expected during failure):
docker compose exec -T kafka-2 kafka-topics --bootstrap-server kafka-2:29092 --describe --under-replicated-partitions --topic balanced_topic

echo [OK] Failure-state check completed.
endlocal

