# Package 7 Admin Retirement And Decoupling Evidence

## Outcome And Frozen Compatibility Surface

Package 7 moves the retained Admin `raw-documents`, `events`, and `source-catalogs` queries behind the Admin-owned `DataServiceClient`. Each valid BFF request invokes exactly one aggregate client method, propagates a safe inbound request ID, preserves existing public DTO/filter/pagination behavior, and does not expose Data-only `content_level`, `parser_key`, or internal repository errors.

The authenticated compatibility routes below remain for one actual deployment window and perform zero Data Service, repository, or database calls:

| Method and path | Result |
|---|---|
| `GET /admin/scheduler/config` | `410 Gone`, code `ADMIN_SCHEDULER_RETIRED`, nonempty request ID |
| `PUT /admin/scheduler/config` | same tombstone without parsing or applying the submitted body |
| `GET /admin/scheduler/runs` | same tombstone without parsing the old `limit` query |

Missing/wrong browser credentials still return `401`; an unconfigured browser token still fails closed with `503`. Health/readiness retain `adminportal` service identity and config readiness. The Admin runtime loader reads only app/server, `ADMIN_API_TOKEN`, `DATA_SERVICE_BASE_URL`, and `DATA_SERVICE_ADMIN_TOKEN`; its service config cannot carry PostgreSQL, migration, or Data DB credentials. Both the service-owned command and thin `cmd/admin-api` alias construct the bounded five-second Admin Data HTTP client and contain no database, migration, domain, or repository wiring.

## TDD And Replacement Evidence

Backend RED was observed after the contract tests and architecture assertions were changed but before production implementation:

```text
GOCACHE=/tmp/tidewise-go-cache go test ./internal/apps/adminapi ./services/adminportal ./internal/architecture -run 'Test(...)' -count=1
# NewRouter signature mismatch; LoadRuntimeConfig undefined; NewHandler dependency arguments undefined;
# architecture still observed the pre-Package-7 direct Data imports. Exit 1.
```

The interrupted predecessor left three frontend mixed tests in test-first RED form and no production edit at accepted HEAD `9077dd3f0d8fd45338419834a495113b94b029cd`: they required exactly three tabs and absence of scheduler styles while the accepted production still imported `SchedulerSettings`, rendered a fourth tab, and retained scheduler CSS. Execution was interrupted before a Vitest RED log was produced; Package 7 continued from those scoped tests without reverting them. Fresh GREEN is recorded below.

Backend `router_test.go` changes from 10 to 8 test functions. Four old scheduler config/run success cases are removed with their production handlers and replaced by one table-driven authenticated 410/request-id/zero-call contract covering all three routes. Retained auth/raw/event/source/health coverage remains, with new invalid-query zero-call, one-call DTO mapping, request-id propagation, and internal-error nonleak assertions.

Admin frontend changes exactly from 7 test files/26 cases to 5 files/16 cases:

- delete `src/api/scheduler.test.ts` (4 cases) with deleted `src/api/scheduler.ts`;
- delete `src/pages/SchedulerSettings.test.tsx` (6 cases) with deleted `src/pages/SchedulerSettings.tsx`;
- retain and rewrite `App.test.tsx`, `DataIngestionCenter.test.tsx`, and `minimalDashboardConformance.test.ts` for login plus source/raw/event three-tab behavior and scheduler-style absence;
- keep `package-lock.json` byte-for-byte unchanged, including the unrelated React transitive `node_modules/scheduler` entry;
- delete no fixture or `testdata` file.

## Fresh Validation

```text
GOCACHE=/tmp/tidewise-go-cache go test ./internal/apps/adminapi ./services/adminportal/... ./cmd/admin-api -count=1
PASS: adminapi, adminportal, adminportal/dataclient, cmd/admin-api; service-owned cmd has no tests

GOCACHE=/tmp/tidewise-go-cache go test ./internal/architecture -run '^(TestTransitionalBFFDataDependencyAllowlist|TestServiceOwnedBFFPackagesDoNotImportDataImplementation|TestLegacyHTTPCommandsUseServiceOwnedCompatibilityFacade|TestServiceOwnedPackagesAndCommandsExist)$' -count=1
PASS

GOCACHE=/tmp/tidewise-go-cache go build ./cmd/admin-api ./services/adminportal/cmd
PASS

cd frontend/admin && npm test
PASS: 5 files, 16 tests

cd frontend/admin && npm run build
PASS: TypeScript no-emit check and Vite production build

openspec validate establish-data-service-bff-boundaries --strict
PASS

OPENSPEC_TASK_LINT_CHANGE=establish-data-service-bff-boundaries GOCACHE=/tmp/tidewise-go-cache go test ./internal/architecture -run '^TestOpenSpecTaskDesignLint$' -count=1
PASS

openspec status --change establish-data-service-bff-boundaries
PASS: spec-driven, 4/4 artifacts complete
```

The first restricted-sandbox invocation of the full Admin Go suite passed the application/service/command packages but the existing Data client tests could not bind an IPv6 loopback `httptest` port. The identical suite was rerun once with loopback permission and passed, including identity, request-id, bounded deadline/timeout, retry, error classification, secret nonleak, and OpenAPI drift cases. This was stateless test wiring only; no external network or service was contacted.

## Exact Package Manifest

Production and tests:

```text
backend/cmd/admin-api/main.go
backend/internal/apps/adminapi/router.go
backend/internal/apps/adminapi/router_test.go
backend/internal/architecture/service_boundary_transition_test.go
backend/internal/architecture/service_skeleton_test.go
backend/services/adminportal/cmd/main.go
backend/services/adminportal/runtime_config.go
backend/services/adminportal/runtime_config_test.go
backend/services/adminportal/service.go
backend/services/adminportal/service_test.go
frontend/admin/src/App.test.tsx
frontend/admin/src/api/scheduler.test.ts                         [deleted]
frontend/admin/src/api/scheduler.ts                              [deleted]
frontend/admin/src/pages/DataIngestionCenter.test.tsx
frontend/admin/src/pages/DataIngestionCenter.tsx
frontend/admin/src/pages/SchedulerSettings.test.tsx              [deleted]
frontend/admin/src/pages/SchedulerSettings.tsx                   [deleted]
frontend/admin/src/styles/app.css
frontend/admin/src/styles/minimalDashboardConformance.test.ts
```

OpenSpec state/evidence:

```text
openspec/changes/establish-data-service-bff-boundaries/tasks.md
openspec/changes/establish-data-service-bff-boundaries/evidence/package-7-admin-retirement-decoupling.md
```

## Scope And Stop Boundary

Architecture checks now reject Data domain/repository/database/migration/graph/connector/parser/runtime/scheduler imports from both Miniapp and Admin BFF application/command owners, including all `services/adminportal/**` packages. No production Admin reference remains to scheduler DTOs, config/run repositories, or scheduler UI code except the approved three-route transport tombstone.

Package 7 did not modify or run migrations, SQL, PostgreSQL, Neo4j, business data, roles/grants/credentials, infra, deployment, UAT/prod/shared, historical tables/repositories/connectors/parsers/import contracts, `backend/Dockerfile`, or `package-lock.json`. Package 8 scheduler/runtime retirement was not started.

`git diff --check` passed. A path-scope query returned no changes under `backend/migrations`, `infra`, `.github`, `backend/Dockerfile`, or `frontend/admin/package-lock.json`. A scoped diff scan found no private key marker, realistic provider token prefix, credential-bearing PostgreSQL URI, or other secret material. The final Package 7 manifest is the 21 files listed above.
