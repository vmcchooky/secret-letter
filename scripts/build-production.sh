#!/bin/bash
# Production build script for secret-letter API

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
BINARY_NAME="secret-letter-api"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION=$(go version | awk '{print $3}')
REQUIRE_SECURITY_TOOLS="${REQUIRE_SECURITY_TOOLS:-0}"

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
if go test ./... -v; then
    echo -e "${GREEN}✓${NC} All tests passed"
else
    echo -e "${RED}✗${NC} Tests failed"
    exit 1
fi
echo ""

# Step 3: Run security audit
echo -e "${YELLOW}[3/7]${NC} Running security audit..."
if command -v gosec &> /dev/null; then
    gosec ./...
    echo -e "${GREEN}✓${NC} gosec passed"
else
    echo -e "${YELLOW}⚠${NC} gosec not installed - skipping static security scan"
    echo "  Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"
    if [ "$REQUIRE_SECURITY_TOOLS" = "1" ]; then
        exit 1
    fi
fi

if command -v govulncheck &> /dev/null; then
    govulncheck ./...
    echo -e "${GREEN}✓${NC} govulncheck passed"
else
    echo -e "${YELLOW}⚠${NC} govulncheck not installed - skipping vulnerability check"
    echo "  Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
    if [ "$REQUIRE_SECURITY_TOOLS" = "1" ]; then
        exit 1
    fi
fi
echo ""

# Step 4: Build for Linux (production target)
echo -e "${YELLOW}[4/7]${NC} Building for Linux (amd64)..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" \
    -o "$BUILD_DIR/${BINARY_NAME}-linux-amd64" \
    ./backend/cmd/api
echo -e "${GREEN}✓${NC} Linux build complete"
echo ""

# Step 5: Build for current platform (for testing)
echo -e "${YELLOW}[5/7]${NC} Building for current platform..."
go build \
    -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" \
    -o "$BUILD_DIR/${BINARY_NAME}" \
    ./backend/cmd/api
echo -e "${GREEN}✓${NC} Local build complete"
echo ""

# Step 6: Create deployment package
echo -e "${YELLOW}[6/7]${NC} Creating deployment package..."
cp deploy/prod/.env.example "$BUILD_DIR/.env.example"
cp deploy/prod/.env.vps-edge.example "$BUILD_DIR/.env.vps-edge.example"
cp -r deploy "$BUILD_DIR/"
echo "$VERSION" > "$BUILD_DIR/VERSION"

# Create tarball
tar -czf "$BUILD_DIR/${BINARY_NAME}-${VERSION}.tar.gz" \
    -C "$BUILD_DIR" \
    "${BINARY_NAME}-linux-amd64" \
    ".env.example" \
    ".env.vps-edge.example" \
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
echo "2. Review deploy/prod/.env.example or deploy/prod/.env.vps-edge.example for your environment"
echo "3. Deploy the tarball: $BUILD_DIR/${BINARY_NAME}-${VERSION}.tar.gz"
echo "4. Follow the production checklist in docs/PRODUCTION_CHECKLIST.md"
echo ""
