@echo off
setlocal

if not exist assets\reassignment.json (
  echo [ERROR] File assets\reassignment.json not found. Run scripts\03-generate-reassignment-json.cmd first.
  exit /b 1
)

echo [STEP 5] Verifying partition reassignment status...
docker compose exec -T kafka-1 kafka-reassign-partitions --bootstrap-server kafka-1:29092 --reassignment-json-file /dev/stdin --verify < assets\reassignment.json
if errorlevel 1 (
  echo [ERROR] Reassignment verify command failed.
  exit /b 1
)

echo [OK] Verify command completed.
endlocal

