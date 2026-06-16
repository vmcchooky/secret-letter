#!/usr/bin/env pwsh

$ErrorActionPreference = "Stop"

$EdgeApiUrl = if ($env:EDGE_API_URL) { $env:EDGE_API_URL } else { "https://api.secret.quorix.io.vn" }
$LogSourceCommand = if ($env:LOG_SOURCE_COMMAND) {
    $env:LOG_SOURCE_COMMAND
} else {
    "docker compose -f deploy/prod/docker-compose.yml -f deploy/prod/docker-compose.vps-edge.yml logs api --since 10m --no-log-prefix"
}
$PollAttempts = if ($env:POLL_ATTEMPTS) { [int]$env:POLL_ATTEMPTS } else { 10 }
$PollDelaySeconds = if ($env:POLL_DELAY_SECONDS) { [int]$env:POLL_DELAY_SECONDS } else { 1 }
$SpoofIpOne = if ($env:SPOOF_IP_ONE) { $env:SPOOF_IP_ONE } else { "198.51.100.77" }
$SpoofIpTwo = if ($env:SPOOF_IP_TWO) { $env:SPOOF_IP_TWO } else { "203.0.113.88" }

$runId = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$normalRequestId = "trusted-proxy-normal-$runId"
$spoofRequestIdOne = "trusted-proxy-spoof-a-$runId"
$spoofRequestIdTwo = "trusted-proxy-spoof-b-$runId"

function Invoke-HealthProbe {
    param(
        [Parameter(Mandatory = $true)]
        [string]$RequestId,
        [string]$SpoofedIp
    )

    $headers = @{
        "X-Request-ID" = $RequestId
        "User-Agent" = "trusted-proxy-check/1.0"
    }

    if ($SpoofedIp) {
        $headers["X-Forwarded-For"] = $SpoofedIp
    }

    $response = Invoke-WebRequest -Uri "$EdgeApiUrl/healthz" -Method Get -Headers $headers
    return [int]$response.StatusCode
}

function Get-LoggedIpHash {
    param(
        [Parameter(Mandatory = $true)]
        [string]$RequestId
    )

    for ($attempt = 1; $attempt -le $PollAttempts; $attempt++) {
        $logOutput = Invoke-Expression $LogSourceCommand 2>$null
        $line = $logOutput | Select-String -Pattern $RequestId | Select-Object -Last 1
        if ($line -and $line.Line -match '"ip_hash":"([^"]+)"') {
            return $Matches[1]
        }
        Start-Sleep -Seconds $PollDelaySeconds
    }

    throw "Could not find request id '$RequestId' in logs."
}

Write-Host "=== Trusted Proxy Verification ===" -ForegroundColor Cyan
Write-Host "EDGE_API_URL: $EdgeApiUrl"
Write-Host "LOG_SOURCE_COMMAND: $LogSourceCommand"
Write-Host ""

Write-Host "[1/4] Sending baseline request through the public edge..." -ForegroundColor Yellow
$normalStatus = Invoke-HealthProbe -RequestId $normalRequestId
if ($normalStatus -ne 200) {
    throw "Expected baseline request to return 200, got $normalStatus"
}

Write-Host "[2/4] Sending spoofed forwarded-header request #1 through the public edge..." -ForegroundColor Yellow
$spoofStatusOne = Invoke-HealthProbe -RequestId $spoofRequestIdOne -SpoofedIp $SpoofIpOne
if ($spoofStatusOne -ne 200) {
    throw "Expected spoofed request #1 to return 200, got $spoofStatusOne"
}

Write-Host "[3/4] Sending spoofed forwarded-header request #2 through the public edge..." -ForegroundColor Yellow
$spoofStatusTwo = Invoke-HealthProbe -RequestId $spoofRequestIdTwo -SpoofedIp $SpoofIpTwo
if ($spoofStatusTwo -ne 200) {
    throw "Expected spoofed request #2 to return 200, got $spoofStatusTwo"
}

Write-Host "[4/4] Comparing trusted client identity hashes from API logs..." -ForegroundColor Yellow
$normalHash = Get-LoggedIpHash -RequestId $normalRequestId
$spoofHashOne = Get-LoggedIpHash -RequestId $spoofRequestIdOne
$spoofHashTwo = Get-LoggedIpHash -RequestId $spoofRequestIdTwo

Write-Host "Baseline ip_hash: $normalHash"
Write-Host "Spoof #1 ip_hash: $spoofHashOne"
Write-Host "Spoof #2 ip_hash: $spoofHashTwo"

if ($normalHash -ne $spoofHashOne -or $normalHash -ne $spoofHashTwo) {
    throw "Trusted proxy verification failed. Check edge header sanitization and TRUSTED_PROXY_CIDRS."
}

Write-Host ""
Write-Host "Trusted proxy verification passed." -ForegroundColor Green
Write-Host "Spoofed X-Forwarded-For headers did not change the client identity seen by the API." -ForegroundColor Green
