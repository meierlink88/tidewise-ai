# UAT OpenSPG Neo4j provider

This directory owns the dedicated UAT graph provider consumed by Tidewise Reason. The provider is
independent from application, AgentOS, Reason Server and the MySQL/MinIO infrastructure release.

## Provider contract

- Container: `tidewise-uat-openspg-neo4j`.
- Image: the same digest-pinned OpenSPG Neo4j image used locally.
- Runtime: Neo4j/DozerDB `5.25.1`, APOC `5.25.1`, OpenGDS `2.12.0`.
- Project isolation: one standard database per OpenSPG project; database name equals the lower-case
  namespace.
- Network: external `tidewise-uat`, alias `release-openspg-neo4j`.
- Browser/Bolt: office-allowlisted ECS ports `7474` and `7687`.
- Persistent volumes: `tidewise-uat-openspg-neo4j-data` and
  `tidewise-uat-openspg-neo4j-logs`.

Generic Neo4j Community 5.26 is not an approved OpenSPG provider because its single standard
database does not preserve OpenSPG project lifecycle, search or GDS isolation. A future provider
upgrade requires an exact compatible DozerDB/APOC/OpenGDS combination and the complete two-project
consumer acceptance suite.

## One-time adoption

The first switch from host-native Neo4j must run as root. Read protected credentials without adding
them to shell history:

```bash
read -rsp 'OpenSPG Neo4j password: ' OPENSPG_NEO4J_PASSWORD
printf '\n'
read -rsp 'OpenSPG MySQL root password: ' OPENSPG_MYSQL_ROOT_PASSWORD
printf '\n'
export OPENSPG_NEO4J_PASSWORD OPENSPG_MYSQL_ROOT_PASSWORD
sudo --preserve-env=OPENSPG_NEO4J_PASSWORD,OPENSPG_MYSQL_ROOT_PASSWORD \
  bash infra/uat-neo4j/adopt-reason-provider.sh
unset OPENSPG_NEO4J_PASSWORD OPENSPG_MYSQL_ROOT_PASSWORD
```

The adoption:

1. validates the existing host-native service, refuses to proceed if its `neo4j` database contains
   any nodes or relationships, and fingerprints all unrelated UAT containers;
2. pulls the exact OpenSPG image and prepares explicit volumes before downtime;
3. disables only host-native `neo4j`, leaving its files untouched for rollback;
4. starts the provider container on `tidewise-uat`;
5. backs up complete OpenSPG project configs in MySQL, changes database names to lower-case
   namespaces and creates those standard databases;
6. verifies versions, multi-database administration, restart persistence, project Graph API calls
   and a real official OpenSPG Server consumer connection.

The protected runtime environment is stored root-only at
`/opt/tidewise/neo4j-uat/runtime.env`. Do not copy it into Git, diagnostics, Issues or PRs.

Neo4j does not support opening a 5.26 store with 5.25, and OpenSPG project data cannot be safely
split out of a shared Community database by copying files. If the inventory gate finds graph data,
the adoption stops before changing either provider. Rebuild that graph from its authoritative source
into the per-project databases, or design and separately approve a logical migration before retrying.

## Rollback

If adoption fails, the script automatically removes only the candidate container, restores the
backed-up OpenSPG project configs and re-enables the untouched host-native Neo4j service. Candidate
volumes are retained and never deleted automatically.

Manual rollback uses the protected runtime file:

```bash
sudo RUNTIME_ENV=/opt/tidewise/neo4j-uat/runtime.env \
  bash infra/uat-neo4j/rollback-host-provider.sh
```

Rollback does not restart MySQL, MinIO, Qdrant, applications, AgentOS or Reason Server.

## Verification

```bash
sudo RUNTIME_ENV=/opt/tidewise/neo4j-uat/runtime.env \
  bash infra/uat-neo4j/verify.sh
```

After provider acceptance, deploy the matching `tidewise-reason` consumer release so
`release-openspg-neo4j` resolves directly through Docker DNS rather than `host-gateway`. Office
access remains bounded by the Huawei Cloud source-IP allowlist.
