---
status: accepted
date: 2026-08-25
issue: 339
supersedes_in_part: 0028-rebuild-event-domain-around-atomic-evidence.md
---

# Publish Events with occurrence identity semantics

## Context

Reasoning now owns the comparison between a proposed Event and historical Events. The former Event
5W1H JSON does not expose the identity dimensions needed to distinguish one announcement,
implementation, update, suspension, or termination from another. Reasoning also needs a safe retry
boundary when Data commits an Event but the HTTP response is lost.

## Decision

- Event `semantic` has exactly seven fields: non-empty `actors`, nonblank `action`, non-empty
  `objects`, controlled `stage`, `jurisdictions`, nullable UTC `effective_at`, and controlled
  `time_precision`.
- `POST /api/data/v1/events` is the only external Event publication operation. The authenticated
  publisher supplies a `publication_key`, one Event without identity or lifecycle status, and at
  least one formal Atomic Evidence ID.
- Data always creates the Event as `ACTIVE`, generates all `EVT` and `EEL` identities, and writes the
  Event, initial Evidence Links at contribution weight `1.0`, and an `EPR` publication receipt in one
  PostgreSQL transaction.
- A replay with the same publisher, key, and payload returns the original Event without writes. Key
  reuse with a different payload is a conflict.
- The receipt is transport idempotency only. It does not identify a real-world Event and Data does
  not perform semantic deduplication.

## Consequences

Migration `000073` and the matching application contract must deploy together. Existing Event rows
must be absent before the constraint is replaced; the migration fails closed rather than inventing
identity semantics from the incompatible legacy 5W1H JSON. The coordinated stop-write deployment
requires a recovery point and does not support mixed old/new binaries. Reasoning may safely retry
publication, while duplicate Event decisions remain entirely outside Data.
