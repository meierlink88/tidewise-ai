#!/usr/bin/env sh
set -eu

: "${POSTGRES_HOST:?POSTGRES_HOST is required}"
: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}"
: "${POSTGRES_DB:?POSTGRES_DB is required}"
: "${AGENTRUN_DATABASE_USER:?AGENTRUN_DATABASE_USER is required}"
: "${AGENTRUN_DATABASE_PASSWORD:?AGENTRUN_DATABASE_PASSWORD is required}"
: "${AGENTRUN_DATABASE_NAME:?AGENTRUN_DATABASE_NAME is required}"

export PGPASSWORD="$POSTGRES_PASSWORD"

psql \
  --host "$POSTGRES_HOST" \
  --username "$POSTGRES_USER" \
  --dbname "$POSTGRES_DB" \
  --set ON_ERROR_STOP=1 \
  --set agentrun_user="$AGENTRUN_DATABASE_USER" \
  --set agentrun_password="$AGENTRUN_DATABASE_PASSWORD" \
  --set agentrun_database="$AGENTRUN_DATABASE_NAME" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'agentrun_user', :'agentrun_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'agentrun_user') \gexec
SELECT format('ALTER ROLE %I WITH LOGIN PASSWORD %L', :'agentrun_user', :'agentrun_password') \gexec
SELECT format('CREATE DATABASE %I OWNER %I', :'agentrun_database', :'agentrun_user')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'agentrun_database') \gexec
SELECT format('ALTER DATABASE %I OWNER TO %I', :'agentrun_database', :'agentrun_user') \gexec
SQL
