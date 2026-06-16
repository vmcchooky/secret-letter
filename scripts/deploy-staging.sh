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
  STAGING_RELEASE_MANIFEST
  STAGING_RELEASE_CHECKSUMS
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
STAGING_KEEP_RELEASES="${STAGING_KEEP_RELEASES:-5}"
DRY_RUN="${DRY_RUN:-0}"

remote_root="${STAGING_DEPLOY_PATH%/}"
remote_release_dir="${remote_root}/releases/${STAGING_RELEASE_VERSION}"
remote_current_link="${remote_root}/current"
remote_previous_link="${remote_root}/previous"
source_archive_basename="$(basename "$STAGING_SOURCE_ARCHIVE")"
backend_archive_basename="$(basename "$STAGING_BACKEND_ARCHIVE")"
frontend_archive_basename=""
release_manifest_basename="$(basename "$STAGING_RELEASE_MANIFEST")"
release_checksums_basename="$(basename "$STAGING_RELEASE_CHECKSUMS")"

if [[ -n "$STAGING_FRONTEND_ARCHIVE" ]]; then
  frontend_archive_basename="$(basename "$STAGING_FRONTEND_ARCHIVE")"
fi

require_command() {
  local command_name="$1"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "Missing required command: $command_name" >&2
    exit 1
  fi
}

require_file() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    echo "Missing required file: $path" >&2
    exit 1
  fi
}

validate_release_version() {
  local version="$1"
  if [[ ! "$version" =~ ^[A-Za-z0-9._-]+$ ]]; then
    echo "Unsafe STAGING_RELEASE_VERSION: $version" >&2
    exit 1
  fi
}

validate_keep_releases() {
  local keep_value="$1"
  if [[ ! "$keep_value" =~ ^[0-9]+$ ]]; then
    echo "STAGING_KEEP_RELEASES must be a positive integer" >&2
    exit 1
  fi
  if (( keep_value < 2 )); then
    echo "STAGING_KEEP_RELEASES must be at least 2" >&2
    exit 1
  fi
}

validate_release_manifest() {
  local manifest_path="$1"
  local release_version="$2"
  local source_name="$3"
  local backend_name="$4"
  local frontend_name="$5"

  python3 - "$manifest_path" "$release_version" "$source_name" "$backend_name" "$frontend_name" <<'PY'
import json
import pathlib
import sys

manifest_path, release_version, source_name, backend_name, frontend_name = sys.argv[1:6]

with open(manifest_path, encoding="utf-8") as handle:
    manifest = json.load(handle)

if manifest.get("version") != release_version:
    raise SystemExit(
        f"Release manifest version mismatch: expected {release_version}, got {manifest.get('version')}"
    )

artifacts = manifest.get("artifacts")
if not isinstance(artifacts, list) or not artifacts:
    raise SystemExit("Release manifest must contain a non-empty artifacts array")

expected_names = {source_name, backend_name}
if frontend_name:
    expected_names.add(frontend_name)

artifact_map = {}
for item in artifacts:
    if not isinstance(item, dict):
        raise SystemExit("Release manifest artifacts must be objects")
    name = item.get("name")
    if not isinstance(name, str) or not name:
        raise SystemExit("Release manifest artifact entries require a name")
    artifact_map[name] = item

missing = sorted(expected_names - set(artifact_map))
if missing:
    raise SystemExit(f"Release manifest missing expected artifacts: {', '.join(missing)}")

for name in expected_names:
    entry = artifact_map[name]
    sha256_value = entry.get("sha256")
    size_bytes = entry.get("size_bytes")
    if not isinstance(sha256_value, str) or len(sha256_value) != 64:
        raise SystemExit(f"Release manifest has invalid sha256 for {name}")
    if not isinstance(size_bytes, int) or size_bytes <= 0:
        raise SystemExit(f"Release manifest has invalid size_bytes for {name}")

print(
    f"Validated release manifest {pathlib.Path(manifest_path).name} for {release_version}",
    file=sys.stderr,
)
PY
}

validate_tar_archive() {
  local archive_path="$1"

  python3 - "$archive_path" <<'PY'
import pathlib
import sys
import tarfile

archive_path = sys.argv[1]

with tarfile.open(archive_path, "r:*") as archive:
    for member in archive.getmembers():
        member_path = pathlib.PurePosixPath(member.name)
        if member_path.is_absolute():
            raise SystemExit(f"Unsafe absolute archive entry in {archive_path}: {member.name}")
        if ".." in member_path.parts:
            raise SystemExit(f"Unsafe parent traversal archive entry in {archive_path}: {member.name}")

print(f"Validated archive layout for {archive_path}", file=sys.stderr)
PY
}

verify_checksum_bundle() {
  local checksums_file="$1"
  shift

  local temp_dir
  temp_dir="$(mktemp -d)"

  cp "$checksums_file" "$temp_dir/$(basename "$checksums_file")"

  local file_path
  for file_path in "$@"; do
    cp "$file_path" "$temp_dir/$(basename "$file_path")"
  done

  local status=0
  (
    cd "$temp_dir"
    sha256sum -c "$(basename "$checksums_file")"
  ) || status=$?

  rm -rf "$temp_dir"
  return "$status"
}

require_command base64
require_command python3
require_command sha256sum
require_command tar
validate_release_version "$STAGING_RELEASE_VERSION"
validate_keep_releases "$STAGING_KEEP_RELEASES"

local_required_files=(
  "$STAGING_SSH_PRIVATE_KEY_FILE"
  "$STAGING_SOURCE_ARCHIVE"
  "$STAGING_BACKEND_ARCHIVE"
  "$STAGING_RELEASE_MANIFEST"
  "$STAGING_RELEASE_CHECKSUMS"
)

if [[ -n "$STAGING_FRONTEND_ARCHIVE" ]]; then
  local_required_files+=("$STAGING_FRONTEND_ARCHIVE")
fi

for file_path in "${local_required_files[@]}"; do
  require_file "$file_path"
done

validation_files=(
  "$STAGING_SOURCE_ARCHIVE"
  "$STAGING_BACKEND_ARCHIVE"
  "$STAGING_RELEASE_MANIFEST"
)

if [[ -n "$STAGING_FRONTEND_ARCHIVE" ]]; then
  validation_files+=("$STAGING_FRONTEND_ARCHIVE")
fi

validate_release_manifest \
  "$STAGING_RELEASE_MANIFEST" \
  "$STAGING_RELEASE_VERSION" \
  "$source_archive_basename" \
  "$backend_archive_basename" \
  "$frontend_archive_basename"
verify_checksum_bundle "$STAGING_RELEASE_CHECKSUMS" "${validation_files[@]}"
validate_tar_archive "$STAGING_SOURCE_ARCHIVE"
validate_tar_archive "$STAGING_BACKEND_ARCHIVE"

if [[ -n "$STAGING_FRONTEND_ARCHIVE" ]]; then
  validate_tar_archive "$STAGING_FRONTEND_ARCHIVE"
fi

if [[ "$DRY_RUN" != "1" ]]; then
  require_command ssh
  require_command scp
fi

ssh_opts=(
  -i "$STAGING_SSH_PRIVATE_KEY_FILE"
  -p "$STAGING_SSH_PORT"
  -o BatchMode=yes
)

scp_opts=(
  -i "$STAGING_SSH_PRIVATE_KEY_FILE"
  -P "$STAGING_SSH_PORT"
  -o BatchMode=yes
)

if [[ -n "${STAGING_KNOWN_HOSTS_FILE:-}" && -f "${STAGING_KNOWN_HOSTS_FILE:-}" ]]; then
  ssh_opts+=(
    -o StrictHostKeyChecking=yes
    -o UserKnownHostsFile="$STAGING_KNOWN_HOSTS_FILE"
  )
  scp_opts+=(
    -o StrictHostKeyChecking=yes
    -o UserKnownHostsFile="$STAGING_KNOWN_HOSTS_FILE"
  )
else
  ssh_opts+=(
    -o StrictHostKeyChecking=accept-new
  )
  scp_opts+=(
    -o StrictHostKeyChecking=accept-new
  )
fi

upload_files=(
  "$STAGING_SOURCE_ARCHIVE"
  "$STAGING_BACKEND_ARCHIVE"
  "$STAGING_RELEASE_MANIFEST"
  "$STAGING_RELEASE_CHECKSUMS"
)

if [[ -n "$STAGING_FRONTEND_ARCHIVE" ]]; then
  upload_files+=("$STAGING_FRONTEND_ARCHIVE")
fi

target="${STAGING_SSH_USER}@${STAGING_SSH_HOST}"
deploy_command_b64="$(printf '%s' "$STAGING_REMOTE_DEPLOY_COMMAND" | base64 | tr -d '\n')"
post_command_b64="$(printf '%s' "$STAGING_REMOTE_POST_DEPLOY_COMMAND" | base64 | tr -d '\n')"

if [[ "$DRY_RUN" == "1" ]]; then
  printf 'Preflight verification passed for release %s\n' "$STAGING_RELEASE_VERSION"
  printf 'Would upload to %s:%s\n' "$target" "$remote_release_dir"
  printf 'Files:\n'
  printf '  %s\n' "${upload_files[@]}"
  printf 'Would verify manifest %s and checksums %s again on the remote host.\n' \
    "$release_manifest_basename" \
    "$release_checksums_basename"
  printf 'Would manage release symlinks:\n'
  printf '  current -> %s\n' "$remote_current_link"
  printf '  previous -> %s\n' "$remote_previous_link"
  printf 'Would keep the newest %s staged releases.\n' "$STAGING_KEEP_RELEASES"
  printf 'Would run remote deploy command:\n%s\n' "$STAGING_REMOTE_DEPLOY_COMMAND"
  if [[ -n "$STAGING_REMOTE_POST_DEPLOY_COMMAND" ]]; then
    printf 'Would run remote post-deploy command:\n%s\n' "$STAGING_REMOTE_POST_DEPLOY_COMMAND"
  fi
  exit 0
fi

ssh "${ssh_opts[@]}" "$target" "mkdir -p '$remote_root/releases' '$remote_release_dir'"
scp "${scp_opts[@]}" "${upload_files[@]}" "$target:$remote_release_dir/"

ssh "${ssh_opts[@]}" "$target" bash -s -- \
  "$remote_root" \
  "$STAGING_RELEASE_VERSION" \
  "$source_archive_basename" \
  "$backend_archive_basename" \
  "$frontend_archive_basename" \
  "$release_manifest_basename" \
  "$release_checksums_basename" \
  "$deploy_command_b64" \
  "$post_command_b64" \
  "$STAGING_KEEP_RELEASES" <<'REMOTE'
set -euo pipefail

remote_root="${1%/}"
release_version="$2"
source_archive_basename="$3"
backend_archive_basename="$4"
frontend_archive_basename="$5"
release_manifest_basename="$6"
release_checksums_basename="$7"
deploy_command_b64="$8"
post_command_b64="$9"
keep_releases="${10}"

remote_release_dir="${remote_root}/releases/${release_version}"
remote_current_link="${remote_root}/current"
remote_previous_link="${remote_root}/previous"

require_command() {
  local command_name="$1"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "Missing required command on remote host: $command_name" >&2
    exit 1
  fi
}

validate_keep_releases() {
  local keep_value="$1"
  if [[ ! "$keep_value" =~ ^[0-9]+$ ]]; then
    echo "STAGING_KEEP_RELEASES must be a positive integer" >&2
    exit 1
  fi
  if (( keep_value < 2 )); then
    echo "STAGING_KEEP_RELEASES must be at least 2" >&2
    exit 1
  fi
}

validate_release_manifest() {
  local manifest_path="$1"
  local expected_version="$2"
  local source_name="$3"
  local backend_name="$4"
  local frontend_name="$5"

  python3 - "$manifest_path" "$expected_version" "$source_name" "$backend_name" "$frontend_name" <<'PY'
import json
import pathlib
import sys

manifest_path, expected_version, source_name, backend_name, frontend_name = sys.argv[1:6]

with open(manifest_path, encoding="utf-8") as handle:
    manifest = json.load(handle)

if manifest.get("version") != expected_version:
    raise SystemExit(
        f"Release manifest version mismatch: expected {expected_version}, got {manifest.get('version')}"
    )

artifacts = manifest.get("artifacts")
if not isinstance(artifacts, list) or not artifacts:
    raise SystemExit("Release manifest must contain a non-empty artifacts array")

expected_names = {source_name, backend_name}
if frontend_name:
    expected_names.add(frontend_name)

artifact_map = {}
for item in artifacts:
    if not isinstance(item, dict):
        raise SystemExit("Release manifest artifacts must be objects")
    name = item.get("name")
    if not isinstance(name, str) or not name:
        raise SystemExit("Release manifest artifact entries require a name")
    artifact_map[name] = item

missing = sorted(expected_names - set(artifact_map))
if missing:
    raise SystemExit(f"Release manifest missing expected artifacts: {', '.join(missing)}")

for name in expected_names:
    entry = artifact_map[name]
    sha256_value = entry.get("sha256")
    size_bytes = entry.get("size_bytes")
    if not isinstance(sha256_value, str) or len(sha256_value) != 64:
        raise SystemExit(f"Release manifest has invalid sha256 for {name}")
    if not isinstance(size_bytes, int) or size_bytes <= 0:
        raise SystemExit(f"Release manifest has invalid size_bytes for {name}")

print(
    f"Validated release manifest {pathlib.Path(manifest_path).name} for {expected_version}",
    file=sys.stderr,
)
PY
}

validate_tar_archive() {
  local archive_path="$1"

  python3 - "$archive_path" <<'PY'
import pathlib
import sys
import tarfile

archive_path = sys.argv[1]

with tarfile.open(archive_path, "r:*") as archive:
    for member in archive.getmembers():
        member_path = pathlib.PurePosixPath(member.name)
        if member_path.is_absolute():
            raise SystemExit(f"Unsafe absolute archive entry in {archive_path}: {member.name}")
        if ".." in member_path.parts:
            raise SystemExit(f"Unsafe parent traversal archive entry in {archive_path}: {member.name}")

print(f"Validated archive layout for {archive_path}", file=sys.stderr)
PY
}

export_release_env() {
  local release_dir="$1"
  local active_version="$2"

  export RELEASE_ROOT="$remote_root"
  export RELEASE_VERSION="$active_version"
  export RELEASE_DIR="$release_dir"
  export SOURCE_DIR="$release_dir/source"
  export BACKEND_ARTIFACT_DIR="$release_dir/backend-artifact"
  export FRONTEND_ARTIFACT_DIR="$release_dir/frontend-artifact"
  export CURRENT_RELEASE_LINK="$remote_current_link"
  export PREVIOUS_RELEASE_LINK="$remote_previous_link"
}

run_release_flow() {
  local release_dir="$1"
  local active_version="$2"
  local deploy_command="$3"
  local post_deploy_command="$4"
  local phase_label="$5"

  export_release_env "$release_dir" "$active_version"
  cd "$SOURCE_DIR"

  echo "Running ${phase_label} deploy command for ${active_version}..."
  bash -lc "$deploy_command"

  if [[ -n "$post_deploy_command" ]]; then
    echo "Running ${phase_label} post-deploy command for ${active_version}..."
    bash -lc "$post_deploy_command"
  fi
}

rollback_previous_release() {
  if [[ -z "$previous_release_dir" || ! -d "$previous_release_dir" ]]; then
    echo "No previous release is available for automatic rollback." >&2
    return 1
  fi

  if [[ ! -f "$previous_release_dir/.deploy-command.sh" ]]; then
    echo "Previous release is missing stored deploy metadata: $previous_release_dir" >&2
    return 1
  fi

  local rollback_deploy_command
  local rollback_post_command=""
  local rollback_version

  rollback_version="$(basename "$previous_release_dir")"
  rollback_deploy_command="$(< "$previous_release_dir/.deploy-command.sh")"

  if [[ -f "$previous_release_dir/.post-deploy-command.sh" ]]; then
    rollback_post_command="$(< "$previous_release_dir/.post-deploy-command.sh")"
  fi

  echo "Attempting automatic rollback to ${rollback_version}..."
  run_release_flow \
    "$previous_release_dir" \
    "$rollback_version" \
    "$rollback_deploy_command" \
    "$rollback_post_command" \
    "rollback"
}

prune_old_releases() {
  local keep_limit="$1"
  local current_target
  local previous_target
  local release_dir
  local kept_count=0

  current_target="$(readlink -f "$remote_current_link" 2>/dev/null || true)"
  previous_target="$(readlink -f "$remote_previous_link" 2>/dev/null || true)"

  mapfile -t release_dirs < <(find "$remote_root/releases" -mindepth 1 -maxdepth 1 -type d -printf '%T@ %p\n' | sort -nr | awk '{print $2}')

  for release_dir in "${release_dirs[@]}"; do
    if [[ "$release_dir" == "$current_target" || "$release_dir" == "$previous_target" ]]; then
      kept_count=$((kept_count + 1))
      continue
    fi

    if (( kept_count < keep_limit )); then
      kept_count=$((kept_count + 1))
      continue
    fi

    rm -rf "$release_dir"
  done
}

require_command base64
require_command bash
require_command python3
require_command sha256sum
require_command tar
validate_keep_releases "$keep_releases"

deploy_command="$(printf '%s' "$deploy_command_b64" | base64 -d)"
post_deploy_command="$(printf '%s' "$post_command_b64" | base64 -d)"

cd "$remote_release_dir"

validate_release_manifest \
  "$release_manifest_basename" \
  "$release_version" \
  "$source_archive_basename" \
  "$backend_archive_basename" \
  "$frontend_archive_basename"
sha256sum -c "$release_checksums_basename"
validate_tar_archive "$source_archive_basename"
validate_tar_archive "$backend_archive_basename"

if [[ -n "$frontend_archive_basename" ]]; then
  validate_tar_archive "$frontend_archive_basename"
fi

rm -rf source backend-artifact frontend-artifact
mkdir -p source backend-artifact frontend-artifact

tar -xzf "$source_archive_basename" -C source
tar -xzf "$backend_archive_basename" -C backend-artifact

if [[ -n "$frontend_archive_basename" ]]; then
  tar -xzf "$frontend_archive_basename" -C frontend-artifact
fi

printf '%s\n' "$deploy_command" > "$remote_release_dir/.deploy-command.sh"
chmod 600 "$remote_release_dir/.deploy-command.sh"

if [[ -n "$post_deploy_command" ]]; then
  printf '%s\n' "$post_deploy_command" > "$remote_release_dir/.post-deploy-command.sh"
  chmod 600 "$remote_release_dir/.post-deploy-command.sh"
else
  rm -f "$remote_release_dir/.post-deploy-command.sh"
fi

previous_release_dir="$(readlink -f "$remote_current_link" 2>/dev/null || true)"
if [[ "$previous_release_dir" == "$remote_release_dir" ]]; then
  previous_release_dir=""
fi

if ! run_release_flow \
  "$remote_release_dir" \
  "$release_version" \
  "$deploy_command" \
  "$post_deploy_command" \
  "release"; then
  echo "Release deployment failed for ${release_version}." >&2

  if rollback_previous_release; then
    echo "Rollback completed successfully." >&2
  else
    echo "Automatic rollback was not successful." >&2
  fi

  exit 1
fi

if [[ -n "$previous_release_dir" && -d "$previous_release_dir" ]]; then
  ln -sfn "$previous_release_dir" "$remote_previous_link"
else
  rm -f "$remote_previous_link"
fi

ln -sfn "$remote_release_dir" "$remote_current_link"
date -u '+%Y-%m-%dT%H:%M:%SZ' > "$remote_release_dir/.deployed-at"
prune_old_releases "$keep_releases"

echo "Release ${release_version} is now active."
echo "Current release link: ${remote_current_link}"
if [[ -L "$remote_previous_link" ]]; then
  echo "Previous release link: ${remote_previous_link}"
fi
REMOTE
