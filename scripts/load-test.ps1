#!/usr/bin/env pwsh
# Load testing script for one-time-link API

param(
    [int]$Concurrent = 10,
    [int]$Requests = 100,
    [string]$Endpoint = "create",
    [string]$BaseUrl = "http://localhost:8080"
)

$ErrorActionPreference = "Stop"

Write-Host "=== Load Testing ===" -ForegroundColor Cyan
Write-Host "Endpoint: $Endpoint" -ForegroundColor Yellow
Write-Host "Concurrent: $Concurrent" -ForegroundColor Yellow
Write-Host "Total Requests: $Requests" -ForegroundColor Yellow
Write-Host ""

# Test data
$testSecret = @{
    ciphertext = "dGVzdCBjaXBoZXJ0ZXh0IGZvciBsb2FkIHRlc3Rpbmc"
    nonce = "MTIzNDU2Nzg5MDEy"
    algorithm = "AES-GCM"
    ttlSeconds = 3600
} | ConvertTo-Json

# Metrics
$successCount = 0
$errorCount = 0
$totalDuration = 0
$durations = @()
$statusCodes = @{}

# Function to make a request
function Invoke-TestRequest {
    param([string]$Url, [string]$Body, [string]$Method = "POST")
    
    $start = Get-Date
    try {
        $response = Invoke-RestMethod -Uri $Url `
            -Method $Method `
            -ContentType "application/json" `
            -Body $Body `
            -TimeoutSec 30 `
            -ErrorAction Stop
        
        $duration = (Get-Date) - $start
        return @{
            Success = $true
            Duration = $duration.TotalMilliseconds
            StatusCode = 200
        }
    }
    catch {
        $duration = (Get-Date) - $start
        $statusCode = 500
        if ($_.Exception.Response) {
            $statusCode = [int]$_.Exception.Response.StatusCode
        }
        return @{
            Success = $false
            Duration = $duration.TotalMilliseconds
            StatusCode = $statusCode
            Error = $_.Exception.Message
        }
    }
}

# Determine endpoint URL
$url = switch ($Endpoint) {
    "create" { "$BaseUrl/api/secrets" }
    "health" { "$BaseUrl/healthz" }
    default { "$BaseUrl/api/secrets" }
}

Write-Host "Starting load test..." -ForegroundColor Green
$overallStart = Get-Date

# Run requests in batches
$batchSize = $Concurrent
$totalBatches = [Math]::Ceiling($Requests / $batchSize)

for ($batch = 0; $batch < $totalBatches; $batch++) {
    $batchStart = Get-Date
    $requestsInBatch = [Math]::Min($batchSize, $Requests - ($batch * $batchSize))
    
    # Create jobs for concurrent requests
    $jobs = @()
    for ($i = 0; $i < $requestsInBatch; $i++) {
        $jobs += Start-Job -ScriptBlock {
            param($Url, $Body, $Method)
            
            $start = Get-Date
            try {
                if ($Method -eq "GET") {
                    $response = Invoke-RestMethod -Uri $Url -Method GET -TimeoutSec 30
                } else {
                    $response = Invoke-RestMethod -Uri $Url `
                        -Method POST `
                        -ContentType "application/json" `
                        -Body $Body `
                        -TimeoutSec 30
                }
                
                $duration = (Get-Date) - $start
                return @{
                    Success = $true
                    Duration = $duration.TotalMilliseconds
                    StatusCode = 200
                }
            }
            catch {
                $duration = (Get-Date) - $start
                $statusCode = 500
                if ($_.Exception.Response) {
                    $statusCode = [int]$_.Exception.Response.StatusCode
                }
                return @{
                    Success = $false
                    Duration = $duration.TotalMilliseconds
                    StatusCode = $statusCode
                }
            }
        } -ArgumentList $url, $testSecret, $(if ($Endpoint -eq "health") { "GET" } else { "POST" })
    }
    
    # Wait for all jobs to complete
    $results = $jobs | Wait-Job | Receive-Job
    $jobs | Remove-Job
    
    # Process results
    foreach ($result in $results) {
        if ($result.Success) {
            $successCount++
        } else {
            $errorCount++
        }
        
        $durations += $result.Duration
        $totalDuration += $result.Duration
        
        $statusCode = $result.StatusCode
        if ($statusCodes.ContainsKey($statusCode)) {
            $statusCodes[$statusCode]++
        } else {
            $statusCodes[$statusCode] = 1
        }
    }
    
    $batchDuration = (Get-Date) - $batchStart
    $completed = ($batch + 1) * $batchSize
    if ($completed > $Requests) { $completed = $Requests }
    
    Write-Host "Batch $($batch + 1)/$totalBatches completed ($completed/$Requests requests) - ${batchDuration}ms" -ForegroundColor Gray
}

$overallDuration = (Get-Date) - $overallStart

Write-Host ""
Write-Host "=== Results ===" -ForegroundColor Cyan

# Calculate statistics
$sortedDurations = $durations | Sort-Object
$p50 = $sortedDurations[[Math]::Floor($sortedDurations.Count * 0.5)]
$p95 = $sortedDurations[[Math]::Floor($sortedDurations.Count * 0.95)]
$p99 = $sortedDurations[[Math]::Floor($sortedDurations.Count * 0.99)]
$min = $sortedDurations[0]
$max = $sortedDurations[-1]
$avg = ($durations | Measure-Object -Average).Average

Write-Host "Total Requests: $Requests" -ForegroundColor White
Write-Host "Successful: $successCount" -ForegroundColor Green
Write-Host "Failed: $errorCount" -ForegroundColor $(if ($errorCount -gt 0) { "Red" } else { "Green" })
Write-Host ""

Write-Host "Duration:" -ForegroundColor White
Write-Host "  Total: $([Math]::Round($overallDuration.TotalSeconds, 2))s" -ForegroundColor White
Write-Host "  Throughput: $([Math]::Round($Requests / $overallDuration.TotalSeconds, 2)) req/s" -ForegroundColor White
Write-Host ""

Write-Host "Response Times (ms):" -ForegroundColor White
Write-Host "  Min: $([Math]::Round($min, 2))" -ForegroundColor White
Write-Host "  Max: $([Math]::Round($max, 2))" -ForegroundColor White
Write-Host "  Avg: $([Math]::Round($avg, 2))" -ForegroundColor White
Write-Host "  P50: $([Math]::Round($p50, 2))" -ForegroundColor White
Write-Host "  P95: $([Math]::Round($p95, 2))" -ForegroundColor White
Write-Host "  P99: $([Math]::Round($p99, 2))" -ForegroundColor White
Write-Host ""

Write-Host "Status Codes:" -ForegroundColor White
foreach ($code in $statusCodes.Keys | Sort-Object) {
    $count = $statusCodes[$code]
    $color = if ($code -ge 200 -and $code -lt 300) { "Green" } elseif ($code -ge 400) { "Red" } else { "Yellow" }
    Write-Host "  $code : $count" -ForegroundColor $color
}

Write-Host ""

# Performance assessment
if ($p95 -lt 50) {
    Write-Host "✓ Performance: Excellent (P95 < 50ms)" -ForegroundColor Green
} elseif ($p95 -lt 100) {
    Write-Host "✓ Performance: Good (P95 < 100ms)" -ForegroundColor Green
} elseif ($p95 -lt 200) {
    Write-Host "⚠ Performance: Acceptable (P95 < 200ms)" -ForegroundColor Yellow
} else {
    Write-Host "✗ Performance: Poor (P95 >= 200ms)" -ForegroundColor Red
}

if ($errorCount -eq 0) {
    Write-Host "✓ Reliability: 100% success rate" -ForegroundColor Green
} elseif ($errorCount / $Requests -lt 0.01) {
    Write-Host "✓ Reliability: >99% success rate" -ForegroundColor Green
} elseif ($errorCount / $Requests -lt 0.05) {
    Write-Host "⚠ Reliability: >95% success rate" -ForegroundColor Yellow
} else {
    Write-Host "✗ Reliability: <95% success rate" -ForegroundColor Red
}

Write-Host ""
Write-Host "=== Load Test Complete ===" -ForegroundColor Cyan
