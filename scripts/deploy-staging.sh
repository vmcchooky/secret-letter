#!/usr/bin/env bash

set -euo pipefail

required_vars=(
  STAGING_SSH_HOST
  STAGING_SSH_USER
  STAGING_SSH_PRIVATE_KEY_FILE
  STAGING_DEPLOY_PATH
  STAGING_REMOTE_DEPLOY_COMMAND
  STAGING_RELEASE_VERSION
  STAGING_SOURCE_ARCHIVE
  STAGING_BACKEND_ARCHIVE
)

for name in "${required_vars[@]}"; do
  if [[ -z "${!name:-}" ]]; then
    echo "Missing required environment variable: $name" >&2
    exit 1
  fi
done

STAGING_SSH_PORT="${STAGING_SSH_PORT:-22}"
STAGING_REMOTE_POST_DEPLOY_COMMAND="${STAGING_REMOTE_POST_DEPLOY_COMMAND:-}"
STAGING_FRONTEND_ARCHIVE="${STAGING_FRONTEND_ARCHIVE:-}"
DRY_RUN="${DRY_RUN:-0}"

remote_release_dir="${STAGING_DEPLOY_PATH%/}/releases/${STAGING_RELEASE_VERSION}"
source_archive_basename="$(basename "$STAGING_SOURCE_ARCHIVE")"
backend_archive_basename="$(basename "$STAGING_BACKEND_ARCHIVE")"
frontend_archive_basename=""

if [[ -n "$STAGING_FRONTEND_ARCHIVE" ]]; then
  frontend_archive_basename="$(basename "$STAGING_FRONTEND_ARCHIVE")"
fi

ssh_opts=(
  -i "$STAGING_SSH_PRIVATE_KEY_FILE"
  -p "$STAGING_SSH_PORT"
  -o BatchMode=yes
)

if [[ -n "${STAGING_KNOWN_HOSTS_FILE:-}" && -f "${STAGING_KNOWN_HOSTS_FILE:-}" ]]; then
  ssh_opts+=(
    -o StrictHostKeyChecking=yes
    -o UserKnownHostsFile="$STAGING_KNOWN_HOSTS_FILE"
  )
else
  ssh_opts+=(
    -o StrictHostKeyChecking=accept-new
  )
fi

upload_files=(
  "$STAGING_SOURCE_ARCHIVE"
  "$STAGING_BACKEND_ARCHIVE"
)

if [[ -n "$STAGING_FRONTEND_ARCHIVE" ]]; then
  upload_files+=("$STAGING_FRONTEND_ARCHIVE")
fi

target="${STAGING_SSH_USER}@${STAGING_SSH_HOST}"
deploy_command_b64="$(printf '%s' "$STAGING_REMOTE_DEPLOY_COMMAND" | base64 | tr -d '\n')"
post_command_b64="$(printf '%s' "$STAGING_REMOTE_POST_DEPLOY_COMMAND" | base64 | tr -d '\n')"

if [[ "$DRY_RUN" == "1" ]]; then
  printf 'Would upload to %s:%s\n' "$target" "$remote_release_dir"
  printf 'Files:\n'
  printf '  %s\n' "${upload_files[@]}"
  printf 'Would run remote deploy command:\n%s\n' "$STAGING_REMOTE_DEPLOY_COMMAND"
  if [[ -n "$STAGING_REMOTE_POST_DEPLOY_COMMAND" ]]; then
    printf 'Would run remote post-deploy command:\n%s\n' "$STAGING_REMOTE_POST_DEPLOY_COMMAND"
  fi
  exit 0
fi

ssh "${ssh_opts[@]}" "$target" "mkdir -p '$remote_release_dir'"
scp "${ssh_opts[@]}" "${upload_files[@]}" "$target:$remote_release_dir/"

ssh "${ssh_opts[@]}" "$target" bash -s -- \
  "$remote_release_dir" \
  "$STAGING_RELEASE_VERSION" \
  "$source_archive_basename" \
  "$backend_archive_basename" \
  "$frontend_archive_basename" \
  "$deploy_command_b64" \
  "$post_command_b64" <<'REMOTE'
set -euo pipefail

remote_release_dir="$1"
release_version="$2"
source_archive_basename="$3"
backend_archive_basename="$4"
frontend_archive_basename="$5"
deploy_command_b64="$6"
post_command_b64="$7"

deploy_command="$(printf '%s' "$deploy_command_b64" | base64 -d)"
post_deploy_command="$(printf '%s' "$post_command_b64" | base64 -d)"

cd "$remote_release_dir"
rm -rf source backend-artifact frontend-artifact
mkdir -p source backend-artifact frontend-artifact

tar -xzf "$source_archive_basename" -C source
tar -xzf "$backend_archive_basename" -C backend-artifact

if [[ -n "$frontend_archive_basename" ]]; then
  tar -xzf "$frontend_archive_basename" -C frontend-artifact
fi

export RELEASE_VERSION="$release_version"
export RELEASE_DIR="$remote_release_dir"
export SOURCE_DIR="$remote_release_dir/source"
export BACKEND_ARTIFACT_DIR="$remote_release_dir/backend-artifact"
export FRONTEND_ARTIFACT_DIR="$remote_release_dir/frontend-artifact"

cd "$SOURCE_DIR"
bash -lc "$deploy_command"

if [[ -n "$post_deploy_command" ]]; then
  bash -lc "$post_deploy_command"
fi
REMOTE
