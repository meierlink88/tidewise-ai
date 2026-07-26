#!/usr/bin/env bash

set -euo pipefail

release_root="${1:?release root is required}"
control_root="${2:?control-plane root is required}"
bundle_root="${3:?bundle output root is required}"
release_sha="${4:?release SHA is required}"
control_plane_sha="${5:?control-plane SHA is required}"

if ! [[ "$release_sha" =~ ^[0-9a-fA-F]{40}$ ]]; then
  echo "release SHA must be a full 40-character Git commit SHA" >&2
  exit 1
fi
if ! [[ "$control_plane_sha" =~ ^[0-9a-fA-F]{40}$ ]]; then
  echo "control-plane SHA must be a full 40-character Git commit SHA" >&2
  exit 1
fi
release_sha="$(printf '%s' "$release_sha" | tr 'A-F' 'a-f')"
control_plane_sha="$(printf '%s' "$control_plane_sha" | tr 'A-F' 'a-f')"
if [ -e "$bundle_root" ] && [ -n "$(find "$bundle_root" -mindepth 1 -maxdepth 1 -print -quit)" ]; then
  echo "bundle output root must be empty: $bundle_root" >&2
  exit 1
fi

install_bundle_file() {
  local source_root="$1"
  local relative_path="$2"
  local destination_root="$3"
  local source="${source_root}/${relative_path}"
  local destination="${destination_root}/${relative_path}"
  local mode=0644

  if [ ! -f "$source" ]; then
    echo "required deployment bundle file is missing: $source" >&2
    exit 1
  fi
  if [[ "$relative_path" == *.sh ]]; then
    mode=0755
  fi
  install -d -m 0755 "$(dirname "$destination")"
  install -m "$mode" "$source" "$destination"
}

install -d -m 0755 "$bundle_root/release" "$bundle_root/control"

for path in \
  infra/uat/docker-compose.yaml \
  analyse-data-service/backend/configs/config.uat.yaml \
  agent-run/backend/configs/config.uat.yaml
do
  install_bundle_file "$release_root" "$path" "$bundle_root/release"
done

for path in \
  infra/uat/preflight.sh \
  infra/uat/deploy.sh \
  infra/uat/collect-diagnostics.sh \
  infra/uat/migration-risk.tsv \
  infra/uat/agentrun-migration-risk.tsv
do
  install_bundle_file "$control_root" "$path" "$bundle_root/control"
done

printf '%s\n' \
  "RELEASE_SHA=${release_sha}" \
  "CONTROL_PLANE_SHA=${control_plane_sha}" > "$bundle_root/metadata.env"

(
  cd "$bundle_root"
  find metadata.env release control -type f -print0 \
    | LC_ALL=C sort -z \
    | xargs -0 sha256sum
) > "$bundle_root/SHA256SUMS"
