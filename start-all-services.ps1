#!/usr/bin/env pwsh

# Script to start all services in parallel
# Services: api-gateway, auth-service, notification-service

$services = @(
    "api-gateway",
    "auth-service", 
    "notification-service"
)

$colors = @(
    "Green",
    "Yellow", 
    "Cyan"
)

Write-Host "Starting all services..." -ForegroundColor White

# Start each service in a new window
for ($i = 0; $i -lt $services.Length; $i++) {
    $service = $services[$i]
    $color = $colors[$i]
    
    Write-Host "Starting $service..." -ForegroundColor $color
    
    # Start the service in a new PowerShell window
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD\$service'; npm run dev" -WindowStyle Normal
}

Write-Host "All services started in separate windows!" -ForegroundColor White
Write-Host "Close this window or press Ctrl+C to stop monitoring." -ForegroundColor Gray

# Keep the script running
try {
    while ($true) {
        Start-Sleep -Seconds 1
    }
}
catch {
    Write-Host "Stopping monitor..." -ForegroundColor Red
}
