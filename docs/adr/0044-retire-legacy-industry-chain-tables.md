---
status: accepted
date: 2026-08-24
issue: 332
supersedes_in_part: 0013-data-entity-domain-and-projection-retirement.md, 0019-database-independent-domain-object-identities.md, 0023-independent-chain-node-and-industry-chain.md
superseded_in_part_by: 0046-simplify-industry-chain-topology-facts.md
---

# Retire legacy Industry Chain relation and import receipt tables

## Context

Data currently has two overlapping-looking node relation stores. `industry_chain_graph_edges` owns
directional topology inside one IndustryChain, with both endpoints constrained to memberships in that
chain. The older `chain_node_relations` instead stores global ChainNode relations and is not read or
written by current application code. Its dependent `chain_node_physical_constraints` is also not part of
the accepted current ChainNode or graph model.

The Industry relationship package importer was retired from the Data runtime by ADR-0013, but its
`industry_relationship_import_receipts` audit table and immutability function remained in PostgreSQL.
They no longer have an owning command, API, or lifecycle.

## Decision

- `industry_chain_graph_edges` is the only current node-to-node topology fact. It uses the
  `IndustryChainGraphRelationType` vocabulary; the legacy `ChainNodeRelationType` name and global
  `is_subcategory_of` relation are retired.
- `industry_chain_node_memberships` remains the only current ChainNode-to-IndustryChain membership fact.
- Migration `000070` drops `chain_node_physical_constraints`, `chain_node_relations`,
  `industry_relationship_import_receipts`, and the receipt immutability function. It does not use
  `CASCADE`, so an unknown dependency fails the migration instead of being deleted implicitly.
- Legacy ChainNode Relation rows are not copied into `industry_chain_graph_edges`: a global relation does
  not identify the IndustryChain context required by a graph edge, so such a conversion would fabricate
  scope. Historical import receipts are also deleted rather than reinterpreted as current facts.
- `CPC`, `CNR`, and `IRI` are removed from the current closed ID registry. Their appearances in historical
  migrations remain immutable ledger history.
- No public API, new import capability, or Reason Server database dependency is introduced.

## Release and rollback

This is a high-risk, forward-only destructive migration. Operators stop Data and direct writers, capture
a current PostgreSQL recovery point, and confirm `000070` is the only pending migration before applying
it with the candidate image. After apply, they verify ledger version `70`, absence of all three tables and
the orphan receipt function, and unchanged IndustryChain, ChainNode, membership, and graph-edge counts;
only then does the candidate Data binary resume traffic. No old/new binary mixed traffic is allowed during
DDL. The previous Data binary has no runtime access to the retired objects and remains schema-compatible
after the drop, but starting it alone does not restore deleted historical rows.

Application rollback alone cannot reconstruct the deleted historical rows. Rollback restores the
pre-migration PostgreSQL recovery point together with the previous application; no down migration is
provided.

ADR-0046 retains these two topology owners but later narrows their current fact columns and changes their
review/lifecycle semantics to row presence.
