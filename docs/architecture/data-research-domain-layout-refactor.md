# Data Research Domain Layout Refactor

## Status And Authority

- Status: approved for implementation.
- Owner: Data Service.
- Date: 2026-08-12.
- Issue: [#207](https://github.com/meierlink88/tidewise-ai/issues/207).
- Governing standard:
  `docs/development-standards/kratos-backend-layout-standard.md`.

This specification governs the source ownership, package convergence, dependency direction and test
seams of the existing Research capabilities. Data Context, ADR-0003, the Research Theme and Reason
Tree specifications, the Analyst-first Snapshot Publication V3 specification and OpenAPI remain
authoritative for product semantics.

## Outcome

Converge all live Research code into one stable singular `research` domain. Research Theme is the
publication aggregate root and one complete publication contains exactly one Theme and one or more
Reason Trees. Research Analysis Context and Research Graph Search remain externally available
Research workflows and are orchestrated by Research Biz through provider-domain capabilities.

The refactor removes the historical Theme-only import and independent Reason Tree import capabilities.
The current aggregate publication API is the only formal Research publication boundary.

## Domain Model

```text
Research Publication Aggregate (analysis_batch_id)
└── Research Theme exactly 1
    ├── Theme Impact 1..N
    ├── Theme Event 0..N
    └── Reason Tree 1..N
        ├── Tree Event 0..N
        └── ordered Node 1..N
            └── Signal 1..5
```

- Research Theme is an immutable conclusion snapshot in one publication aggregate.
- Reason Tree is an immutable child explanation projection and has no independent write lifecycle.
- Theme and Reason Tree retain separate read resources but become visible atomically.
- Research Analysis Context is an externally consumed read workflow that assembles eligible Event,
  accepted Event Semantic and the minimum Entity/TBox reference closure.
- Research Graph Search is an externally consumed read workflow that returns a bounded, stable and
  reference-complete Entity/Industry Chain subgraph.

## Ownership And Dependency Direction

The Research HTTP Service remains a thin transport adapter. It converts wire values, calls the
Research UseCase, maps results and preserves public errors. It does not orchestrate provider domains,
open transactions or query PostgreSQL.

Research Biz owns:

- Theme and Reason Tree entities, values, rules and reads;
- aggregate publication validation, deterministic identity, canonical hashing, replay and conflict;
- Analysis Context time windows, cursor, budgets, fingerprints and reference-closure validation;
- Graph Search request validation, budgets, deterministic normalization and result mapping;
- cross-domain workflow orchestration through explicitly injected provider capabilities.

Provider responsibilities are:

- Event Biz provides bulk eligible formal Event facts required by Analysis Context;
- Event Semantic Biz provides accepted/latest Semantic facts and versioned semantic definitions;
- Entity Biz provides Entity/TBox reference closure and bounded graph traversal facts;
- provider Data adapters own their SQL and validate persisted rows before returning them to Biz;
- Research Data owns only Research Theme, Reason Tree, publication Receipt and their persistence.

The dependency direction is:

```text
Research HTTP -> Research Service -> Research Biz
                                      ├── Event Biz capability
                                      ├── Event Semantic Biz capability
                                      ├── Entity Biz capability
                                      ├── Research read Port <- Research Data
                                      └── Research transaction Port <- Research Data transaction
```

Research Biz may depend on stable provider-domain values and small read capabilities. It must not
import provider Data adapters. Provider domains must not depend back on Research. The composition root
constructs the provider UseCases once and injects them into Research.

## Target Source Layout

```text
analyse-data-service/backend/
├── api/data/v1/research/
│   ├── api.go
│   └── http.go
└── internal/
    ├── biz/research/
    │   ├── biz.go
    │   └── transaction.go
    ├── data/research/
    │   ├── data.go
    │   └── transaction.go
    └── service/research/
        └── service.go
```

Tests follow the responsibility names `api_test.go`, `http_test.go`, `biz_test.go`,
`transaction_test.go`, `data_test.go` and `service_test.go` only.

No scenario or predicted-responsibility production files are introduced. Theme publication, Theme
read, Reason Tree read, Analysis Context and Graph Search appear in public method and test names, not
in package or production file names.

## Current-to-target Convergence

Converge the live behavior from the current Research read, publication, Theme import, Reason Tree
import, Analysis Context and Graph Search Biz packages into the target Research Biz module. Valid
models, canonicalization, deterministic identity, validation, cursor, budget and error behavior are
preserved while obsolete standalone workflows are removed.

Converge Research wire DTOs, operation IDs, error codes and HTTP binding from the flat API package into
the Research API module. Shared envelope, strict binding helpers, middleware and OpenAPI embedding stay
in the parent API package.

Converge Research Service conversion and error mapping into one Research Service adapter. The generic
Data Service no longer contains Research methods or Research-specific dependency interfaces.

Converge Research-owned PostgreSQL reads and aggregate transaction implementation into the Research
Data module. Remove pass-through repositories and Research SQL from the generic PostgreSQL package.
Move provider-owned query portions into Event, Event Semantic and Entity Data modules together with
their fail-closed validation.

Delete the superseded scenario packages and all empty directories after callers are migrated. Do not
leave compatibility aliases or forwarding wrappers.

## Historical Capability Removal

The following runtime capabilities are obsolete and removed:

- Theme-only import with a batch containing Theme rows but no Reason Trees;
- independent Reason Tree import and its separate transaction/receipt workflow;
- pass-through repositories and aliases dedicated to those standalone workflows;
- the local Theme-only development seed command, its manifest and its usage documentation.

The local Research reset command remains because it clears the current Theme plus Reason Tree
publication aggregate and does not provide a competing publication path.

Historical migration files remain immutable. Tables and Receipt rows still used by the current
aggregate publication are not historical merely because their physical names contain `import`.
No persisted Research row is deleted or rewritten by this refactor.

## Preserved External Contract

The following paths remain unchanged:

```text
POST /api/data/v1/research-theme-imports
GET  /api/data/v1/research/themes
GET  /api/data/v1/research/themes/{theme_id}
GET  /api/data/v1/research/themes/{theme_id}/reasoning-trees
GET  /api/data/v1/research/themes/{theme_id}/reasoning-trees/{reasoning_tree_id}
GET  /api/data/v1/research-analysis-context
POST /api/data/v1/research-graph:search
```

Preserve the current OpenAPI schemas, methods, operation IDs, Bearer service-token scopes, request
limits, timeout budgets, status codes, safe error codes, stable ordering, pagination/cursor behavior,
resource budgets, fingerprints and response envelopes.

The publication endpoint continues to support the existing formal publication variant and
`publication_mode=analyst_snapshot`. One request contains one Theme and its complete Reason Tree set.
`analysis_batch_id` remains the idempotency identity; identical replay returns the first Receipt and a
changed payload or publisher returns the existing conflict behavior.

Analysis Context keeps its live retrospective-reconstruction semantics. Provider facts are assembled
and the final closure is revalidated; concurrent reference drift remains a safe `409` restart result
rather than being hidden. Graph Search keeps its PostgreSQL-backed deterministic bounded traversal and
does not introduce Neo4j or Qdrant.

## Transaction Ownership

Aggregate publication follows:

```text
Research Service -> Research Biz UseCase -> Research transaction Port <- Research Data transaction
```

- Biz decides transaction scope, operation order, replay, conflict and publication result.
- Biz computes domain identities, payload hashes and the shared publication timestamp.
- Data implements begin, identity lock, validated reference reads, writes, verification, commit,
  rollback and cancellation propagation.
- Data returns validated state and executes Biz-authored commands; it does not independently choose a
  business result.
- Any Theme, Reason Tree, lineage or write failure leaves no visible partial aggregate.

## Testing Decisions

Tests observe supported behavior and public module interfaces rather than file movement, method call
counts or directory lists. Existing coverage is relocated or consolidated at the strongest applicable
seam.

### Highest observable seam

The unchanged wire contract is protected by the real Research HTTP registration/binding suite. A
separate real PostgreSQL provider/consumer fixture publishes a complete Theme and Reason Tree
aggregate, replays it, reads Theme and Tree projections and exercises Analysis Context through Event,
Event Semantic and Entity provider UseCases. Entity Data separately exercises bounded Graph Search
against real PostgreSQL. This split keeps transport and persistence failures independently observable.

### Focused seams

- Research HTTP tests cover all seven routes, strict binding, query validation, scope, timeout and
  stable error/status behavior against the unchanged OpenAPI contract.
- Research Biz tests cover formal and Analyst-first validation, identity, canonical replay/conflict,
  Theme/Tree reads, cursor behavior, Analysis Context closure/budgets/drift and Graph Search
  normalization/budgets.
- Research transaction tests cover aggregate commit/rollback through the public transaction Port;
  the real PostgreSQL aggregate fixture covers reference validation, replay and late-failure rollback.
- Research Data tests cover Theme/Tree persistence, ordering, reconstruction and malformed persisted
  row fail-closed behavior.
- Event, Event Semantic and Entity Biz/Data tests cover the new bulk provider capabilities and ensure
  provider facts remain validated before Research receives them.
- Existing production fixtures remain the independent expected source where they protect a stable
  contract. Tests do not recreate response mapping, authentication or route wiring in private harnesses.

Obsolete standalone Theme-only and independent Reason Tree import tests are removed. Their still-live
aggregate validation, identity, replay, rollback and read behaviors are retained at Research seams.

No migration-specific test is added. The migration ledger is unchanged and remains covered by the
stable full-ledger forward smoke. The self-developed migration control layer is outside this refactor.

## Rollout And Rollback

The change ships as one behavior-preserving Data Service revision. There is no database rollout or
compatibility window. Source rollback restores the previous package layout while reading the same
schema and rows. The removed local Theme-only seed is not restored by runtime compatibility code.

## Non-goals

- No Research API, DTO, authentication, timeout or error-contract change.
- No Theme, Reason Tree, Analysis Context or Graph Search semantic change.
- No database schema, migration, trigger, index, backfill or historical-row change.
- No Miniapp, Admin Portal, Research Analyst or other consumer implementation change.
- No independent Theme or Reason Tree publication endpoint.
- No Research Thesis, mutable Theme lifecycle or cross-publication merge identity.
- No Neo4j, Qdrant, embedding, graph projection or new infrastructure.
- No deletion of the current aggregate reset command.
- No compatibility wrapper for removed scenario packages.

## Acceptance

- All live Research code uses the singular Research domain across API, Biz, Data and Service.
- Theme and Reason Tree publish only through the current atomic aggregate API.
- Historical Theme-only and independent Reason Tree import code and the Theme-only seed are absent.
- Analysis Context and Graph Search remain externally observable Research APIs and Research Biz owns
  their orchestration through Event, Event Semantic and Entity capabilities.
- Service remains a thin wire adapter and provider Data ownership remains in the provider domains.
- Existing OpenAPI and runtime behavior pass regression and provider/consumer tests.
- PostgreSQL transaction, rollback, cancellation, replay, conflict and fail-closed seams pass.
- No migration or persisted-data change appears in the delivery diff.
- Formatting, vet, build, architecture, contract and secret checks pass.
