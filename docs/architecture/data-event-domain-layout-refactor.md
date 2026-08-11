# Data Event Domains Layout Refactor

> ADR-0013 已取代本文关于 `semanticprojection` 继续由 Data 维护的过渡性边界；Event 与
> Event Semantic 的领域布局和外部合同仍然有效。

## Status And Authority

- Status: Frozen; user confirmed on 2026-08-11.
- Owner: Data Service.
- Governing standard:
  `docs/development-standards/kratos-backend-layout-standard.md`.
- Delivery: one behavior-preserving pull request covering both strongly coupled domains; the PR does
  not change product behavior.

This specification governs only source ownership, package convergence, dependency direction and test
seams for the existing Event and Event Semantic capabilities. Existing Context terms, accepted ADRs,
OpenAPI, fixtures, PostgreSQL schema and runtime behavior remain authoritative for product semantics.

## Outcome

Converge the current scenario-named and cross-layer Event code into two stable domain modules:

1. `event` owns the formal Event aggregate, Event Evidence Record and Link, Event Tag Catalog and
   Assignment, Event Publication Batch and Receipt, and Event reads.
2. `eventsemantic` owns Context Lease, Semantic Submission, candidate precheck and review, semantic
   result reads, and their persistence lifecycle.

The modules are deliberately separate. Event Semantic has independent entities, rules, lifecycle,
external HTTP contract and persistence. It is not a publication helper or a file-level subdivision of
`event`.

## Non-goals

- No OpenAPI path, method, DTO, operation ID, status code, scope or error-contract change.
- No PostgreSQL schema, table, column, constraint, index, migration or stored-data change.
- No Event Publication V3 or Event Semantic V3 behavior change.
- No AgentRun-to-AgentOS ownership migration or consumer implementation change.
- No Qdrant, embedding, semantic projection or deployment change.
- No new timeout, retry, idempotency, authentication or Secret behavior.
- No compatibility package, re-export alias or long-lived transitional code.
- No layout assertions or mechanical directory-list tests in CI.

## Domain Boundaries

### Event

`event` owns:

- formal Event identity, status, fact payload and convergence;
- Event Evidence Record and Event Evidence Link;
- Event Tag, active Tag Catalog and Event Tag Assignment;
- Event Publication Batch validation, natural-identity conflict handling and Receipt;
- Event list queries currently exposed for the Admin Portal BFF.

Internal ubiquitous language is tightened without changing wire or storage names:

| Current internal name | Target internal name | Stable external name |
| --- | --- | --- |
| `RawDocument` in Event Publication | `EventEvidenceRecord` | `raw_documents` remains unchanged |
| `EventSource` | `EventEvidenceLink` | `event_sources` remains unchanged |
| `EventTagDef` | `EventTag` | existing Event Tag DTO/table remains unchanged |
| `EventTagMap` | `EventTagAssignment` | existing assignment DTO/table remains unchanged |
| `Publication` | `PublicationBatch` | existing publication request remains unchanged |

The new Raw Evidence and atomic Evidence domain remains separate. An Event Evidence Record is the
existing lightweight Event-publication lineage object; it must not be renamed or treated as Raw
Evidence.

### Event Semantic

`eventsemantic` owns:

- eligible Event discovery and stable cursor behavior;
- Context Lease creation, pinned manifest and context hydration;
- Semantic Submission identity, replay, deterministic precheck and candidate decisions;
- independent review, supersession and final semantic status;
- Event Semantic result reads and historical compatibility.

Event, Evidence and Entity values inside an Event Semantic Context are pinned snapshots or references;
they do not transfer ownership of the formal Event, Evidence or Entity aggregates into this module.

### Excluded Neighbor: Semantic Projection

`semanticprojection` remains outside both modules in this delivery. It projects Entity and Variable
Definition facts through an independently runnable command and external embedding/Qdrant adapters.
Its later layout convergence requires a separate reviewed change.

## Target Source Layout

```text
analyse-data-service/backend/
├── api/data/v1/
│   ├── openapi.yaml
│   ├── document.go
│   ├── http.go
│   ├── event/
│   │   ├── api.go
│   │   └── http.go
│   └── eventsemantic/
│       ├── api.go
│       └── http.go
└── internal/
    ├── biz/
    │   ├── event/
    │   │   ├── biz.go
    │   │   └── transaction.go
    │   └── eventsemantic/
    │       ├── biz.go
    │       └── transaction.go
    ├── data/
    │   ├── event/
    │   │   ├── data.go
    │   │   └── transaction.go
    │   └── eventsemantic/
    │       ├── data.go
    │       └── transaction.go
    └── service/
        ├── event/service.go
        └── eventsemantic/service.go
```

Only the listed production files are permitted initially. Use cases such as publication, list,
context creation, submission and review appear in method and test names, never in package or production
file names. File size alone does not authorize `publication.go`, `query.go`, `precheck.go`, `types.go`,
`model.go`, `port.go`, `repository.go`, `mapping.go` or similar responsibility fragments.

## File Responsibilities

| File | Responsibility |
| --- | --- |
| `api/.../<domain>/api.go` | Domain wire DTOs, HTTP Service interface and stable wire error codes |
| `api/.../<domain>/http.go` | Domain route registration, binding and existing request execution policy |
| `internal/biz/<domain>/biz.go` | Entities, values, errors, non-transaction Ports, rules, UseCase construction and public methods |
| `internal/biz/<domain>/transaction.go` | Explicit business transaction Port and atomicity interface |
| `internal/data/<domain>/data.go` | PostgreSQL reads, scans, persisted-invariant validation, conversion and error cleaning |
| `internal/data/<domain>/transaction.go` | PostgreSQL transaction implementation, identity locks, commit and rollback behavior |
| `internal/service/<domain>/service.go` | Wire/Biz conversion, transport validation and stable error mapping |

The public module interfaces stay small:

- Event exposes publication, active Tag Catalog and Event list use cases through one Event UseCase.
- Event Semantic exposes eligible scan, lease, context, submission, review and result-read use cases
  through one Event Semantic UseCase.
- Transaction details remain behind the Biz transaction seam and are not exposed to API or Service.

### Event Semantic Transaction Seam

Event Semantic follows the same transaction ownership as the Event and Evidence domains:

- the Biz UseCase opens the business atomicity boundary through a Biz-owned
  `TransactionStore.InTransaction` Port;
- the transaction callback reads validated, locked persistence state through the Biz-owned
  `Transaction` interface, makes replay, conflict, status, review, lease-consumption and
  supersession decisions in Biz, and submits explicit persistence commands;
- the Data Adapter implements `sql.Tx`, row locking, queries, inserts, updates, commit, rollback and
  persisted-row validation only;
- Data may assemble persistence snapshots and execute a Biz-authored write command, but it must not
  independently choose a domain status, conflict result, replay result, retry outcome or
  supersession transition;
- Service performs wire conversion and error mapping only and never opens or coordinates a database
  transaction.

The transaction interface uses one validated state read and one explicit persistence command per
lease, submission or review workflow. It does not expose `sql.Tx`, driver errors or a wide collection
of SQL-shaped methods to Biz. Workflow aggregate IDs, business transition timestamps and domain
transition decisions are created by Biz; Data persists them unchanged. Persistence-only row IDs stay
inside Data. This is a behavior-preserving ownership correction: existing locks, atomicity, replay,
concurrency, status, error and rollback behavior remain unchanged.

## Current-to-target Mapping

### Event work package

Move and converge:

- `internal/biz/eventpublication/*` into `internal/biz/event/{biz.go,transaction.go}`;
- Event-owned types from `internal/biz/model/event.go` and `internal/biz/model/source.go` into
  `internal/biz/event/biz.go`;
- `internal/biz/eventtagcatalog/*` into `internal/biz/event/biz.go`;
- the Event-list portion of `internal/biz/adminquery` into `internal/biz/event/biz.go`;
- Event Publication and Event Tag wire DTO/binding from flat `api/data/v1` files into
  `api/data/v1/event/{api.go,http.go}`;
- `internal/data/eventpublication/*` and Event-owned `internal/data/postgres/event*.go` into
  `internal/data/event/{data.go,transaction.go}`;
- Event publication, Tag and list conversion/error logic into
  `internal/service/event/service.go`.

Delete superseded scenario packages and pass-through adapters in the same PR. The composition root
constructs one Event data adapter, one Event UseCase and one Event API Service.

### Event Semantic work package

Move and converge:

- flat `api/data/v1/event_semantics_*` files into
  `api/data/v1/eventsemantic/{api.go,http.go}`;
- `internal/biz/eventsemantics/{types.go,precheck.go,service.go}` into
  `internal/biz/eventsemantic/{biz.go,transaction.go}`;
- `internal/data/postgres/event_semantics.go` and
  `internal/data/postgres/event_semantic_history.go` into
  `internal/data/eventsemantic/{data.go,transaction.go}`;
- `internal/service/event_semantics.go` into
  `internal/service/eventsemantic/service.go`.

Delete the superseded plural package after all callers use the target module. The composition root
constructs one Event Semantic data adapter, one UseCase and one API Service.

## Dependency Direction And Composition

```text
HTTP binding → API Service interface ← Service adapter → Biz UseCase → Biz Port ← Data adapter
```

- API domain packages must not import Biz or Data.
- Biz domain packages must not import API, Service, SQL drivers or Kratos transport types.
- Data implements Biz-owned Ports and validates every PostgreSQL row before returning it to Biz.
- Service implements the domain API interface and owns all wire/Biz conversion.
- Server owns shared middleware, authentication, envelope, docs and domain route registration only.
- `cmd/server/app.go` is the only runtime composition root and does not contain domain rules.
- Research consumers may depend on Event Semantic Biz values only through an explicit consumer-owned
  Port or immutable result value; they must not import Event Semantic Data implementation.

## Preserved Runtime Contract

The following routes and their current semantics remain byte-for-byte contract compatible:

```text
GET  /api/data/v1/event-tags
GET  /api/data/v1/events
POST /api/data/v1/reviewed-event-imports
GET  /api/data/v1/event-semantics/eligible-events
POST /api/data/v1/event-semantics/context-leases
GET  /api/data/v1/event-semantics/context-leases/{context_lease_id}/context
POST /api/data/v1/event-semantics/submissions
POST /api/data/v1/event-semantics/submissions/{submission_id}/reviews
GET  /api/data/v1/events/{event_id}/semantics
```

Existing Bearer service-token scopes, request-size limits, deadlines, retries, replay rules, natural
identity conflicts, transaction atomicity, security errors and Receipt behavior are preserved. The
refactor must not introduce a client-provided idempotency header or a new global HTTP timeout.

## Test Seams And Acceptance

No test is added merely because a file moved. Existing tests are relocated or consolidated under the
strongest observable seam.

### Event

- Highest observable seam: real PostgreSQL Event Publication integration test through the real HTTP
  server, proving create/reuse, conflict, Receipt, transaction rollback and unchanged response.
- Biz seam: validation, convergence, Evidence Link and Tag invariants through fake Ports.
- API/HTTP seam: strict JSON, unchanged route/status/error/request-ID/scope behavior and OpenAPI drift.
- Data seam: persisted Event, Event Evidence and Tag rows are validated before Biz receives them;
  malformed persisted values fail closed.
- Event list and Tag Catalog keep focused read behavior tests without duplicating publication rules.

### Event Semantic

- Highest observable seam: real PostgreSQL Context Lease → Submission → Review → Result read flow
  through the real HTTP server.
- Biz seam: eligibility cursor, pinned context, precheck, replay, review propagation, supersession and
  stable error classification.
- API/HTTP seam: all existing Event Semantic routes, strict DTOs, provider-consumer fixtures and
  historical read compatibility.
- Data seam: persisted manifests, references, enums, status and candidate sets are validated before
  Biz receives them; transaction rollback and cancellation remain observable.

Implementation completion requires focused tests, affected PostgreSQL integration suites, OpenAPI and
fixture checks, `gofmt`, `go vet`, Data binary build, repository architecture/contract checks, secret
scan, and Standards/Spec review. No migration test is triggered because no schema changes are allowed.

## Delivery And Rollback

Event and Event Semantic converge in the same PR because their composition root, Event eligibility,
Event/Evidence snapshots, HTTP server and PostgreSQL acceptance flow are strongly coupled. They remain
separate domain modules and may depend on each other only through the explicit interfaces and immutable
values defined above; sharing a PR does not collapse their ownership boundary.

The PR must be behavior preserving and ready for review, and must not merge itself. Rollback is a normal
binary/code rollback because wire, schema and stored data do not change. There is no mixed-version
provider/consumer rollout requirement.

## Resolved Decisions

- The user confirmed `event` and `eventsemantic` as two separate domains.
- Event Publication, Event list and Event Tag Catalog are use cases or owned values inside `event`, not
  scenario packages.
- Event Semantic is an independent domain, not a file subdivision of Event.
- Internal Event evidence terminology follows the Data Context while existing wire and table names stay
  compatible.
- Semantic Projection is outside this delivery.
- The two domains converge together in one PR so their shared composition and acceptance seams cannot
  drift between deliveries; their source modules and ownership remain separate.

There are no unresolved design questions blocking implementation.
