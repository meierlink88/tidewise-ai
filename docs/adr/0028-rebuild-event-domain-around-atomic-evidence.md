# ADR 0028: Rebuild Event Around Atomic Evidence

- Status: Accepted
- Date: 2026-08-18
- Issue: #277

## Context

The accepted Event model still depended on lightweight `raw_documents`, `event_sources`, Event
Tags, publication receipts, dedupe keys, observation timestamps and two review statuses. Data now
owns Raw Evidence and Atomic Evidence directly, so that parallel provenance model duplicates the
Evidence boundary and prevents Event from adopting its intended compact fact shape.

The old rows cannot be converted without inventing semantic, modality and lifecycle values. The
product owner explicitly authorized deleting all Event facts and dependent Research aggregates.

## Decision

Migration `000060` is a forward-only, zero-compatibility cutover. It deletes old Event and dependent
Research facts, retires lightweight Raw Document, Event Source, Event Tag and Event Publication
Receipt persistence, and rebuilds the Event domain around:

- `events`, containing only prefixed ID, title, summary, strict six-key semantic JSON, modality,
  nullable occurred/announced timestamps and lifecycle status;
- `event_evidence_links`, linking every Event directly to at least one Atomic Evidence with an
  independent contribution weight;
- `event_actor_links` and `event_asset_links`, Event-owned relationship snapshots whose target IDs
  are opaque and have no target foreign keys.

Event, Evidence Link, Actor Link and Asset Link identities use `EVT`, `EEL`, `EAC` and `EAS`
respectively. No Event Type, Tag replacement, Actor/Asset entity, external write API or publication
receipt is introduced. The retained Event read contract changes only where removed fields require
it. Research resolves Event evidence through Atomic Evidence and Raw Evidence.

This decision supersedes ADR 0005 and the Event-isolation clauses of ADR 0011. Their historical
migration record remains valid, but they no longer describe the current Event boundary.

## Consequences

Old and new applications cannot share a schema window. Release must stop writers, take a PostgreSQL
recovery point, use the bounded `data_60_cutover`, apply the matching application and database
together, and verify ledger 60 and the retired/new table set. Rollback restores the pre-cutover
snapshot and previous applications together; the Down migration is intentionally unavailable.

Data-generated Raw Evidence and Atomic Evidence remain intact. Existing Event and dependent
Research data are intentionally lost. Actor and Asset referential integrity is deferred until those
domains have an owner and lifecycle.
