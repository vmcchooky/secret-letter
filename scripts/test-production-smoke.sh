#!/usr/bin/env bash

set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:18080}"
EDGE_API_URL="${EDGE_API_URL:-https://api.secret.quorix.io.vn}"
READY_TIMEOUT_SECONDS="${READY_TIMEOUT_SECONDS:-60}"
RESTART_COMMAND="${RESTART_COMMAND:-docker compose -f deploy/prod/docker-compose.yml -f deploy/prod/docker-compose.vps-edge.yml restart api}"
TEST_CIPHERTEXT="${TEST_CIPHERTEXT:-cHJvZHVjdGlvbi1zbW9rZS10ZXN0LWNpcGhlcnRleHQ}"
TEST_NONCE="${TEST_NONCE:-MTIzNDU2Nzg5MDEy}"

extract_json_field() {
  local field_name="$1"
  local payload="${2:-}"
  python3 -c '
import json
import sys

field_name = sys.argv[1]
payload = json.load(sys.stdin)
value = payload
for part in field_name.split("."):
    if isinstance(value, dict):
        value = value.get(part)
    else:
        value = None
        break

if value is None:
    sys.exit(1)

print(value)
' "$field_name" <<<"$payload"
}

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

echo "[1/8] Checking local health and readiness endpoints..."
health_status="$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE_URL/healthz")"
ready_status="$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE_URL/readyz")"
if [ "$health_status" != "200" ] || [ "$ready_status" != "200" ]; then
  echo "Expected healthz=200 and readyz=200 before the test, got healthz=$health_status readyz=$ready_status"
  exit 1
fi

echo "[2/8] Creating a secret before restart..."
create_response="$(curl -s -X POST "$API_BASE_URL/api/secrets" \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: production-smoke-create-$(date +%s)" \
  -d "{
    \"ciphertext\": \"$TEST_CIPHERTEXT\",
    \"nonce\": \"$TEST_NONCE\",
    \"algorithm\": \"AES-GCM\",
    \"ttlSeconds\": 3600
  }")"

secret_id="$(extract_json_field secretId "$create_response" || true)"
if [ -z "$secret_id" ]; then
  echo "Failed to create a secret. Response:"
  echo "$create_response"
  exit 1
fi
echo "Created secret: $secret_id"

echo "[3/8] Verifying status before restart and checking rate-limit headers..."
status_headers_file="$(mktemp)"
status_response="$(curl -s -D "$status_headers_file" "$API_BASE_URL/api/secrets/$secret_id/status")"
status_value="$(extract_json_field status "$status_response" || true)"
if [ "$status_value" != "active" ]; then
  echo "Expected active status before restart, got '$status_value'"
  echo "$status_response"
  rm -f "$status_headers_file"
  exit 1
fi
for header_name in X-RateLimit-Limit X-RateLimit-Remaining X-RateLimit-Reset; do
  if ! grep -qi "^${header_name}:" "$status_headers_file"; then
    echo "Expected $header_name header to be present on status response."
    cat "$status_headers_file"
    rm -f "$status_headers_file"
    exit 1
  fi
done
rm -f "$status_headers_file"

echo "[4/8] Restarting the API..."
if [ "$RESTART_COMMAND" = "manual" ]; then
  echo "Restart the API service/container now, then press Enter to continue."
  read -r _
else
  sh -c "$RESTART_COMMAND"
fi

echo "[5/8] Waiting for readyz to recover..."
if ! wait_for_ready; then
  echo "readyz did not return 200 within ${READY_TIMEOUT_SECONDS}s after restart."
  exit 1
fi

echo "[6/8] Revealing the secret created before restart..."
consume_response="$(curl -s -X POST "$API_BASE_URL/api/secrets/$secret_id/consume" \
  -H "Content-Type: application/json" \
  -d '{}')"
returned_ciphertext="$(printf '%s' "$consume_response" | extract_json_field ciphertext || true)"
if [ "$returned_ciphertext" != "$TEST_CIPHERTEXT" ]; then
  echo "Expected ciphertext '$TEST_CIPHERTEXT' after restart, got '$returned_ciphertext'"
  echo "$consume_response"
  exit 1
fi

echo "[7/8] Verifying the same secret cannot be consumed twice..."
second_consume_body_file="$(mktemp)"
second_consume_status="$(curl -s -o "$second_consume_body_file" -w "%{http_code}" \
  -X POST "$API_BASE_URL/api/secrets/$secret_id/consume" \
  -H "Content-Type: application/json" \
  -d '{}')"
if [ "$second_consume_status" != "410" ]; then
  echo "Expected second consume to return 410, got $second_consume_status"
  cat "$second_consume_body_file"
  rm -f "$second_consume_body_file"
  exit 1
fi
second_consume_error="$(extract_json_field error "$(cat "$second_consume_body_file")" || true)"
if [ "$second_consume_error" != "SECRET_CONSUMED" ]; then
  echo "Expected second consume error to be SECRET_CONSUMED, got '$second_consume_error'"
  cat "$second_consume_body_file"
  rm -f "$second_consume_body_file"
  exit 1
fi
rm -f "$second_consume_body_file"

echo "[8/8] Sending an oversized request through the public edge..."
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
echo "The API preserved SECRET_ENCRYPTION_KEY continuity, emitted rate-limit headers, rejected double consume, and blocked oversized edge traffic."
