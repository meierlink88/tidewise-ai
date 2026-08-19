---
status: accepted
date: 2026-08-19
issue: 286
---

# Move Source management ownership to Data Service

## Decision

Data Service owns the current `Source` domain: validation, initialization, persistence, complete
management reads, dynamic Source creation/deletion, mutable-field replacement, enable/disable and
the active runtime snapshot. This is a new current domain and does not restore or reuse the retired
`source_catalogs` model. Historical migrations remain immutable ledger evidence; no live table,
API, code or documentation may present `source_catalogs` as the current contract.

The `sources` table uses a Data-generated deterministic `SRC + UUID` identity and globally unique,
immutable `code`. A fixed Source cannot be deleted and its `code`, `ownership_type` and
`channel_type` cannot change. Its `adapter_key` and other operational fields can change. Data
validates that `adapter_key` is known but, by explicit product decision, does not validate its
compatibility with `channel_type`. A dynamic Source is created only as `dynamic + rss +
generic_rss` and can be deleted.

The complete current constraints are:

- at most 200 Sources in total and at most 200 active Sources;
- at most one active `web_search` Source;
- `priority` is 1..5, `timeout_seconds` is 1..300 and `max_results` is 1..100;
- `default_source_level` and `config.source_levels` use exactly `L1_OFFICIAL`, `L2_WIRE`,
  `L3_MEDIA` or `L4_SOCIAL`;
- compact UTF-8 `config` is a JSON object no larger than 4096 bytes;
- RSS `config.max_bytes`, when present, is 65,536..10,485,760 bytes; 5,000,000 is the recommended
  default;
- the complete runtime envelope is at most 500,000 bytes.

## API and runtime boundary

The versioned Data API exposes a non-paginated management list, dynamic create, full mutable-field
`PUT`, dynamic delete and a separate non-paginated active snapshot. Admin Backend is the intended
management caller, but its BFF/API and Admin Portal UI are outside Issue #286. All operations use
the existing `DATA_SERVICE_TOKEN` bearer principal with `data.sources.read` or
`data.sources.write`; Data does not introduce a second Source token.

The runtime snapshot sorts by `channel_type`, `priority`, `code`, then `id`. It returns a complete
active set or fails closed within a 3-second server budget. Empty is a valid complete snapshot.
Persistence, integrity, capacity or envelope-size failures never return a partial set and never
fall back to cache. Per the accepted trust boundary, `app_key` remains plaintext in PostgreSQL and
in authenticated management/snapshot responses; there is no `credential_ref` or Secret Manager
resolution in this decision.

External AgentOS will be changed in its own repository and delivery to read exactly one complete
snapshot immediately before each Raw Collection Workflow, freeze it for that workflow, and retain
its existing planning, web/API/RSS adapter concurrency, failure isolation, result processing and
publication behavior. AgentOS must use a small typed client and the provider fixture at
`data-service/backend/api/data/v1/source/testdata/source-snapshot.v1.json`. The accepted removal of
adapter/channel compatibility validation in Data means the future AgentOS consumer must also
accept known mismatched values or deliberately report a whole-snapshot contract failure; silently
dropping an item is forbidden.

Data never connects to the AgentOS PostgreSQL database and never runs AgentOS connectors, parsers,
prompts, schedules, providers or workflows. The one-time operator-controlled import consumes an
export file; it is not a runtime database dependency or a proxy for third-party Provider requests.
Raw Evidence `source_id` and `source_name` remain publication-time evidence snapshots and are not
foreign keys to `sources`.

## Publication, mixed versions and rollback

Migration `000061` installs only schema and database invariants. It does not seed or import Source
facts. A fresh environment runs `source-initialize`, which publishes the seven reviewed fixed
defaults by natural `code`, inserts only missing rows and preserves existing mutable values. A
local/UAT ownership transfer instead freezes Source management, exports the complete AgentOS set
with timestamps, takes recovery points for both databases, applies `000061`, then runs
`source-import` against an empty Data `sources` table. Exact replay is safe; any partial existing
set or content drift fails atomically.

The schema and new Data endpoints are additive, so the new Data release can coexist with old
Admin Backend and old AgentOS versions that do not call them. There is no dual write: AgentOS
remains the sole owner before its later cutover; after the frozen export/import verification and
consumer deployment, Data becomes the sole owner and the old AgentOS management path must be
disabled by that external delivery. Admin Backend adopts the management API only after Data owns
the facts.

Before external consumer cutover, application rollback keeps migration `000061` and imported rows
and deploys the previous Data image; no down migration runs. After consumer cutover, rollback is a
coordinated external operation: stop management and collection starts, restore the pre-cutover
AgentOS Source snapshot and its compatible AgentOS version, then roll back the Data application if
needed. Data's forward-only schema is retained or a reviewed forward repair is applied. A partial
snapshot, reverse synchronization or automatic fallback across the two stores is not a rollback.

## Consequences

Source policy and capacity become one Data-owned contract usable by Admin Backend and AgentOS
without database coupling. Data must retain provider-side OpenAPI, Biz/Data and migration tests;
the later AgentOS delivery must consume the exact fixture and verify fail-closed acquisition startup.
Issue #286 does not implement Admin Portal, Admin Backend or AgentOS changes, and it does not
authorize Data to execute or proxy collection.
