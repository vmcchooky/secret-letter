#!/usr/bin/env bash

set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:18080}"
EDGE_API_URL="${EDGE_API_URL:-https://api.secret.quorix.io.vn}"
READY_TIMEOUT_SECONDS="${READY_TIMEOUT_SECONDS:-60}"
RESTART_COMMAND="${RESTART_COMMAND:-docker compose -f deploy/prod/docker-compose.yml -f deploy/prod/docker-compose.vps-edge.yml restart api}"
TEST_CIPHERTEXT="${TEST_CIPHERTEXT:-cHJvZHVjdGlvbi1zbW9rZS10ZXN0LWNpcGhlcnRleHQ}"
TEST_NONCE="${TEST_NONCE:-MTIzNDU2Nzg5MDEy}"

wait_for_ready() {
  local deadline
  deadline=$((SECONDS + READY_TIMEOUT_SECONDS))

  while [ "$SECONDS" -lt "$deadline" ]; do
    local status
    status="$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE_URL/readyz")"
    if [ "$status" = "200" ]; then
      return 0
    fi
    sleep 2
  done

  return 1
}

echo "=== Production Smoke Test ==="
echo "API_BASE_URL: $API_BASE_URL"
echo "EDGE_API_URL: $EDGE_API_URL"
echo

echo "[1/7] Checking local health and readiness endpoints..."
health_status="$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE_URL/healthz")"
ready_status="$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE_URL/readyz")"
if [ "$health_status" != "200" ] || [ "$ready_status" != "200" ]; then
  echo "Expected healthz=200 and readyz=200 before the test, got healthz=$health_status readyz=$ready_status"
  exit 1
fi

echo "[2/7] Creating a secret before restart..."
create_response="$(curl -s -X POST "$API_BASE_URL/api/secrets" \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: production-smoke-create-$(date +%s)" \
  -d "{
    \"ciphertext\": \"$TEST_CIPHERTEXT\",
    \"nonce\": \"$TEST_NONCE\",
    \"algorithm\": \"AES-GCM\",
    \"ttlSeconds\": 3600
  }")"

secret_id="$(printf "%s\n" "$create_response" | grep -o '"secretId":"[^"]*"' | cut -d'"' -f4 || true)"
if [ -z "$secret_id" ]; then
  echo "Failed to create a secret. Response:"
  echo "$create_response"
  exit 1
fi
echo "Created secret: $secret_id"

echo "[3/7] Verifying status before restart..."
status_response="$(curl -s "$API_BASE_URL/api/secrets/$secret_id/status")"
status_value="$(printf "%s\n" "$status_response" | grep -o '"status":"[^"]*"' | cut -d'"' -f4 || true)"
if [ "$status_value" != "pending" ]; then
  echo "Expected pending status before restart, got '$status_value'"
  echo "$status_response"
  exit 1
fi

echo "[4/7] Restarting the API..."
if [ "$RESTART_COMMAND" = "manual" ]; then
  echo "Restart the API service/container now, then press Enter to continue."
  read -r _
else
  sh -c "$RESTART_COMMAND"
fi

echo "[5/7] Waiting for readyz to recover..."
if ! wait_for_ready; then
  echo "readyz did not return 200 within ${READY_TIMEOUT_SECONDS}s after restart."
  exit 1
fi

echo "[6/7] Revealing the secret created before restart..."
consume_response="$(curl -s -X POST "$API_BASE_URL/api/secrets/$secret_id/consume" \
  -H "Content-Type: application/json" \
  -d '{}')"
returned_ciphertext="$(printf "%s\n" "$consume_response" | grep -o '"ciphertext":"[^"]*"' | cut -d'"' -f4 || true)"
if [ "$returned_ciphertext" != "$TEST_CIPHERTEXT" ]; then
  echo "Expected ciphertext '$TEST_CIPHERTEXT' after restart, got '$returned_ciphertext'"
  echo "$consume_response"
  exit 1
fi

echo "[7/7] Sending an oversized request through the public edge..."
oversized_ciphertext="$(head -c 40960 /dev/zero | tr '\0' 'A')"
oversized_body="$(printf '{"ciphertext":"%s","nonce":"%s","algorithm":"AES-GCM","ttlSeconds":3600}' "$oversized_ciphertext" "$TEST_NONCE")"
oversized_status="$(curl -s -o /dev/null -w "%{http_code}" \
  -X POST "$EDGE_API_URL/api/secrets" \
  -H "Content-Type: application/json" \
  --data-binary "$oversized_body")"
if [ "$oversized_status" != "413" ]; then
  echo "Expected oversized edge request to return 413, got $oversized_status"
  exit 1
fi

echo
echo "Production smoke test passed."
echo "The API survived a restart, preserved SECRET_ENCRYPTION_KEY continuity, and rejected oversized edge traffic."
