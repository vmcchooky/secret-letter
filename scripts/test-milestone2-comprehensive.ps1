# Comprehensive Milestone 2 Testing Script
# Tests all edge cases and validation scenarios

$API_BASE_URL = if ($env:API_BASE_URL) { $env:API_BASE_URL } else { "http://localhost:8080" }

Write-Host "=== MILESTONE 2 COMPREHENSIVE TESTING ===" -ForegroundColor Cyan
Write-Host "API Base URL: $API_BASE_URL"
Write-Host ""

$testResults = @()

function Test-Endpoint {
    param(
        [string]$TestName,
        [hashtable]$Body,
        [int]$ExpectedStatus,
        [string]$Description
    )
    
    Write-Host "Test: $TestName" -ForegroundColor Yellow
    Write-Host "Description: $Description"
    
    $bodyJson = $Body | ConvertTo-Json
    
    try {
        $response = Invoke-WebRequest -Uri "$API_BASE_URL/api/secrets" `
            -Method POST `
            -Headers @{
                "Content-Type"="application/json"
                "X-Request-ID"="test-$(Get-Date -Format 'yyyyMMddHHmmss')"
            } `
            -Body $bodyJson `
            -ErrorAction Stop
        
        $actualStatus = $response.StatusCode
        $success = ($actualStatus -eq $ExpectedStatus)
        
        if ($success) {
            Write-Host "✓ PASS - Status: $actualStatus (Expected: $ExpectedStatus)" -ForegroundColor Green
            if ($actualStatus -eq 201) {
                $result = $response.Content | ConvertFrom-Json
                Write-Host "  Secret ID: $($result.secretId)" -ForegroundColor Gray
                Write-Host "  Expires At: $($result.expiresAt)" -ForegroundColor Gray
            }
        } else {
            Write-Host "✗ FAIL - Status: $actualStatus (Expected: $ExpectedStatus)" -ForegroundColor Red
        }
        
        $script:testResults += @{
            Test = $TestName
            Expected = $ExpectedStatus
            Actual = $actualStatus
            Success = $success
        }
        
    } catch {
        $actualStatus = $_.Exception.Response.StatusCode.value__
        $success = ($actualStatus -eq $ExpectedStatus)
        
        if ($success) {
            Write-Host "✓ PASS - Status: $actualStatus (Expected: $ExpectedStatus)" -ForegroundColor Green
        } else {
            Write-Host "✗ FAIL - Status: $actualStatus (Expected: $ExpectedStatus)" -ForegroundColor Red
        }
        
        $script:testResults += @{
            Test = $TestName
            Expected = $ExpectedStatus
            Actual = $actualStatus
            Success = $success
        }
    }
    
    Write-Host ""
}

# ===== SIZE LIMIT TESTS =====
Write-Host "===== CIPHERTEXT SIZE TESTS =====" -ForegroundColor Magenta

# Test: Small ciphertext (should pass)
Test-Endpoint `
    -TestName "Small ciphertext (100 bytes)" `
    -Body @{
        ciphertext = "dGVzdC1zbWFsbC1wbGFpbnRleHQ"
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-GCM"
        ttlSeconds = 3600
    } `
    -ExpectedStatus 201 `
    -Description "Small ciphertext should be accepted"

# Test: Medium ciphertext (~5KB base64 encoded)
$mediumText = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes(("a" * 5000)))
Test-Endpoint `
    -TestName "Medium ciphertext (~5KB)" `
    -Body @{
        ciphertext = $mediumText
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-GCM"
        ttlSeconds = 3600
    } `
    -ExpectedStatus 201 `
    -Description "5KB ciphertext should be accepted"

# Test: Large ciphertext (~10KB base64 encoded)
$largeText = [Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes(("a" * 10000)))
Test-Endpoint `
    -TestName "Large ciphertext (~10KB)" `
    -Body @{
        ciphertext = $largeText
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-GCM"
        ttlSeconds = 3600
    } `
    -ExpectedStatus 201 `
    -Description "10KB ciphertext should be accepted"

# Test: Too large ciphertext (>15KB should fail)
$tooLargeText = "a" * 16000
Test-Endpoint `
    -TestName "Too large ciphertext (>15KB)" `
    -Body @{
        ciphertext = $tooLargeText
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-GCM"
        ttlSeconds = 3600
    } `
    -ExpectedStatus 413 `
    -Description "Ciphertext exceeding 15KB should be rejected"

# ===== TTL VALIDATION TESTS =====
Write-Host "===== TTL VALIDATION TESTS =====" -ForegroundColor Magenta

# Test: TTL 3600 (1 hour)
Test-Endpoint `
    -TestName "TTL: 3600 seconds (1 hour)" `
    -Body @{
        ciphertext = "dGVzdC10dGwtMzYwMA"
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-GCM"
        ttlSeconds = 3600
    } `
    -ExpectedStatus 201 `
    -Description "TTL 3600 should be accepted"

# Test: TTL 86400 (24 hours)
Test-Endpoint `
    -TestName "TTL: 86400 seconds (24 hours)" `
    -Body @{
        ciphertext = "dGVzdC10dGwtODY0MDA"
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-GCM"
        ttlSeconds = 86400
    } `
    -ExpectedStatus 201 `
    -Description "TTL 86400 should be accepted"

# Test: TTL 604800 (7 days)
Test-Endpoint `
    -TestName "TTL: 604800 seconds (7 days)" `
    -Body @{
        ciphertext = "dGVzdC10dGwtNjA0ODAw"
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-GCM"
        ttlSeconds = 604800
    } `
    -ExpectedStatus 201 `
    -Description "TTL 604800 should be accepted"

# Test: Invalid TTL (7200)
Test-Endpoint `
    -TestName "TTL: 7200 seconds (invalid)" `
    -Body @{
        ciphertext = "dGVzdC1pbnZhbGlkLXR0bA"
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-GCM"
        ttlSeconds = 7200
    } `
    -ExpectedStatus 400 `
    -Description "Invalid TTL should be rejected"

# Test: Invalid TTL (0)
Test-Endpoint `
    -TestName "TTL: 0 seconds (invalid)" `
    -Body @{
        ciphertext = "dGVzdC16ZXJvLXR0bA"
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-GCM"
        ttlSeconds = 0
    } `
    -ExpectedStatus 400 `
    -Description "Zero TTL should be rejected"

# Test: Negative TTL
Test-Endpoint `
    -TestName "TTL: -1 seconds (invalid)" `
    -Body @{
        ciphertext = "dGVzdC1uZWdhdGl2ZS10dGw"
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-GCM"
        ttlSeconds = -1
    } `
    -ExpectedStatus 400 `
    -Description "Negative TTL should be rejected"

# ===== ALGORITHM VALIDATION TESTS =====
Write-Host "===== ALGORITHM VALIDATION TESTS =====" -ForegroundColor Magenta

# Test: Valid algorithm (AES-GCM)
Test-Endpoint `
    -TestName "Algorithm: AES-GCM (valid)" `
    -Body @{
        ciphertext = "dGVzdC1hZXMtZ2Nt"
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-GCM"
        ttlSeconds = 3600
    } `
    -ExpectedStatus 201 `
    -Description "AES-GCM algorithm should be accepted"

# Test: Invalid algorithm (AES-CBC)
Test-Endpoint `
    -TestName "Algorithm: AES-CBC (invalid)" `
    -Body @{
        ciphertext = "dGVzdC1hZXMtY2Jj"
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-CBC"
        ttlSeconds = 3600
    } `
    -ExpectedStatus 400 `
    -Description "AES-CBC algorithm should be rejected"

# Test: Invalid algorithm (ChaCha20)
Test-Endpoint `
    -TestName "Algorithm: ChaCha20 (invalid)" `
    -Body @{
        ciphertext = "dGVzdC1jaGFjaGEyMA"
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "ChaCha20"
        ttlSeconds = 3600
    } `
    -ExpectedStatus 400 `
    -Description "ChaCha20 algorithm should be rejected"

# Test: Empty algorithm
Test-Endpoint `
    -TestName "Algorithm: empty string (invalid)" `
    -Body @{
        ciphertext = "dGVzdC1lbXB0eS1hbGdv"
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = ""
        ttlSeconds = 3600
    } `
    -ExpectedStatus 400 `
    -Description "Empty algorithm should be rejected"

# ===== NONCE VALIDATION TESTS =====
Write-Host "===== NONCE VALIDATION TESTS =====" -ForegroundColor Magenta

# Test: Valid nonce (12 bytes)
Test-Endpoint `
    -TestName "Nonce: 12 bytes (valid)" `
    -Body @{
        ciphertext = "dGVzdC12YWxpZC1ub25jZQ"
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-GCM"
        ttlSeconds = 3600
    } `
    -ExpectedStatus 201 `
    -Description "12-byte nonce should be accepted"

# Test: Short nonce (8 bytes)
Test-Endpoint `
    -TestName "Nonce: 8 bytes (invalid)" `
    -Body @{
        ciphertext = "dGVzdC1zaG9ydC1ub25jZQ"
        nonce = "MTIzNDU2Nzg"
        algorithm = "AES-GCM"
        ttlSeconds = 3600
    } `
    -ExpectedStatus 400 `
    -Description "8-byte nonce should be rejected"

# Test: Long nonce (16 bytes)
Test-Endpoint `
    -TestName "Nonce: 16 bytes (invalid)" `
    -Body @{
        ciphertext = "dGVzdC1sb25nLW5vbmNl"
        nonce = "MTIzNDU2Nzg5MDEyMzQ1Ng"
        algorithm = "AES-GCM"
        ttlSeconds = 3600
    } `
    -ExpectedStatus 400 `
    -Description "16-byte nonce should be rejected"

# Test: Empty nonce
Test-Endpoint `
    -TestName "Nonce: empty string (invalid)" `
    -Body @{
        ciphertext = "dGVzdC1lbXB0eS1ub25jZQ"
        nonce = ""
        algorithm = "AES-GCM"
        ttlSeconds = 3600
    } `
    -ExpectedStatus 400 `
    -Description "Empty nonce should be rejected"

# Test: Invalid base64url nonce
Test-Endpoint `
    -TestName "Nonce: invalid base64url (invalid)" `
    -Body @{
        ciphertext = "dGVzdC1pbnZhbGlkLW5vbmNl"
        nonce = "invalid!!!nonce"
        algorithm = "AES-GCM"
        ttlSeconds = 3600
    } `
    -ExpectedStatus 400 `
    -Description "Invalid base64url nonce should be rejected"

# ===== CIPHERTEXT VALIDATION TESTS =====
Write-Host "===== CIPHERTEXT VALIDATION TESTS =====" -ForegroundColor Magenta

# Test: Empty ciphertext
Test-Endpoint `
    -TestName "Ciphertext: empty string (invalid)" `
    -Body @{
        ciphertext = ""
        nonce = "MTIzNDU2Nzg5MDEy"
        algorithm = "AES-GCM"
        ttlSeconds = 3600
    } `
    -ExpectedStatus 400 `
    -Description "Empty ciphertext should be rejected"

# ===== MALFORMED REQUEST TESTS =====
Write-Host "===== MALFORMED REQUEST TESTS =====" -ForegroundColor Magenta

# Test: Missing fields
try {
    Write-Host "Test: Missing required fields" -ForegroundColor Yellow
    $response = Invoke-WebRequest -Uri "$API_BASE_URL/api/secrets" `
        -Method POST `
        -Headers @{"Content-Type"="application/json"} `
        -Body '{"ciphertext":"test"}' `
        -ErrorAction Stop
    Write-Host "✗ FAIL - Should have returned 400" -ForegroundColor Red
} catch {
    $status = $_.Exception.Response.StatusCode.value__
    if ($status -eq 400) {
        Write-Host "✓ PASS - Status: 400 (Expected: 400)" -ForegroundColor Green
    } else {
        Write-Host "✗ FAIL - Status: $status (Expected: 400)" -ForegroundColor Red
    }
}
Write-Host ""

# Test: Invalid JSON
try {
    Write-Host "Test: Invalid JSON syntax" -ForegroundColor Yellow
    $response = Invoke-WebRequest -Uri "$API_BASE_URL/api/secrets" `
        -Method POST `
        -Headers @{"Content-Type"="application/json"} `
        -Body 'invalid json {' `
        -ErrorAction Stop
    Write-Host "✗ FAIL - Should have returned 400" -ForegroundColor Red
} catch {
    $status = $_.Exception.Response.StatusCode.value__
    if ($status -eq 400) {
        Write-Host "✓ PASS - Status: 400 (Expected: 400)" -ForegroundColor Green
    } else {
        Write-Host "✗ FAIL - Status: $status (Expected: 400)" -ForegroundColor Red
    }
}
Write-Host ""

# ===== SUMMARY =====
Write-Host "===== TEST SUMMARY =====" -ForegroundColor Cyan
$totalTests = $testResults.Count
$passedTests = ($testResults | Where-Object { $_.Success -eq $true }).Count
$failedTests = $totalTests - $passedTests

Write-Host "Total Tests: $totalTests"
Write-Host "Passed: $passedTests" -ForegroundColor Green
Write-Host "Failed: $failedTests" -ForegroundColor $(if ($failedTests -eq 0) { "Green" } else { "Red" })
Write-Host ""

if ($failedTests -gt 0) {
    Write-Host "Failed Tests:" -ForegroundColor Red
    $testResults | Where-Object { $_.Success -eq $false } | ForEach-Object {
        Write-Host "  - $($_.Test): Expected $($_.Expected), Got $($_.Actual)" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "Testing completed!" -ForegroundColor Cyan
