---
status: accepted
date: 2026-08-18
issue: 282
---

# Operate the dedicated UAT Neo4j as independent infrastructure

## Decision

Tidewise AI owns the host-native UAT Neo4j installation, plugins, configuration, credentials,
backup/reset procedure, network exposure and infrastructure verification. Neo4j remains an
independent infrastructure release unit: the four Tidewise application services, AgentOS and
Reason Server may consume it, but none of their deployment transactions may install, upgrade,
clear, restart or roll it back.

The adopted Reason provider contract is Neo4j 5.26 with its compatible Graph Data Science 2.13
plugin and matching APOC Core plugin. HTTP listens only on loopback. Bolt listens only on the
Docker host-gateway address and advertises the stable `release-openspg-neo4j` name used by Reason.
The `neo4j` database and its credential are infrastructure-owned state. Neo4j Community Edition
supports a single native user, so Reason and the infrastructure's read-only health probe use the
same UAT-only `neo4j` identity. This accepted UAT exception avoids introducing an unlicensed
Enterprise runtime; it does not authorize broader application use of the administrator credential.

## Adoption and failure boundary

The one-time adoption script is service-scoped to host-native Neo4j. It fingerprints all unrelated
UAT containers, downloads a checksum-pinned GDS artifact before downtime, stops Neo4j, moves the
pre-adoption data directory into a timestamped recoverable backup, installs the reviewed plugins
and configuration, initializes new credentials, then performs authenticated host and Docker-network
smokes. The existing graph may be reset because UAT Neo4j is dedicated to Tidewise Reason.

Any failure before acceptance restores the previous data, configuration and plugins and restarts
the previous Neo4j service. MySQL, MinIO, Qdrant, application and AgentOS containers are never
restarted. The generated credential enters GitHub `uat` Environment Secrets only after the new
provider passes verification.

The Neo4j 5.26 package's `neo4j-admin dbms set-initial-password` command accepts the initial
password only as a positional argument; it has no file-descriptor or standard-input option. The
one-time adoption therefore runs that command only as root on the dedicated UAT host and never with
shell tracing. This transient upstream CLI boundary is accepted for UAT; it does not permit logging
the command, copying process diagnostics, or retaining the cleartext outside GitHub Environment
Secrets. Revisit the boundary when Neo4j provides a non-argv input mechanism.

## Consequences

Reason deployment can fail closed on Neo4j authentication and `gds.version()` without owning
middleware lifecycle. The UAT public address does not expose Neo4j HTTP or Bolt. Future Neo4j
upgrades require an infrastructure change with a compatibility decision, backup and the same real
provider-consumer acceptance seam; changing the Reason deployment alone is insufficient.
