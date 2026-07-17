# Package 8 Scheduler/Runtime Retirement Evidence

Date: 2026-07-18

Base checkpoint: `f7c70992713f514146268bdc2926efb3389834db`

Scope: tasks 8.1-8.7 only. Package 9 assets/local orchestration/CI, Package 10 roles and credentials, database execution, Neo4j, deployment, UAT/prod/shared environments and agent-run repository changes were not executed.

## 8.1 fail-closed gate

- Worktree started clean at the accepted Package 7 checkpoint.
- Package 4 raw-document import and reviewed-event import contracts remain separate and covered by Data handler/application/repository tests.
- Package 5 evidence remains complete with ledger version 22 and zero receipt residue; no database command was run in Package 8.
- Historical migrations `000001`-`000021` aggregate SHA-256 remained `2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc`.
- Migration `000022_add_raw_document_import_receipts.sql` SHA-256 remained `3966172b267b1cb27e1538fde35a4125734201628c9ea308d62e0a1314685c26`.
- Static caller scans found no retained production caller for the retired commands/packages, scheduler/run repositories/domain/config, or `SourceRegistry`, `RateLimiter`, `LocalRawObjectStore` and `RawDocumentWriter`.

## Keep/remove/replace result

### Removed production paths

- Commands: `backend/cmd/ingestion-scheduler`, `backend/cmd/source-ingest`, `backend/cmd/ingest-smoke`.
- Application packages: `backend/internal/apps/ingestion/scheduler`, `runtime`, and `health/doc.go`.
- Runtime-only persistence: `repositories/scheduler.go`, `repositories/ingestion_run.go`, and in-memory scheduler/run state.
- Runtime-only domain/config: scheduler/run types and validation, `config.IngestionConfig`, validation and three repo config-template `ingestion` blocks.
- Runtime-only core helpers: `SourceRegistry`, `RateLimiter`, `LocalRawObjectStore`, and `RawDocumentWriter`.

No replacement scheduler, worker or Tidewise execution command was added.

### Retained contracts and data ownership

- All 22 repository migration artifacts, including the immutable historical 21 migrations, remain byte-identical. Historical scheduler/run tables and rows are preserved; no drop/truncate/rewrite or database connection occurred.
- `Connector`, `Parser`, `Registry`, `EnvCredentialResolver`, connector/parser/sourcecatalog packages, source metadata/seed, prompts, raw-document repository, raw import/status receipt contract, reviewed-event import and graph projector remain.
- Connector/parser code remains temporarily owned in Tidewise without an executable Tidewise scheduler command. Independent agent-run owns future scheduling/runtime and must use authenticated `/internal/data/v1` metadata/import contracts rather than Data DB access.

## Tests and testdata

Immediate Package 8 baseline and result:

| Inventory | Before | After | Delta | Reason |
|---|---:|---:|---:|---|
| Go `*_test.go` files | 134 | 124 | -10 | Ten direct test files belonged only to deleted command/scheduler/runtime/repository production files. |
| Go `func Test` entries | 612 | 580 | -32 | 23 direct tests + 5 scheduler/run domain tests + 4 runtime-only core tests. |
| testdata files | 13 | 13 | 0 | All architecture fixtures and reviewed-event fixture remain referenced. |

The Package 7 Admin `410 Gone`/zero-Data-call replacement tests and three-tab frontend tests remain unchanged. No connector, parser, sourcecatalog, API, migration, idempotency, transaction, security, raw receipt or reviewed-event test was deleted.

Architecture/config replacements now assert the retired paths, repository files and ingestion config keys are absent while retained packages and dependency rules remain present. RED failed on the still-present retired repository and commands; GREEN passed after the scoped removal.

The testdata reference scan reported one test reference for each of the 12 task-design fixture files and two references for `backend/testdata/event-import/reviewed-outbox-v1.json`; there are no dangling fixtures.

## Fresh validation

- PASS: `go test ./internal/apps/ingestion/core ./internal/apps/ingestion/connectors ./internal/apps/ingestion/parsers ./internal/apps/ingestion/sourcecatalog ./internal/apps/ingestion/eventimport -count=1` (connector loopback package rerun with local bind permission after sandbox-only `operation not permitted`).
- PASS: `go test ./services/data/... ./internal/repositories ./internal/platform/dbmigration -count=1`.
- PASS: `go test ./internal/architecture ./internal/config ./internal/domain ./cmd/... -count=1`.
- PASS: `go test ./services/adminportal/... ./services/miniapp/... ./internal/apps/adminapi ./internal/apps/miniappapi -count=1` with local loopback permission.
- PASS: production/reference scan has no retired symbol, import or command reference outside explicit absence assertions and retirement documentation.
- PASS: all 13 retained testdata files have test loaders/references.
- PASS: `openspec validate establish-data-service-bff-boundaries --strict`.
- PASS: explicit `OPENSPEC_TASK_LINT_CHANGE=establish-data-service-bff-boundaries` task-design lint.
- PASS: OpenSpec artifact status 4/4 complete.
- PASS: migration hashes above, `git diff --check`, scoped path and secret checks.

No PostgreSQL/Neo4j command, migration apply, role/grant/credential change, seed/import, business-data write, infra service asset, CI/Docker, deployment or Package 9+ action was performed.
