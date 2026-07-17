# Package 3 Service Skeleton Evidence

## Scope And Outcome

Package 3 adds service-owned `data`, `miniapp`, and `adminportal` HTTP process boundaries under `backend/services/`, with one command, facade, liveness endpoint, and readiness endpoint per service. A business-free `internal/platform/servicehttp` helper owns only common server mechanics. The two legacy HTTP commands now pass their existing handlers through the matching service facade, so their route behavior remains unchanged during the compatibility window.

The new Miniapp and Admin service packages import no Data domain, repository, database, migration, graph, connector, or parser implementation. Existing direct Data wiring remains frozen only in the legacy packages already listed by the Package 2 transitional allowlist; removing that wiring belongs to Packages 6 and 7. No domain/repository/data bulk move occurred.

The Leader-requested stale architecture diagnostics now identify Packages 6–7 for BFF decoupling and Package 8 for scheduler/runtime retirement. `TestTransitionalBFFDataDependencyAllowlist` and `TestLegacyIngestionPreRetirementImportAllowlist` cover those diagnostics and the preserved pre-retirement manifest.

## TDD Evidence

RED was observed before production files were added:

```text
go test ./services/data ./services/miniapp ./services/adminportal
# undefined: NewHandler, ServiceName, NewServer; exit 1

go test ./internal/architecture -run 'Test(ServiceOwned|LegacyHTTPCommands|ServiceSkeleton|TransitionalBFF|LegacyIngestionPreRetirement)' -count=1
# missing three service command packages and legacy compatibility imports; exit 1
```

Minimal GREEN and compatibility verification:

```text
go test ./services/data ./services/miniapp ./services/adminportal
ok services/data; ok services/miniapp; ok services/adminportal

go test ./internal/architecture -run 'Test(ServiceOwned|LegacyHTTPCommands|ServiceSkeleton|TransitionalBFF|LegacyIngestionPreRetirement)' -count=1
ok internal/architecture

go test ./cmd/api ./cmd/admin-api
cmd/api [no test files]; ok cmd/admin-api
```

## Fresh Package Validation

```text
go build ./services/data/cmd ./services/miniapp/cmd ./services/adminportal/cmd
PASS

go test ./services/... ./cmd/api ./cmd/admin-api ./internal/http ./internal/apps/miniappapi ./internal/apps/adminapi
PASS

go test ./internal/architecture -count=1
PASS

OPENSPEC_TASK_LINT_CHANGE=establish-data-service-bff-boundaries go test ./internal/architecture -run '^TestOpenSpecTaskDesignLint$' -count=1
PASS

openspec validate establish-data-service-bff-boundaries --strict
PASS

openspec status --change establish-data-service-bff-boundaries
PASS: 4/4 artifacts complete

git diff --check
PASS
```

Reference audit found none of the stale strings `Packages 5-6`, `Package 7 manifest verification`, or `Package 7 replaces`; the corrected Package 6–7 and Package 8 diagnostics are present. Scope review is limited to the approved skeleton/facade/health/architecture diagnostics, task state, and this evidence. Secret scanning found no credential-like assignment, connection string, or token material.

## Exclusions And Stop Boundary

No PostgreSQL or Neo4j connection was made; no SQL, migration, seed, import, role, credential, environment, deployment, scheduler/runtime deletion, BFF decoupling, or Data API/import code was executed. Package 4 owns the API/import and `000022` artifact work.
