@echo off
setlocal

echo [STEP 2] Current partition distribution for balanced_topic:
docker compose exec -T kafka-1 kafka-topics --bootstrap-server kafka-1:29092 --describe --topic balanced_topic
if errorlevel 1 (
  echo [ERROR] Failed to describe topic before reassignment.
  exit /b 1
)

echo [OK] Describe-before step completed.
endlocal

