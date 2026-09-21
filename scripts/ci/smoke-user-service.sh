#!/usr/bin/env bash
# Disposable container smoke; never targets an existing database.
set -euo pipefail
image="${1:?User Service image required}"
name="user-smoke-${RANDOM}-${RANDOM}"
network="$name-network"
db="$name-db"
service="$name-service"
cleanup() {
  docker rm -f "$service" "$db" >/dev/null 2>&1 || true
  docker network rm "$network" >/dev/null 2>&1 || true
}
trap cleanup EXIT
docker network create "$network" >/dev/null
docker run -d --name "$db" --network "$network" \
  -e POSTGRES_PASSWORD=user-smoke-password -e POSTGRES_DB=tidewise_user_test postgres:16 >/dev/null
for attempt in {1..30}; do
  if docker exec "$db" pg_isready -U postgres -d tidewise_user_test >/dev/null 2>&1; then break; fi
  sleep 1
done
dsn="postgres://postgres:user-smoke-password@${db}:5432/tidewise_user_test?sslmode=disable"
for attempt in 1 2; do
  docker run --rm --network "$network" --entrypoint /app/dbmigrate \
    -e USER_DATABASE_URL="$dsn" -e USER_DATABASE_NAME=tidewise_user_test \
    "$image" -dir /app/migrations
done
# Runtime role cannot create or alter tables.
docker exec "$db" psql -U postgres -d tidewise_user_test -v ON_ERROR_STOP=1 -c \
  "CREATE ROLE user_runtime LOGIN PASSWORD 'user-runtime-test'; GRANT CONNECT ON DATABASE tidewise_user_test TO user_runtime; GRANT USAGE ON SCHEMA public TO user_runtime; GRANT SELECT,INSERT,UPDATE ON users,wechat_identities,user_sessions TO user_runtime; GRANT SELECT ON goose_db_version TO user_runtime;" >/dev/null
runtime_dsn="postgres://user_runtime:user-runtime-test@${db}:5432/tidewise_user_test?sslmode=disable"
docker run -d --name "$service" --network "$network" \
  -e USER_DATABASE_URL="$runtime_dsn" -e USER_DATABASE_NAME=tidewise_user_test \
  -e USER_SERVICE_TOKEN=user-service-smoke-01234567890123456789 \
  -e WECHAT_APP_ID=smoke-app -e WECHAT_APP_SECRET=smoke-secret \
  "$image" >/dev/null
ready=false
for attempt in {1..30}; do
  if docker exec "$service" wget -q -O /dev/null http://127.0.0.1:9015/readyz; then ready=true; break; fi
  sleep 1
done
if [[ "$ready" != true ]]; then echo 'User Service did not become ready' >&2; exit 1; fi
docker exec "$service" wget -q -O /dev/null http://127.0.0.1:9015/healthz
docker stop -t 15 "$service" >/dev/null
exit_code="$(docker inspect --format '{{.State.ExitCode}}' "$service")"
[[ "$exit_code" == 0 ]]
echo 'User Service container migration, restricted-role readiness and graceful stop passed'
