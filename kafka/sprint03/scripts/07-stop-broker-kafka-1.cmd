@echo off
setlocal

echo [STEP 7.1] Stopping broker kafka-1...
docker compose stop kafka-1
if errorlevel 1 (
  echo [ERROR] Failed to stop kafka-1.
  exit /b 1
)

echo [OK] kafka-1 stopped.
endlocal

