#!/usr/bin/env bash
set -euo pipefail

# Stop local development services for one-time-link.
# Stops frontend/backend listeners on the default dev ports, then stops local Redis.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$REPO_ROOT/deploy/local/docker-compose.yml"

if [ "$#" -eq 0 ]; then
  PORTS=(5173 8080)
else
  PORTS=("$@")
fi

echo "Stopping one-time-link local development services..."

for port in "${PORTS[@]}"; do
  pids=""

  if command -v lsof >/dev/null 2>&1; then
    pids="$(lsof -ti TCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
  elif command -v fuser >/dev/null 2>&1; then
    pids="$(fuser "$port"/tcp 2>/dev/null || true)"
  else
    echo "Port $port: install lsof or fuser to stop listeners automatically."
    continue
  fi

  if [ -z "$pids" ]; then
    echo "Port $port: no listener found."
    continue
  fi

  for pid in $pids; do
    kill "$pid" 2>/dev/null || true
    echo "Port $port: stopped PID $pid."
  done
done

if [ -f "$COMPOSE_FILE" ]; then
  echo "Redis: stopping Docker Compose service..."
  docker compose -f "$COMPOSE_FILE" down
else
  echo "Redis: compose file not found at $COMPOSE_FILE."
fi

echo "Done."
