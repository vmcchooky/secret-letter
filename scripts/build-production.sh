#!/bin/bash
# Production build script for one-time-link API

set -e

echo "=== Production Build Script ==="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BUILD_DIR="build"
BINARY_NAME="one-time-link-api"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION=$(go version | awk '{print $3}')

echo "Version: $VERSION"
echo "Build Time: $BUILD_TIME"
echo "Go Version: $GO_VERSION"
echo ""

# Step 1: Clean previous builds
echo -e "${YELLOW}[1/7]${NC} Cleaning previous builds..."
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"
echo -e "${GREEN}✓${NC} Clean complete"
echo ""

# Step 2: Run tests
echo -e "${YELLOW}[2/7]${NC} Running tests..."
cd backend
if go test ./... -v; then
    echo -e "${GREEN}✓${NC} All tests passed"
else
    echo -e "${RED}✗${NC} Tests failed"
    exit 1
fi
cd ..
echo ""

# Step 3: Run security audit
echo -e "${YELLOW}[3/7]${NC} Running security audit..."
cd backend
if command -v govulncheck &> /dev/null; then
    if govulncheck ./...; then
        echo -e "${GREEN}✓${NC} No vulnerabilities found"
    else
        echo -e "${RED}⚠${NC} Vulnerabilities detected - review before deploying"
        read -p "Continue anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
else
    echo -e "${YELLOW}⚠${NC} govulncheck not installed - skipping vulnerability check"
    echo "  Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi
cd ..
echo ""

# Step 4: Build for Linux (production target)
echo -e "${YELLOW}[4/7]${NC} Building for Linux (amd64)..."
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" \
    -o "../$BUILD_DIR/${BINARY_NAME}-linux-amd64" \
    ./cmd/api
echo -e "${GREEN}✓${NC} Linux build complete"
cd ..
echo ""

# Step 5: Build for current platform (for testing)
echo -e "${YELLOW}[5/7]${NC} Building for current platform..."
cd backend
go build \
    -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" \
    -o "../$BUILD_DIR/${BINARY_NAME}" \
    ./cmd/api
echo -e "${GREEN}✓${NC} Local build complete"
cd ..
echo ""

# Step 6: Create deployment package
echo -e "${YELLOW}[6/7]${NC} Creating deployment package..."
cp backend/.env.production "$BUILD_DIR/.env.example"
cp -r deploy "$BUILD_DIR/"
echo "$VERSION" > "$BUILD_DIR/VERSION"

# Create tarball
tar -czf "$BUILD_DIR/${BINARY_NAME}-${VERSION}.tar.gz" \
    -C "$BUILD_DIR" \
    "${BINARY_NAME}-linux-amd64" \
    ".env.example" \
    "deploy" \
    "VERSION"

echo -e "${GREEN}✓${NC} Deployment package created: ${BINARY_NAME}-${VERSION}.tar.gz"
echo ""

# Step 7: Display build info
echo -e "${YELLOW}[7/7]${NC} Build Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Version:        $VERSION"
echo "Build Time:     $BUILD_TIME"
echo "Go Version:     $GO_VERSION"
echo ""
echo "Artifacts:"
ls -lh "$BUILD_DIR" | grep -E "${BINARY_NAME}|tar.gz" | awk '{print "  " $9 " (" $5 ")"}'
echo ""
echo -e "${GREEN}✓${NC} Production build complete!"
echo ""

# Display next steps
echo "Next Steps:"
echo "1. Test the binary: ./$BUILD_DIR/${BINARY_NAME}"
echo "2. Review .env.production and configure for your environment"
echo "3. Deploy the tarball: $BUILD_DIR/${BINARY_NAME}-${VERSION}.tar.gz"
echo "4. Follow the production checklist in docs/PRODUCTION_CHECKLIST.md"
echo ""
