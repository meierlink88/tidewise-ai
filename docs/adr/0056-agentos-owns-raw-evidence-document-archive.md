---
status: accepted
date: 2026-09-04
issue: 245
supersedes_in_part:
  - 0011-data-owns-raw-evidence-and-evidence-publication.md
  - 0055-retire-uat-reasoning-runtime.md
---

# AgentOS owns the Raw Evidence document archive

## Context

Raw Collection must preserve the complete collected Markdown before publishing the formal Raw
Evidence record. Data Service currently stores `raw_text` immutably and returns it unchanged, while
the historical contract describes that value as the complete article body. The DGX AgentOS UAT
runtime now owns its own MinIO dependency. Data Service and Admin Portal must not acquire a direct
dependency on that AgentOS infrastructure.

Historical UAT objects and links can remain on the retained Huawei Cloud MinIO lifecycle. New DGX
objects do not require a public route; a browser outside the reachable network may therefore be
unable to open a newly published path. This does not change Admin Portal behavior.

## Decision

- AgentOS uploads the complete immutable Markdown document to its owned MinIO before publishing the
  Raw Evidence record to Data Service.
- New publications keep the existing `raw_text` JSON field and PostgreSQL column, but the value is an
  environment-neutral `/{bucket}/{object_key}` relative path. It excludes scheme, host and port.
- Data Service stores, hashes and returns that value unchanged. It does not fetch, validate, proxy,
  authenticate to or operate AgentOS MinIO. `content_hash` protects the persisted field value; it is
  not an object-content checksum.
- Existing rows whose `raw_text` contains article bodies remain valid and readable. No migration or
  backfill rewrites them, so an identical historical publication can still replay safely.
- Admin Portal code, API contracts and link-construction behavior remain unchanged. This decision
  adds no public route and no availability guarantee for the DGX object origin.

## Consequences

Raw Evidence remains the Data-owned formal record and identity, while the complete raw document and
its storage lifecycle are AgentOS-owned. Consumers must tolerate both historical body values and new
relative paths. Moving, exposing or migrating archived objects is a separate storage decision and
cannot be inferred from a Data or Admin release.
