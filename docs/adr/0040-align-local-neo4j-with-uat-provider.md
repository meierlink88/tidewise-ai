---
status: superseded
date: 2026-08-20
issue: 317
superseded_by: 0041
---

# Align the local Reason Neo4j provider with UAT

This decision was reversed after source and runtime verification showed that OpenSPG v0.8 relies on
one standard database per project and that generic Neo4j Community 5.26 does not preserve that
contract. ADR 0041 owns the replacement decision.

## Decision

The independently operated local Neo4j provider uses the same reviewed version contract as UAT:
Neo4j Community 5.26.28, APOC Core 5.26.28 and Graph Data Science 2.13.4. The Neo4j image and GDS
artifact are digest/checksum pinned. Reason continues to consume the stable `neo4j:7687` service
contract and remains outside the infrastructure lifecycle.

The former local provider is Neo4j Enterprise 5.25.1 and contains multiple standard databases.
Because Community supports a single standard database, local adoption is not an in-place edition
conversion. The target provider starts on explicit versioned data, log and plugin volumes. The
former `tidewise-reason_neo4j-data` and `tidewise-reason_neo4j-logs` volumes remain detached and are
not cleared or migrated implicitly.

Existing local OpenSPG project configurations may name former Enterprise databases such as
`tidewise` or `reasonsmoke`. Adoption saves those complete project configs in the local MySQL
`local_neo4j_526_project_config_backup` table, then changes only their graph-store database field to
`neo4j`. Namespaces and graph labels continue to separate project data inside the single Community
database. The backup table is retained with the legacy Neo4j volumes and is consumed by rollback.

## Rollout and failure boundary

The local upgrade command prepares checksum-reviewed plugins, fingerprints unrelated
infrastructure containers, recreates only the Neo4j service and verifies the real authenticated
provider contract. Acceptance requires the exact server edition/version, exact APOC/GDS versions,
a read/write Cypher smoke, the expected volume mounts and preservation of the legacy data volume.
The current Reason Server must also execute `graph/allLabels` successfully for every local project
whose graph-store URI targets this provider, and its bundled KAG and KNEXT CLIs must load.

If target acceptance fails, the upgrade command recreates only Neo4j with the former image and
legacy volumes. PostgreSQL, MySQL, MinIO, Qdrant, application services, AgentOS and Reason Server
are not restarted. Manual rollback remains available until explicit retirement of the legacy
volumes.

## Consequences

Local and UAT Reason evaluations no longer differ on Neo4j, APOC or GDS versions. The target local
graph starts empty; publication or test fixtures must populate it through their owning interfaces.
Legacy `reasonsmoke` or `tidewise` databases are not silently merged into the Community `neo4j`
database. Any later migration of those stores requires an explicit export/import decision.
