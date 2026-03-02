@echo off
setlocal

echo [STEP 6] Partition distribution after reassignment:
docker compose exec -T kafka-1 kafka-topics --bootstrap-server kafka-1:29092 --describe --topic balanced_topic
if errorlevel 1 (
  echo [ERROR] Failed to describe topic after reassignment.
  exit /b 1
)

echo [OK] Describe-after step completed.
endlocal

