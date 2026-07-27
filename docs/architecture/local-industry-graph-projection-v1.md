# Local Industry Graph Projection V1

Status: approved for implementation
Owner: Data Domain Service
Environment: local only
Date: 2026-07-27
Issue: [#125](https://github.com/meierlink88/tidewise-ai/issues/125)

## Outcome

Provide one supported, repeatable command that rebuilds the approved Industry–Concept–Industry
Chain–Chain Node structure from local PostgreSQL into local Neo4j.

PostgreSQL remains the fact source. Neo4j is a disposable query projection. V1 proves the complete
projection and traversal contract locally before any UAT Neo4j topology is designed.

The frozen Industry relationship package is the independent projection baseline:

- package SHA-256:
  `7c737410ac6af562af19f8b9dad9e8e1c802f8f782625bd360bb2e8f20768608`;
- `neo4j/nodes.csv` SHA-256:
  `2710ca446884fbdb2d54a52f730d2d5c4bf991608074197b3096859e039e3ea7`;
- `neo4j/relationships.csv` SHA-256:
  `50a3d59e277724e4c4071e3ea545466a2c503b3c05dffb1337cbd68991b18e56`;
- expected graph: 4,449 nodes and 7,867 relationships.

## Non-goals

V1 does not:

- deploy Neo4j to UAT or production;
- make Data Server depend on Neo4j or expose a new HTTP API;
- implement CDC, incremental projection, scheduling, polling or dual writes;
- project Event, Research Theme, Research Anchor, Evidence nodes or investment conclusions;
- infer new relationships from graph connectivity;
- restore the retired generic `graph-projector` or its permissive `RELATED_TO` fallback;
- delete or cut over the historical local `projection_namespace=tidewise` graph.

The current local Neo4j contains a historical 981-node/345-relationship projection. V1 preserves it
and writes the new graph under a distinct namespace.

## Owner map

| Responsibility | Owner |
|---|---|
| Entity and relationship facts | Data Domain Service PostgreSQL |
| Frozen approved package and projection baseline | Data Domain Service versioned data assets |
| Projection rules and validation | Data Domain Service Biz module |
| PostgreSQL snapshot extraction | Data Domain Service PostgreSQL Adapter |
| Neo4j atomic replacement and inspection | Data Domain Service Neo4j Adapter |
| Local command composition and target safety | `cmd/industry-graph-projector` |
| Product reads | unchanged; no caller in V1 |

## Chosen reference

Local authority:

- `relation_spec.md` section 16 defines the allowed labels, relationship types, direction and
  chain-scoped properties.
- `industry-relationship-import-v1.md` requires approved/active PostgreSQL facts, `chain_id` in
  chain-scoped edge identity and PostgreSQL/Neo4j count reconciliation.
- the frozen Neo4j CSV files define the independent expected node and relationship sets.
- the local runtime is Neo4j Community 5.26.28.

Official Neo4j reference:

- Neo4j Go Driver v6.2.0 is compatible with Neo4j 5.x and the repository Go 1.25 baseline.
- an explicit Neo4j transaction is used so multiple writes and client-side verification occur
  before one explicit commit; any error rolls the replacement back.
- named composite property uniqueness constraints are available in Neo4j Community and apply only
  when all constrained properties exist.

Adopted:

- one explicitly selected Neo4j database per session;
- query parameters rather than string-interpolated data;
- one explicit transaction for namespace deletion, creation and complete verification;
- named node and relationship property uniqueness constraints;
- bounded context and transaction timeouts.

Rejected:

- `neo4j-admin database import full`, because this graph is small and the command would require an
  offline whole-database replacement;
- APOC and Graph Data Science, because neither is needed for a deterministic structural projection;
- managed write transaction retries, because V1 performs a deliberate namespace replacement and
  verifies it within one explicit transaction;
- generic dynamic relationship strings; only the seven frozen relation types are emitted.

## External module interface

The deep module exposes one operation:

```go
Project(ctx context.Context, request Request) (Result, error)
```

The request fixes:

- validated frozen package and projection baseline;
- projection namespace;
- dry-run or apply mode.

The result reports:

- package SHA-256 and projection contract version;
- PostgreSQL node/relationship counts and semantic fingerprints;
- current and final Neo4j counts and semantic fingerprints;
- `dry_run`, `applied` and `unchanged`;
- per-node-type and per-relationship-type counts;
- orphan, duplicate, self-loop and missing-chain-identity counts.

The module hides PostgreSQL query ordering, CSV parsing, canonical hashing, relationship routing,
Neo4j batching, Cypher, constraints and transaction handling.

Internal seams:

- `SnapshotReader` returns one repeatable-read, read-only PostgreSQL snapshot.
- `ProjectionStore` inspects or atomically replaces one Neo4j namespace.

The production PostgreSQL/Neo4j Adapters and focused fakes are the two implementations used at
these seams. They are not exposed by the CLI.

## CLI contract and target safety

Command:

```text
industry-graph-projector \
  -package <approved-package-directory> \
  -expected-sha256 <64-lowercase-hex> \
  -dry-run
```

Writing requires:

```text
-apply -allow-env local
```

Rules:

- exactly one of `-dry-run` or `-apply` is required;
- `-expected-sha256` is mandatory;
- V1 accepts only `APP_ENV=local`;
- PostgreSQL must use a loopback/local host and database `tidewise_local`;
- Neo4j must use a loopback host and database `neo4j`;
- credentials come only from `TIDEWISW_DB_PASSWORD`, `NEO4J_USERNAME` and
  `NEO4J_PASSWORD`;
- the fixed namespace is `tidewise-industry-v1`;
- callers cannot select another namespace or delete arbitrary graph data;
- output and errors never include credentials, database URLs, SQL or raw driver errors.

The command uses a bounded total timeout and closes both database clients explicitly.

## Projection contract

### Nodes

Every node has:

```text
(:TidewiseEntity:<SpecificType> {
  entity_id,
  entity_key,
  entity_type,
  canonical_name,
  aliases,
  status: "active",
  projection_namespace: "tidewise-industry-v1",
  projection_contract_version: "industry-graph-projection-v1",
  source_package_sha256
})
```

Specific labels and exact counts:

| Label | Count |
|---|---:|
| `Industry` | 512 |
| `Concept` | 180 |
| `IndustryChain` | 708 |
| `ChainNode` | 3,049 |

Only approved, active typed entities referenced by at least one projected relationship are emitted.
The 14 approved Concepts marked for future chain expansion and all 18 candidate Concepts remain in
PostgreSQL but are not emitted as isolated nodes.

### Relationships

Every relationship has:

```text
relation_key
projection_namespace
projection_contract_version
source_package_sha256
status: "active"
```

| PostgreSQL source | Neo4j type | Direction | Count |
|---|---|---|---:|
| `industry_profiles.parent_industry_entity_id` | `IS_SUBCATEGORY_OF` | child Industry → parent Industry | 480 |
| `entity_edges(mapped_to_industry)` | `MAPPED_TO_INDUSTRY` | Industry Chain → Industry | 716 |
| `entity_edges(mapped_to_concept)` | `MAPPED_TO_CONCEPT` | Industry Chain → Concept | 521 |
| `industry_chain_node_memberships` | `HAS_NODE` | Industry Chain → Chain Node | 3,350 |
| `industry_chain_graph_edges(input_to)` | `INPUT_TO` | input/supplier → user/transformer | 1,537 |
| `industry_chain_graph_edges(is_component_of)` | `IS_COMPONENT_OF` | component → assembly/system | 704 |
| `industry_chain_graph_edges(depends_on)` | `DEPENDS_ON` | dependent → dependency | 404 |
| `chain_node_relations(is_subcategory_of)` | `IS_SUBCATEGORY_OF` | child Chain Node → parent Chain Node | 155 |

Properties by family:

- `MAPPED_TO_*`: `chain_id`, `mechanism` from the persisted mapping evidence note;
- `HAS_NODE`: `chain_id`, `contextual_stage`, `position`, `mechanism` from inclusion reason;
- chain graph edges: `chain_id`, `mechanism`;
- global Chain Node hierarchy: `mechanism`;
- Industry hierarchy: fixed mechanism `Authoritative Industry classification hierarchy`.

V1 does not fabricate Evidence nodes or identifiers. PostgreSQL does not currently persist the full
9,736-item relationship evidence registry, and `mapped_to_*` rows retain only their evidence note.

## PostgreSQL snapshot

The PostgreSQL Adapter opens one `REPEATABLE READ READ ONLY` transaction and:

1. requires the exact `industry_relationship_import_receipts.package_sha256`;
2. loads approved, active typed entities;
3. loads all eight relationship families in deterministic order;
4. constructs canonical relation keys from endpoint entity keys and the frozen relation type;
5. filters nodes to the relationship endpoint closure;
6. commits the read-only transaction only after the complete snapshot is materialized.

The Biz module then compares the complete semantic node and relationship sets with the frozen CSV
baseline. Count equality alone is insufficient.

## Neo4j replacement

Schema setup is idempotent and precedes the data transaction:

- one composite uniqueness constraint for
  `:TidewiseEntity(projection_namespace, entity_id)`;
- one metadata uniqueness constraint for
  `:TidewiseProjection(projection_namespace)`;
- one composite relationship uniqueness constraint per allowed relationship type for
  `(projection_namespace, relation_key)`.

Apply behavior:

1. inspect the fixed namespace;
2. if its canonical content equals the validated source snapshot, return `unchanged=true`;
3. begin one explicit write transaction;
4. delete only `:TidewiseEntity` and `:TidewiseProjection` records in the fixed namespace;
5. create all nodes in deterministic bounded batches;
6. create all relationships in deterministic bounded batches using static Cypher per allowed type;
7. create projection metadata with package SHA, fingerprints, counts and projection time;
8. read the full namespace inside the same transaction and recompute canonical fingerprints;
9. require exact counts, type counts, endpoint closure and fingerprints;
10. commit; on any error explicitly roll back.

The historical `projection_namespace=tidewise` nodes do not carry the new base label and are outside
the deletion match.

## Hard gates

Before any Neo4j write:

1. package, manifest, relation Spec and projection-file SHA values are valid;
2. the exact package receipt exists in local PostgreSQL;
3. every source entity is approved and active;
4. node IDs and keys are unique;
5. relationship keys are unique;
6. relationship types, endpoint types and directions match the frozen registry;
7. no self-loop or missing endpoint exists;
8. every chain-scoped relationship has a nonblank `chain_id`;
9. every chain graph edge joins active memberships in that same chain;
10. no projected node is isolated;
11. the PostgreSQL semantic sets exactly equal the frozen CSV sets;
12. expected totals are 4,449 nodes and 7,867 relationships.

The same gates run on the transaction-visible Neo4j result before commit.

## Rollout and rollback

Rollout:

1. implement and test the command without changing Data Server;
2. run `-dry-run` against local PostgreSQL and local Neo4j;
3. run `-apply -allow-env local`;
4. rerun the same command and require `unchanged=true`;
5. execute traversal and count queries in Neo4j Browser/cypher-shell.

Rollback:

- a failed transaction leaves the previous V1 namespace unchanged;
- after a successful local apply, V1 can be removed by deleting only
  `:TidewiseEntity`/`:TidewiseProjection` with
  `projection_namespace=tidewise-industry-v1`;
- PostgreSQL is never changed by the projector;
- the historical local graph remains untouched.

UAT requires a separate topology, credentials, deployment and restore/cutover design.

## Verification seams

Frozen acceptance criteria already establish the test seams.

Default:

- Biz seam: complete package/PG/Neo4j projection behavior through `Project`;
- API/HTTP seam: not involved.

Conditional:

- Data seam: real PostgreSQL snapshot extraction and Neo4j transaction Adapter;
- Conf seam: local-only target and credential validation;
- Architecture seam: Data Server remains free of Neo4j configuration/dependencies;
- Container seam: one local Neo4j 5.26.28 smoke.

No Migration, provider/consumer, Frontend or AgentRun seam is involved.

Required final checks:

- focused Red/Green tests for the Biz module;
- Data Adapter tests;
- CLI target-safety tests;
- Data affected suite, format, vet and binary build;
- repository architecture contracts;
- local dry-run, apply, replay and Cypher count/traversal smoke.
