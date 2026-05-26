#!/usr/bin/env pwsh
# Production build script for one-time-link API

$ErrorActionPreference = "Stop"

Write-Host "=== Production Build Script ===" -ForegroundColor Cyan
Write-Host ""

# Configuration
$BUILD_DIR = "build"
$BINARY_NAME = "one-time-link-api"
$VERSION = (git describe --tags --always --dirty 2>$null) ?? "dev"
$BUILD_TIME = (Get-Date).ToUniversalTime().ToString("yyyy-MM-dd_HH:mm:ss")
$GO_VERSION = (go version).Split()[2]

Write-Host "Version: $VERSION" -ForegroundColor White
Write-Host "Build Time: $BUILD_TIME" -ForegroundColor White
Write-Host "Go Version: $GO_VERSION" -ForegroundColor White
Write-Host ""

# Step 1: Clean previous builds
Write-Host "[1/7] Cleaning previous builds..." -ForegroundColor Yellow
if (Test-Path $BUILD_DIR) {
    Remove-Item -Recurse -Force $BUILD_DIR
}
New-Item -ItemType Directory -Path $BUILD_DIR | Out-Null
Write-Host "✓ Clean complete" -ForegroundColor Green
Write-Host ""

# Step 2: Run tests
Write-Host "[2/7] Running tests..." -ForegroundColor Yellow
Push-Location backend
$testResult = go test ./... -v
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ All tests passed" -ForegroundColor Green
} else {
    Write-Host "✗ Tests failed" -ForegroundColor Red
    Pop-Location
    exit 1
}
Pop-Location
Write-Host ""

# Step 3: Run security audit
Write-Host "[3/7] Running security audit..." -ForegroundColor Yellow
Push-Location backend
if (Get-Command govulncheck -ErrorAction SilentlyContinue) {
    $vulnResult = govulncheck ./...
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ No vulnerabilities found" -ForegroundColor Green
    } else {
        Write-Host "⚠ Vulnerabilities detected - review before deploying" -ForegroundColor Red
        $continue = Read-Host "Continue anyway? (y/N)"
        if ($continue -ne "y" -and $continue -ne "Y") {
            Pop-Location
            exit 1
        }
    }
} else {
    Write-Host "⚠ govulncheck not installed - skipping vulnerability check" -ForegroundColor Yellow
    Write-Host "  Install with: go install golang.org/x/vuln/cmd/govulncheck@latest" -ForegroundColor Gray
}
Pop-Location
Write-Host ""

# Step 4: Build for Linux (production target)
Write-Host "[4/7] Building for Linux (amd64)..." -ForegroundColor Yellow
Push-Location backend
$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build `
    -ldflags="-w -s -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" `
    -o "../$BUILD_DIR/${BINARY_NAME}-linux-amd64" `
    ./cmd/api
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Linux build complete" -ForegroundColor Green
} else {
    Write-Host "✗ Linux build failed" -ForegroundColor Red
    Pop-Location
    exit 1
}
Pop-Location
Write-Host ""

# Step 5: Build for Windows (for testing)
Write-Host "[5/7] Building for Windows..." -ForegroundColor Yellow
Push-Location backend
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build `
    -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" `
    -o "../$BUILD_DIR/${BINARY_NAME}.exe" `
    ./cmd/api
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Windows build complete" -ForegroundColor Green
} else {
    Write-Host "✗ Windows build failed" -ForegroundColor Red
    Pop-Location
    exit 1
}
Pop-Location
Write-Host ""

# Step 6: Create deployment package
Write-Host "[6/7] Creating deployment package..." -ForegroundColor Yellow
Copy-Item "backend/.env.production" "$BUILD_DIR/.env.example"
Copy-Item -Recurse "deploy" "$BUILD_DIR/"
Set-Content -Path "$BUILD_DIR/VERSION" -Value $VERSION

# Create zip archive
$zipPath = "$BUILD_DIR/${BINARY_NAME}-${VERSION}.zip"
Compress-Archive -Path "$BUILD_DIR/${BINARY_NAME}-linux-amd64", "$BUILD_DIR/.env.example", "$BUILD_DIR/deploy", "$BUILD_DIR/VERSION" `
    -DestinationPath $zipPath -Force

Write-Host "✓ Deployment package created: ${BINARY_NAME}-${VERSION}.zip" -ForegroundColor Green
Write-Host ""

# Step 7: Display build info
Write-Host "[7/7] Build Summary" -ForegroundColor Yellow
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Gray
Write-Host "Version:        $VERSION" -ForegroundColor White
Write-Host "Build Time:     $BUILD_TIME" -ForegroundColor White
Write-Host "Go Version:     $GO_VERSION" -ForegroundColor White
Write-Host ""
Write-Host "Artifacts:" -ForegroundColor White
Get-ChildItem $BUILD_DIR | Where-Object { $_.Name -match $BINARY_NAME -or $_.Extension -eq ".zip" } | ForEach-Object {
    $size = if ($_.Length -gt 1MB) { "{0:N2} MB" -f ($_.Length / 1MB) } else { "{0:N2} KB" -f ($_.Length / 1KB) }
    Write-Host "  $($_.Name) ($size)" -ForegroundColor Gray
}
Write-Host ""
Write-Host "✓ Production build complete!" -ForegroundColor Green
Write-Host ""

# Display next steps
Write-Host "Next Steps:" -ForegroundColor Cyan
Write-Host "1. Test the binary: .\$BUILD_DIR\${BINARY_NAME}.exe" -ForegroundColor White
Write-Host "2. Review .env.production and configure for your environment" -ForegroundColor White
Write-Host "3. Deploy the package: $BUILD_DIR\${BINARY_NAME}-${VERSION}.zip" -ForegroundColor White
Write-Host "4. Follow the production checklist in docs\PRODUCTION_CHECKLIST.md" -ForegroundColor White
Write-Host ""
