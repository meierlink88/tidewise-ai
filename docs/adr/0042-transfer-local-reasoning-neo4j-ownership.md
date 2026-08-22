---
status: accepted
date: 2026-08-21
issue: 12
supersedes: 0041 (local ownership only)
---

# Transfer local reasoning Neo4j ownership

## Decision

The `tidewise-reason` repository owns the local reasoning-specific Neo4j provider, its volumes and
its lifecycle. Tidewise AI no longer provisions or verifies local Neo4j because Data, Miniapp and
Admin do not read from or write to it.

The current local evaluation uses Graphiti with a dedicated generic Neo4j provider. Removal of the
former OpenSPG provider and its volumes was explicitly authorized as evaluation cleanup; no data
migration is claimed.

ADR 0041 remains authoritative for the UAT OpenSPG-specialized Neo4j provider. This decision does
not move UAT ownership, change UAT versions or widen the Reason deployment transaction.

## Consequences

The local `tidewise-infra` project contains PostgreSQL, MySQL, MinIO and Qdrant only. Application
startup no longer creates an unused Neo4j container. Reasoning experiments can change their local
graph backend without coupling that lifecycle to Tidewise AI applications.
