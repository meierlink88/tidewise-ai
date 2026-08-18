#!/usr/bin/env bash

set -euo pipefail

runtime_env="${RUNTIME_ENV:?RUNTIME_ENV is required}"
compose_file="${COMPOSE_FILE:?COMPOSE_FILE is required}"

{
  docker compose --env-file "$runtime_env" -f "$compose_file" ps 2>&1 || true
  docker compose --env-file "$runtime_env" -f "$compose_file" logs --tail 150 --no-color 2>&1 || true
} | sed -E \
  -e 's#((PASSWORD|SECRET|TOKEN|ACCESS_KEY|ROOT_USER)[=:][[:space:]]*)[^,[:space:]]+#\1***#Ig' \
  -e 's#(Authorization:[[:space:]]*(Bearer|Basic)[[:space:]]+)[^[:space:]]+#\1***#Ig'
