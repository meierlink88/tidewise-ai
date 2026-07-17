# Package 2 Inventory And Architecture Evidence

## Snapshot

| Item | Frozen value |
|---|---|
| Original Proposal checkpoint | `bfd6e7e1648ca6fd98526cc2c358e8edeb329e02`；后续raw receipt amendment另建checkpoint |
| Input `origin/main` | `3f0f779d2c332a74f31fd398adb47adb306a60c3` |
| Go module | one module at `backend/go.mod` |
| Historical SQL migrations | 21 `backend/migrations/*.sql`；按路径排序的逐文件SHA-256 manifest聚合hash=`2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc`；Package 4只能新增`000022`，21文件保持byte-for-byte不变 |
| Historical scheduler tables | `ingestion_scheduler_configs`, `ingestion_runs`, `ingestion_run_sources`; tables and rows are retained |
| Go tests before removal | 114 `_test.go` files; 541 `func Test` cases |
| Frontend UI tests before removal | Admin 7 files/26 cases; Miniapp 1 file/18 cases; total 8 files/44 cases |
| On-disk `testdata` | 13 files, all referenced; expected deletion is zero |

This is a read-only manifest. No database, migration, seed, graph, environment, credential, or deployment operation was used to produce it.

## Production Keep/Remove/Replace Manifest

| Area | Exact path or symbol | Current production caller | Package action | Replacement or retained contract |
|---|---|---|---|---|
| command | `backend/cmd/ingestion-scheduler/main.go` | independent binary | remove in Package 8 | no Tidewise replacement; scheduling/execution belongs to future external `agent-run` |
| command | `backend/cmd/source-ingest/main.go` | independent binary | remove in Package 8 | no Tidewise manual runner; Data import API is the supported write boundary |
| command | `backend/cmd/ingest-smoke/main.go` | independent binary | remove in Package 8 | no production ingest smoke; connector/parser tests remain |
| scheduler app | `backend/internal/apps/ingestion/scheduler/{doc.go,planner.go,service.go}` | `cmd/ingestion-scheduler` | remove in Package 8 | no replacement scheduler in this repository |
| runtime app | `backend/internal/apps/ingestion/runtime/{doc.go,ingestion_job.go,ingestion_smoke.go}` | three retired commands and scheduler app | remove in Package 8 | raw/reviewed-event HTTP import contracts replace the Data write boundary, not connector execution |
| health marker | `backend/internal/apps/ingestion/health/doc.go` | no Go importer | remove in Package 8 | service-owned liveness/readiness, not source-run health |
| runtime-only core | `SourceRegistry`, `NewSourceRegistry`, `RateLimiter`, `NewRateLimiter`, `RateLimitPolicy`, `LocalRawObjectStore`, `RawObject`, `RawDocumentWriter`, `NewRawDocumentWriter`, `contentHash` | only retired runtime/commands or their tests | remove only after Package 8 caller proof | keep `Registry`, `Connector`, `Parser`, `RawResponse`, `RawDocumentCandidate`, `EnvCredentialResolver` |
| scheduler persistence | `backend/internal/repositories/scheduler.go`, `ingestion_run.go`; matching fields/state in `memory.go` | Admin scheduler handlers and retired runtime | remove in Package 8 | authenticated 410 tombstones perform no DB reads/writes; historical tables remain |
| scheduler domain | scheduler/run/source-run types and validation in `backend/internal/domain/models.go` | retired scheduler/repositories/Admin DTO mapping | remove in Package 8 | machine-readable transport tombstone only |
| runtime config | `IngestionConfig`, `Config.Ingestion`, scheduler tick/timezone and runtime-only fields in `internal/config/config.go`; `ingestion:` blocks in `config.{local,uat,prod}.yaml` | only three retired commands | remove in Package 8 | per-service HTTP identity/timeout/import limits use independent config |
| Admin scheduler API | three routes plus scheduler DTO/handlers/repository methods in `internal/apps/adminapi/router.go` | Admin frontend scheduler UI | replace in Package 7 | authenticated, machine-readable, no-DB `410 Gone` for one deployment window |
| Admin scheduler UI | `api/scheduler.ts`, `SchedulerSettings.tsx`, scheduler tab and dedicated styles | Admin browser | remove/replace in Package 7 | retain raw document, event, source catalog, auth and pagination behavior |
| historical schema | all 21 SQL migrations, especially `000005_add_ingestion_scheduler.sql`; three scheduler tables and existing rows | migration history/audit | keep unchanged | no drop, truncate, rewrite, backfill, or data deletion |
| Data repositories | `raw_document.go`, `source_catalog.go`, `event*.go`, `event_import*.go`, `admin_query.go` | Data API/import/query paths | keep | Data-owned repositories remain covered by domain/transaction/idempotency tests |
| raw import receipt | Package 4 adds `000022_add_raw_document_import_receipts.sql` plus dedicated repository/API contracts | raw-document import and status only | add/keep | caller-scoped immutable receipt; Package 5 applies once to local only after order 1 preflight evidence passes exactly; fully separate from event receipts |
| source metadata | `cmd/source-seed`, `internal/apps/ingestion/sourcecatalog/**`, `data/source_catalogs/**` | source seed and scoped read contract | keep | Data Service exposes only approved, redacted metadata and `credential_ref` names |
| connectors/parsers | `internal/apps/ingestion/connectors/**`, `parsers/**`, retained core contracts, `data/prompts/ingestion/**` | unit/contract tests; no retained production runner | keep in Data Service ownership | independent command execution is removed; migration to external repo is out of scope |
| reviewed-event import | `cmd/event-import`, ingestion/domain event-import packages and repositories | controlled import CLI/Data API | keep/adapt | canonical hash, receipt and event/source/tag/raw transaction stay atomic |
| other Data jobs | entity seed, graph projector, dbmigrate | independent Data maintenance commands | keep | none is an ingestion scheduler/runtime path; none is executed in R1 |

## Transitional Import/Reference Manifest

The strict target test was first run with an empty Miniapp allowlist and failed with the existing `internal/domain` and `internal/repositories` imports. The committed Package 2 guard freezes the exact transitional graph below; Packages 6 and 7 must reduce it to zero rather than add callers.

| Owner | Temporarily allowed direct Data internals |
|---|---|
| `internal/apps/miniappapi` | `internal/domain`, `internal/repositories` |
| `internal/apps/adminapi` | `internal/domain`, `internal/repositories` |
| `cmd/api` | `internal/repositories`, `internal/platform/database`, `internal/platform/dbmigration` |
| `cmd/admin-api` | `internal/repositories`, `internal/platform/database`, `internal/platform/dbmigration` |

No Miniapp/Admin application package currently imports connectors, parsers, scheduler, runtime, or graph database. Platform currently has no business application/domain/repository/HTTP import; the architecture test now makes that invariant permanent.

The only direct production importers before retirement are:

| Imported package | Exact production importers |
|---|---|
| `internal/apps/ingestion/runtime` | `cmd/ingest-smoke`, `cmd/ingestion-scheduler`, `cmd/source-ingest`, `internal/apps/ingestion/scheduler` |
| `internal/apps/ingestion/scheduler` | `cmd/ingestion-scheduler` |
| `internal/apps/ingestion/health` | none; package marker only |

The pre-retirement architecture test rejects any new importer and freezes the six retiring paths plus the exact runtime config markers. Package 8 must replace this positive inventory with absence assertions.

## Test Keep/Remove/Replace Classification

| Classification | Exact tests/cases | Count before | Production reason |
|---|---|---:|---|
| remove with retired commands | `cmd/ingestion-scheduler/main_test.go`, `cmd/source-ingest/main_test.go` | 6 | their only production commands are removed |
| remove with scheduler app | scheduler `planner_test.go`, `service_test.go` | 6 | scheduler planning and run orchestration are removed |
| remove with runtime app | runtime `ingestion_job_test.go`, `ingestion_smoke_test.go`, `ai_web_research_runtime_test.go` | 7 | connector-to-parser-to-writer orchestration and real smoke entry are removed |
| remove with scheduler repositories | `repositories/scheduler_test.go`, `ingestion_run_test.go`, scheduler Postgres integration case | 4 | repositories no longer have a supported caller; migration test remains |
| remove cases from mixed domain file | five scheduler/run validation cases in `domain/models_test.go` | 5 | matching scheduler/run domain symbols are removed; all retained domain cases stay |
| remove cases from mixed core file | source registry, limiter, local raw object store, raw writer cases in `ingestion/core/core_test.go` | 4 | matching runtime-only symbols are removed; connector/parser contracts stay |
| replace Admin cases | four scheduler config/run cases in `adminapi/router_test.go` | 4 | replaced by auth/410/no-repository/no-write tombstone tests |
| remove Admin scheduler-only UI cases | `api/scheduler.test.ts` (4), `SchedulerSettings.test.tsx` (6) | 10 | matching client/page are removed |
| rewrite mixed Admin UI assertions | `DataIngestionCenter.test.tsx` scheduler mocks/tab (3 cases), `App.test.tsx` setup, one scheduler layout assertion in `minimalDashboardConformance.test.ts` | 3 cases plus shared setup/assertion | retain raw/event/source/auth/design-system coverage while deleting only scheduler behavior |
| keep Miniapp UI cases | all existing `frontend/miniapp/src` UI tests | 18 | external Miniapp contract remains supported |
| keep retained backend coverage | migration, raw/source/event repository, event-import, connector/parser/sourcecatalog, idempotency/transaction/security tests | all non-retired cases | these validate retained capabilities and may not be removed for speed |

The 23 “direct backend tests” are 6 command + 6 scheduler + 7 runtime + 4 scheduler-repository cases. The wider old-behavior count is 36 when the five mixed domain, four mixed core and four Admin cases are included. Replacement/new import, 410 and architecture tests are counted separately in the final before/after report.

## Testdata And Versioned Asset Audit

| Files | Reference evidence | Action |
|---|---|---|
| `backend/testdata/event-import/reviewed-outbox-v1.json` | loaded by `cmd/event-import/main_test.go` | keep |
| 12 files under `backend/internal/architecture/testdata/task_design/**` | loaded through `task_design_lint_test.go` | keep |
| 8 JSON files under `backend/data/source_catalogs/**` | loaded by sourcecatalog default seed-path tests and `source-seed` | keep as versioned business assets, not testdata |
| 2 Markdown files under `backend/data/prompts/ingestion/**` | loaded/validated by promptstore and AI connector tests | keep as versioned connector assets, not testdata |

No fixture is zero-reference or exclusively supports a retired behavior. Expected testdata deletion is therefore exactly zero unless later evidence changes and replacement coverage is added.

## Reproduction Commands

```text
find backend -type f -name '*_test.go' | wc -l
rg '^func Test' backend --glob '*_test.go' | wc -l
find backend -type f -path '*/testdata/*' | sort
rg '\b(it|test)\(' frontend/admin/src --glob '*.{test,spec}.{ts,tsx}' | wc -l
rg '\b(it|test)\(' frontend/miniapp/src --glob '*.{test,spec}.{ts,tsx}' | wc -l
find backend/migrations -maxdepth 1 -type f -name '*.sql' | wc -l
find backend/migrations -maxdepth 1 -type f -name '*.sql' ! -name '000022_add_raw_document_import_receipts.sql' | sort | xargs shasum -a 256 | shasum -a 256
shasum -a 256 backend/migrations/000022_add_raw_document_import_receipts.sql
rg -n 'testdata|reviewed-outbox-v1|task_design' backend --glob '*.{go,md}'
rg -n 'NewIngestionJob|NewIngestionSmokeRunner|NewSourceRegistry|NewRawDocumentWriter|NewRateLimiter|LoadSchedulerConfig|SaveSchedulerConfig|CreateIngestionRun|RecordIngestionRunSource' backend --glob '*.go'
```

其中`000022`独立hash命令只在Package 4创建artifact后运行；Package 2/current amendment checkpoint仍应确认该文件不存在。历史聚合命令显式排除`000022`，因此Package 4之后仍可复现冻结的21-file hash。

Package validation records the fresh test, diff, scope, and secret-check results in the Package 2 checkpoint commit message/review package; this evidence file intentionally contains no credentials or environment values.
