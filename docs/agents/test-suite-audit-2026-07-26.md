# Test Suite Risk-Boundary Audit

Date: 2026-07-26

Issue: [#108 精简全仓测试套件并统一风险边界 TDD 规范](https://github.com/meierlink88/tidewise-ai/issues/108)

Baseline: `main` at `ef27de7`, after Issue #105 was merged

## Outcome

This cleanup keeps production behavior, APIs, schemas, migrations, runtime
configuration, and deployment topology unchanged. It removes tests that restated
implementation details, consolidates repeated repository invariants and migration
source checks, and makes CI select test suites by changed application and risk
boundary.

The repository now has 130 test files instead of 181, a reduction of 51 files
(28.2%). Go test functions fell from 692 to 502 (27.5%), and frontend test cases
fell from 91 to 70 (23.1%).

## Inventory

### Test files by application

| Application           |  Before |   After |  Change |
| --------------------- | ------: | ------: | ------: |
| Data Service Backend  |      97 |      61 |     -36 |
| Miniapp Backend       |      17 |      13 |      -4 |
| Admin Portal Backend  |      14 |      10 |      -4 |
| AgentRun Backend      |      25 |      22 |      -3 |
| Miniapp Frontend      |      14 |      12 |      -2 |
| Admin Portal Frontend |       6 |       4 |      -2 |
| Repository contracts  |       8 |       8 |       0 |
| **Total**             | **181** | **130** | **-51** |

### Backend test files by seam

The API, Biz, configuration, and retained observable Server/Service seams are
unchanged. Reductions are concentrated in Architecture, Data, command glue, and
duplicate runtime-contract tests.

| Application  | Seam                          |        Before |         After |
| ------------ | ----------------------------- | ------------: | ------------: |
| Data Service | API / Biz                     |        3 / 34 |        3 / 34 |
| Data Service | Data / Migration              |            48 |            15 |
| Data Service | Cmd / Conf / Server / Service | 6 / 1 / 1 / 3 | 5 / 1 / 1 / 2 |
| Data Service | Architecture                  |             1 |             0 |
| Miniapp      | API / Biz                     |         2 / 2 |         2 / 2 |
| Miniapp      | Data                          |             4 |             3 |
| Miniapp      | Cmd / Conf / Server / Service | 1 / 1 / 5 / 1 | 1 / 1 / 4 / 0 |
| Miniapp      | Architecture                  |             1 |             0 |
| Admin Portal | API / Biz                     |         2 / 1 |         2 / 1 |
| Admin Portal | Data                          |             5 |             3 |
| Admin Portal | Cmd / Conf / Server           |     1 / 1 / 3 |     1 / 1 / 2 |
| Admin Portal | Architecture                  |             1 |             0 |
| AgentRun     | API / Biz                     |         1 / 7 |         1 / 7 |
| AgentRun     | Data                          |             8 |             8 |
| AgentRun     | Cmd / Conf / Server / Service | 4 / 1 / 1 / 1 | 3 / 1 / 1 / 1 |
| AgentRun     | Architecture                  |             2 |             0 |

The Miniapp retains 12 behavior, navigation, state, platform, API-adapter, and
component test files. The Admin Portal retains four page, application, and
API-adapter test files.

## Deletion classification

Every removed file belongs to one of the Issue #108 classifications:

| Classification                 |  Files | Stronger retained guarantee                                                                             |
| ------------------------------ | -----: | ------------------------------------------------------------------------------------------------------- |
| Duplicated by a stronger seam  |      9 | Central repository architecture rules, generated/OpenAPI contract checks, and existing Biz/API behavior |
| Implementation-only            |     31 | Compilation, vet/typecheck, binary builds, real boundary tests, or user-visible frontend behavior       |
| Consolidated                   |     11 | One ordered forward migration-chain suite with critical table and destructive-reset checks              |
| Obsolete with removed behavior |      0 | No product behavior was removed                                                                         |
| **Total**                      | **51** |                                                                                                         |

### Duplicated by a stronger seam

- Five per-application Architecture tests were replaced by centralized rules in
  `scripts/ci/repository-contracts/service_skeleton_test.go`. The central suite
  checks required Kratos layers, cross-service import boundaries, Wire exclusion,
  Data-to-API exclusion, the Gin exclusion in binary closures, AgentRun's Eino
  dependency closure, and concrete Eino provider ownership.
- Three `openapi_runtime_test.go` files were removed from Data Service, Miniapp,
  and Admin Portal. Existing API, Server, generated contract, and smoke seams
  retain externally observable HTTP guarantees.
- `miniapp/backend/internal/service/research_test.go` duplicated request/result
  mapping already protected at the Biz and API boundaries.

### Implementation-only

- Removed command-construction tests:
  `analyse-data-service/backend/cmd/dbmigrate/main_test.go` and
  `agent-run/backend/cmd/artifacts/main_test.go`.
- Removed Admin Portal Wire assembly tests and Miniapp's mechanical Data contract
  mapping test.
- Removed frontend CSS-token conformance, Vite plumbing, Taro config plumbing,
  and duplicate mock-port tests.
- Removed 13 Data Service Entity Seed tests based on in-memory repositories,
  `sqlmock`, source-shape assertions, or per-method behavior. The retained real
  PostgreSQL concurrency and transactional boundaries remain.
- Removed nine Data Service PostgreSQL adapter tests based on `sqlmock`,
  in-memory doubles, or duplicated mapping matrices. Retained tests exercise the
  meaningful SQL, transaction, constraint, pagination, and import boundaries
  against PostgreSQL when `TIDEWISE_TEST_DATABASE_URL` is available.

Despite names containing `postgres`, the deleted
`entityseed/convergence_postgres_test.go` and the deleted adapter files did not
connect to a real PostgreSQL server. No real PostgreSQL integration test was
removed solely to reduce the count.

### Consolidated migration coverage

Eleven migration source-contract and runner files were consolidated into
`internal/data/dbmigration/migrations_test.go`. The retained suite now proves:

- migrations are versioned, ordered, and contain Goose Up/Down markers;
- the complete forward chain applies to an isolated real PostgreSQL database;
- critical current tables, constraints, and indexes exist after the chain;
- destructive database-wide reset statements and unreviewed row deletion are
  forbidden.

`readiness_test.go` remains as a Lifecycle seam: it proves startup readiness is
read-only and fails closed when the migration ledger is missing or migrations
remain pending.

### File-level deletion manifest

The following 51 entries are the complete deletion manifest. No deleted test file
is represented only by an aggregate count.

Duplicated by a stronger seam:

- `admin-portal/backend/internal/architecture/architecture_test.go` — central
  repository Architecture seam.
- `agent-run/backend/internal/architecture/ci_workflow_test.go` — central
  repository CI/Architecture seam.
- `agent-run/backend/internal/architecture/layout_test.go` — central repository
  Architecture and Eino ownership seam.
- `analyse-data-service/backend/internal/architecture/architecture_test.go` —
  central repository Architecture seam.
- `miniapp/backend/internal/architecture/architecture_test.go` — central
  repository Architecture seam.
- `admin-portal/backend/internal/server/openapi_runtime_test.go` — retained
  Admin API, generated OpenAPI, Server, and smoke seams.
- `analyse-data-service/backend/internal/service/openapi_runtime_test.go` —
  retained Data API, generated OpenAPI, Server, and smoke seams.
- `miniapp/backend/internal/server/openapi_runtime_test.go` — retained Miniapp
  API, generated OpenAPI, Server, and smoke seams.
- `miniapp/backend/internal/service/research_test.go` — retained Miniapp Biz and
  API mapping seams.

Implementation-only:

- `analyse-data-service/backend/cmd/dbmigrate/main_test.go` — compilation and
  command build.
- `agent-run/backend/cmd/artifacts/main_test.go` — compilation and command build.
- `admin-portal/backend/internal/data/agentrun_wire_test.go` — compilation,
  provider/consumer contract, and smoke.
- `admin-portal/backend/internal/data/data_wire_test.go` — compilation,
  provider/consumer contract, and smoke.
- `miniapp/backend/internal/data/contract_mapping_test.go` — typed Data API
  Adapter contract.
- `admin-portal/frontend/src/styles/minimalDashboardConformance.test.ts` —
  page/component behavior and visual review.
- `admin-portal/frontend/src/vite-config.test.ts` — frontend typecheck and build.
- `miniapp/frontend/config/index.test.ts` — both target builds and platform
  behavior.
- `miniapp/frontend/src/mocks/research-reasoning-trees/mock-port.test.ts` —
  retained API-port and page behavior.
- `analyse-data-service/backend/internal/data/entityseed/alliance_economy_profile_fingerprint_test.go`
  — retained real PostgreSQL Entity Seed boundary.
- `analyse-data-service/backend/internal/data/entityseed/alliance_economy_rebuild_test.go`
  — retained real PostgreSQL Entity Seed boundary.
- `analyse-data-service/backend/internal/data/entityseed/alliance_economy_test_helpers_test.go`
  — no independent guarantee; retained consumers own their fixtures.
- `analyse-data-service/backend/internal/data/entityseed/chain_node_relation_batch_test.go`
  — retained real PostgreSQL transaction boundary.
- `analyse-data-service/backend/internal/data/entityseed/chain_node_relation_preflight_test.go`
  — retained Biz validation and real PostgreSQL boundary.
- `analyse-data-service/backend/internal/data/entityseed/chain_node_relation_test.go`
  — retained real PostgreSQL transaction boundary.
- `analyse-data-service/backend/internal/data/entityseed/convergence_postgres_test.go`
  — retained real PostgreSQL convergence boundary; deleted test used `sqlmock`.
- `analyse-data-service/backend/internal/data/entityseed/external_identifier_batch_test.go`
  — retained real PostgreSQL concurrency boundary.
- `analyse-data-service/backend/internal/data/entityseed/external_identifier_test.go`
  — retained real PostgreSQL concurrency boundary.
- `analyse-data-service/backend/internal/data/entityseed/industry_chain_repository_test.go`
  — retained real PostgreSQL Entity Seed boundary.
- `analyse-data-service/backend/internal/data/entityseed/postgres_repository_test.go`
  — retained real PostgreSQL Entity Seed boundary.
- `analyse-data-service/backend/internal/data/entityseed/preflight_test.go` —
  retained Biz validation and real PostgreSQL boundary.
- `analyse-data-service/backend/internal/data/entityseed/relationship_batch_test.go`
  — retained real PostgreSQL transaction boundary.
- `analyse-data-service/backend/internal/data/postgres/admin_query_postgres_test.go`
  — retained API/Biz query behavior and real PostgreSQL complex-query boundary.
- `analyse-data-service/backend/internal/data/postgres/admin_query_test.go` —
  retained API/Biz query behavior and real PostgreSQL complex-query boundary.
- `analyse-data-service/backend/internal/data/postgres/benchmark_observation_test.go`
  — retained Biz rule and real PostgreSQL boundary.
- `analyse-data-service/backend/internal/data/postgres/event_contract_test.go` —
  retained event publication transaction boundary.
- `analyse-data-service/backend/internal/data/postgres/industry_chain_test.go` —
  retained typed master-data PostgreSQL boundary.
- `analyse-data-service/backend/internal/data/postgres/research_anchor_import_postgres_test.go`
  — retained real PostgreSQL import boundary.
- `analyse-data-service/backend/internal/data/postgres/research_contract_test.go`
  — retained Biz/API research contract and real PostgreSQL boundary.
- `analyse-data-service/backend/internal/data/postgres/research_reasoning_tree_read_test.go`
  — retained Biz/API query behavior and real PostgreSQL boundary.
- `analyse-data-service/backend/internal/data/postgres/research_theme_import_postgres_test.go`
  — retained real PostgreSQL import boundary.

Consolidated into `internal/data/dbmigration/migrations_test.go`:

- `analyse-data-service/backend/internal/data/dbmigration/alliance_economy_contract_test.go`
- `analyse-data-service/backend/internal/data/dbmigration/benchmark_contract_test.go`
- `analyse-data-service/backend/internal/data/dbmigration/event_import_contract_test.go`
- `analyse-data-service/backend/internal/data/dbmigration/event_publication_v2_contract_test.go`
- `analyse-data-service/backend/internal/data/dbmigration/event_schema_contract_test.go`
- `analyse-data-service/backend/internal/data/dbmigration/identity_contract_test.go`
- `analyse-data-service/backend/internal/data/dbmigration/industry_chain_contract_test.go`
- `analyse-data-service/backend/internal/data/dbmigration/legacy_migration_contract_test.go`
- `analyse-data-service/backend/internal/data/dbmigration/raw_document_import_receipt_contract_test.go`
- `analyse-data-service/backend/internal/data/dbmigration/research_anchor_reasoning_tree_contract_test.go`
- `analyse-data-service/backend/internal/data/dbmigration/runner_test.go`

## Retained risk boundaries

| Risk boundary           | Retained verification                                                                                                                              |
| ----------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Business behavior       | Biz tests in all four Kratos services                                                                                                              |
| Public contract         | Protobuf/API tests plus observable Server/Service HTTP tests                                                                                       |
| Data and transactions   | Minimal real PostgreSQL, concurrency, rollback, complex query, and adapter protocol tests                                                          |
| Migration               | Ordered forward-chain application and critical postconditions                                                                                      |
| Configuration/lifecycle | Required settings, security-sensitive validation, startup/shutdown, and resource behavior                                                          |
| Agent/Eino              | Collector workflow decisions, planning, scheduling, cancellation, materialization, and observable outcomes using compiled Eino workflows and fakes |
| Frontend                | Critical pages/components, navigation, state transitions, API adapters, and loading/error/empty behavior                                           |
| Architecture            | One repository-level invariant suite                                                                                                               |

## CI selection

Previously, application jobs ran focused packages and then immediately ran an
unconditional recursive suite, repeating the same packages. The workflow now:

1. detects the changed application;
2. selects `default`, `frontend`, `data`, `migration`, `conf_lifecycle`,
   `provider_consumer`, `container`, and `architecture` risks;
3. runs each selected seam once;
4. runs repository governance once, centrally;
5. keeps formatting, vet/typecheck, backend binary builds, both Miniapp target
   builds (`weapp` and `tt`), and affected container builds.

`scripts/ci/detect-test-risk.sh` is covered by a failing-before/fixed-after
repository contract test. Data Service's API/Service package may also be selected
for a narrowly named PostgreSQL publication integration check; this is a distinct
database guarantee rather than a repeated default behavior matrix.

## Reference-first audits

### Taro

The Miniapp cleanup was checked against the current official
[React 18 support](https://docs.taro.zone/docs/react-18),
[multi-platform build](https://docs.taro.zone/docs/envs), and
[testing utilities](https://docs.taro.zone/docs/3.x/test-utils) guidance.
The project stays on its existing React/Vitest/Taro setup. No new testing
dependency was introduced, and both supported target builds remain in CI.

### Eino

The local reference clones were audited at the required commits:

- Eino: `922b6a8a233b5233fe47eecee6cd2c005e8c39cd`
- Eino Extensions: `9137edd89e72b72735ede69db1c5ae29178a6e41`
- Eino Examples: `171220631fb7068ead50b7cd964b8c471647117d`

The audit compared AgentRun's collector workflow with Eino's
`compose/workflow_test.go`, the examples' field-mapping workflows, and the
OpenAI model adapter test. It confirmed that compiled workflow behavior,
field-mapping decisions, provider normalization, cancellation, and observable
outcomes are the valuable seams. Those tests remain. Directory-layout and CI
source-text assertions were rejected as duplicate repository governance.

## Representative local duration

Measurements used the same warm shared Go build cache and parallel application
shape before and after. Database integration tests skip when the local
`TIDEWISE_TEST_DATABASE_URL` is absent, so these figures compare developer
feedback time rather than hosted PostgreSQL CI time.

| Entry point           | Before |  After | Change |
| --------------------- | -----: | -----: | -----: |
| Data Service Backend  | 27.49s | 18.42s | -33.0% |
| Miniapp Backend       | 13.54s |  6.26s | -53.8% |
| Admin Portal Backend  |  9.30s |  4.20s | -54.8% |
| AgentRun Backend      | 28.91s | 18.86s | -34.8% |
| Miniapp Frontend      |  7.36s |  3.17s | -56.9% |
| Admin Portal Frontend |  9.45s |  5.18s | -45.2% |

These are representative local wall-clock measurements, not a promise about
hosted runner duration. Hosted CI duration should be observed after the PR runs.

## Verification

The cleanup was verified with:

- all four recursive backend suites;
- both complete frontend Vitest suites;
- repository governance tests;
- focused Data migration and Entity Seed boundary tests;
- YAML parsing and workflow contract tests;
- formatting and diff whitespace checks.

The repository testing policy and future Issue test-plan template are defined in
`docs/agents/testing.md`.
