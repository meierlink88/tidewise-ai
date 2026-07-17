# Package 11 Apply-final Review Evidence

## Result And Scope

**PASS — ready for independent Apply-final Review.** The reviewed implementation starts at change baseline `3f0f779d2c332a74f31fd398adb47adb306a60c3` and Package 11 starts from accepted checkpoint `4cb2524e142cfa3b57e799748aa405fb8b84cc9f` on branch `codex/establish-data-service-bff-boundaries`.

Package 11 changed only this evidence file and the four task checkboxes in `tasks.md`. It did not change production code, API/schema artifacts, migrations, tests, frontend code, service assets, configuration, infrastructure, database state, Neo4j, deployment, UAT/prod/shared, or the external `agent-run` repository. No migration, seed, import, projection, synthetic/business DML, role/password/owner/grant change, service start, environment write, Sync, Archive, Deliver, push or PR operation ran.

## 11.1 Artifact-to-Implementation Audit

| Contract area | Result | Final evidence |
|---|---|---|
| Three-service topology | PASS | Data owns PostgreSQL/Neo4j access and repositories; Miniapp and Admin are service-owned BFFs with consumer-owned typed HTTP clients. Architecture tests reject BFF imports of Data implementation/repositories and local Compose gives datastore configuration only to Data. |
| Data API and typed clients | PASS | Namespace is `/internal/data/v1`; OpenAPI fixes scopes, DTOs, cursors, request IDs, structured errors and timeout budgets. Miniapp/Admin handwritten clients have independent schema-drift and HTTP tests; no generator was introduced. |
| Research semantic authority | PASS | OpenAPI, Data handler golden, typed clients and Miniapp public mapping preserve the published `research-theme-anchor-foundation` contract, including distinct theme `impact_summary` and anchor `relation_summary` DTOs and the frozen enums/nonempty natural-language fields. |
| Raw receipt/import/status | PASS | `raw_document_import_receipts` has the frozen seven-column immutable contract. Whole-batch validation precedes one PostgreSQL transaction that writes raw documents plus receipt atomically. Caller-scoped key/hash conflict, exact result replay, status lookup, deterministic membership and zero-partial-success behavior are covered. Both receipt adapters take transaction-scoped advisory locks before plain `SELECT`; no row-lock clause or UPDATE privilege is required. |
| Reviewed-event import | PASS | Existing event receipt semantics remain separate from raw receipts; CLI compatibility uses the scoped Data HTTP import with a 15-second default and no mutation retry. Event/source/tag/review contracts were not collapsed into the raw receipt table. |
| Scheduler/runtime retirement | PASS | Tidewise-owned `ingestion-scheduler`, `source-ingest`, `ingest-smoke`, scheduler/runtime/health orchestration and runtime-only scheduler/run state are removed. No replacement scheduler, worker or source runner was added to Tidewise. Historical scheduler tables, rows and migrations remain. |
| Retained ingestion/data capabilities | PASS | Connectors, parsers, registry, `EnvCredentialResolver`, source catalog/assets, prompts, source seed, event import, raw import/status, repositories, historical migrations/tables and graph projector remain. Retained capability suites pass. |
| Agent handoff and 410 compatibility | PASS with operational follow-up | Future scheduling/execution belongs to external `agent-run`; this repository neither edits it nor supplies an in-repo substitute. Three authenticated Admin scheduler endpoints return machine-readable `410 Gone` with zero Data/DB calls. The required one actual deployment window and observed zero-use retirement are deployment-stage work and were not exercised here. |
| Independent R2 layers | PASS | Package 5 applied migration `000022` and rollback-only receipt integration without roles/credentials. Package 10 separately established local-only owners/roles/minimum privileges and process-only Data credential cutover. Neither layer is conflated with the other. |
| Tests and fixtures | PASS | Exact before/after formulas appear below. All 13 testdata files remain referenced; deletion and dangling-reference counts are zero. |

No business, API, database, permission or environment drift was found. No production repair was needed in Package 11.

## 11.2 Migration And Stateful Before/After Manifest

| Assertion | Before | After | Result |
|---|---:|---:|---|
| Repository SQL migration files | 21 | 22 | PASS; the only addition is `000022_add_raw_document_import_receipts.sql`. |
| Historical `000001`-`000021` aggregate SHA-256 | `2ed0dd004ab3b0bad633af5f0107a9ddffa28b83422b643f821c2cd47fb02dfc` | same | PASS; all 21 historical files are byte-for-byte unchanged. |
| `000022` SHA-256 | absent | `3966172b267b1cb27e1538fde35a4125734201628c9ea308d62e0a1314685c26` | PASS; repository artifact matches the applied migration reviewed in Package 5. |
| Local applied positive migrations | `1..21` | `1..22` | PASS; version 22 is present exactly once and no higher applied version exists. |
| Public schema structure | Package 5 frozen baseline | `43` tables / `373` columns / `203` constraints / `104` indexes / `1` public function / `1` non-internal public trigger | PASS; only approved migration/ownership effects are present. |
| Frozen 41-table business counts | Package 5 baseline | exact Package 10 post-DCL manifest | PASS; fresh server-enforced read-only audit matched every table. |
| Raw receipt rows | `0` before apply and after rollback-only integration | `0` | PASS; no seed/import/business data persisted. |

The receipt table remains exactly: `id uuid`, `caller_identity text`, `idempotency_key text`, `payload_hash character(64)`, `raw_document_ids uuid[]`, `result_payload jsonb`, `imported_at timestamptz DEFAULT now()`, all non-null; seven named constraints, three named indexes, immutable mutation-prevention function and enabled statement trigger remain present. Existing receipt row means completed immutable result; missing row means unknown/not committed; same caller/key with another canonical hash returns conflict without mutation.

The fresh Package 11 read-only run of `/private/tmp/tidewise_p10_post.sql` returned `P10_POST_DCL_PASS`. It proves local PostgreSQL 16 / `tidewise_local`, `transaction_read_only=on`, ledger 22, version 22 once, complete role/owner/grant/PUBLIC/negative-privilege contract, the structural counts above, all 41 business counts and receipt rows zero. Package 11 performed no database write.

Migration recovery remains fail-closed and forward-only: the accepted repo-external pre-role backup is `/private/tmp/tidewise-p10-review-119f735e6263-pre-roles.dump`, size `1,379,080` bytes, SHA-256 `434e1eb2cc64ba145160d20a36b8d9852b77ea0309009d060337212f231c1e66`, listable as 262 entries. It was not restored and is not claimed restore-tested. A failure requires stop and explicit recovery approval; no automatic Down, retry, restore, reverse DCL or forward-fix is permitted.

## Test Before/After Manifest

### Go

Exact repository inventory is:

```text
114 files / 541 Test functions
- 10 deleted files / 23 Test functions
+ 21 added files / 74 Test functions
- 7 net Test functions across retained modified files
= 125 files / 585 Test functions
```

An identity comparison gives the equivalent contract formula `541 - 50 removed identities + 94 replacement/new identities = 585`. The structural per-file accounting is authoritative for counts; identity differences additionally capture renamed/replaced assertions in retained files.

Deleted files are production-removal paired, not time-saving deletions:

| Deleted test file | Tests | Production reason/replacement |
|---|---:|---|
| `backend/cmd/ingestion-scheduler/main_test.go` | 3 | Command removed; architecture/reference tests forbid its return. |
| `backend/cmd/source-ingest/main_test.go` | 3 | Command removed; Data raw import and external-agent boundary replace the old entry path. |
| `backend/internal/apps/ingestion/runtime/ai_web_research_runtime_test.go` | 2 | Retired runtime orchestration removed; connector/parser tests remain. |
| `backend/internal/apps/ingestion/runtime/ingestion_job_test.go` | 2 | Retired job runtime removed; no replacement worker was added. |
| `backend/internal/apps/ingestion/runtime/ingestion_smoke_test.go` | 3 | Retired smoke runtime removed; retained contracts are covered at Data/import boundaries. |
| `backend/internal/apps/ingestion/scheduler/planner_test.go` | 4 | Planner production removed; architecture tests guard owner handoff/no substitute. |
| `backend/internal/apps/ingestion/scheduler/service_test.go` | 2 | Scheduler service production removed; 410 and absence tests replace old behavior. |
| `backend/internal/repositories/ingestion_run_postgres_test.go` | 1 | Runtime-only repository removed; historical tables/migrations stay. |
| `backend/internal/repositories/ingestion_run_test.go` | 1 | Runtime-only in-memory repository behavior removed. |
| `backend/internal/repositories/scheduler_test.go` | 2 | Scheduler repository removed; no supported scheduler write contract remains. |

Added files and replacement/new test counts:

| Added test file | Tests | Coverage |
|---|---:|---|
| `backend/internal/architecture/service_assets_test.go` | 3 | Three service assets/CI/local consumers. |
| `backend/internal/architecture/service_boundary_transition_test.go` | 2 | BFF/Data and scheduler-retirement boundaries. |
| `backend/internal/architecture/service_skeleton_test.go` | 4 | Service ownership/import rules. |
| `backend/internal/platform/dbmigration/raw_document_import_receipt_contract_test.go` | 4 | Static `000022` contract. |
| `backend/internal/platform/dbmigration/readiness_test.go` | 2 | Read-only fail-closed readiness. |
| `backend/internal/repositories/admin_query_postgres_test.go` | 1 | Retained admin projection contract. |
| `backend/services/adminportal/dataclient/client_test.go` | 6 | Auth/request-ID/query/error/timeout HTTP client. |
| `backend/services/adminportal/dataclient/contract_drift_test.go` | 1 | Admin/OpenAPI drift. |
| `backend/services/adminportal/runtime_config_test.go` | 2 | No DB credential and bounded client config. |
| `backend/services/adminportal/service_test.go` | 3 | 410/zero-call and retained Data calls. |
| `backend/services/data/api/openapi_test.go` | 5 | Namespace/scopes/DTO/import/status/reference contract. |
| `backend/services/data/cmd/main_test.go` | 1 | Data wiring and read-only readiness. |
| `backend/services/data/internalapi/handler_test.go` | 8 | Auth/scope/request-ID/cursor/errors/goldens/imports. |
| `backend/services/data/rawimport/postgresstore/store_test.go` | 5 | Transaction/advisory/plain-select/conflict/status contracts. |
| `backend/services/data/rawimport/service_test.go` | 7 | Atomic validation/replay/conflict/source attribution. |
| `backend/services/data/research/service_test.go` | 4 | Authoritative research DTO/query behavior. |
| `backend/services/data/service_test.go` | 2 | Data facade/health/readiness. |
| `backend/services/miniapp/dataclient/client_test.go` | 8 | Auth/request-ID/DTO/cursor/error/timeout HTTP client. |
| `backend/services/miniapp/dataclient/contract_drift_test.go` | 1 | Miniapp/OpenAPI drift. |
| `backend/services/miniapp/runtime_config_test.go` | 2 | No DB credential and bounded client config. |
| `backend/services/miniapp/service_test.go` | 3 | Public mapping and one aggregate call. |

Retained modified Go files account for the remaining net `-7`: `cmd/event-import` `5→8`, Admin router `10→8`, ingestion core `6→2`, eventimport service `7→7`, Miniapp research router `2→3`, Miniapp research service `5→4`, architecture core/cmd/local-infra/repository-organization each unchanged, config `12→12`, domain `15→10`, migration source `2→2`, event-import Postgres `2→3`. Removed domain/core tests correspond only to removed scheduler/runtime symbols; retained domain, repository, connector, API, migration, idempotency, transaction and security behavior remains covered.

### Frontend

```text
8 files / 44 cases
- scheduler API test 1 file / 4 cases
- SchedulerSettings test 1 file / 6 cases
= 6 files / 34 cases
```

Admin now has 5 files / 16 cases: `App` 2, `DataIngestionCenter` 3, `DataIngestionCenter` page 3, minimal-dashboard conformance 7, Vite config 1. Miniapp has 1 Vitest file / 18 cases. Mixed Admin tests were count-neutral rewrites from scheduler tabs to source/raw/event tabs; unrelated React transitive packages and `package-lock.json` were retained.

### Testdata And Fixtures

Testdata is exactly `13→13`, deletion zero, dangling-reference zero:

- 12 task-design Markdown fixtures in six directories are loaded by the architecture fixture loader;
- `backend/testdata/event-import/reviewed-outbox-v1.json` is loaded by two test files at seven literal reference sites.

No fixture was removed merely to shorten tests.

## Final Repository Manifest

The final baseline-to-Package-11 manifest has **143 path entries**: `A75 / D28 / M39 / R053=1`. Mutually exclusive groups are production 48, tests 50, OpenSpec 12, evidence 16, assets/CI/local 7, config 7 and docs/rules 3. Package 11 itself is exactly two paths: this new evidence file plus the checkbox-only update to `tasks.md`.

The exact high-level removal/retention mapping is:

- 26 direct scheduler/runtime/Admin scheduler UI paths are deleted: three commands, two command tests, health doc, three runtime production plus three runtime tests, three scheduler production plus two scheduler tests, two scheduler/run repositories plus three tests, and scheduler client/page plus their two tests;
- two split local Compose files are deleted only after a unified three-service Compose consumer replaces them;
- runtime-only domain/config/memory/core symbols are removed only where caller scans are zero;
- connector, parser, source-catalog, prompt, source-seed, event-import, raw-import/status, repository, graph-projector, migration and testdata paths remain, with the retained adaptations described above;
- `.github/workflows/deploy-uat.yml`, `infra/uat/**`, prod/shared and `agent-run/**` have zero diff.

The exact 143-path list is reproducible with `git diff --name-status 3f0f779d2c332a74f31fd398adb47adb306a60c3..HEAD`; Package 11 records the full categorized result without creating a generated repository artifact:

```text
Production 48: M14 D14 A20
Tests 50: M17 D12 A21
OpenSpec 12: A12
Evidence 16: A16
Assets/CI/local 7: M1 D2 A3 R053=1
Config 7: M4 A3
Docs/rules 3: M3
```

Exact path membership:

```text
[Production]
M backend/cmd/admin-api/main.go
M backend/cmd/api/main.go
M backend/cmd/event-import/main.go
D backend/cmd/ingest-smoke/main.go
D backend/cmd/ingestion-scheduler/main.go
D backend/cmd/source-ingest/main.go
M backend/internal/apps/adminapi/router.go
M backend/internal/apps/ingestion/core/core.go
M backend/internal/apps/ingestion/eventimport/service.go
D backend/internal/apps/ingestion/health/doc.go
D backend/internal/apps/ingestion/runtime/doc.go
D backend/internal/apps/ingestion/runtime/ingestion_job.go
D backend/internal/apps/ingestion/runtime/ingestion_smoke.go
D backend/internal/apps/ingestion/scheduler/doc.go
D backend/internal/apps/ingestion/scheduler/planner.go
D backend/internal/apps/ingestion/scheduler/service.go
M backend/internal/apps/miniappapi/research_service.go
M backend/internal/config/config.go
M backend/internal/domain/models.go
A backend/internal/platform/dbmigration/readiness.go
A backend/internal/platform/servicehttp/servicehttp.go
M backend/internal/repositories/admin_query.go
M backend/internal/repositories/event_import_postgres.go
D backend/internal/repositories/ingestion_run.go
M backend/internal/repositories/memory.go
D backend/internal/repositories/scheduler.go
A backend/migrations/000022_add_raw_document_import_receipts.sql
A backend/services/adminportal/cmd/main.go
A backend/services/adminportal/dataclient/http.go
A backend/services/adminportal/dataclient/port.go
A backend/services/adminportal/runtime_config.go
A backend/services/adminportal/service.go
A backend/services/data/api/openapi.yaml
A backend/services/data/cmd/main.go
A backend/services/data/internalapi/handler.go
A backend/services/data/rawimport/postgresstore/store.go
A backend/services/data/rawimport/service.go
A backend/services/data/research/service.go
A backend/services/data/service.go
A backend/services/miniapp/cmd/main.go
A backend/services/miniapp/dataclient/http.go
A backend/services/miniapp/dataclient/port.go
A backend/services/miniapp/runtime_config.go
A backend/services/miniapp/service.go
D frontend/admin/src/api/scheduler.ts
M frontend/admin/src/pages/DataIngestionCenter.tsx
D frontend/admin/src/pages/SchedulerSettings.tsx
M frontend/admin/src/styles/app.css

[Tests]
M backend/cmd/event-import/main_test.go
D backend/cmd/ingestion-scheduler/main_test.go
D backend/cmd/source-ingest/main_test.go
M backend/internal/apps/adminapi/router_test.go
M backend/internal/apps/ingestion/core/core_test.go
M backend/internal/apps/ingestion/eventimport/service_test.go
D backend/internal/apps/ingestion/runtime/ai_web_research_runtime_test.go
D backend/internal/apps/ingestion/runtime/ingestion_job_test.go
D backend/internal/apps/ingestion/runtime/ingestion_smoke_test.go
D backend/internal/apps/ingestion/scheduler/planner_test.go
D backend/internal/apps/ingestion/scheduler/service_test.go
M backend/internal/apps/miniappapi/research_router_test.go
M backend/internal/apps/miniappapi/research_service_test.go
M backend/internal/architecture/architecture_test.go
M backend/internal/architecture/cmd_imports_test.go
M backend/internal/architecture/local_infra_test.go
M backend/internal/architecture/repository_organization_test.go
A backend/internal/architecture/service_assets_test.go
A backend/internal/architecture/service_boundary_transition_test.go
A backend/internal/architecture/service_skeleton_test.go
M backend/internal/config/config_test.go
M backend/internal/domain/models_test.go
A backend/internal/platform/dbmigration/raw_document_import_receipt_contract_test.go
A backend/internal/platform/dbmigration/readiness_test.go
M backend/internal/platform/dbmigration/source_test.go
A backend/internal/repositories/admin_query_postgres_test.go
M backend/internal/repositories/event_import_postgres_test.go
D backend/internal/repositories/ingestion_run_postgres_test.go
D backend/internal/repositories/ingestion_run_test.go
D backend/internal/repositories/scheduler_test.go
A backend/services/adminportal/dataclient/client_test.go
A backend/services/adminportal/dataclient/contract_drift_test.go
A backend/services/adminportal/runtime_config_test.go
A backend/services/adminportal/service_test.go
A backend/services/data/api/openapi_test.go
A backend/services/data/cmd/main_test.go
A backend/services/data/internalapi/handler_test.go
A backend/services/data/rawimport/postgresstore/store_test.go
A backend/services/data/rawimport/service_test.go
A backend/services/data/research/service_test.go
A backend/services/data/service_test.go
A backend/services/miniapp/dataclient/client_test.go
A backend/services/miniapp/dataclient/contract_drift_test.go
A backend/services/miniapp/runtime_config_test.go
A backend/services/miniapp/service_test.go
M frontend/admin/src/App.test.tsx
D frontend/admin/src/api/scheduler.test.ts
M frontend/admin/src/pages/DataIngestionCenter.test.tsx
D frontend/admin/src/pages/SchedulerSettings.test.tsx
M frontend/admin/src/styles/minimalDashboardConformance.test.ts

[OpenSpec]
A openspec/changes/establish-data-service-bff-boundaries/.openspec.yaml
A openspec/changes/establish-data-service-bff-boundaries/design.md
A openspec/changes/establish-data-service-bff-boundaries/proposal.md
A openspec/changes/establish-data-service-bff-boundaries/specs/admin-console/spec.md
A openspec/changes/establish-data-service-bff-boundaries/specs/backend-service-boundaries/spec.md
A openspec/changes/establish-data-service-bff-boundaries/specs/backend-subsystem-boundaries/spec.md
A openspec/changes/establish-data-service-bff-boundaries/specs/data-ingestion-layer/spec.md
A openspec/changes/establish-data-service-bff-boundaries/specs/event-import-and-tag-catalog/spec.md
A openspec/changes/establish-data-service-bff-boundaries/specs/ingestion-scheduler/spec.md
A openspec/changes/establish-data-service-bff-boundaries/specs/persistence-and-contracts/spec.md
A openspec/changes/establish-data-service-bff-boundaries/specs/technical-architecture/spec.md
A openspec/changes/establish-data-service-bff-boundaries/tasks.md

[Evidence]
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-10-applied-recovery-review.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-10-credential-recovery-cutover.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-10-receipt-lock-compatibility.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-10-role-credential-review-refresh.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-10-role-credential-review.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-11-apply-final-review.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-2-inventory.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-3-service-skeleton.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-4-data-api-import.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-5-order-1-preflight.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-5-order-2-apply-integration.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-5-raw-receipt-schema-review.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-6-miniapp-bff-decoupling.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-7-admin-retirement-decoupling.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-8-scheduler-runtime-retirement.md
A openspec/changes/establish-data-service-bff-boundaries/evidence/package-9-assets-local-ci.md

[Assets/CI/local]
M .github/workflows/ci.yml
A backend/services/adminportal/Dockerfile
R053 backend/Dockerfile backend/services/data/Dockerfile
A backend/services/miniapp/Dockerfile
D infra/local/docker-compose.neo4j.yaml
D infra/local/docker-compose.postgres.yaml
A infra/local/docker-compose.yaml

[Config]
M backend/config/config.local.yaml
M backend/config/config.prod.yaml
M backend/config/config.uat.yaml
A backend/services/adminportal/config/config.local.yaml
A backend/services/data/config/config.local.yaml
A backend/services/miniapp/config/config.local.yaml
M infra/local/.env.example

[Docs/rules]
M .agents/backend-boundaries.md
M backend/internal/apps/ingestion/README.md
M infra/local/README.md
```

This membership was cross-checked against every prior package evidence. The scheduler/runtime deletions and retained paths above match Package 8; service assets/local/CI paths match Package 9; migration/API/client/service paths match Packages 3-7; role/credential work remains evidence/task-only in the repository.

## 11.3 Fresh Validation

| Command/check | Result |
|---|---|
| `GOCACHE=/private/tmp/tidewise-p11-go-cache go test ./... -count=1` from `backend/`, DB integration URL unset | PASS; all backend packages. No real database write path ran. |
| Admin `npm test` | PASS; 5 files / 16 tests. |
| Admin `npm run build` | PASS; TypeScript + Vite, 41 modules. |
| Miniapp `npm test` | PASS; 3 pure Node contract scripts plus 1 Vitest file / 18 tests. |
| Miniapp `npm run typecheck` | PASS. |
| Miniapp `npm run build:weapp` and `npm run build:tt` | PASS; only pre-existing Sass `@import` deprecation warnings. |
| Data/Miniapp/Admin Go binary builds | PASS; SHA-256 respectively `9ae60642207758bafe6a9be79aada992ad6220f8db8758fa8bcf5f7451bbcdbe`, `458cc2e66a0651930ba4d88715fb2afa37d43dd32eafb54a447ae515e069d4c0`, `0812e4b1c741b02c34f48415519f61980b9d9b1ed2fc28f5ef2b87d4`. Output is repo-external under `/private/tmp`. |
| Three service image builds with `--pull=false` | PASS; local image IDs Data `sha256:1acc304510139cc4f6320c7dcf024910cf2efcf6c196cf81f3391fe846190d37`, Miniapp `sha256:e1dacefab164dcaf536bfe4245ed6236d4198bb66fb4afe09c10946231bb7748`, Admin `sha256:8f73d5372e2b15b771524957254f7857bbc8adb363ad160aa9342704eb4175cd`. Each has one service-owned CMD and combined `/healthz` + `/readyz` probe. Images were not run or pushed. |
| `go test ./internal/architecture -count=1` | PASS; architecture/import/assets/CI/reference/task-fixture/testdata-loader checks. |
| `go test ./services/data/api ./services/miniapp/dataclient ./services/adminportal/dataclient -count=1` | Data API PASS; the first sandboxed client run was blocked only because `httptest` could not bind loopback. The exact two client packages were rerun with loopback permission and both PASS, with no code change. |
| `docker compose --env-file .env.example -f docker-compose.yaml config --quiet` from `infra/local` | PASS; dry config only, no `up`, service start or datastore connection. |
| `/private/tmp/tidewise_p10_post.sql` under `BEGIN READ ONLY` | PASS; `P10_POST_DCL_PASS`. |
| Historical migration aggregate and `000022` SHA-256 commands | PASS; exact frozen values above. |
| `openspec validate establish-data-service-bff-boundaries --strict` | PASS. |
| `OPENSPEC_TASK_LINT_CHANGE=establish-data-service-bff-boundaries go test ./internal/architecture -run '^TestOpenSpecTaskDesignLint$' -count=1` | PASS. |
| `openspec status --change establish-data-service-bff-boundaries --json` | PASS; 4/4 artifacts complete. |
| Test inventory/reference scans | PASS; Go `125/585`, frontend `6/34`, testdata `13`, zero dangling references. |
| `git diff --check`, staged diff check, scoped manifest, whitespace and high-risk secret scans | PASS; final checkpoint is tasks + this evidence only and no credential/DSN/token/private key is present. |

## 11.4 Risk, Compatibility And Stop Review

| Risk | Final assessment and recovery/stop condition |
|---|---|
| Compatibility | Public Miniapp behavior is golden-tested. Admin retains source/raw/event behavior; only the three approved authenticated scheduler routes become structured 410. Keep tombstones for one actual deployment window, observe zero use, then retire only under a separately reviewed change. |
| Performance | Each BFF request has a bounded Data call count and timeout; no unbounded retries were added. The extra HTTP boundary is intentional and covered by client/service tests, but production latency/load was not exercised because deployment is excluded. |
| Authentication/secrets | Data service identity/scope and request-ID propagation are tested. Miniapp/Admin contain no datastore credentials. Package 10 uses least-privilege local roles; runtime has exact SELECT/INSERT only and no UPDATE/DELETE/TRUNCATE/sequence/function privilege. No secret is committed. |
| Transaction/race | Raw batch validation precedes one atomic transaction; advisory transaction locks serialize caller/key races and winner re-read uses plain SELECT. Reviewed-event keeps its separate receipt contract. SQL-mock, unit and rollback-only local integration evidence cover constraints, immutability, rollback and two-connection locking. A persistent winner-commit fixture was deliberately not created. |
| Migration/schema | Historical 21 files/tables/data remain; `000022` is forward-only and immutable. Failures stop; no retry, auto-Down, restore or forward-fix is authorized. The accepted backup is readable/listable but not restore-tested. |
| Scheduler stop/agent handoff | Tidewise has no scheduler/runtime substitute. Connectors/parsers/import contracts remain available to the Data boundary and future external agent integration. External `agent-run` implementation/delivery was not part of this repository and remains operational follow-up. |
| Environment/deployment | UAT/prod/shared, deploy workflow execution, Neo4j runtime, real ingestion/Agent execution and the actual 410 observation window were intentionally not validated. Any environment-specific drift requires a new reviewed layer; Package 11 must not infer approval. |

Apply stops at this checkpoint. Independent Leader approval is required before Sync, Archive or Deliver; push, PR and deployment remain unauthorized.
