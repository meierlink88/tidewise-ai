# R2 Local PostgreSQL Review Package

Status: prepared only; no database connection, backup, migration, seed, import,
or DSN-enabled integration test was run to prepare this package. Tasks 2.1–2.3
remain unchecked. A future authorization must explicitly authorize both writes
in this one local R2 layer: one `000020` migration and synthetic repository
fixture writes with mandatory cleanup.

## Scope, credential route, and stop policy

Scope is only the local development PostgreSQL in
`backend/config/config.local.yaml`: `APP_ENV=local`, `localhost:5432`,
`tidewise_local`, user `tidewise`. Excluded: UAT/prod/shared, Neo4j,
`event_entity_links` writes, Agent work, real outbox/business Event imports,
frontend, deployment, repair, retry, automatic restore, and forward-fix.

Before every command, use one selected local credential route. `R2_DSN` is an
already-injected value copied from `TIDEWISE_DATABASE_URL` or `DATABASE_URL`;
it is never printed. Password fallback is deliberately not mixed into the R2
package: if no local DSN can be injected, stop before backup. The same `R2_DSN`
is used by backup, psql, dbmigrate and the integration test.

```bash
cd /Users/meierlink/.codex/worktrees/6900/tidewise-ai/backend
set -euo pipefail; set +x; umask 077
export APP_ENV=local
R2_DSN="${TIDEWISE_DATABASE_URL:-${DATABASE_URL:-}}"
: "${R2_DSN:?inject one local DSN through TIDEWISE_DATABASE_URL or DATABASE_URL}"
export TIDEWISE_DATABASE_URL="$R2_DSN"
export TIDEWISE_TEST_DATABASE_URL="$R2_DSN"
```

Any fingerprint, backup, identity, schema, count, hash, cleanup, timeout,
test, or scope failure stops immediately. Do not retry, restore, or forward-fix
automatically. Recovery is manual into a separate local recovery database after
a separate decision.

## Frozen artifact fingerprint gate

Expected SHA-256 values are recalculated in this checkpoint and must be
recomputed and compared again before connecting; do not proceed on a mismatch.

```bash
hash256() { command -v sha256sum >/dev/null && sha256sum "$1" | awk '{print $1}' || shasum -a 256 "$1" | awk '{print $1}'; }
test "$(hash256 migrations/000020_add_event_import_receipts_and_tag_seed.sql)" = "c14bf5952d133fd88ffc2881105fb599b366f6c5a2ad0b3fcea8bdce7fa4fc51"
test "$(hash256 internal/domain/eventimport/tag_catalog.go)" = "ee9959b1df8bacb37782a5dd6af4fbf0b1fef8f11653f5e096a82770a6370906"
test "$(hash256 testdata/event-import/reviewed-outbox-v1.json)" = "acd100ed7778812981490a31015b33384c83c0f4d5457465df273f9e5764905d"
test "$(hash256 ../openspec/changes/add-event-import-and-tag-catalog/r2-preflight.sql)" = "6b20202d4f76f386d4bb8c474314be6152da4815eb85fab58f02149de5308b79"
test "$(hash256 ../openspec/changes/add-event-import-and-tag-catalog/r2-baseline.sql)" = "717b609e44fb355c0956ad6b7fc2f58c172a3cc20033c9b04fbeddef844afafc"
test "$(hash256 ../openspec/changes/add-event-import-and-tag-catalog/r2-postflight.sql)" = "e810f471d06395fac842b48a937f1be2e286b511046cc169541ac5257484c271"
```

The fixture is frozen at 22 tuples (10 `news_category`, 12 `index_category`).
Migration static tests compare UUID/kind/code/name/order/active values to the
same `FrozenTags` source.

## Authorized execution order (not executed now)

Expected elapsed time: fingerprint <1 min; backup 1–3 min; read-only status and
preflight/baseline 1–2 min; migration <1 min; postflight <2 min; synthetic
integration 1–5 min; dry-run/recheck <1 min.

### 1. Backup

```bash
backup_dir="$(pwd)/.data/r2-backups"; mkdir -p "$backup_dir"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
backup_path="$backup_dir/tidewise_local_event_import_${stamp}_pre-000020.dump"
pg_dump --format=custom --no-owner --file="$backup_path" --dbname="$R2_DSN"
test -s "$backup_path"; pg_restore --list "$backup_path" >"${backup_path}.inventory"
hash256 "$backup_path" >"${backup_path}.sha256"
```

The custom dump covers schema and all rows (including source/tag/event and
`event_entity_links`); inventory and dump hash are recorded locally without
printing credentials.

### 2. Read-only dbmigrate status, SQL preflight, and baseline

```bash
status_json=".data/r2-backups/event_import_dbmigrate_status_$(date -u +%Y%m%dT%H%M%SZ).json"
go run ./cmd/dbmigrate >"$status_json"
jq -e '.current_version == "19" and (.pending | length == 1) and (.pending[0].Version == "000020")' "$status_json" >/dev/null
psql "$R2_DSN" -X -v ON_ERROR_STOP=1 -f ../openspec/changes/add-event-import-and-tag-catalog/r2-preflight.sql > .data/r2-backups/event_import_preflight.txt
baseline_json=".data/r2-backups/event_import_baseline.json"
psql "$R2_DSN" -X -qAt -v ON_ERROR_STOP=1 -f ../openspec/changes/add-event-import-and-tag-catalog/r2-baseline.sql >"$baseline_json"
jq -e '.receipt_table_present_before == false and (.fixed_source_present_before == 0 or .fixed_source_present_before == 1) and (.matching_frozen_tuple_count_before >= 0 and .matching_frozen_tuple_count_before <= 22)' "$baseline_json" >/dev/null
```

`r2-preflight.sql` begins `BEGIN TRANSACTION READ ONLY` and nonzero-fails on
wrong database/user/loopback address/version, receipt partial apply, source
UUID/manifest split, frozen UUID occupation, or any matching frozen tag drift.
It also emits table/column/constraint/index inventory. The JSON `dbmigrate`
check proves both database version and the filesystem pending set.

### 3. Single migration write

```bash
go run ./cmd/dbmigrate -apply -target-version 20
```

This is the only schema/master-data write: receipt table, fixed source master,
and absent frozen tags. It must execute once only after steps 1–2 pass. No real
outbox import is permitted.

### 4. Fail-closed postflight

```bash
psql "$R2_DSN" -X -v ON_ERROR_STOP=1 \
  -v source_before="$(jq -er .source_before "$baseline_json")" \
  -v fixed_source_present_before="$(jq -er .fixed_source_present_before "$baseline_json")" \
  -v tag_defs_before="$(jq -er .tag_defs_before "$baseline_json")" \
  -v matching_frozen_tuple_count_before="$(jq -er .matching_frozen_tuple_count_before "$baseline_json")" \
  -v raw_documents_before="$(jq -er .raw_documents_before "$baseline_json")" \
  -v events_before="$(jq -er .events_before "$baseline_json")" \
  -v event_sources_before="$(jq -er .event_sources_before "$baseline_json")" \
  -v event_tag_maps_before="$(jq -er .event_tag_maps_before "$baseline_json")" \
  -v event_entity_links_before="$(jq -er .event_entity_links_before "$baseline_json")" \
  -f ../openspec/changes/add-event-import-and-tag-catalog/r2-postflight.sql
```

It asserts `source_after = source_before + (1 - fixed_source_present_before)`,
`tag_defs_after = tag_defs_before + (22 - matching_frozen_tuple_count_before)`,
unchanged raw/event/source/tag-map/entity-link counts, zero receipts before
dry-run, active fixed source identity, version 20, receipt schema presence, and
exactly 22 fully matching active frozen tuples. It does not require the global
active-tag total to be 22.

### 5. Synthetic repository integration assertions and cleanup

```bash
go test ./internal/repositories -run '^TestEventImportPostgresIntegration$' -count=1
```

The narrow test requires the same `TIDEWISE_TEST_DATABASE_URL` route, runs
immediately after `r2-postflight.sql`, asserts `tidewise_local` and complete
receipt schema compatibility before its first fixture write, then writes only a
timestamp-keyed synthetic fixture after 000020,
then deferred-cleans receipts → tag maps → sources → events → raw documents and
asserts zero residual rows. It never modifies fixed source/22 Tag masters or
uses a real outbox. Coverage: UUID[] receipt scan/write; barrier-synchronized
concurrent same-hash replay through the advisory lock (two identical results
and one persisted result set); different-hash conflict; rollback;
`VerifyReceiptResults` missing/cross-event; and full receipt column
type/not-null/default, PK/FK/idempotency-unique/check/index compatibility.
Cleanup failure fails the test.

### 6. CLI dry-run and zero-write recheck

```bash
go run ./cmd/event-import --file testdata/event-import/reviewed-outbox-v1.json --dry-run --json > .data/r2-backups/event_import_dry_run.json
psql "$R2_DSN" -X -tA -v ON_ERROR_STOP=1 -c 'SELECT count(*) = 0 FROM event_import_receipts' | grep -qx t
```

An integration failure or any recheck failure stops the package; no automatic
restore/retry/forward fix follows.
