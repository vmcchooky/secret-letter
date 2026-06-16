#!/usr/bin/env bash

set -euo pipefail

EDGE_API_URL="${EDGE_API_URL:-https://api.secret.quorix.io.vn}"
LOG_SOURCE_COMMAND="${LOG_SOURCE_COMMAND:-docker compose -f deploy/prod/docker-compose.yml -f deploy/prod/docker-compose.vps-edge.yml logs api --since 10m --no-log-prefix}"
POLL_ATTEMPTS="${POLL_ATTEMPTS:-10}"
POLL_DELAY_SECONDS="${POLL_DELAY_SECONDS:-1}"
SPOOF_IP_ONE="${SPOOF_IP_ONE:-198.51.100.77}"
SPOOF_IP_TWO="${SPOOF_IP_TWO:-203.0.113.88}"

run_id="$(date +%s)"
normal_request_id="trusted-proxy-normal-${run_id}"
spoof_request_id_one="trusted-proxy-spoof-a-${run_id}"
spoof_request_id_two="trusted-proxy-spoof-b-${run_id}"

extract_ip_hash() {
  local request_id="$1"
  local attempt
  for attempt in $(seq 1 "$POLL_ATTEMPTS"); do
    local log_output
    log_output="$(sh -c "$LOG_SOURCE_COMMAND" 2>/dev/null || true)"
    local ip_hash
    ip_hash="$(printf "%s\n" "$log_output" | grep "$request_id" | tail -n 1 | sed -n 's/.*"ip_hash":"\([^"]*\)".*/\1/p' || true)"
    if [ -n "$ip_hash" ]; then
      printf "%s" "$ip_hash"
      return 0
    fi
    sleep "$POLL_DELAY_SECONDS"
  done
  return 1
}

request_status() {
  local request_id="$1"
  local spoofed_ip="${2:-}"

  if [ -n "$spoofed_ip" ]; then
    curl -s -o /dev/null -w "%{http_code}" \
      -H "X-Request-ID: $request_id" \
      -H "X-Forwarded-For: $spoofed_ip" \
      -H "User-Agent: trusted-proxy-check/1.0" \
      "$EDGE_API_URL/healthz"
    return
  fi

  curl -s -o /dev/null -w "%{http_code}" \
    -H "X-Request-ID: $request_id" \
    -H "User-Agent: trusted-proxy-check/1.0" \
    "$EDGE_API_URL/healthz"
}

echo "=== Trusted Proxy Verification ==="
echo "EDGE_API_URL: $EDGE_API_URL"
echo "LOG_SOURCE_COMMAND: $LOG_SOURCE_COMMAND"
echo

echo "[1/4] Sending baseline request through the public edge..."
normal_status="$(request_status "$normal_request_id")"
if [ "$normal_status" != "200" ]; then
  echo "Expected baseline request to return 200, got $normal_status"
  exit 1
fi

echo "[2/4] Sending spoofed forwarded-header request #1 through the public edge..."
spoof_status_one="$(request_status "$spoof_request_id_one" "$SPOOF_IP_ONE")"
if [ "$spoof_status_one" != "200" ]; then
  echo "Expected spoofed request #1 to return 200, got $spoof_status_one"
  exit 1
fi

echo "[3/4] Sending spoofed forwarded-header request #2 through the public edge..."
spoof_status_two="$(request_status "$spoof_request_id_two" "$SPOOF_IP_TWO")"
if [ "$spoof_status_two" != "200" ]; then
  echo "Expected spoofed request #2 to return 200, got $spoof_status_two"
  exit 1
fi

echo "[4/4] Comparing trusted client identity hashes from API logs..."
normal_hash="$(extract_ip_hash "$normal_request_id")" || {
  echo "Could not find baseline request in logs. Check LOG_SOURCE_COMMAND and API log access."
  exit 1
}
spoof_hash_one="$(extract_ip_hash "$spoof_request_id_one")" || {
  echo "Could not find spoofed request #1 in logs. Check LOG_SOURCE_COMMAND and API log access."
  exit 1
}
spoof_hash_two="$(extract_ip_hash "$spoof_request_id_two")" || {
  echo "Could not find spoofed request #2 in logs. Check LOG_SOURCE_COMMAND and API log access."
  exit 1
}

echo "Baseline ip_hash: $normal_hash"
echo "Spoof #1 ip_hash: $spoof_hash_one"
echo "Spoof #2 ip_hash: $spoof_hash_two"

if [ "$normal_hash" != "$spoof_hash_one" ] || [ "$normal_hash" != "$spoof_hash_two" ]; then
  echo "Trusted proxy verification failed."
  echo "The API observed different client identities when spoofed X-Forwarded-For headers were supplied."
  echo "Check the edge proxy header sanitization and TRUSTED_PROXY_CIDRS."
  exit 1
fi

echo
echo "Trusted proxy verification passed."
echo "Spoofed X-Forwarded-For headers did not change the client identity seen by the API."
