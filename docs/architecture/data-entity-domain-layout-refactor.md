# Data Entity Domain Layout Refactor

Status: approved for implementation

Owner: Data context

Date: 2026-08-12

Issue: [#205](https://github.com/meierlink88/tidewise-ai/issues/205)

## Outcome

Converge Data-owned Entity code on the repository Kratos domain layout, remove the historical shared
model bucket, and retire every Data-owned Entity seed, relationship-package import, Neo4j projection,
and Qdrant projection capability. PostgreSQL remains the authoritative Entity fact store.

The refactor preserves existing Data HTTP paths, methods, authentication, status codes, DTOs, database
schema, migration ledger, persisted rows, and PostgreSQL consumer query behavior except for the
explicitly approved runtime-health contraction that removes Neo4j.

## Domain ownership

The Entity domain owns:

- formal Entity identity, type, stable key, names, aliases, status, external identifiers and redirects;
- Entity profiles including Company, Market, Sector, Industry, Concept, Industry Chain, Chain Node,
  Benchmark and the other currently persisted profile types;
- global Entity Relations, Industry Chain Definitions, contextual Memberships and Industry Chain Graph
  Edges, Chain Node Relations and physical constraints;
- Entity Type Definition TBox records, including their version, names, business definition, inclusion
  and exclusion criteria, Event-link and Signal-subject capabilities, allowed Event roles and status;
- Benchmark Observation facts, including observed time, value, unit, source and quality state.

`Industry Chain Graph Edge` remains a PostgreSQL domain fact. Its use of “Graph” describes directed
Industry Chain topology and does not make it a Neo4j projection.

Event Semantic and Research Analysis Context consume Entity Type Definitions but do not own duplicate
canonical definitions. Consumer-owned queries may continue to read Entity tables inside their own
consistent PostgreSQL seam, but they map into and validate against Entity-owned types before returning
persisted data to Biz.

Raw Document and Ingest Status do not belong to Entity or Event. They form a separately named,
deprecated Raw Document compatibility domain so the existing administrative read API can remain
stable and later be removed as one vertical slice. The compatibility domain owns no new write path.

## Removed capabilities

Data Service no longer owns or ships:

- Entity seed parsing, validation, idempotent application or historical seed modes;
- Phase A, Sector convergence, Alliance/Economy rebuild, frozen external-identifier batches or frozen
  Chain Node relation batches;
- Industry relationship package loading, validation, preflight, import, replay or receipt creation;
- Entity-to-Neo4j graph projection, Neo4j graph inspection or replacement;
- Entity/Variable Definition-to-Qdrant projection, embedding calls or Qdrant collection replacement;
- CLI binaries, Docker entries, local Compose operations, UAT deployment inputs, secrets or CI
  contracts dedicated to those retired capabilities.

Those authoring and construction capabilities move to the separately owned Tidewise Reason project.
This change does not create a Tidewise Reason integration or allow Tidewise Reason to access the Data
PostgreSQL database. A future Reason-to-Data publication contract requires a separate reviewed design.
Until then, the formal Entity catalog is operationally read-only through Data Service, apart from the
already existing Benchmark Observation persistence capability.

## Target modules and dependency direction

Entity uses the stable singular domain name in Biz and Data. It initially needs only the fixed `biz.go`
and `data.go` responsibilities. No Entity API, Service or transaction responsibility is created because
this change adds no Entity publication wire contract or multi-step atomic Entity write use case.

The Raw Document compatibility capability uses the stable singular `rawdocument` domain across API,
Biz, Data and Service while keeping its current public route and wire contract unchanged.

The intended dependency direction is:

```text
Raw Document HTTP -> Raw Document Service -> Raw Document Biz Port <- Raw Document Data Adapter

Event Semantic / Research Analysis Context -> Entity-owned types and validation
                                         -> consumer-owned PostgreSQL query seams

Entity Biz Port <- Entity Data Adapter -> PostgreSQL
```

Biz owns domain types, controlled enums, validation, ordinary Ports and public business methods. Data
owns SQL, row scanning, null mapping, stable ordering, persistence errors and fail-closed validation.
No Biz package performs filesystem, SQL, Neo4j, Qdrant, embedding or environment access.

`transaction.go` is not pre-created. If a future Entity publication use case needs an atomic Bundle,
Biz will define the business transaction Port and commands and Data will implement begin, lock, read,
write, commit and rollback in the two domain transaction responsibility files.

## Type convergence

- The Go domain name for a formal row is `Entity`; the physical `entity_nodes` table remains unchanged.
- The Go domain name for a global relationship is `EntityRelation`; the physical `entity_edges` table
  remains unchanged.
- Entity Type Definition moves from Event Semantic ownership to Entity ownership without changing its
  current Event Semantic or Research Analysis wire projections.
- Benchmark Observation moves into Entity ownership as a fact attached to a Benchmark Entity.
- Raw Document and Ingest Status move out of the generic model bucket into the Raw Document
  compatibility domain.
- Research-only historical constants do not move into Entity. Unused constants and tests may be removed
  as dead implementation; live Research types remain owned by Research.
- Retired generic `model`, scenario-named Entity seed/import packages, wrappers and aliases are removed
  once all live consumers use their owning domains.

## Runtime-health contract

Data Runtime Health continues to expose the same authenticated route, request, envelope, status and
timeout contract, but its service list contains only Data Service. It no longer probes, configures,
starts, closes or reports Neo4j.

Admin Portal continues to combine Data and AgentRun provider health. Its stable ordered service list is
Data Service, AgentRun and Qdrant. Neo4j is removed from the Admin provider DTO, validation, aggregate
logic and monitoring UI. AgentRun Qdrant health remains unchanged; this task does not modify AgentRun's
Qdrant reader contract.

## Historical data, schema and artifacts

- Existing migration files remain immutable and the complete forward ledger must still apply.
- No table, index, constraint, receipt or persisted row is dropped or rewritten.
- Historical graph-projection and Industry-import tables remain dormant audit history with no runtime
  reader or writer.
- Frozen seed and relationship artifacts may remain in repository history/current audit assets, but
  Data commands, Docker image operations and deployment automation do not consume them.
- Removing active projection code does not delete or mutate independently operated Neo4j or Qdrant
  infrastructure or stored data.

## Compatibility, rollout and rollback

The Data image stops shipping the four retired operational binaries: Entity seed, Industry relationship
import, Industry graph projection and Event Semantic projection. UAT no longer accepts or executes their
deployment gates.

Data Server loses its Neo4j driver/configuration lifecycle. PostgreSQL readiness, healthz, readyz and all
non-runtime-health APIs remain unchanged. Application rollback restores the previous image and code but
does not run down migrations or mutate external Neo4j/Qdrant state.

The accepted Event Semantic Qdrant design is paused at its consumer boundary: AgentRun retains its
Qdrant reader and health contracts but does not start or notify the Event Semantic worker. No Data-owned
process refreshes the collections. Worker activation remains disabled until a new projection owner and
rollout contract are approved.

## Testing decisions

Tests observe behavior rather than package moves or file lists.

- Entity Biz tests cover the existing Entity/Profile/Entity Type Definition/Benchmark Observation
  validation behavior that remains live after consolidation.
- Entity Data tests cover Benchmark Observation persistence, stable reads and malformed persisted-row
  fail-closed behavior. Existing consumer Data tests continue to cover Entity Type Definition reads.
- Raw Document HTTP/OpenAPI tests prove its existing route, query validation, envelope, error mapping and
  response DTO remain unchanged after vertical-slice convergence.
- Data and Admin Runtime Health provider/consumer tests prove the new three-service aggregate and removal
  of the Neo4j dependency.
- Architecture/build/deployment checks prove retired binaries, adapters, credentials and UAT projection
  inputs are absent from active runtime delivery. They do not test mechanical directory layouts.
- Projection and retired-tool tests are removed as `obsolete` rather than copied into the new Entity
  domain.
- No migration-specific unit test is added. The unchanged complete ledger remains covered by the stable
  PostgreSQL forward smoke.

## Non-goals

- No Tidewise Reason implementation or cross-project publication API.
- No Entity HTTP publication API, CRUD API or management UI.
- No direct Tidewise Reason access to Data PostgreSQL.
- No Entity schema change, migration rewrite, destructive cleanup or historical data backfill.
- No deletion or lifecycle management of external Neo4j/Qdrant infrastructure or data.
- No AgentRun Qdrant retrieval rewrite.
- No semantic change to Entity, Relation, Industry Chain, Entity Type Definition or Benchmark Observation
  vocabularies.
- No change to Event, Event Semantic, Evidence or Research business behavior beyond their type ownership
  imports and the explicitly paused projection freshness assumption.

## Acceptance

- No live Data package uses the generic Entity model bucket or scenario-named Entity seed/import modules.
- Entity and Raw Document source follows the current Kratos backend layout responsibility names.
- Existing Raw Document and non-runtime-health Data HTTP contracts remain byte-compatible at their
  observable seams.
- Data and Admin Runtime Health omit Neo4j and retain safe failure mapping for remaining providers.
- Data binaries and deployment automation contain no seed/import/Neo4j projection/Qdrant projection
  execution path.
- Data long-running binary has no Neo4j driver or Neo4j credential/config dependency.
- Entity Type Definition and Benchmark Observation are owned and validated by Entity.
- Historical migrations, tables and rows remain untouched and the forward ledger smoke passes.
