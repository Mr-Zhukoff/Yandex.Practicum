@echo off
setlocal

echo [RUN] 01-create-topic
call scripts\01-create-topic.cmd || exit /b 1

echo [RUN] 02-describe-topic-before
call scripts\02-describe-topic-before.cmd || exit /b 1

echo [RUN] 03-generate-reassignment-json
call scripts\03-generate-reassignment-json.cmd || exit /b 1

echo [RUN] 04-execute-reassignment
call scripts\04-execute-reassignment.cmd || exit /b 1

echo [RUN] 05-verify-reassignment
call scripts\05-verify-reassignment.cmd || exit /b 1

echo [RUN] 06-describe-topic-after
call scripts\06-describe-topic-after.cmd || exit /b 1

echo [INFO] Main reassignment flow completed.
echo [INFO] For failure simulation run scripts 07..10 manually in order.

endlocal

