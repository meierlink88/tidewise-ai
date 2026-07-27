# Industry Relationship Import V1

Status: implemented
Owner: Data Domain Service
Date: 2026-07-27
Issue: [#123](https://github.com/meierlink88/tidewise-ai/issues/123)

## Goal

Import one frozen, approved Industry–Concept–Industry Chain–Chain Node relationship package into
PostgreSQL as a single, replay-safe transaction. PostgreSQL remains the fact source. Neo4j is a
rebuildable projection and is not part of this importer.

The importer is deliberately separate from the legacy `entity-seed` command. That command rejects
Industry Chain entities in its production Phase A contract and does not write the typed master-data
tables introduced by migration 000027.

## Authorized scope

The command may write only:

- `entity_nodes` rows whose type is `industry_chain`, plus approved new `chain_node` rows explicitly
  present in the frozen package;
- `industry_chain_definitions`;
- `chain_node_profiles` for approved package additions;
- `entity_edges` with `mapped_to_industry` or `mapped_to_concept`;
- `industry_chain_node_memberships`;
- `industry_chain_graph_edges`;
- approved `chain_node_relations` included in the same package;
- one import receipt and its deterministic package counts.

It must not modify existing Industry, Concept or Chain Node identity/content, follow redirects
implicitly, write Event/Theme facts, or project Neo4j.

## Package gate

The package is accepted only when all of the following are true:

1. the relationship Spec path, version and SHA-256 match the manifest;
2. the source inventory freezes the four canonical registries, mapping decisions, topology contract,
   all 18 topology work-item batches, local PostgreSQL snapshot time and evidence cutoff time by
   path and SHA-256;
3. every payload file SHA-256 matches the manifest;
4. `approval_basis=user_explicit_delegated_review`;
5. all relationship rows have `review_status=approved` and `status=active`;
6. all existing endpoints resolve to active entities with approved typed profiles;
7. all IDs match the repository deterministic UUID algorithm;
8. every relation type and direction matches the relationship Spec;
9. all active graph-edge endpoints are active memberships in the same chain;
10. every chain graph is weakly connected and acyclic;
11. every heuristic topology warning is either corrected or retained by an approved, machine-matched
    semantic review record;
12. the global Chain Node `is_subcategory_of` hierarchy is deterministic and acyclic, including
    approved topology-derived nodes added by this package;
13. all 588 frozen Chain Nodes have a formal membership or hierarchy relation;
14. concepts marked `mapped` have a formal chain edge; concepts marked `needs_chain_expansion` are
    deliberately excluded from the current Neo4j projection;
15. the closed-world coverage report has no unresolved, unmapped or projected-orphan rows.

The command does not accept confidence as evidence and does not turn discovery provenance into formal
relationships.

## Build audit manifest

The construction Spec's `relationship_build_manifest.json` and the runtime import manifest are
separate contracts. The build audit is stored at
`analyse-data-service/backend/data/industry_relationships/audit/2026-07-27-v1/relationship_build_manifest.json`.
It records the four input registry versions and counts, deduplication/rejection/pending counts,
relation-decision outcomes, review state, and the exact import package, manifest, source inventory and
Spec SHA-256 values.

The build audit records `database_write_at_freeze.executed=false` because it is immutable once the
package is frozen. Local and UAT writes are recorded afterward by the immutable
`industry_relationship_import_receipts` row rather than by rewriting either manifest.

## Transaction and replay behavior

The package SHA is the batch identity. Import runs under a PostgreSQL advisory lock and one database
transaction.

First application:

1. acquire the batch lock;
2. ensure no conflicting receipt exists;
3. resolve and validate all existing endpoints;
4. insert new Chain Node entities/profiles;
5. insert Industry Chain entities/definitions;
6. insert Industry/Concept mappings;
7. insert memberships;
8. insert chain graph edges;
9. insert approved global node relations;
10. execute post-write counts, topology and orphan checks;
11. persist the import receipt;
12. commit.

Replay with the same package SHA verifies every persisted tuple and returns `unchanged`. The same
semantic identity with different content fails closed; the importer never updates an existing
approved fact in place.

Any error rolls back the entire batch.

## CLI safety

Proposed command:

```text
industry-relationship-import \
  -package /data/industry_relationships/2026-07-27-v1 \
  -expected-sha256 <sha> \
  -dry-run
```

Write additionally requires:

```text
-apply -allow-env local|uat
```

Rules:

- `-dry-run` performs package validation, target database identity validation and a read-only database
  preflight with zero writes.
- `-apply` and `-dry-run` are mutually exclusive.
- `prod` is always rejected.
- the configured app environment must exactly match `-allow-env`.
- UAT apply must run inside the controlled ECS deployment network after an RDS restore point is
  confirmed.

## Minimal Schema delta

The typed tables already support chain definitions, contextual stage/position and the three graph
edge types. The importer requires an additive migration that:

- adds nonblank `industry_chain_node_memberships.inclusion_reason`;
- adds optional nonblank `industry_chain_definitions.technology_route_qualifier`;
- preserves non-empty `industry_chain_definitions.observable_variables` and relation
  `evidence_ids` arrays instead of leaving them only in the release artifact;
- adds primary provenance (`source_name`, `source_url`, `verified_at`) to memberships and graph edges;
  `source_url` accepts an HTTP(S) primary source or an immutable `artifact://` package locator whose
  target is pinned by the manifest SHA-256;
- adds an import receipt table keyed by package SHA;
- fails closed if legacy rows prevent the new nonblank contract from being established.

Business data is not embedded in the migration.

## UAT delivery

The Data image must contain:

- `/usr/local/bin/industry-relationship-import`;
- the exact frozen package directory referenced by its manifest.

UAT execution is an optional, explicitly enabled phase of the manual UAT deployment. It runs after
Data migrations and before the candidate services are activated. The workflow requires:

- `apply_industry_relationship_package=true`;
- the exact 64-character `industry_relationship_package_sha`;
- `confirm_high_risk_backup=true`, confirming an RDS recovery point.

The deployment executor then:

1. run the importer in `-dry-run` mode in ECS;
2. compare counts and package SHA with the approved release artifact;
3. run `-apply -allow-env uat`;
4. rerun the same package and require `unchanged`;
5. require identical package counts across dry-run, apply and replay.

No local workstation connection to the private UAT RDS is assumed.

## Neo4j follow-up

This import makes the PostgreSQL facts projection-ready but does not populate Neo4j. A separate
projector must read only approved, active PostgreSQL facts, use `chain_id` in chain-edge identity and
reconcile projected counts against PostgreSQL. UAT currently has no Neo4j service or configuration, so
UAT graph deployment is a separate change.

The release artifact nevertheless includes a deterministic Neo4j bulk-projection package. Its
validator requires every relationship endpoint to exist, rejects duplicate semantic tuples and
self-loops, and rejects any projected Industry, Concept, Industry Chain or Chain Node with no formal
relationship. Concepts awaiting a structurally valid new chain remain PostgreSQL master data but are
not emitted as isolated Neo4j nodes.
