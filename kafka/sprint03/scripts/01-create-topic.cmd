@echo off
setlocal

echo [STEP 1] Creating topic balanced_topic with 8 partitions and RF=3...
docker compose exec -T kafka-1 kafka-topics --bootstrap-server kafka-1:29092 --create --if-not-exists --topic balanced_topic --partitions 8 --replication-factor 3
if errorlevel 1 (
  echo [ERROR] Failed to create topic.
  exit /b 1
)

echo [OK] Topic creation step completed.
endlocal

