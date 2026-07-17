# Package 6 Miniapp BFF Decoupling Evidence

## Scope And Outcome

Package 6 corrects the Package 4 Research transport drift to the approved published foundation contract, moves repository-backed Research aggregate ownership into the Data Service, and makes both Miniapp entry commands call that boundary through the Miniapp-owned handwritten `DataServiceClient`. The public `/api/v1/miniapp/research/*` JSON, opaque cursor, ordering ranks, query bounds, and `400/404/500` behavior remain stable. Each accepted BFF request performs one aggregate client method call; the existing typed HTTP adapter retains its bounded timeout, service identity, request-id propagation, and safe bounded GET retry.

Miniapp startup now loads only app/server plus `DATA_SERVICE_BASE_URL` and `DATA_SERVICE_MINIAPP_TOKEN`. Its runtime type cannot carry PostgreSQL, migration, Neo4j, or shared secret configuration, and both `cmd/api` and `services/miniapp/cmd` have no database, migration, domain, or repository imports. No Miniapp image exists in the current repository; Package 9 remains the approved owner of new service image/assets, so Package 6 had no existing Miniapp image credential reference to remove.

No Admin, ingestion scheduler/runtime, migration, database, Neo4j, deployment, Package 7+, or frontend source file was changed. No test or testdata file was deleted; testdata remains 13 files.

## Exact Change Manifest

| Files | Purpose |
|---|---|
| `backend/services/data/api/openapi.yaml` | authoritative Research enums, free-text `trading_direction`, distinct Theme/Anchor chain-node schemas |
| `backend/services/data/research/{service.go,service_test.go}` | Data-owned repository aggregate, window/cursor/rank semantics, explicit DTOs, safe error classification |
| `backend/services/data/internalapi/{handler.go,handler_test.go}` | Data Research adapter plus nonempty golden covering all frozen values, fields, cursor, one-call and safe errors |
| `backend/services/data/cmd/main.go` | Data command owns repository-backed Research service wiring |
| `backend/services/miniapp/dataclient/{port.go,client_test.go,contract_drift_test.go}` | consumer-owned corrected DTOs, HTTP decoding and OpenAPI drift guards |
| `backend/internal/apps/miniappapi/{research_service.go,research_service_test.go,research_router_test.go}` | BFF-only validation, explicit DTO mapping, request-id/error mapping, public nonempty goldens and call counts |
| `backend/services/miniapp/{runtime_config.go,runtime_config_test.go,service.go,service_test.go}` | DB-free fail-closed runtime config, health identity, Data client route composition |
| `backend/{cmd/api,services/miniapp/cmd}/main.go` | DB-free Miniapp startup adapters |
| `backend/internal/architecture/{service_boundary_transition_test.go,service_skeleton_test.go}` | remove Miniapp transition allowlist and forbid persistence/Data implementation imports |
| this evidence and `tasks.md` | Package 6 completion and validation record |

## TDD And Contract Evidence

Observed RED conditions before implementation:

- typed-client tests did not compile because `focus`, `infrastructure`, `market_structure`, `contextual`, neutral index direction, and the two chain-node DTOs did not exist;
- Miniapp tests did not compile against a `DataServiceClient` constructor or DB-free runtime loader;
- the transition architecture test failed because `internal/apps/miniappapi` still imported `internal/domain` and repositories;
- service-owned route tests could not compose the BFF client; and
- Data Research tests initially had no Data-owned service/package.

GREEN coverage establishes:

- `impact_level=high|focus|watch`, `transmission_stage=upstream|midstream|downstream|infrastructure|service`, all seven `anchor_type` values, `importance=primary|secondary|contextual`, index `impact_direction=positive|negative|mixed|neutral`, and `evidence_role=driver|supporting|contradicting|context`;
- `trading_direction` remains a nonempty natural-language string with no enum;
- Theme nodes serialize only `impact_summary`; Anchor nodes serialize only `relation_summary`;
- Data owns opaque cursor construction and the existing `high > focus > watch` / `primary > secondary > contextual` ranks;
- BFF list/detail routes preserve public DTOs, empty arrays, cursor forwarding, `400/404/500`, sanitized internal failures, inbound request-id propagation, and one aggregate method call;
- invalid local query/UUID input performs zero Data calls; and
- Miniapp runtime config succeeds without any PostgreSQL setting, fails closed without Data service identity, and carries no Data DB URL/password.

Final OpenAPI SHA-256:

```text
8f9dee88269d1ed6d5777afc13778614d482a227ce03595c4690d88ea8214252
```

Repository test inventory at this checkpoint is 133 Go test files / 611 `func Test` cases. Testdata is unchanged at 13 files. All 22 migration files, including the already-applied Package 5 artifact, are byte-untouched by this package.

## Fresh Package Validation

```text
GOCACHE=/tmp/tidewise-go-cache go test \
  ./services/data/api ./services/data/research ./services/data/internalapi ./services/data/cmd \
  ./services/miniapp/dataclient ./internal/apps/miniappapi ./internal/http \
  ./services/miniapp ./services/miniapp/cmd ./cmd/api -count=1
PASS (typed-client loopback tests run with loopback permission)

GOCACHE=/tmp/tidewise-go-cache go test ./internal/architecture \
  -run '^(TestTransitionalBFFDataDependencyAllowlist|TestServiceOwnedBFFPackagesDoNotImportDataImplementation|TestLegacyHTTPCommandsUseServiceOwnedCompatibilityFacade|TestServiceOwnedPackagesAndCommandsExist)$' -count=1
PASS

GOCACHE=/tmp/tidewise-go-cache go vet \
  ./internal/apps/miniappapi ./internal/http ./services/miniapp/... \
  ./services/data/research ./services/data/internalapi ./services/data/cmd ./cmd/api
PASS

GOCACHE=/tmp/tidewise-go-cache go build ./cmd/api ./services/miniapp/cmd ./services/data/cmd
PASS

cd frontend/miniapp && npm test
PASS: 3 Node script tests + 18 Vitest tests

cd frontend/miniapp && npm run typecheck
PASS
```

The first frontend invocation correctly reported missing local `vitest`/`tsc` executables. A single workspace `npm install --ignore-scripts` restored the lockfile-defined dependencies; `package-lock.json` remained unchanged, and both commands then passed. No frontend source or generated output entered the diff.

Strict OpenSpec/task-design lint, artifact status, and final diff/whitespace/scope/secret checks are recorded after this evidence and task state are added.

## Stop Boundary

No PostgreSQL/Neo4j connection or command, SQL/migration, seed/import/business write, role/grant/credential change, Admin decoupling, scheduler/runtime cleanup, image/deployment action, or Package 7+ work was performed. Package 6 stops at this committed R1 checkpoint for independent Leader review.
