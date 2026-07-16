# Apply-final Review Package

Status: prepared for human Apply-final Review. This evidence does **not**
approve Sync, Archive, Deliver, push, PR creation, or any additional database
operation.

## Scope reviewed

The branch diff against `origin/main` contains only this change's backend and
OpenSpec artifacts: the reviewed-outbox CLI and its tests; the event-import
domain, application service, PostgreSQL transaction repository and tests;
`000020_add_event_import_receipts_and_tag_seed.sql`; the required-v1 fixture;
and R2 review/recovery/evidence artifacts. It does not change the Agent
project, `event_entity_links`, Neo4j, frontend, deployment, `doc/`, or
`prototype/`.

The reviewed implementation preserves the approved boundary:

- CLI input remains the single-event top-level reviewed-outbox transport and
  exposes `--file`, `--dir`, `--dry-run`, strict machine JSON, deterministic
  IDs/counts/hashes, and the frozen exit-code classes.
- `EventImportService` plans deterministic identities and performs source
  resolution, raw/event/evidence/Tag persistence, receipt insert, replay, and
  result verification inside one transaction. Same-key replay is serialized by
  advisory transaction lock plus receipt row lock; a different payload hash
  returns the idempotency conflict before same-hash identity checks.
- Migration `000020` is forward-only, preserves existing data, seeds only the
  fixed Event Agent source and the 22 frozen Tag tuples, and blocks source or
  Tag UUID identity drift instead of silently accepting it.

## Task and R2 state

`openspec instructions apply --change add-event-import-and-tag-catalog --json`
reports 9/9 tasks complete. Tasks 2.1--2.3 are factual records of the accepted
local R2 execution, not authorization for another database operation.

R2 evidence records the sole `000020` execution from migration version 19 to
20, a new pre-migration custom backup, postflight schema/source/Tag assertions,
the authorized synthetic integration cleanup/recovery, and reviewed-outbox
dry-run zero-write verification. Final local assertions were:

| Assertion | Recorded value |
|---|---|
| environment identity | `tidewise_local` / `tidewise` / loopback |
| migration version | 20 |
| active frozen Tags | 22 (10 `news_category`, 12 `index_category`) |
| fixed Event Agent source | 1 active row |
| synthetic receipts/events/raw documents | 0 / 0 / 0 |
| final receipt/event/source/tag-map/raw-document counts | 0 / 0 / 0 / 0 / 407 |

No real Event import, UAT/prod/shared operation, Neo4j operation,
`event_entity_links` write, restore, repeat migration, or forward fix is part
of this evidence.

## Self-review focus and residual risks

The review checked transaction rollback propagation, receipt replay identity
and result-ID validation, UUID-array JSON decoding, deterministic cleanup
registration, migration identity guards and forward recovery, dry-run's
frozen-Tag validation, CLI machine-output/error classification, and diff
secrets/scope.

Residual operational risks are intentionally fail-closed: a current-v0 Agent
outbox lacking required-v1 review/evidence/Tag fields is rejected; local R2
evidence exercises synthetic fixtures and dry-run rather than a real business
Event import; and future Agent contract changes require their own reviewed
change. No credentials, DSNs, passwords, tokens, or backup contents are stored
in this package.

## Required next gate

Human Apply-final Review must approve this scoped diff and fresh validation
before any Sync. Until that approval, this change remains active and must not
be synced, archived, delivered, pushed, or proposed as a PR.
