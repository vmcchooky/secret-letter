#!/usr/bin/env pwsh
# Test script for rate limiting functionality

$ErrorActionPreference = "Stop"

$API_URL = "http://localhost:8080"

Write-Host "=== Rate Limiting Test ===" -ForegroundColor Cyan
Write-Host ""

# Function to create a secret
function Create-Secret {
    $body = @{
        ciphertext = "dGVzdC1jaXBoZXJ0ZXh0"
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-GCM"
        ttlSeconds = 3600
    } | ConvertTo-Json

    try {
        $response = Invoke-RestMethod -Uri "$API_URL/api/secrets" `
            -Method Post `
            -ContentType "application/json" `
            -Body $body `
            -ResponseHeadersVariable headers

        return @{
            Success = $true
            SecretID = $response.secretId
            Headers = $headers
        }
    }
    catch {
        $statusCode = $_.Exception.Response.StatusCode.value__
        return @{
            Success = $false
            StatusCode = $statusCode
            Headers = $_.Exception.Response.Headers
        }
    }
}

# Test 1: Verify rate limit headers are present
Write-Host "Test 1: Checking rate limit headers..." -ForegroundColor Yellow
$result = Create-Secret

if ($result.Success) {
    Write-Host "✓ Request succeeded" -ForegroundColor Green
    
    $limit = $result.Headers["X-RateLimit-Limit"]
    $remaining = $result.Headers["X-RateLimit-Remaining"]
    $reset = $result.Headers["X-RateLimit-Reset"]
    
    if ($limit -and $remaining -and $reset) {
        Write-Host "✓ Rate limit headers present:" -ForegroundColor Green
        Write-Host "  Limit: $limit"
        Write-Host "  Remaining: $remaining"
        Write-Host "  Reset: $reset"
    }
    else {
        Write-Host "✗ Rate limit headers missing" -ForegroundColor Red
        exit 1
    }
}
else {
    Write-Host "✗ Request failed with status $($result.StatusCode)" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Test 2: Test rate limit enforcement
Write-Host "Test 2: Testing rate limit enforcement (dynamic limit)..." -ForegroundColor Yellow
Write-Host "Making requests rapidly until the API returns 429..."

$successCount = 0
$rateLimitedCount = 0

if (-not $result.Success) {
    Write-Host "✗ Initial request failed with status $($result.StatusCode)" -ForegroundColor Red
    exit 1
}

$limitHeader = $result.Headers["X-RateLimit-Limit"]
if (-not $limitHeader) {
    Write-Host "✗ Rate limit headers missing; make sure RATE_LIMIT_ENABLED=true for this backend" -ForegroundColor Red
    exit 1
}

$limit = [int]$limitHeader
$successCount = 1
$remaining = $result.Headers["X-RateLimit-Remaining"]
Write-Host "  Request 1 : Success (Remaining: $remaining)" -ForegroundColor Green

for ($i = 2; $i -le ($limit + 1); $i++) {
    $result = Create-Secret
    
    if ($result.Success) {
        $successCount++
        $remaining = $result.Headers["X-RateLimit-Remaining"]
        Write-Host "  Request $i : Success (Remaining: $remaining)" -ForegroundColor Green
    }
    elseif ($result.StatusCode -eq 429) {
        $rateLimitedCount++
        Write-Host "  Request $i : Rate limited (429)" -ForegroundColor Yellow
    }
    else {
        Write-Host "  Request $i : Failed with status $($result.StatusCode)" -ForegroundColor Red
    }
    
    Start-Sleep -Milliseconds 100
}

Write-Host ""
Write-Host "Results:" -ForegroundColor Cyan
Write-Host "  Successful: $successCount"
Write-Host "  Rate limited: $rateLimitedCount"

if ($successCount -eq $limit -and $rateLimitedCount -eq 1) {
    Write-Host "✓ Rate limiting working correctly!" -ForegroundColor Green
}
elseif ($successCount -gt $limit) {
    Write-Host "✗ Rate limiting not enforced (too many requests succeeded)" -ForegroundColor Red
    exit 1
}
else {
    Write-Host "⚠ Unexpected results (may need to wait for rate limit reset)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Rate Limiting Test Complete ===" -ForegroundColor Cyan
