---
status: accepted
date: 2026-08-20
issue: 319
supersedes: 0030, 0040
---

# Use the OpenSPG-specialized Neo4j provider

## Decision

Local and UAT Reason use the same digest-pinned OpenSPG Neo4j provider: Neo4j/DozerDB 5.25.1,
APOC Core 5.25.1 and OpenGDS 2.12.0. OpenSPG project isolation remains physical: project database
names are the lower-case namespaces, and OpenSPG owns their create/drop lifecycle.

Generic Neo4j Community 5.26 is not an approved OpenSPG provider. Connecting to it is insufficient:
its single standard database breaks project creation/deletion, full-database label/search paths and
GDS projection isolation. Official changelogs do not establish a broad performance improvement for
OpenSPG Graph API, vector search or PageRank that could justify that correctness loss.

The approved image is
`spg-registry.cn-hangzhou.cr.aliyuncs.com/spg/openspg-neo4j@sha256:4bc5b7f6b83d333b1d2c8f60ac145c068d77d50bca65b3a07c927f9e2a541eb9`.
Tidewise AI owns its local and UAT lifecycle. Reason consumes the stable `neo4j` local alias and
`release-openspg-neo4j` UAT alias without owning middleware.

## Rollout and failure boundary

Local Compose permanently mounts the retained OpenSPG data/log volumes; abandoned 5.26 plugin and
volume preparation is removed. UAT replaces only the dedicated host-native Neo4j provider with the
approved container on explicit persistent volumes and the shared `tidewise-uat` network. The
one-time UAT adoption backs up project configuration, creates namespace databases and retains a
recoverable host-native provider boundary until container and Reason acceptance pass.

No rollout may use unscoped Compose shutdown, `--remove-orphans`, delete a data volume, restart an
unrelated service or log protected credentials. Failure removes only the candidate Neo4j container
and restores the prior host-native service. Reason rollback never changes Neo4j state.

## Consequences

OpenSPG project lifecycle and graph/search/GDS isolation remain correct in local and UAT. The
provider is older than Neo4j 5.26 LTS and does not receive its later security fixes, so UAT ports
remain office-allowlisted and the image remains digest-pinned. A future upgrade requires an exact
DozerDB/APOC/OpenGDS combination or a licensed multi-database alternative plus the complete
two-project provider-consumer acceptance suite; version number alone is not sufficient.
