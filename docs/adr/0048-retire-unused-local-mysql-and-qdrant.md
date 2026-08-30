---
status: accepted
date: 2026-08-30
issue: 349
supersedes_in_part: 0012-docker-only-service-runtime.md, 0020-local-docker-application-and-infrastructure-projects.md, 0027-retire-agent-run.md, 0042-transfer-local-reasoning-neo4j-ownership.md
---

# Retire unused local MySQL and Qdrant infrastructure

## Context

The current local runtime uses PostgreSQL for Data facts and MinIO for public evidence artifacts,
but none of its four application services connects to the MySQL or Qdrant containers provisioned by
`tidewise-infra`. AgentRun is retired, and local reasoning infrastructure is owned by other
repositories without a MySQL or Qdrant runtime dependency. Keeping the two unused services running
consumes resources and falsely presents them as part of the supported local contract.

UAT has separate infrastructure ownership. Its OpenSPG MySQL stores provider project metadata, and
its independently operated Qdrant remains outside the local developer-machine lifecycle.

## Decision

- The local `tidewise-infra` project owns PostgreSQL, MinIO and the shared `tidewise-local` network.
- Local MySQL and Qdrant services, ports, image declarations, environment examples and automatic
  volume provisioning are retired.
- Rollout explicitly stops and removes only the local Compose services `mysql` and `qdrant`.
- Existing named volumes `tidewise-reason_mysql-data` and `tidewise-qdrant-local-storage` leave the
  active contract but remain physically preserved. Docker image cache is also retained.
- UAT MySQL, UAT MinIO, UAT Neo4j provider metadata and independently operated UAT Qdrant are not
  changed by this decision.

## Consequences

Normal local infrastructure startup creates and reconciles only PostgreSQL and MinIO, while the four
application services keep their existing runtime and health contracts. No database migration, API
change or application compatibility window is required.

Rollback restores the previous local Compose, environment example and volume bootstrap definitions,
then recreates the two services against the preserved volumes. Deleting either retired volume or
changing UAT infrastructure requires a separate reviewed decision and explicit data-destruction or
environment authorization.
