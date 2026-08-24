---
status: accepted
date: 2026-08-24
issue: 330
extends: 0022-independent-industry-and-concept.md, 0023-independent-chain-node-and-industry-chain.md
---

# Use typed Links for IndustryChain mappings

## Context

Formal IndustryChain-to-Industry and IndustryChain-to-Concept mappings were stored as polymorphic
`entity_edges`. Their relation type was a free string and their endpoint kinds had no direct foreign-key
contract. Every row also carried lifecycle, note, source, verification, and update fields that are not part
of the accepted current mapping fact.

The local database also contained one isolated simulated wafer IndustryChain fixture. Its two nodes,
memberships, mapping, and `produces` relation were test-only and had no Storyline, graph edge, physical
constraint, shared membership, or other retained dependency.

## Decision

- IndustryChain owns dedicated `industry_chain_industry_links` and
  `industry_chain_concept_links` endpoint sets. Each Link contains only its existing ERL identity, typed
  endpoints, and `created_at`; the physical table implies `mapped_to_industry` or `mapped_to_concept`.
- Both endpoints use restrictive foreign keys, each endpoint pair is unique, and ERL identities remain
  unique across the two Link tables and generic `entity_edges`.
- `status`, `evidence_note`, `source_name`, `source_url`, `verified_at`, and `updated_at` are deliberately
  retired. Link presence is the complete current business fact. The migration rejects any formal legacy
  mapping whose `updated_at` differs from `created_at`, because its historical cutoff behavior could not be
  preserved after removing update time.
- Generic `entity_edges` rejects both reserved mapping relation types after cutover.
- Research Graph V1 keeps its existing wire contract and stable ERL identities through a logical union of
  generic Entity Relations and the two typed Link sets. Typed Links project constant active status; their
  immutable creation time is also the effective update time for historical cutoff behavior.
- Migration `000069` deletes the complete known simulated wafer fixture, migrates every formal endpoint
  mapping exactly once, and fails unless every remaining IndustryChain has at least one Industry Link.
- This change adds no authoring/import API, UI, publication workflow, provenance model, or mapping history.

## Release and rollback

`000069` is a coordinated stop-write, forward-only cutover. Operators stop Data and direct writers, capture
a PostgreSQL restore point, confirm the candidate migration is the only pending Data migration, apply it,
deploy the matching Data image, and verify that formal Link counts match the pre-cutover source sets,
reserved mapping rows are absent, the simulated fixture is absent, and every remaining IndustryChain has an
Industry target. Rollback restores the pre-cutover PostgreSQL snapshot and prior Data image together; the
down section does not reconstruct retired metadata.
