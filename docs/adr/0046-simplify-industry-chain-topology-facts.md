---
status: accepted
date: 2026-08-24
issue: 336
supersedes_in_part: 0023-independent-chain-node-and-industry-chain.md, 0044-retire-legacy-industry-chain-tables.md
---

# Simplify Industry Chain topology facts

## Context

`industry_chain_node_memberships` and `industry_chain_graph_edges` mixed current topology with review,
lifecycle, provenance, evidence, and explanatory fields left by the retired relationship-package model.
Every current row is `approved` and `active`, so those flags do not distinguish any retained business
state. Provenance and narrative text have no current writer or independent lifecycle on these topology
facts. Their presence also forced graph reads and downstream DTOs to repeat policy fields that were not
part of topology identity.

## Decision

- A membership row is the current formal membership fact. It retains only `industry_chain_id`,
  `chain_node_id`, `position`, `contextual_stage`, `created_at`, and `updated_at`.
- A graph-edge row is the current formal directional topology fact. It retains only `id`,
  `industry_chain_id`, `from_chain_node_id`, `to_chain_node_id`, `relation_type`, `created_at`, and
  `updated_at`.
- Migration `000072` removes the obsolete review, lifecycle, provenance, evidence, mechanism, condition,
  segment, omission, and inclusion columns without `CASCADE`. It replaces status-dependent indexes and
  the status-dependent membership guard.
- The graph stays acyclic. The rebuilt cycle trigger locks endpoint memberships and serializes mutation
  per IndustryChain, but evaluates every row because row presence now means a current fact. Existing
  endpoint membership foreign keys, uniqueness, controlled relation types, and timestamp ordering remain
  enforced.
- Data Research Graph Search publishes `research-graph-search.v2`. Membership and edge objects contain
  only the retained topology fields. There is no V1 compatibility alias or mixed-version response.
- The Miniapp Reasoning Tree `incoming_graph_edge` projection retains only `id` and `relation_type`.
  `incoming_transmission_mechanism` and `incoming_condition_summary` remain independent immutable analyst
  snapshot facts; they are not sourced from the formal graph-edge row.
- The test “compressed edge requires omitted step” is obsolete because compressed-segment classification
  and omission notes no longer belong to the topology model. Acyclicity and retained vocabulary remain
  covered.

## Release and rollback

This is a high-risk, forward-only coordinated schema and contract cutover. Operators stop Data and all
direct writers, capture a PostgreSQL recovery point, confirm migration `000072` is the only pending
migration, and deploy the matching Data and Miniapp binaries without old/new mixed traffic. Before apply,
they record membership and graph-edge row counts and confirm no timestamp-order violations. After apply,
they verify ledger version `72`, the exact retained columns, unchanged row counts and endpoints, the four
replacement indexes, and rejection of a cycle-forming edge.

Application-only rollback is incompatible with the narrowed schema and V2 contract. Full rollback restores
the pre-migration PostgreSQL recovery point together with the previous Data and Miniapp versions; no down
migration is provided.
