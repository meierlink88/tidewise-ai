#!/usr/bin/env bash
set -euo pipefail

: "${BASE_SHA:?BASE_SHA is required}"
: "${HEAD_SHA:?HEAD_SHA is required}"
: "${GITHUB_OUTPUT:?GITHUB_OUTPUT is required}"

scope="${1:-}"
case "$scope" in
  data | miniapp | adminportal | agentrun | repository) ;;
  *)
    echo "unsupported test-risk scope: ${scope:-<empty>}" >&2
    exit 2
    ;;
esac

base="$BASE_SHA"
if [[ -z "$base" || "$base" =~ ^0+$ ]]; then
  base="$(git rev-list --max-parents=0 "$HEAD_SHA" | tail -n 1)"
fi
changed_paths="$(git diff --name-only "$base" "$HEAD_SHA")"

matches() {
  grep -Eq "$1" <<<"$changed_paths"
}

default=false
frontend=false
data=false
migration=false
conf_lifecycle=false
provider_consumer=false
container=false
architecture=false

shared_go='^(go\.mod|go\.sum)$'
shared_frontend='^(package\.json|package-lock\.json)$'
container_assets='(^|/)(Dockerfile|docker-compose[^/]*\.ya?ml)$|^infra/(local|uat)/|^\.github/workflows/(ci|deploy-uat)\.yml$'

case "$scope" in
  data)
    if matches '^analyse-data-service/backend/(api/|cmd/|configs/|internal/|migrations/)' || matches "$shared_go"; then
      default=true
    fi
    if matches '^analyse-data-service/backend/(internal/data/|migrations/)' || matches "$shared_go"; then
      data=true
    fi
    if matches '^analyse-data-service/backend/(migrations/|internal/data/dbmigration/)' || matches "$shared_go"; then
      migration=true
    fi
    if matches '^analyse-data-service/backend/(cmd/server/|configs/|internal/conf/|internal/server/)' || matches "$shared_go"; then
      conf_lifecycle=true
    fi
    if matches '^analyse-data-service/backend/api/|^testdata/' || matches "$shared_go"; then
      provider_consumer=true
    fi
    if matches '^analyse-data-service/backend/Dockerfile$' || matches "$container_assets"; then
      container=true
    fi
    ;;
  miniapp)
    if matches '^miniapp/backend/(api/|cmd/|configs/|internal/)' || matches "$shared_go"; then
      default=true
    fi
    if matches '^miniapp/frontend/' || matches "$shared_frontend"; then
      frontend=true
    fi
    if matches '^miniapp/backend/internal/data/' || matches "$shared_go"; then
      data=true
    fi
    if matches '^miniapp/backend/(api/|internal/data/)|^analyse-data-service/backend/api/|^testdata/' || matches "$shared_go"; then
      provider_consumer=true
    fi
    if matches '^miniapp/backend/(cmd/server/|configs/|internal/conf/|internal/server/)' || matches "$shared_go"; then
      conf_lifecycle=true
    fi
    if matches '^miniapp/(backend|frontend)/Dockerfile$' || matches "$container_assets"; then
      container=true
    fi
    ;;
  adminportal)
    if matches '^admin-portal/backend/(api/|cmd/|configs/|internal/)' || matches "$shared_go"; then
      default=true
    fi
    if matches '^admin-portal/frontend/' || matches "$shared_frontend"; then
      frontend=true
    fi
    if matches '^admin-portal/backend/internal/data/' || matches "$shared_go"; then
      data=true
    fi
    if matches '^admin-portal/backend/(api/|internal/data/)|^(analyse-data-service|agent-run)/backend/api/|^testdata/' || matches "$shared_go"; then
      provider_consumer=true
    fi
    if matches '^admin-portal/backend/(cmd/server/|configs/|internal/conf/|internal/server/)' || matches "$shared_go"; then
      conf_lifecycle=true
    fi
    if matches '^admin-portal/(backend|frontend)/Dockerfile$' || matches "$container_assets"; then
      container=true
    fi
    ;;
  agentrun)
    if matches '^agent-run/backend/(api/|cmd/|configs/|internal/|migrations/)' || matches "$shared_go"; then
      default=true
    fi
    if matches '^agent-run/backend/(internal/data/|migrations/)' || matches "$shared_go"; then
      data=true
    fi
    if matches '^agent-run/backend/(migrations/|internal/data/postgres/migrations(\.go|/))' || matches "$shared_go"; then
      migration=true
    fi
    if matches '^agent-run/backend/(cmd/server/|configs/|internal/conf/|internal/server/)' || matches "$shared_go"; then
      conf_lifecycle=true
    fi
    if matches '^agent-run/backend/(api/|internal/data/connectors/|internal/data/modelprovider/)|^admin-portal/backend/(api/|internal/data/)' || matches "$shared_go"; then
      provider_consumer=true
    fi
    if matches '^agent-run/backend/Dockerfile$' || matches "$container_assets"; then
      container=true
    fi
    ;;
  repository)
    if matches '^(go\.mod|go\.sum|AGENTS\.md|CONTEXT-MAP\.md)$|^(analyse-data-service|miniapp|admin-portal|agent-run)/backend/.*\.go$|^docs/(agents/|adr/|architecture/|contexts/)|^scripts/ci/|^\.github/workflows/'; then
      architecture=true
    fi
    ;;
esac

{
  echo "default=$default"
  echo "frontend=$frontend"
  echo "data=$data"
  echo "migration=$migration"
  echo "conf_lifecycle=$conf_lifecycle"
  echo "provider_consumer=$provider_consumer"
  echo "container=$container"
  echo "architecture=$architecture"
} >>"$GITHUB_OUTPUT"
