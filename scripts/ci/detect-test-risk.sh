#!/usr/bin/env bash
set -euo pipefail

: "${BASE_SHA:?BASE_SHA is required}"
: "${HEAD_SHA:?HEAD_SHA is required}"
: "${GITHUB_OUTPUT:?GITHUB_OUTPUT is required}"

scope="${1:-}"
case "$scope" in
  data | miniapp | adminportal | repository) ;;
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

matches_excluding() {
  grep -E "$1" <<<"$changed_paths" | grep -Ev "$2" >/dev/null
}

default=false
frontend=false
data=false
migration=false
migration_smoke=false
migration_framework=false
conf_lifecycle=false
provider_consumer=false
container=false
architecture=false
object_schema=false
uat_infra=false
uat_neo4j=false

shared_go='^(go\.mod|go\.sum)$'
shared_frontend='^(package\.json|package-lock\.json)$'
container_assets='(^|/)(Dockerfile|docker-compose[^/]*\.ya?ml)$|^infra/(local|uat|uat-infra)/|^\.github/workflows/(ci|deploy-uat|deploy-uat-infra|recover-uat-pre-v74-evidence)\.yml$'

if matches '^infra/uat-neo4j/|^scripts/ci/verify-uat-neo4j-contract\.sh$'; then
  uat_neo4j=true
fi

case "$scope" in
  data)
    if matches '^infra/uat-infra/|^scripts/ci/smoke-uat-infra\.sh$|^\.github/workflows/deploy-uat-infra\.yml$'; then
      uat_infra=true
    fi
    if matches '^data-service/doctype/|^scripts/ci/(verify-openspg-schemas\.py|verify-openspg-schemas\.sh|openspg-parser-requirements\.txt|openspg-kag-revision\.txt)$|^docs/development-standards/openspg-schema\.md$'; then
      object_schema=true
    fi
    if matches_excluding '^data-service/backend/(api/|cmd/|configs/|internal/)' '^data-service/backend/internal/data/dbmigration/' || matches "$shared_go"; then
      default=true
    fi
    if matches_excluding '^data-service/backend/internal/data/' '^data-service/backend/internal/data/dbmigration/' || matches "$shared_go"; then
      data=true
    fi
    if matches '^data-service/backend/(migrations/.*\.sql$|internal/data/dbmigration/.*\.go$)' || matches "$shared_go"; then
      migration=true
      migration_smoke=true
    fi
    if matches '^data-service/backend/internal/data/dbmigration/.*\.go$' || matches "$shared_go"; then
      migration_framework=true
    fi
    if matches '^data-service/backend/(cmd/server/|configs/|internal/conf/|internal/server/)' || matches "$shared_go"; then
      conf_lifecycle=true
    fi
    if matches '^data-service/backend/api/' || matches "$shared_go"; then
      provider_consumer=true
    fi
    if matches '^data-service/backend/Dockerfile$' || matches "$container_assets"; then
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
    if matches '^miniapp/backend/(api/|internal/data/)|^data-service/backend/api/' || matches "$shared_go"; then
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
    if matches '^admin-portal/backend/(api/|internal/data/)|^data-service/backend/api/' || matches "$shared_go"; then
      provider_consumer=true
    fi
    if matches '^admin-portal/backend/(cmd/server/|configs/|internal/conf/|internal/server/)' || matches "$shared_go"; then
      conf_lifecycle=true
    fi
    if matches '^admin-portal/(backend|frontend)/Dockerfile$' || matches "$container_assets"; then
      container=true
    fi
    ;;
  repository)
    if matches '^(go\.mod|go\.sum|AGENTS\.md|CONTEXT-MAP\.md)$|^(data-service|miniapp|admin-portal)/backend/.*\.go$|^docs/(agents/|adr/|contexts/|development-standards/)|^infra/(uat|uat-infra|uat-neo4j)/|^scripts/ci/|^\.github/workflows/'; then
      architecture=true
    fi
    ;;
esac

{
  echo "default=$default"
  echo "frontend=$frontend"
  echo "data=$data"
  echo "migration=$migration"
  echo "migration_smoke=$migration_smoke"
  echo "migration_framework=$migration_framework"
  echo "conf_lifecycle=$conf_lifecycle"
  echo "provider_consumer=$provider_consumer"
  echo "container=$container"
  echo "architecture=$architecture"
  echo "object_schema=$object_schema"
  echo "uat_infra=$uat_infra"
  echo "uat_neo4j=$uat_neo4j"
} >>"$GITHUB_OUTPUT"
