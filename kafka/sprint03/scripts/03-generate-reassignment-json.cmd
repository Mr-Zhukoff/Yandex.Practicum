@echo off
setlocal

if not exist assets (
  mkdir assets
)

echo [STEP 3] Generating reassignment plan to assets\reassignment.json ...
docker compose exec -T kafka-1 kafka-reassign-partitions --bootstrap-server kafka-1:29092 --topics-to-move-json-file /dev/stdin --broker-list "1,2,3" --generate < assets\topics-to-move.json > assets\reassignment.raw.txt
if errorlevel 1 (
  echo [ERROR] Failed to generate reassignment plan.
  exit /b 1
)

powershell -NoProfile -Command "$raw = Get-Content -Raw 'assets/reassignment.raw.txt'; $parts = $raw -split 'Proposed partition reassignment configuration'; if($parts.Count -lt 2){ Write-Error 'Cannot find proposed reassignment section'; exit 1 }; $tail = $parts[$parts.Count-1]; $m = [regex]::Match($tail, '\{[\s\S]*\}'); if(-not $m.Success){ Write-Error 'Cannot extract proposed JSON'; exit 1 }; $m.Value | Set-Content -NoNewline 'assets/reassignment.json'"
if errorlevel 1 (
  echo [ERROR] Failed to extract reassignment JSON to assets\reassignment.json.
  exit /b 1
)

echo [OK] Reassignment JSON saved to assets\reassignment.json
endlocal

