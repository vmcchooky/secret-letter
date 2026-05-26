#!/bin/bash
# Milestone 3 Reveal Flow Test Script
# Tests the complete create -> status -> consume flow

set -e

API_BASE="http://localhost:8080"

echo "=== Milestone 3 Reveal Flow Test ==="
echo ""

# Test 1: Health Check
echo "[1/6] Testing health endpoint..."
HEALTH=$(curl -s "$API_BASE/healthz")
if echo "$HEALTH" | grep -q "healthy"; then
    echo "✓ Health check passed"
else
    echo "✗ Health check failed"
    exit 1
fi

# Test 2: Create Secret
echo "[2/6] Creating secret..."
CREATE_RESP=$(curl -s -X POST "$API_BASE/api/secrets" \
    -H "Content-Type: application/json" \
    -d '{
        "ciphertext": "dGVzdC1yZXZlYWwtZmxvdy1taWxlc3RvbmUz",
        "nonce": "MTIzNDU2Nzg5MDEy",
        "algorithm": "AES-GCM",
        "ttlSeconds": 3600
    }')

SECRET_ID=$(echo "$CREATE_RESP" | grep -o '"secretId":"[^"]*"' | cut -d'"' -f4)
if [ -z "$SECRET_ID" ]; then
    echo "✗ Failed to create secret"
    exit 1
fi
echo "✓ Secret created: $SECRET_ID"

# Test 3: Check Status (should be pending)
echo "[3/6] Checking secret status..."
STATUS_RESP=$(curl -s "$API_BASE/api/secrets/$SECRET_ID/status")
STATUS=$(echo "$STATUS_RESP" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)

if [ "$STATUS" = "pending" ]; then
    echo "✓ Status check passed: $STATUS"
else
    echo "✗ Expected status 'pending', got '$STATUS'"
    exit 1
fi

# Test 4: Consume Secret (first time should succeed)
echo "[4/6] Consuming secret (first attempt)..."
CONSUME_RESP=$(curl -s -X POST "$API_BASE/api/secrets/$SECRET_ID/consume" \
    -H "Content-Type: application/json" \
    -d '{}')

if echo "$CONSUME_RESP" | grep -q "ciphertext"; then
    echo "✓ Secret consumed successfully"
else
    echo "✗ Failed to consume secret"
    exit 1
fi

# Test 5: Try to Consume Again (should fail with 410)
echo "[5/6] Attempting to consume again (should fail)..."
CONSUME2_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_BASE/api/secrets/$SECRET_ID/consume" \
    -H "Content-Type: application/json" \
    -d '{}')

if [ "$CONSUME2_STATUS" = "410" ]; then
    echo "✓ Second consume correctly rejected (410 Gone)"
else
    echo "✗ Expected status 410, got $CONSUME2_STATUS"
    exit 1
fi

# Test 6: Check Status Again (should be not_found)
echo "[6/6] Checking status after consumption..."
STATUS_AFTER_RESP=$(curl -s "$API_BASE/api/secrets/$SECRET_ID/status")
STATUS_AFTER=$(echo "$STATUS_AFTER_RESP" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)

if [ "$STATUS_AFTER" = "not_found" ]; then
    echo "✓ Status correctly shows 'not_found'"
else
    echo "✗ Expected status 'not_found', got '$STATUS_AFTER'"
    exit 1
fi

echo ""
echo "=== All Tests Passed! ==="
echo ""
echo "Milestone 3 reveal flow is working correctly:"
echo "  ✓ Create secret"
echo "  ✓ Check status (pending)"
echo "  ✓ Consume secret (success)"
echo "  ✓ Prevent double consumption (410)"
echo "  ✓ Status after consumption (not_found)"
