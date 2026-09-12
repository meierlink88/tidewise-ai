# ADR-0064: Retire generic Entity storage and clarify node table names

- Status: accepted by user request (Issue #488)
- Date: 2026-09-12

## Decision

Rename `chain_node` to `industry_chain_node` and `industry_chain_graph_edges` to
`industry_chain_node_graph` without changing rows, IDs, membership or topology.
These are node-to-node edges scoped to an IndustryChain, not edges between IndustryChains.
Preserve existing API paths, JSON names and CND/IGE IDs for retained objects.
This updates ADR-0044 and ADR-0046 physical naming without weakening their invariants.

Drop `entity_nodes`, `entity_edges`, `policy_body_profiles`, `person_profiles`,
`instrument_profiles`, `index_profiles`, `security_profiles`, `theme_profiles`,
`commodity_profiles` and `market_profiles`, including all their rows.
The user explicitly included market_profiles after its dependency was identified.
Remove their runtime SQL, generic Entity/Profile models and obsolete tests.
The standalone generic CRUD/profile endpoints were already absent; retained Research
Graph routes now read only independent facts, typed links and Organization memberships.
Retired ENT IDs are no longer accepted as Research Graph object seeds. ERL remains
valid for typed mappings and projected Organization membership relations.

## Migration and ownership

Data owns forward-only migration 000091. Historical migrations remain immutable.
Use explicit drops, update identity/reference and cycle functions, and retain foreign keys,
row identity, typed-link collision protection and cycle rejection. Update initialization SQL
physical table selection without rewriting the reviewed short-name catalog.

Stop Data traffic and take a database backup before migrating and starting the matching
application. No mixed-version compatibility or alias tables. Restore both the backup and
matching old application to roll back. UAT is marked high-risk and is not applied by this task.

## Verification

Compare every retained business table's row count and content fingerprint before/after,
including the renamed tables. Verify no retired tables or runtime SQL references remain.
Run the complete forward ledger on an empty disposable database and affected real PostgreSQL
Biz/API/Data tests, vet and build. Keep topology cycle and typed-link invariant coverage.
Retired generic/profile and old Economy cutover tests are obsolete; the current country
catalog checks and generic forward-ledger smoke remain.
