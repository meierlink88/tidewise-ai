#!/usr/bin/env bash

set -euo pipefail

runtime_env="${RUNTIME_ENV:?RUNTIME_ENV is required}"
images_env="${CANDIDATE_IMAGES:?CANDIDATE_IMAGES is required}"
compose_file="${COMPOSE_FILE:-infra/uat/docker-compose.yaml}"

{
  docker compose --env-file "$runtime_env" --env-file "$images_env" -f "$compose_file" ps 2>&1 || true
  docker compose --env-file "$runtime_env" --env-file "$images_env" -f "$compose_file" logs --tail 100 --no-color 2>&1 || true
} | sed -E \
  -e 's#(postgres(ql)?://[^:/[:space:]]+:)[^@[:space:]]+@#\1***@#g' \
  -e 's#(Authorization:[[:space:]]*(Bearer|Basic)[[:space:]]+)[^[:space:]]+#\1***#Ig' \
  -e 's#((TOKEN|PASSWORD|SECRET|DATABASE_URL)[=:][[:space:]]*)[^,[:space:]]+#\1***#Ig'
