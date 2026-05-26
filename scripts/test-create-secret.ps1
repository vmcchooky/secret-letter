# Manual test script for POST /api/secrets endpoint
# PowerShell version for Windows

$API_BASE_URL = if ($env:API_BASE_URL) { $env:API_BASE_URL } else { "http://localhost:8080" }

Write-Host "Testing POST /api/secrets endpoint" -ForegroundColor Cyan
Write-Host "API Base URL: $API_BASE_URL"
Write-Host ""

# Test 1: Valid request with 1 hour TTL
Write-Host "Test 1: Valid request with 1 hour TTL" -ForegroundColor Yellow
$body1 = @{
    ciphertext = "dGVzdC1jaXBoZXJ0ZXh0LWJhc2U2NHVybA"
    nonce = "MTIzNDU2Nzg5MDEy"
    algorithm = "AES-GCM"
    ttlSeconds = 3600
} | ConvertTo-Json

try {
    $response1 = Invoke-WebRequest -Uri "$API_BASE_URL/api/secrets" `
        -Method POST `
        -Headers @{"Content-Type"="application/json"; "X-Request-ID"="test-$(Get-Date -Format 'yyyyMMddHHmmss')"} `
        -Body $body1
    Write-Host "Status: $($response1.StatusCode)" -ForegroundColor Green
    Write-Host $response1.Content
} catch {
    Write-Host "Status: $($_.Exception.Response.StatusCode.value__)" -ForegroundColor Red
    Write-Host $_.Exception.Message
}
Write-Host ""

# Test 2: Valid request with 24 hours TTL
Write-Host "Test 2: Valid request with 24 hours TTL" -ForegroundColor Yellow
$body2 = @{
    ciphertext = "YW5vdGhlci10ZXN0LWNpcGhlcnRleHQ"
    nonce = "bm9uY2UtMTIzNDU2"
    algorithm = "AES-GCM"
    ttlSeconds = 86400
} | ConvertTo-Json

try {
    $response2 = Invoke-WebRequest -Uri "$API_BASE_URL/api/secrets" `
        -Method POST `
        -Headers @{"Content-Type"="application/json"} `
        -Body $body2
    Write-Host "Status: $($response2.StatusCode)" -ForegroundColor Green
    Write-Host $response2.Content
} catch {
    Write-Host "Status: $($_.Exception.Response.StatusCode.value__)" -ForegroundColor Red
}
Write-Host ""

# Test 3: Invalid algorithm (should return 400)
Write-Host "Test 3: Invalid algorithm (should return 400)" -ForegroundColor Yellow
$body3 = @{
    ciphertext = "dGVzdA"
    nonce = "MTIzNDU2Nzg5MDEy"
    algorithm = "AES-CBC"
    ttlSeconds = 3600
} | ConvertTo-Json

try {
    $response3 = Invoke-WebRequest -Uri "$API_BASE_URL/api/secrets" `
        -Method POST `
        -Headers @{"Content-Type"="application/json"} `
        -Body $body3
    Write-Host "Status: $($response3.StatusCode)"
} catch {
    Write-Host "Status: $($_.Exception.Response.StatusCode.value__) (Expected 400)" -ForegroundColor Green
}
Write-Host ""

# Test 4: Invalid TTL (should return 400)
Write-Host "Test 4: Invalid TTL (should return 400)" -ForegroundColor Yellow
$body4 = @{
    ciphertext = "dGVzdA"
    nonce = "MTIzNDU2Nzg5MDEy"
    algorithm = "AES-GCM"
    ttlSeconds = 7200
} | ConvertTo-Json

try {
    $response4 = Invoke-WebRequest -Uri "$API_BASE_URL/api/secrets" `
        -Method POST `
        -Headers @{"Content-Type"="application/json"} `
        -Body $body4
    Write-Host "Status: $($response4.StatusCode)"
} catch {
    Write-Host "Status: $($_.Exception.Response.StatusCode.value__) (Expected 400)" -ForegroundColor Green
}
Write-Host ""

# Test 5: Empty ciphertext (should return 400)
Write-Host "Test 5: Empty ciphertext (should return 400)" -ForegroundColor Yellow
$body5 = @{
    ciphertext = ""
    nonce = "MTIzNDU2Nzg5MDEy"
    algorithm = "AES-GCM"
    ttlSeconds = 3600
} | ConvertTo-Json

try {
    $response5 = Invoke-WebRequest -Uri "$API_BASE_URL/api/secrets" `
        -Method POST `
        -Headers @{"Content-Type"="application/json"} `
        -Body $body5
    Write-Host "Status: $($response5.StatusCode)"
} catch {
    Write-Host "Status: $($_.Exception.Response.StatusCode.value__) (Expected 400)" -ForegroundColor Green
}
Write-Host ""

Write-Host "All tests completed!" -ForegroundColor Cyan
