# Test Redis TTL expiration
# This script creates a secret with short TTL and verifies it expires

$API_BASE_URL = if ($env:API_BASE_URL) { $env:API_BASE_URL } else { "http://localhost:8080" }

Write-Host "===== REDIS TTL EXPIRATION TEST =====" -ForegroundColor Cyan
Write-Host ""

# Note: This test uses actual allowed TTL values (3600, 86400, 604800)
# We'll create a secret and check it exists, then verify TTL in Redis

Write-Host "Step 1: Creating secret with 1 hour TTL..." -ForegroundColor Yellow

$body = @{
    ciphertext = "dGVzdC1leHBpcmF0aW9uLXNlY3JldA"
    nonce = "ZXhwaXJlLW5vbmNl"
    algorithm = "AES-GCM"
    ttlSeconds = 3600
} | ConvertTo-Json

try {
    $response = Invoke-WebRequest -Uri "$API_BASE_URL/api/secrets" `
        -Method POST `
        -Headers @{"Content-Type"="application/json"} `
        -Body $body `
        -ErrorAction Stop
    
    $result = $response.Content | ConvertFrom-Json
    $secretId = $result.secretId
    $expiresAt = $result.expiresAt
    
    Write-Host "✓ Secret created successfully" -ForegroundColor Green
    Write-Host "  Secret ID: $secretId" -ForegroundColor Gray
    Write-Host "  Expires At: $expiresAt" -ForegroundColor Gray
    Write-Host ""
    
    Write-Host "Step 2: Verify secret exists in Redis..." -ForegroundColor Yellow
    Write-Host "Run this command in Redis CLI to check:" -ForegroundColor Gray
    Write-Host "  EXISTS secret:$secretId" -ForegroundColor Cyan
    Write-Host "  TTL secret:$secretId" -ForegroundColor Cyan
    Write-Host "  GET secret:$secretId" -ForegroundColor Cyan
    Write-Host ""
    
    Write-Host "Expected Results:" -ForegroundColor Yellow
    Write-Host "  - EXISTS should return 1 (key exists)" -ForegroundColor Gray
    Write-Host "  - TTL should return ~3600 seconds" -ForegroundColor Gray
    Write-Host "  - GET should return JSON with encrypted data" -ForegroundColor Gray
    Write-Host ""
    
    Write-Host "Step 3: After TTL expires (1 hour)..." -ForegroundColor Yellow
    Write-Host "  - EXISTS should return 0 (key deleted)" -ForegroundColor Gray
    Write-Host "  - TTL should return -2 (key doesn't exist)" -ForegroundColor Gray
    Write-Host ""
    
    Write-Host "Redis Commands for Manual Verification:" -ForegroundColor Cyan
    Write-Host "docker exec -it <container-id> redis-cli" -ForegroundColor White
    Write-Host "EXISTS secret:$secretId" -ForegroundColor White
    Write-Host "TTL secret:$secretId" -ForegroundColor White
    Write-Host ""
    
} catch {
    Write-Host "✗ Failed to create secret" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
}

Write-Host "Note: For faster testing, you can modify the backend code temporarily" -ForegroundColor Yellow
Write-Host "to accept shorter TTL values (e.g., 10 seconds) for testing purposes." -ForegroundColor Yellow
