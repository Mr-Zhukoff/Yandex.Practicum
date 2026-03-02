@echo off
setlocal

if not exist assets\reassignment.json (
  echo [ERROR] File assets\reassignment.json not found. Run scripts\03-generate-reassignment-json.cmd first.
  exit /b 1
)

echo [STEP 4] Executing partition reassignment...
docker compose exec -T kafka-1 kafka-reassign-partitions --bootstrap-server kafka-1:29092 --reassignment-json-file /dev/stdin --execute < assets\reassignment.json
if errorlevel 1 (
  echo [ERROR] Reassignment execution failed.
  exit /b 1
)

echo [OK] Reassignment execution command sent.
endlocal

