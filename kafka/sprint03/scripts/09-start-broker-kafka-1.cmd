@echo off
setlocal

echo [STEP 7.3] Starting broker kafka-1...
docker compose start kafka-1
if errorlevel 1 (
  echo [ERROR] Failed to start kafka-1.
  exit /b 1
)

echo [OK] kafka-1 started.
endlocal

