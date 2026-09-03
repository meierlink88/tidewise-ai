#!/usr/bin/env bash
set -euo pipefail

: "${BASE_SHA:?BASE_SHA is required}"
: "${HEAD_SHA:?HEAD_SHA is required}"
: "${GITHUB_OUTPUT:?GITHUB_OUTPUT is required}"

scope="${1:-}"
case "$scope" in
  data)
    pattern='^(data-service/|go\.mod$|go\.sum$|AGENTS\.md$|CONTEXT-MAP\.md$|docs/(agents/|contexts/data/)|infra/(local|uat|uat-infra)/|scripts/ci/|\.github/workflows/(ci|deploy-uat|deploy-uat-infra|recover-uat-pre-v74-evidence|replace-uat-public-schema)\.yml$)'
    ;;
  miniapp)
    pattern='^(miniapp/|data-service/backend/api/|go\.mod$|go\.sum$|package\.json$|package-lock\.json$|AGENTS\.md$|CONTEXT-MAP\.md$|docs/(agents/|contexts/miniapp/)|infra/(local|uat)/|scripts/ci/|\.github/workflows/(ci|deploy-uat)\.yml$)'
    ;;
  adminportal)
    pattern='^(admin-portal/|data-service/backend/api/|go\.mod$|go\.sum$|package\.json$|package-lock\.json$|AGENTS\.md$|CONTEXT-MAP\.md$|docs/(agents/|contexts/adminportal/)|infra/(local|uat)/|scripts/ci/|\.github/workflows/(ci|deploy-uat)\.yml$)'
    ;;
  *)
    echo "unsupported application scope: ${scope:-<empty>}" >&2
    exit 2
    ;;
esac

base="$BASE_SHA"
if [[ -z "$base" || "$base" =~ ^0+$ ]]; then
  base="$(git rev-list --max-parents=0 "$HEAD_SHA" | tail -n 1)"
fi

if git diff --name-only "$base" "$HEAD_SHA" | grep -Eq "$pattern"; then
  echo "changed=true" >>"$GITHUB_OUTPUT"
else
  echo "changed=false" >>"$GITHUB_OUTPUT"
fi
