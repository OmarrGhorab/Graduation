@echo off
REM Script to start all services in parallel
REM Services: api-gateway, auth-service, notification-service

echo Starting all services...

REM Start each service in a new command prompt window
start "API Gateway" cmd /k "cd api-gateway && npm run dev"
start "Auth Service" cmd /k "cd auth-service && npm run dev"
start "Notification Service" cmd /k "cd notification-service && npm run dev"

echo All services started in separate windows!
echo Close this window or press any key to exit.
pause
