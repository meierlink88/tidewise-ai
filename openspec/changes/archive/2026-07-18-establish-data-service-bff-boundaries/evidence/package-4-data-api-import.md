# Package 4 Data API And Import Evidence

## Scope And Outcome

Package 4 implements the approved `/internal/data/v1` Data boundary without applying schema or changing a BFF runtime. The Data Service now owns its OpenAPI, authenticated handler, Research/Admin/source-metadata reads, raw-document batch import/status, reviewed-event import, PostgreSQL adapters, health composition, repository wiring, and fail-closed read-only migration readiness. Miniapp and Admin own small handwritten read-only typed clients plus fakes and OpenAPI drift tests; they are not wired into either BFF until Packages 6 and 7.

`event-import` retains file/dir/input, dry-run, machine JSON and exit-code behavior. Its non-dry-run path no longer imports `database` or `repositories`: it performs one authenticated Data HTTP mutation with no retry and a bounded request-context timeout (15 seconds when the compatible flag value is `0`, matching OpenAPI `x-timeout-budget-ms: 15000`; a positive flag value supplies an explicit bounded deadline). It does not log the token or response body.

The Data command assembles existing repositories and application services. Startup migration readiness opens a server-enforced read-only transaction, requires an existing Goose ledger, compares it with repository migration files, and fails on a missing ledger, unknown applied version, or pending migration. It does not call `goose.EnsureDBVersionContext`, create the ledger, auto-apply, or execute migration SQL.

## Exact Change Manifest

| Files | Purpose |
|---|---|
| `backend/services/data/api/{openapi.yaml,openapi_test.go}` | 11-path versioned contract, DTO/error/auth/request-id/bounds and schema tests |
| `backend/services/data/internalapi/{handler.go,handler_test.go}` | scoped transport, aggregates, cursor/key/body bounds, redacted source metadata and imports |
| `backend/services/data/rawimport/{service.go,service_test.go}` | canonical plan, caller-scoped receipts, validation/replay/status/conflict state machine |
| `backend/services/data/rawimport/postgresstore/{store.go,store_test.go}` | one-transaction locks, separate identity reads, conflict-safe insert/winner reread and receipt storage |
| `backend/services/data/{service.go,service_test.go}` | Data API plus health/readiness composition |
| `backend/services/data/cmd/{main.go,main_test.go}` | fail-closed service identities and production dependency assembly |
| `backend/services/{miniapp,adminportal}/dataclient/{port.go,http.go,client_test.go,contract_drift_test.go}` | consumer-owned typed ports/adapters/fakes, safe GET retry and drift checks |
| `backend/internal/config/{config.go,config_test.go}` | environment-only Data service identity secrets, excluded from YAML serialization |
| `backend/internal/platform/dbmigration/{readiness.go,readiness_test.go,raw_document_import_receipt_contract_test.go,source_test.go}` | existing-ledger read-only readiness and 22-file/frozen migration static contracts |
| `backend/migrations/000022_add_raw_document_import_receipts.sql` | unapplied forward-only immutable seven-column raw receipt artifact |
| `backend/internal/repositories/{admin_query.go,admin_query_postgres_test.go}` | approved raw filters and exact 17-column scanner projection including `content_level` |
| `backend/internal/apps/ingestion/eventimport/{service.go,service_test.go}` | transport-only replay marker while preserving stored business result |
| `backend/cmd/event-import/{main.go,main_test.go}` | controlled Data HTTP compatibility adapter and bounded default timeout |
| this evidence, `package-5-raw-receipt-schema-review.md`, and `tasks.md` | checkpoint evidence, next R2 review, completed Package 4 state |

No existing migration was modified. No test or testdata file was removed.

## TDD And Review Findings

The following RED conditions were observed before minimal GREEN changes:

- raw import packages/types were absent; canonical/idempotency/transaction tests did not compile;
- `TestImportValidatesEveryCandidateAgainstCachedSourceAttribution` accepted the second same-source item with a mismatched channel (`error=<nil>`); GREEN caches one lookup but validates every item and performs zero raw lock/document/receipt DML on failure;
- the Data server with an API handler returned `404` for `/healthz`; GREEN composes the API with service health;
- Data command credential assembly tests did not compile before environment-only scoped credentials were added;
- read-only readiness tests did not compile before `RequirePostgresReadyReadOnly`; GREEN uses only `BEGIN READ ONLY`, two `SELECT` statements, and `COMMIT`/`ROLLBACK` under sqlmock, with missing-ledger/pending fail-closed cases;
- non-dry-run CLI tests had no controlled HTTP adapter; after the adapter existed, the default-timeout test reported `request context has no deadline`; GREEN always supplies the OpenAPI-aligned 15-second default or explicit positive deadline;
- the existing migration source test still expected only `000001`–`000021`; GREEN explicitly includes approved `000022`;
- Admin client drift caught the four preserved event-time filters and `content_level`; the repository review found a 16-column query feeding a 17-value scanner. GREEN preserves all filters and asserts the corrected `content_text, content_level, raw_object_uri` projection;
- source-metadata final-cursor and raw status-key boundary tests guard an empty final page and transport-level 1..200 validation with OpenAPI `400`.
- reviewed-event handler tests first exposed a repository-like error in a `422` body; GREEN prevalidates known package errors with a fixed safe `422`, maps only the typed idempotency conflict to `409`, and returns a fixed `500` for unexpected service/store failures. Raw source-store tests likewise keep only the explicit source-not-found sentinel in safe validation while unclassified SQL/connection errors become generic `500`; token, SQL, connection and password-like text never reaches the response.

## Frozen Migration And Test Manifest

```text
repository migration files: 22
historical 000001..000021 manifest aggregate SHA-256:
2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc
000022 SHA-256:
3966172b267b1cb27e1538fde35a4125734201628c9ea308d62e0a1314685c26
OpenAPI SHA-256:
e2fbb4328d8a8f16107ed46f01a7f2dbee12872abd6b9109f2c789f1744fd6fd
```

Static migration tests assert the exact seven columns; named PK/unique/checks/index/function/statement trigger; transactional Goose Up; absence of `NO TRANSACTION`, `IF NOT EXISTS`, `OR REPLACE`, existing-table DML, event/source/tag/review placeholders and jobs/runs; and a failing forward-only Down. The repo-level before/after test inventory is `114 -> 131` Go test files and `541 -> 601` `func Test` cases across completed Packages 3–4. Testdata remains exactly 13 files; deletion count is zero. The reviewed-event fixture has both CLI and Data handler references, and the 12 task-design fixtures remain loaded by the architecture lint fixture loader.

## Fresh Package Validation

```text
go test ./services/data/... -count=1
PASS

go test ./services/miniapp/dataclient ./services/adminportal/dataclient -count=1
PASS (loopback-only httptest run)

go test ./cmd/event-import ./internal/config -count=1
PASS

env -u TIDEWISE_TEST_DATABASE_URL -u TIDEWISE_DATABASE_URL -u DATABASE_URL \
  go test ./internal/apps/ingestion/eventimport ./internal/repositories -count=1
PASS; live PostgreSQL integration disabled

go test ./internal/platform/dbmigration -count=1
PASS

go test -race ./services/data/internalapi ./services/data/rawimport/... \
  ./services/miniapp/dataclient ./services/adminportal/dataclient \
  ./internal/apps/ingestion/eventimport ./cmd/event-import -count=1
PASS

go test ./services/data/api ./services/miniapp/dataclient ./services/adminportal/dataclient \
  -run '^(TestOpenAPI|TestOpenAPIContractMatches)' -count=1
PASS

go test ./internal/architecture \
  -run '^(TestTransitionalBFFDataDependencyAllowlist|TestServiceOwnedPackagesAndCommandsExist|TestServiceOwnedBFFPackagesDoNotImportDataImplementation|TestServiceSkeletonKeepsSingleBackendModule)$' \
  -count=1
PASS

go vet ./services/data/... ./services/miniapp/dataclient ./services/adminportal/dataclient \
  ./cmd/event-import ./internal/apps/ingestion/eventimport ./internal/repositories \
  ./internal/platform/dbmigration ./internal/config
PASS

go build ./services/data/cmd ./cmd/event-import
PASS
```

Strict OpenSpec/task-design lint, artifact status, final diff/whitespace/scope/secret checks are recorded after this evidence and task state are added.

## Stop Boundary

No PostgreSQL/Neo4j connection, SQL/migration command, seed/import business write, role/credential change, UAT/prod/shared access, deployment, BFF decoupling, scheduler/runtime deletion, restore, retry, or forward-fix was performed. `000022` is an unapplied artifact. Package 5 remains an R2 Human=yes gate and is not started by this checkpoint.
