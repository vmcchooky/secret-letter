#!/usr/bin/env pwsh

$ErrorActionPreference = "Stop"

$ApiBaseUrl = if ($env:API_BASE_URL) { $env:API_BASE_URL } else { "http://127.0.0.1:18080" }
$EdgeApiUrl = if ($env:EDGE_API_URL) { $env:EDGE_API_URL } else { "https://api.secret.quorix.io.vn" }
$ReadyTimeoutSeconds = if ($env:READY_TIMEOUT_SECONDS) { [int]$env:READY_TIMEOUT_SECONDS } else { 60 }
$RestartCommand = if ($env:RESTART_COMMAND) {
    $env:RESTART_COMMAND
} else {
    "docker compose -f deploy/prod/docker-compose.yml -f deploy/prod/docker-compose.vps-edge.yml restart api"
}
$TestCiphertext = if ($env:TEST_CIPHERTEXT) { $env:TEST_CIPHERTEXT } else { "cHJvZHVjdGlvbi1zbW9rZS10ZXN0LWNpcGhlcnRleHQ" }
$TestNonce = if ($env:TEST_NONCE) { $env:TEST_NONCE } else { "MTIzNDU2Nzg5MDEy" }

function Get-StatusCode {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Url
    )

    try {
        return [int](Invoke-WebRequest -Uri $Url -Method Get).StatusCode
    } catch {
        if ($_.Exception.Response) {
            return [int]$_.Exception.Response.StatusCode.value__
        }
        throw
    }
}

function Wait-ForReady {
    $deadline = (Get-Date).AddSeconds($ReadyTimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        if ((Get-StatusCode -Url "$ApiBaseUrl/readyz") -eq 200) {
            return
        }
        Start-Sleep -Seconds 2
    }

    throw "readyz did not return 200 within ${ReadyTimeoutSeconds}s after restart."
}

Write-Host "=== Production Smoke Test ===" -ForegroundColor Cyan
Write-Host "API_BASE_URL: $ApiBaseUrl"
Write-Host "EDGE_API_URL: $EdgeApiUrl"
Write-Host ""

Write-Host "[1/7] Checking local health and readiness endpoints..." -ForegroundColor Yellow
$healthStatus = Get-StatusCode -Url "$ApiBaseUrl/healthz"
$readyStatus = Get-StatusCode -Url "$ApiBaseUrl/readyz"
if ($healthStatus -ne 200 -or $readyStatus -ne 200) {
    throw "Expected healthz=200 and readyz=200 before the test, got healthz=$healthStatus readyz=$readyStatus"
}

Write-Host "[2/7] Creating a secret before restart..." -ForegroundColor Yellow
$createBody = @{
    ciphertext = $TestCiphertext
    nonce = $TestNonce
    algorithm = "AES-GCM"
    ttlSeconds = 3600
} | ConvertTo-Json

$createResponse = Invoke-RestMethod -Uri "$ApiBaseUrl/api/secrets" `
    -Method Post `
    -ContentType "application/json" `
    -Headers @{ "X-Request-ID" = "production-smoke-create-$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())" } `
    -Body $createBody

$secretId = $createResponse.secretId
if (-not $secretId) {
    throw "Failed to create a secret."
}
Write-Host "Created secret: $secretId"

Write-Host "[3/7] Verifying status before restart..." -ForegroundColor Yellow
$statusResponse = Invoke-RestMethod -Uri "$ApiBaseUrl/api/secrets/$secretId/status" -Method Get
if ($statusResponse.status -ne "pending") {
    throw "Expected pending status before restart, got '$($statusResponse.status)'"
}

Write-Host "[4/7] Restarting the API..." -ForegroundColor Yellow
if ($RestartCommand -eq "manual") {
    Read-Host "Restart the API service/container now, then press Enter to continue"
} else {
    Invoke-Expression $RestartCommand
}

Write-Host "[5/7] Waiting for readyz to recover..." -ForegroundColor Yellow
Wait-ForReady

Write-Host "[6/7] Revealing the secret created before restart..." -ForegroundColor Yellow
$consumeResponse = Invoke-RestMethod -Uri "$ApiBaseUrl/api/secrets/$secretId/consume" `
    -Method Post `
    -ContentType "application/json" `
    -Body "{}"

if ($consumeResponse.ciphertext -ne $TestCiphertext) {
    throw "Expected ciphertext '$TestCiphertext' after restart, got '$($consumeResponse.ciphertext)'"
}

Write-Host "[7/7] Sending an oversized request through the public edge..." -ForegroundColor Yellow
$oversizedBody = @{
    ciphertext = ("A" * 40KB)
    nonce = $TestNonce
    algorithm = "AES-GCM"
    ttlSeconds = 3600
} | ConvertTo-Json -Compress

try {
    $oversizedResponse = Invoke-WebRequest -Uri "$EdgeApiUrl/api/secrets" `
        -Method Post `
        -ContentType "application/json" `
        -Body $oversizedBody

    throw "Expected oversized edge request to return 413, got $($oversizedResponse.StatusCode)"
} catch {
    if (-not $_.Exception.Response) {
        throw
    }

    $statusCode = [int]$_.Exception.Response.StatusCode.value__
    if ($statusCode -ne 413) {
        throw "Expected oversized edge request to return 413, got $statusCode"
    }
}

Write-Host ""
Write-Host "Production smoke test passed." -ForegroundColor Green
Write-Host "The API survived a restart, preserved SECRET_ENCRYPTION_KEY continuity, and rejected oversized edge traffic." -ForegroundColor Green
