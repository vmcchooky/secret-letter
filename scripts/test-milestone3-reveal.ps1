#!/usr/bin/env pwsh
# Milestone 3 Reveal Flow Test Script
# Tests the complete create -> status -> consume flow

$ErrorActionPreference = "Stop"
$API_BASE = "http://localhost:8080"

Write-Host "=== Milestone 3 Reveal Flow Test ===" -ForegroundColor Cyan
Write-Host ""

# Test 1: Health Check
Write-Host "[1/6] Testing health endpoint..." -ForegroundColor Yellow
try {
    $health = Invoke-RestMethod -Uri "$API_BASE/healthz" -Method Get
    Write-Host "✓ Health check passed: $($health.status)" -ForegroundColor Green
} catch {
    Write-Host "✗ Health check failed: $_" -ForegroundColor Red
    exit 1
}

# Test 2: Create Secret
Write-Host "[2/6] Creating secret..." -ForegroundColor Yellow
$createBody = @{
    ciphertext = "dGVzdC1yZXZlYWwtZmxvdy1taWxlc3RvbmUz"
    nonce = "MTIzNDU2Nzg5MDEy"
    algorithm = "AES-GCM"
    ttlSeconds = 3600
} | ConvertTo-Json

try {
    $createResp = Invoke-RestMethod -Uri "$API_BASE/api/secrets" `
        -Method Post `
        -ContentType "application/json" `
        -Body $createBody
    
    $secretId = $createResp.secretId
    Write-Host "✓ Secret created: $secretId" -ForegroundColor Green
    Write-Host "  Expires at: $($createResp.expiresAt)" -ForegroundColor Gray
} catch {
    Write-Host "✗ Failed to create secret: $_" -ForegroundColor Red
    exit 1
}

# Test 3: Check Status (should be pending)
Write-Host "[3/6] Checking secret status..." -ForegroundColor Yellow
try {
    $status = Invoke-RestMethod -Uri "$API_BASE/api/secrets/$secretId/status" -Method Get
    
    if ($status.status -eq "pending") {
        Write-Host "✓ Status check passed: $($status.status)" -ForegroundColor Green
        Write-Host "  Created at: $($status.createdAt)" -ForegroundColor Gray
        Write-Host "  Expires at: $($status.expiresAt)" -ForegroundColor Gray
    } else {
        Write-Host "✗ Expected status 'pending', got '$($status.status)'" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "✗ Status check failed: $_" -ForegroundColor Red
    exit 1
}

# Test 4: Consume Secret (first time should succeed)
Write-Host "[4/6] Consuming secret (first attempt)..." -ForegroundColor Yellow
try {
    $consumeResp = Invoke-RestMethod -Uri "$API_BASE/api/secrets/$secretId/consume" `
        -Method Post `
        -ContentType "application/json" `
        -Body "{}"
    
    Write-Host "✓ Secret consumed successfully" -ForegroundColor Green
    Write-Host "  Ciphertext: $($consumeResp.ciphertext)" -ForegroundColor Gray
    Write-Host "  Nonce: $($consumeResp.nonce)" -ForegroundColor Gray
    Write-Host "  Algorithm: $($consumeResp.algorithm)" -ForegroundColor Gray
    Write-Host "  Consumed at: $($consumeResp.consumedAt)" -ForegroundColor Gray
} catch {
    Write-Host "✗ Failed to consume secret: $_" -ForegroundColor Red
    exit 1
}

# Test 5: Try to Consume Again (should fail with 410)
Write-Host "[5/6] Attempting to consume again (should fail)..." -ForegroundColor Yellow
try {
    $consumeResp2 = Invoke-RestMethod -Uri "$API_BASE/api/secrets/$secretId/consume" `
        -Method Post `
        -ContentType "application/json" `
        -Body "{}"
    
    Write-Host "✗ Second consume should have failed but succeeded" -ForegroundColor Red
    exit 1
} catch {
    $statusCode = $_.Exception.Response.StatusCode.value__
    if ($statusCode -eq 410) {
        Write-Host "✓ Second consume correctly rejected (410 Gone)" -ForegroundColor Green
    } else {
        Write-Host "✗ Expected status 410, got $statusCode" -ForegroundColor Red
        exit 1
    }
}

# Test 6: Check Status Again (should be not_found)
Write-Host "[6/6] Checking status after consumption..." -ForegroundColor Yellow
try {
    $statusAfter = Invoke-RestMethod -Uri "$API_BASE/api/secrets/$secretId/status" -Method Get
    
    if ($statusAfter.status -eq "not_found") {
        Write-Host "✓ Status correctly shows 'not_found'" -ForegroundColor Green
    } else {
        Write-Host "✗ Expected status 'not_found', got '$($statusAfter.status)'" -ForegroundColor Red
        exit 1
    }
} catch {
    # 404 is also acceptable
    $statusCode = $_.Exception.Response.StatusCode.value__
    if ($statusCode -eq 404) {
        Write-Host "✓ Status correctly returns 404" -ForegroundColor Green
    } else {
        Write-Host "✗ Unexpected error: $_" -ForegroundColor Red
        exit 1
    }
}

Write-Host ""
Write-Host "=== All Tests Passed! ===" -ForegroundColor Green
Write-Host ""
Write-Host "Milestone 3 reveal flow is working correctly:" -ForegroundColor Cyan
Write-Host "  ✓ Create secret" -ForegroundColor Green
Write-Host "  ✓ Check status (pending)" -ForegroundColor Green
Write-Host "  ✓ Consume secret (success)" -ForegroundColor Green
Write-Host "  ✓ Prevent double consumption (410)" -ForegroundColor Green
Write-Host "  ✓ Status after consumption (not_found)" -ForegroundColor Green
