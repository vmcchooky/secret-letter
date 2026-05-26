#!/bin/bash

# Manual test script for POST /api/secrets endpoint
# This script tests the create secret flow with various scenarios

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"

echo "Testing POST /api/secrets endpoint"
echo "API Base URL: $API_BASE_URL"
echo ""

# Test 1: Valid request with 1 hour TTL
echo "Test 1: Valid request with 1 hour TTL"
curl -X POST "$API_BASE_URL/api/secrets" \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: test-$(date +%s)" \
  -d '{
    "ciphertext": "dGVzdC1jaXBoZXJ0ZXh0LWJhc2U2NHVybA",
    "nonce": "MTIzNDU2Nzg5MDEy",
    "algorithm": "AES-GCM",
    "ttlSeconds": 3600
  }' \
  -w "\nHTTP Status: %{http_code}\n\n"

# Test 2: Valid request with 24 hours TTL
echo "Test 2: Valid request with 24 hours TTL"
curl -X POST "$API_BASE_URL/api/secrets" \
  -H "Content-Type: application/json" \
  -d '{
    "ciphertext": "YW5vdGhlci10ZXN0LWNpcGhlcnRleHQ",
    "nonce": "bm9uY2UtMTIzNDU2",
    "algorithm": "AES-GCM",
    "ttlSeconds": 86400
  }' \
  -w "\nHTTP Status: %{http_code}\n\n"

# Test 3: Invalid algorithm (should return 400)
echo "Test 3: Invalid algorithm (should return 400)"
curl -X POST "$API_BASE_URL/api/secrets" \
  -H "Content-Type: application/json" \
  -d '{
    "ciphertext": "dGVzdA",
    "nonce": "MTIzNDU2Nzg5MDEy",
    "algorithm": "AES-CBC",
    "ttlSeconds": 3600
  }' \
  -w "\nHTTP Status: %{http_code}\n\n"

# Test 4: Invalid TTL (should return 400)
echo "Test 4: Invalid TTL (should return 400)"
curl -X POST "$API_BASE_URL/api/secrets" \
  -H "Content-Type: application/json" \
  -d '{
    "ciphertext": "dGVzdA",
    "nonce": "MTIzNDU2Nzg5MDEy",
    "algorithm": "AES-GCM",
    "ttlSeconds": 7200
  }' \
  -w "\nHTTP Status: %{http_code}\n\n"

# Test 5: Empty ciphertext (should return 400)
echo "Test 5: Empty ciphertext (should return 400)"
curl -X POST "$API_BASE_URL/api/secrets" \
  -H "Content-Type: application/json" \
  -d '{
    "ciphertext": "",
    "nonce": "MTIzNDU2Nzg5MDEy",
    "algorithm": "AES-GCM",
    "ttlSeconds": 3600
  }' \
  -w "\nHTTP Status: %{http_code}\n\n"

# Test 6: Invalid nonce length (should return 400)
echo "Test 6: Invalid nonce length (should return 400)"
curl -X POST "$API_BASE_URL/api/secrets" \
  -H "Content-Type: application/json" \
  -d '{
    "ciphertext": "dGVzdA",
    "nonce": "c2hvcnQ",
    "algorithm": "AES-GCM",
    "ttlSeconds": 3600
  }' \
  -w "\nHTTP Status: %{http_code}\n\n"

# Test 7: Payload too large (should return 413)
echo "Test 7: Payload too large (should return 413)"
LARGE_PAYLOAD=$(printf 'a%.0s' {1..16384})
curl -X POST "$API_BASE_URL/api/secrets" \
  -H "Content-Type: application/json" \
  -d "$LARGE_PAYLOAD" \
  -w "\nHTTP Status: %{http_code}\n\n"

echo "All tests completed!"
