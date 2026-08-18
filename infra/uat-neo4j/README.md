# UAT Neo4j provider for Tidewise Reason

This directory owns the dedicated host-native UAT Neo4j infrastructure consumed by Tidewise
Reason. It is independent from application, AgentOS, Reason Server and the `tidewise-infra-uat`
MySQL/MinIO Compose release units.

## Provider contract

- Ubuntu package runtime: Neo4j `5.26.28`.
- Plugin: Neo4j Graph Data Science `2.13.4`, checksum-pinned from the official GDS distribution.
- Plugin: APOC Core `5.26.28`, bundled with the installed Neo4j release.
- Database: `neo4j`.
- HTTP: office-allowlisted ECS port `0.0.0.0:7474`.
- Bolt: office-allowlisted ECS port `0.0.0.0:7687`.
- Advertised Bolt name: `release-openspg-neo4j:7687`.

Reason maps `release-openspg-neo4j` to `host-gateway`. Office users load Neo4j Browser at
`http://123.60.99.198:7474` and enter `bolt://123.60.99.198:7687` as the connection URL. Using
`bolt://` keeps this human connection direct while Reason retains its internal `neo4j://` routing
contract. Huawei Cloud source-IP rules restrict both native ports to the office network.

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

## Existing-provider office access

After merging the reviewed infrastructure change, read the existing credential without placing it
in shell history and apply only the network contract:

```bash
read -rsp 'Neo4j administrator password: ' NEO4J_ADMIN_PASSWORD
printf '\n'
export NEO4J_ADMIN_PASSWORD
./infra/uat-neo4j/configure-office-access.sh
unset NEO4J_ADMIN_PASSWORD
```

The operation backs up `neo4j.conf`, shares both UAT locks, restarts only Neo4j and restores the
previous configuration if service, plugin or Reason-consumer verification fails. It never clears,
migrates or rewrites graph data.

After success, update the GitHub `uat` Environment Secrets:

- `tidewise-ai`: `NEO4J_PASSWORD` and `DATA_NEO4J_HEALTH_PASSWORD`.
- `tidewise-reason`: `OPENSPG_NEO4J_PASSWORD`.

Neo4j Community Edition supports one native user, so set both `NEO4J_USERNAME` and
`DATA_NEO4J_HEALTH_USERNAME` to `neo4j`, and store the same generated value in the two Tidewise AI
Secrets. The health path performs only a read-only probe. Do not print the credential in logs,
diagnostics, Issue comments or PRs.

## Verification

`verify.sh` checks exact Neo4j/APOC versions, compatible GDS, exact office-listener ports and an
authenticated Reason consumer session from the `tidewise-uat` Docker network. After configuration,
probe Browser and Bolt from the office network; probes from sources outside the Huawei allowlist
must remain denied.

This directory never deploys OpenSPG Server/KAG. The Reason repository owns that release and only
consumes this provider contract.
