# R2 Local PostgreSQL Review Package

Status: prepared only; not authorized or executed. This artifact defines the
execution package for tasks 2.1–2.3. No database, backup, migration, seed, or
import command was run while preparing it.

## Scope, environment, and secrets

Only local development PostgreSQL is in scope. The actual repository config is
`backend/config/config.local.yaml`: `APP_ENV=local`, `localhost:5432`, database
`tidewise_local`, user `tidewise`, migration directory `migrations`, and a
5-second connection timeout. UAT/prod/shared, Neo4j, `event_entity_links`
writes, Agent changes, real Event import, deployment, frontend, and data repair
are excluded. The only permitted write is one application of migration 000020;
the reviewed-outbox CLI may run only afterward with `--dry-run` and must be
zero-write.

Credentials must remain injected through existing `TIDEWISE_DATABASE_URL` or
`DATABASE_URL`, or through `DATABASE_PASSWORD` with the local host/user/name.
Never print, echo, persist, commit, or report a DSN, password, token, or
`PGPASSWORD`; run shell commands with tracing disabled.

Current read-only artifact fingerprints (recompute immediately before R2):

| Artifact | SHA-256 |
|---|---|
| `backend/migrations/000020_add_event_import_receipts_and_tag_seed.sql` | `c14bf5952d133fd88ffc2881105fb599b366f6c5a2ad0b3fcea8bdce7fa4fc51` |
| `backend/internal/domain/eventimport/tag_catalog.go` | `ee9959b1df8bacb37782a5dd6af4fbf0b1fef8f11653f5e096a82770a6370906` |
| `backend/testdata/event-import/reviewed-outbox-v1.json` | `acd100ed7778812981490a31015b33384c83c0f4d5457465df273f9e5764905d` |
| `r2-preflight.sql` | `cae739e5317edbb7c9cd445de44d97036e8459677a0ebe2dca91abdb5edae43c` |

The Tag fixture is exactly 22 ordered tuples: 10 `news_category` and 12
`index_category`. Migration static tests compare every UUID, kind, code, name,
display order, and active value against `FrozenTags`.

## Command order and stop policy

Expected elapsed time: fingerprint/config checks 1 minute; backup 1–3 minutes;
read-only preflight 1–2 minutes; isolated repository assertions 1–5 minutes;
single migration under 1 minute; after assertions/dry-run 1–3 minutes.

Any backup, identity, count, hash, schema, test, cleanup, timeout, or scope
failure stops the package. Do not retry, auto-restore, destructive-down, or
forward-fix.

### 1. Backup — prepare only

Run from `backend/`, with output under the local ignored data directory:

```bash
cd /Users/meierlink/.codex/worktrees/6900/tidewise-ai/backend
set -euo pipefail; set +x; umask 077
backup_dir="$(pwd)/.data/r2-backups"; mkdir -p "$backup_dir"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
backup_path="$backup_dir/tidewise_local_event_import_${stamp}_pre-000020.dump"
if [[ -n "${TIDEWISE_DATABASE_URL:-${DATABASE_URL:-}}" ]]; then
  pg_dump --format=custom --no-owner --file="$backup_path" --dbname="${TIDEWISE_DATABASE_URL:-${DATABASE_URL}}"
else
  : "${DATABASE_PASSWORD:?DATABASE_PASSWORD must already be injected}"
  PGPASSWORD="$DATABASE_PASSWORD" pg_dump --format=custom --no-owner --file="$backup_path" \
    --host=localhost --port=5432 --username=tidewise --dbname=tidewise_local
fi
test -s "$backup_path"; pg_restore --list "$backup_path" >/dev/null
printf '%s\n' "$backup_path"
```

Success means a non-empty custom-format dump and successful `pg_restore --list`.
The dump covers schema and all rows, including event/source/tag/receipt and
`event_entity_links`. Recovery is manual only, into an isolated local recovery
database after a separate decision; never restore automatically into the source
database.

### 2. Read-only preflight

Recompute the fingerprints, then run the SQL artifact without exposing secrets:

```bash
cd /Users/meierlink/.codex/worktrees/6900/tidewise-ai/backend
set -euo pipefail; set +x
: "${TIDEWISE_DATABASE_URL:?inject the local DSN through the environment}"
psql "$TIDEWISE_DATABASE_URL" -X -v ON_ERROR_STOP=1 \
  -f ../openspec/changes/add-event-import-and-tag-catalog/r2-preflight.sql \
  > .data/r2-backups/event_import_preflight_$(date -u +%Y%m%dT%H%M%SZ).txt
```

The report must record: PostgreSQL database/user/address/port/version; Goose
current version and pending migrations; full table/column/constraint/index
inventory for source, raw, event, evidence, Tag, receipt, and
`event_entity_links`; fixed source UUID
`cd209afe-2ea9-54b8-bdd7-db64eebf0d71`, manifest identity
`tidewise:agent:event-reviewed-outbox`, active status and conflict check; all
22 Tag tuples with exact UUID/identity/order/active state; and before counts for
`source_catalogs`, `raw_documents`, `events`, `event_sources`,
`event_tag_defs`, `event_tag_maps`, `event_import_receipts`, and
`event_entity_links`.

The runtime baseline is intentionally not invented here. For this single-write
package the preflight must assert current migration version exactly `19`, only
`000020` pending, and no other pending migration. Any other value stops. After
values must preserve every existing count, have exactly 22 active frozen Tags,
and leave receipts at their pre-count (normally zero for a new table).

### 3. Isolated repository assertions

Use a disposable local PostgreSQL database or isolated schema supplied through
`TIDEWISE_TEST_DATABASE_URL`; never point it at `tidewise_local`. Reuse the
existing `sql.Open("pgx", os.Getenv("TIDEWISE_TEST_DATABASE_URL"))` convention,
apply the reviewed schema fixture only inside that isolation, run the focused
repository contract test, then drop the database/schema in deferred cleanup.
Cleanup failure is a stop condition.

The exact focused command is:

```bash
cd /Users/meierlink/.codex/worktrees/6900/tidewise-ai/backend
set -euo pipefail; set +x
: "${TIDEWISE_TEST_DATABASE_URL:?point this only at the disposable isolated database}"
TIDEWISE_TEST_DATABASE_URL="$TIDEWISE_TEST_DATABASE_URL" \
  go test ./internal/repositories -run '^TestEventImportPostgresIntegration$' -count=1
```

The current R1 checkpoint has no database integration test and therefore does
not claim this command has passed. Before R2 Write authorization, the narrowly
scoped test named above must be present, reviewed, and run against the isolated
fixture; absence of the test is a stop condition, not permission to substitute
static SQL checks.

Required assertions: UUID[] insert/scan preserves order and values; concurrent
same-key same-hash requests serialize on
`pg_advisory_xact_lock(hashtextextended(...))` and replay identical IDs; same
key/different hash conflicts without changing the receipt; post-write failure
rolls back all raw/event/source/tag/receipt rows; `VerifyReceiptResults` rejects
missing or cross-event IDs; and all 000020 columns, constraints, unique keys,
and indexes are schema-compatible. No real outbox package is used.

### 4. Single write — prepare only

Only after backup, preflight, and isolated assertions pass:

```bash
cd /Users/meierlink/.codex/worktrees/6900/tidewise-ai/backend
set -euo pipefail; set +x
APP_ENV=local DATABASE_PASSWORD="${DATABASE_PASSWORD:?DATABASE_PASSWORD must already be injected}" \
  go run ./cmd/dbmigrate -apply -target-version 20
```

Because preflight requires version 19, this writes only 000020, including the
fixed source master, 22 Tag seed, receipt schema, constraints and indexes. Do
not run source/entity seed or real `event-import`.

### 5. After assertions and zero-write dry-run

Rerun identity, migration, schema, source, Tag and count queries and assert:

- migration version 20 is applied and nothing remains pending;
- receipt schema/constraints/indexes match 000020;
- fixed source is active with the exact manifest identity;
- exactly 22 active frozen tuples and their recorded fingerprints match;
- existing source/raw/event/evidence/Tag/tag-map counts and
  `event_entity_links` count are unchanged;
- receipt count is the pre-count plus zero; and
- this dry-run succeeds without changing that receipt count:

```bash
cd /Users/meierlink/.codex/worktrees/6900/tidewise-ai/backend
set -euo pipefail; set +x
go run ./cmd/event-import --file testdata/event-import/reviewed-outbox-v1.json --dry-run --json \
  > .data/r2-backups/event_import_dry_run_$(date -u +%Y%m%dT%H%M%SZ).json
```

Tasks 2.1, 2.2 and 2.3 remain unchecked until a separate R2 authorization and
the complete preflight → write → after-assert sequence succeeds.
