#!/usr/bin/env bash

set -euo pipefail

release_root="${1:?release root is required}"
control_root="${2:?control-plane root is required}"
bundle_root="${3:?bundle output root is required}"
release_sha="${4:?release SHA is required}"
control_plane_sha="${5:?control-plane SHA is required}"
bundle_manifest="${control_root}/infra/uat/deploy-bundle-files.txt"

if ! [[ "$release_sha" =~ ^[0-9a-fA-F]{40}$ ]]; then
  echo "release SHA must be a full 40-character Git commit SHA" >&2
  exit 1
fi
if ! [[ "$control_plane_sha" =~ ^[0-9a-fA-F]{40}$ ]]; then
  echo "control-plane SHA must be a full 40-character Git commit SHA" >&2
  exit 1
fi
if [ ! -f "$bundle_manifest" ]; then
  echo "deployment bundle manifest is missing: $bundle_manifest" >&2
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

while IFS=$'\t' read -r scope path extra; do
  if [ -z "$scope" ] && [ -z "$path" ]; then
    continue
  fi
  if [ -n "${extra:-}" ] || ! [[ "$path" =~ ^[A-Za-z0-9._/-]+$ ]] || [[ "/$path/" == *"/../"* ]]; then
    echo "invalid deployment bundle manifest row: ${scope} ${path} ${extra:-}" >&2
    exit 1
  fi
  case "$scope" in
    release) install_bundle_file "$release_root" "$path" "$bundle_root/release" ;;
    control) install_bundle_file "$control_root" "$path" "$bundle_root/control" ;;
    *)
      echo "invalid deployment bundle scope: $scope" >&2
      exit 1
      ;;
  esac
done < "$bundle_manifest"

printf '%s\n' \
  "RELEASE_SHA=${release_sha}" \
  "CONTROL_PLANE_SHA=${control_plane_sha}" > "$bundle_root/metadata.env"
install -m 0644 "$bundle_manifest" "$bundle_root/files.manifest"

(
  cd "$bundle_root"
  find metadata.env files.manifest release control -type f -print0 \
    | LC_ALL=C sort -z \
    | xargs -0 sha256sum
) > "$bundle_root/SHA256SUMS"
