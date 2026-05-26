# Stop local development services for one-time-link.
# Stops frontend/backend listeners on the default dev ports, then stops local Redis.

[CmdletBinding()]
param(
    [int[]]$Ports = @(5173, 8080),
    [switch]$SkipRedis
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$composeFile = Join-Path $repoRoot "deploy/local/docker-compose.yml"

Write-Host "Stopping one-time-link local development services..." -ForegroundColor Cyan

foreach ($port in $Ports) {
    $connections = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue

    if (-not $connections) {
        Write-Host "Port ${port}: no listener found." -ForegroundColor DarkGray
        continue
    }

    $processIds = $connections | Select-Object -ExpandProperty OwningProcess -Unique

    foreach ($processId in $processIds) {
        try {
            $process = Get-Process -Id $processId -ErrorAction Stop
            Stop-Process -Id $processId -Force
            Write-Host "Port ${port}: stopped PID $processId ($($process.ProcessName))." -ForegroundColor Green
        } catch {
            Write-Host "Port ${port}: could not stop PID $processId. $($_.Exception.Message)" -ForegroundColor Yellow
        }
    }
}

if ($SkipRedis) {
    Write-Host "Redis: skipped." -ForegroundColor DarkGray
} elseif (Test-Path $composeFile) {
    Write-Host "Redis: stopping Docker Compose service..." -ForegroundColor Cyan
    docker compose -f $composeFile down
} else {
    Write-Host "Redis: compose file not found at $composeFile." -ForegroundColor Yellow
}

Write-Host "Done." -ForegroundColor Green
