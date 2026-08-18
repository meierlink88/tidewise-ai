# UAT Neo4j provider for Tidewise Reason

This directory owns the dedicated host-native UAT Neo4j infrastructure consumed by Tidewise
Reason. It is independent from application, AgentOS, Reason Server and the `tidewise-infra-uat`
MySQL/MinIO Compose release units.

## Provider contract

- Ubuntu package runtime: Neo4j `5.26.28`.
- Plugin: Neo4j Graph Data Science `2.13.4`, checksum-pinned from the official GDS distribution.
- Plugin: APOC Core `5.26.28`, bundled with the installed Neo4j release.
- Database: `neo4j`.
- HTTP: `127.0.0.1:7474` only.
- Bolt: Docker host gateway `172.17.0.1:7687` only.
- Advertised Bolt name: `release-openspg-neo4j:7687`.

Reason maps `release-openspg-neo4j` to `host-gateway`. No public security-group rule is required or
allowed for 7474 or 7687.

## One-time adoption

Issue #282 authorizes clearing the existing graph. Generate a 24-64 character URL-safe password,
then read it without putting the value in shell history and run as root:

```bash
read -rsp 'Neo4j administrator password: ' NEO4J_ADMIN_PASSWORD
printf '\n'
export NEO4J_ADMIN_PASSWORD
./infra/uat-neo4j/adopt-reason-provider.sh
unset NEO4J_ADMIN_PASSWORD
```

The script downloads and verifies GDS before downtime, shares `/opt/tidewise/uat/deploy.lock`,
fingerprints unrelated containers, and moves the complete pre-adoption data directory to
`/opt/tidewise/neo4j-uat/backups/<UTC timestamp>/data`. It restores the previous data, config and
plugins automatically if adoption fails.

After success, update the GitHub `uat` Environment Secrets:

- `tidewise-ai`: `NEO4J_PASSWORD` and `DATA_NEO4J_HEALTH_PASSWORD`.
- `tidewise-reason`: `OPENSPG_NEO4J_PASSWORD`.

Neo4j Community Edition supports one native user, so set both `NEO4J_USERNAME` and
`DATA_NEO4J_HEALTH_USERNAME` to `neo4j`, and store the same generated value in the two Tidewise AI
Secrets. The health path performs only a read-only probe. Do not print the credential in logs,
diagnostics, Issue comments or PRs.

## Verification

`verify.sh` checks exact Neo4j/APOC versions, compatible GDS, restricted listeners and an
authenticated Reason consumer session from the `tidewise-uat` Docker network. The external public
port denial must also be probed from outside the ECS after adoption.

This directory never deploys OpenSPG Server/KAG. The Reason repository owns that release and only
consumes this provider contract.
